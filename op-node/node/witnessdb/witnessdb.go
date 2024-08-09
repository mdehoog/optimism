package witnessdb

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"

	"github.com/cockroachdb/pebble"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

var (
	hashPrefix    = []byte("h") // hashPrefix + blockNum (uint64 big endian) -> blockHash
	witnessPrefix = []byte("w") // witnessPrefix + blockHash -> witness
)

type WitnessDB struct {
	m   sync.RWMutex
	log log.Logger
	db  *pebble.DB

	writeOpts *pebble.WriteOptions

	closed bool
}

func NewWitnessDB(logger log.Logger, path string) (*WitnessDB, error) {
	db, err := pebble.Open(path, &pebble.Options{})
	if err != nil {
		return nil, err
	}
	return &WitnessDB{
		log:       logger,
		db:        db,
		writeOpts: &pebble.WriteOptions{Sync: true},
	}, nil
}

func (w *WitnessDB) RecordWitness(ctx context.Context, envelope *eth.ExecutionPayloadEnvelope) error {
	if envelope.Witness == nil {
		return errors.New("witness is nil")
	}
	w.m.Lock()
	defer w.m.Unlock()
	w.log.Info("Record witness", "blockHash", envelope.ExecutionPayload.BlockHash, "blockNumber", envelope.ExecutionPayload.BlockNumber)
	batch := w.db.NewBatch()
	defer func() {
		_ = batch.Close()
	}()
	if err := batch.Set(witnessKey(envelope.ExecutionPayload.BlockHash), *envelope.Witness, w.writeOpts); err != nil {
		return fmt.Errorf("failed to set witness: %w", err)
	}
	hashes, closer, err := w.db.Get(blockHashKey(uint64(envelope.ExecutionPayload.BlockNumber)))
	if err == nil {
		defer func() {
			_ = closer.Close()
		}()
	} else if !errors.Is(err, pebble.ErrNotFound) {
		return fmt.Errorf("failed to get existing block hashes: %w", err)
	}
	hashes = append(hashes, envelope.ExecutionPayload.BlockHash.Bytes()...)
	if err := batch.Set(blockHashKey(uint64(envelope.ExecutionPayload.BlockNumber)), hashes, w.writeOpts); err != nil {
		return fmt.Errorf("failed to set block hash: %w", err)
	}
	if err := batch.Commit(w.writeOpts); err != nil {
		return fmt.Errorf("failed to commit witness record: %w", err)
	}
	return nil
}

func (w *WitnessDB) GetWitness(ctx context.Context, blockHash common.Hash) ([]byte, error) {
	w.m.RLock()
	defer w.m.RUnlock()
	witness, closer, err := w.db.Get(witnessKey(blockHash))
	if err != nil {
		return nil, fmt.Errorf("failed to get witness: %w", err)
	}
	defer func() {
		_ = closer.Close()
	}()
	c := make([]byte, len(witness))
	copy(c, witness)
	return c, nil
}

func (w *WitnessDB) PurgeOldWitnesses(ctx context.Context, beforeBlockNumber uint64) error {
	w.m.Lock()
	defer w.m.Unlock()
	iter, err := w.db.NewIter(&pebble.IterOptions{
		LowerBound: blockHashKey(0),
		UpperBound: blockHashKey(beforeBlockNumber),
	})
	if err != nil {
		return fmt.Errorf("failed to create iterator: %w", err)
	}
	defer func() {
		_ = iter.Close()
	}()
	batch := w.db.NewBatch()
	defer func() {
		_ = batch.Close()
	}()
	for iter.First(); iter.Valid(); iter.Next() {
		hashes, err := iter.ValueAndErr()
		if err != nil {
			return fmt.Errorf("failed to get block hashes iterator value: %w", err)
		}
		for i := 0; i < len(hashes); i += common.HashLength {
			if err := batch.Delete(witnessKey(common.BytesToHash(hashes[i:i+common.HashLength])), w.writeOpts); err != nil {
				return fmt.Errorf("failed to delete witness: %w", err)
			}
		}
		if err := batch.Delete(iter.Key(), w.writeOpts); err != nil {
			return fmt.Errorf("failed to delete block hashes: %w", err)
		}
	}
	if err := iter.Error(); err != nil {
		return fmt.Errorf("failed to iterate block hashes: %w", err)
	}
	if err := batch.Commit(w.writeOpts); err != nil {
		return fmt.Errorf("failed to commit purge: %w", err)
	}
	return nil
}

func (w *WitnessDB) Close() error {
	w.m.Lock()
	defer w.m.Unlock()
	if w.closed {
		// Already closed
		return nil
	}
	w.closed = true
	return w.db.Close()
}

func witnessKey(blockHash common.Hash) []byte {
	return append(witnessPrefix, blockHash.Bytes()...)
}

func blockHashKey(blockNumber uint64) []byte {
	return binary.BigEndian.AppendUint64(hashPrefix, blockNumber)
}
