package batcher

import (
	"fmt"
	"strings"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/log"
)

// TxData represents the data for a single transaction.
//
// Note: The batcher currently sends exactly one frame per transaction. This
// might change in the future to allow for multiple frames from possibly
// different channels.
type TxData struct {
	Frames []FrameData
	AsBlob bool // indicates whether this should be sent as blob
}

func singleFrameTxData(frame FrameData) TxData {
	return TxData{Frames: []FrameData{frame}}
}

// ID returns the id for this transaction data. Its String() can be used as a map key.
func (td *TxData) ID() TxID {
	id := make(txID, 0, len(td.Frames))
	for _, f := range td.Frames {
		id = append(id, f.ID)
	}
	return id
}

// CallData returns the transaction data as calldata.
// It's a version byte (0) followed by the concatenated frames for this transaction.
func (td *TxData) CallData() []byte {
	data := make([]byte, 1, 1+td.Len())
	data[0] = derive.DerivationVersion0
	for _, f := range td.Frames {
		data = append(data, f.Data...)
	}
	return data
}

func (td *TxData) Blobs() ([]*eth.Blob, error) {
	blobs := make([]*eth.Blob, 0, len(td.Frames))
	for _, f := range td.Frames {
		var blob eth.Blob
		if err := blob.FromData(append([]byte{derive.DerivationVersion0}, f.Data...)); err != nil {
			return nil, err
		}
		blobs = append(blobs, &blob)
	}
	return blobs, nil
}

// Len returns the sum of all the sizes of data in all frames.
// Len only counts the data itself and doesn't account for the version byte(s).
func (td *TxData) Len() (l int) {
	for _, f := range td.Frames {
		l += len(f.Data)
	}
	return l
}

// TxID is an opaque identifier for a transaction.
// Its String() can be used for comparisons and works as a map key.
type TxID interface {
	fmt.Stringer
	log.TerminalStringer
}

type txID []FrameID

func (id txID) String() string {
	return id.string(func(id derive.ChannelID) string { return id.String() })
}

// TerminalString implements log.TerminalStringer, formatting a string for console
// output during logging.
func (id txID) TerminalString() string {
	return id.string(func(id derive.ChannelID) string { return id.TerminalString() })
}

func (id txID) string(chIDStringer func(id derive.ChannelID) string) string {
	var (
		sb      strings.Builder
		curChID derive.ChannelID
	)
	for _, f := range id {
		if f.ChID == curChID {
			sb.WriteString(fmt.Sprintf("+%d", f.FrameNumber))
		} else {
			if curChID != (derive.ChannelID{}) {
				sb.WriteString("|")
			}
			curChID = f.ChID
			sb.WriteString(fmt.Sprintf("%s:%d", chIDStringer(f.ChID), f.FrameNumber))
		}
	}
	return sb.String()
}
