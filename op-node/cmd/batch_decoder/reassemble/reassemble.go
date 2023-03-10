package reassemble

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"sort"

	"github.com/ethereum-optimism/optimism/op-node/cmd/batch_decoder/fetch"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum/go-ethereum/common"
)

type ChannelWithMeta struct {
	ID            derive.ChannelID    `json:"id"`
	SkippedFrames []FrameWithMetadata `json:"skipped_frames"`
	IsReady       bool                `json:"is_ready"`
	Frames        []FrameWithMetadata `json:"frames"`
	Batches       []*derive.BatchData `json:"batches"`
}

type FrameWithMetadata struct {
	TxHash         common.Hash  `json:"transaction_hash"`
	InclusionBlock uint64       `json:"inclusion_block"`
	Timestamp      uint64       `json:"timestamp"`
	Frame          derive.Frame `json:"frame"`
}

type Config struct {
	BatchInbox   common.Address
	InDirectory  string
	OutDirectory string
}

// Channels loads all transactions from the given input directory that are submitted to the
// specified batch inbox and then re-assembles all channels & writes the re-assembled channels
// to the out directory.
func Channels(config Config) {
	if err := os.MkdirAll(config.OutDirectory, 0750); err != nil {
		log.Fatal(err)
	}
	txns := loadTransactions(config.InDirectory, config.BatchInbox)
	// Sort first by block number then by transaction index inside the block number range.
	// This is to match the order they are processed in derivation.
	sort.Slice(txns, func(i, j int) bool {
		if txns[i].BlockNumber == txns[j].BlockNumber {
			return txns[i].TxIndex < txns[j].TxIndex
		} else {
			return txns[i].BlockNumber < txns[j].BlockNumber
		}

	})
	frames := transactionsToFrames(txns)
	framesByChannel := make(map[derive.ChannelID][]FrameWithMetadata)
	for _, frame := range frames {
		framesByChannel[frame.Frame.ID] = append(framesByChannel[frame.Frame.ID], frame)
	}
	for id, frames := range framesByChannel {
		ch := processFrames(id, frames)
		filename := path.Join(config.OutDirectory, fmt.Sprintf("%s.json", id.String()))
		if err := writeChannel(ch, filename); err != nil {
			log.Fatal(err)
		}
	}
}

func writeChannel(ch ChannelWithMeta, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	enc := json.NewEncoder(file)
	return enc.Encode(ch)
}

func processFrames(id derive.ChannelID, frames []FrameWithMetadata) ChannelWithMeta {
	sort.Slice(frames, func(i, j int) bool {
		return frames[i].Frame.FrameNumber < frames[j].Frame.FrameNumber
	})

	var fullData []byte
	var lastData []byte
	lastIndex := -1
	ready := false
	for _, frame := range frames {
		if int(frame.Frame.FrameNumber) == lastIndex+1 {
			// next frame found
			if ready {
				log.Fatal("Additional frame found after last frame")
			}
			fullData = append(fullData, frame.Frame.Data...)
			lastIndex++
		} else if int(frame.Frame.FrameNumber) == lastIndex {
			// duplicate frame found
			if !bytes.Equal(lastData, frame.Frame.Data) {
				log.Fatal("Duplicate frame number has different data")
			}
		} else {
			// next frame not found
			log.Fatalf("Missing frame %d", lastIndex+1)
		}
		lastData = frame.Frame.Data
		ready = ready || frame.Frame.IsLast
	}

	if !ready {
		fmt.Printf("Found channel that was not closed: %v\n", id.String())
	}

	batchReader, err := derive.BatchReader(bytes.NewBuffer(fullData), eth.L1BlockRef{Number: frames[0].InclusionBlock})
	if err != nil {
		panic(err)
	}

	var batches []*derive.BatchData
	for {
		batch, err := batchReader()
		if err == io.EOF {
			break
		} else if err == io.ErrUnexpectedEOF && !ready {
			break
		} else if err != nil {
			log.Fatalf("WARNING: unexpected error reading batches: %v\n", err)
		}
		batches = append(batches, batch.Batch)
	}

	sort.Slice(batches, func(i, j int) bool {
		return batches[i].Timestamp < batches[j].Timestamp
	})

	return ChannelWithMeta{
		ID:            id,
		Frames:        frames,
		SkippedFrames: nil,
		IsReady:       ready,
		Batches:       batches,
	}
}

func transactionsToFrames(txns []fetch.TransactionWithMeta) []FrameWithMetadata {
	var out []FrameWithMetadata
	for _, tx := range txns {
		for _, frame := range tx.Frames {
			fm := FrameWithMetadata{
				TxHash:         tx.Tx.Hash(),
				InclusionBlock: tx.BlockNumber,
				Timestamp:      tx.BlockTime,
				Frame:          frame,
			}
			out = append(out, fm)
		}
	}
	return out
}

func loadTransactions(dir string, inbox common.Address) []fetch.TransactionWithMeta {
	files, err := os.ReadDir(dir)
	if err != nil {
		log.Fatal(err)
	}
	var out []fetch.TransactionWithMeta
	for _, file := range files {
		f := path.Join(dir, file.Name())
		txm := loadTransactionsFile(f)
		if txm.InboxAddr == inbox && txm.ValidSender {
			out = append(out, txm)
		}
	}
	return out
}

func loadTransactionsFile(file string) fetch.TransactionWithMeta {
	f, err := os.Open(file)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	var txm fetch.TransactionWithMeta
	if err := dec.Decode(&txm); err != nil {
		log.Fatalf("Failed to decode %v. Err: %v\n", file, err)
	}
	return txm
}
