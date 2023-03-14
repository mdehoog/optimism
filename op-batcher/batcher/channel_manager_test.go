package batcher_test

import (
	"io"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-batcher/batcher"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	derivetest "github.com/ethereum-optimism/optimism/op-node/rollup/derive/test"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/stretchr/testify/require"
)

// TestChannelManagerReturnsErrReorg ensures that the channel manager
// detects a reorg when it has cached L1 blocks.
func TestChannelManagerReturnsErrReorg(t *testing.T) {
	log := testlog.Logger(t, log.LvlCrit)
	m := batcher.NewChannelManager(log, batcher.ChannelConfig{})

	a := types.NewBlock(&types.Header{
		Number: big.NewInt(0),
	}, nil, nil, nil, nil)
	b := types.NewBlock(&types.Header{
		Number:     big.NewInt(1),
		ParentHash: a.Hash(),
	}, nil, nil, nil, nil)
	c := types.NewBlock(&types.Header{
		Number:     big.NewInt(2),
		ParentHash: b.Hash(),
	}, nil, nil, nil, nil)
	x := types.NewBlock(&types.Header{
		Number:     big.NewInt(2),
		ParentHash: common.Hash{0xff},
	}, nil, nil, nil, nil)

	err := m.AddL2Block(a)
	require.NoError(t, err)
	err = m.AddL2Block(b)
	require.NoError(t, err)
	err = m.AddL2Block(c)
	require.NoError(t, err)
	err = m.AddL2Block(x)
	require.ErrorIs(t, err, batcher.ErrReorg)
}

// TestChannelManagerReturnsErrReorgWhenDrained ensures that the channel manager
// detects a reorg even if it does not have any blocks inside it.
func TestChannelManagerReturnsErrReorgWhenDrained(t *testing.T) {
	log := testlog.Logger(t, log.LvlCrit)
	m := batcher.NewChannelManager(log, batcher.ChannelConfig{
		TargetFrameSize:  0,
		MaxFrameSize:     120_000,
		ApproxComprRatio: 1.0,
	})
	l1Block := types.NewBlock(&types.Header{
		BaseFee:    big.NewInt(10),
		Difficulty: common.Big0,
		Number:     big.NewInt(100),
	}, nil, nil, nil, trie.NewStackTrie(nil))
	l1InfoTx, err := derive.L1InfoDeposit(0, l1Block, eth.SystemConfig{}, false)
	require.NoError(t, err)
	txs := []*types.Transaction{types.NewTx(l1InfoTx)}

	a := types.NewBlock(&types.Header{
		Number: big.NewInt(0),
	}, txs, nil, nil, trie.NewStackTrie(nil))
	x := types.NewBlock(&types.Header{
		Number:     big.NewInt(1),
		ParentHash: common.Hash{0xff},
	}, txs, nil, nil, trie.NewStackTrie(nil))

	err = m.AddL2Block(a)
	require.NoError(t, err)

	_, err = m.TxData(eth.BlockID{})
	require.NoError(t, err)
	_, err = m.TxData(eth.BlockID{})
	require.ErrorIs(t, err, io.EOF)

	err = m.AddL2Block(x)
	require.ErrorIs(t, err, batcher.ErrReorg)
}

func TestChannelManager_TxResend(t *testing.T) {
	require := require.New(t)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	log := testlog.Logger(t, log.LvlError)
	m := batcher.NewChannelManager(log, batcher.ChannelConfig{
		TargetNumFrames:  2,
		TargetFrameSize:  1000,
		MaxFrameSize:     2000,
		ApproxComprRatio: 1.0,
		ChannelTimeout:   1000,
	})

	a, _ := derivetest.RandomL2Block(rng, 4)

	err := m.AddL2Block(a)
	require.NoError(err)

	txdata0, err := m.TxData(eth.BlockID{})
	require.NoError(err)

	// confirm one frame to keep the channel open
	m.TxConfirmed(txdata0.ID(), eth.BlockID{})

	txdata1, err := m.TxData(eth.BlockID{})
	require.NoError(err)
	txdata1bytes := txdata1.Bytes()
	data1 := make([]byte, len(txdata1bytes))
	// make sure we have a clone for later comparison
	copy(data1, txdata1bytes)

	// ensure channel is drained
	_, err = m.TxData(eth.BlockID{})
	require.ErrorIs(err, io.EOF)

	// requeue frame
	m.TxFailed(txdata1.ID())

	txdata2, err := m.TxData(eth.BlockID{})
	require.NoError(err)

	data2 := txdata2.Bytes()
	require.Equal(data2, data1)
	fs, err := derive.ParseFrames(data2)
	require.NoError(err)
	require.Len(fs, 1)
}

// TestChannelManagerCloseBeforeFirstUse ensures that the channel manager
// will not produce any frames if closed immediately.
func TestChannelManagerCloseBeforeFirstUse(t *testing.T) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	log := testlog.Logger(t, log.LvlCrit)
	m := batcher.NewChannelManager(log, batcher.ChannelConfig{
		TargetFrameSize:  0,
		MaxFrameSize:     100,
		ApproxComprRatio: 1.0,
		ChannelTimeout:   1000,
	})

	a, _ := derivetest.RandomL2Block(rng, 4)

	err := m.Close()
	require.NoError(t, err)

	err = m.AddL2Block(a)
	require.NoError(t, err)

	_, err = m.TxData(eth.BlockID{})
	require.ErrorIs(t, err, io.EOF)
}

// TestChannelManagerCloseNoPendingChannel ensures that the channel manager
// can gracefully close with no pending channels, and will not emit any new
// channel frames.
func TestChannelManagerCloseNoPendingChannel(t *testing.T) {
	log := testlog.Logger(t, log.LvlCrit)
	m := batcher.NewChannelManager(log, batcher.ChannelConfig{
		TargetFrameSize:  0,
		MaxFrameSize:     100,
		ApproxComprRatio: 1.0,
		ChannelTimeout:   1000,
	})
	lBlock := types.NewBlock(&types.Header{
		BaseFee:    big.NewInt(10),
		Difficulty: common.Big0,
		Number:     big.NewInt(100),
	}, nil, nil, nil, trie.NewStackTrie(nil))
	l1InfoTx, err := derive.L1InfoDeposit(0, lBlock, eth.SystemConfig{}, false)
	require.NoError(t, err)
	txs := []*types.Transaction{types.NewTx(l1InfoTx)}

	a := types.NewBlock(&types.Header{
		Number: big.NewInt(0),
	}, txs, nil, nil, trie.NewStackTrie(nil))

	l1InfoTx, err = derive.L1InfoDeposit(1, lBlock, eth.SystemConfig{}, false)
	require.NoError(t, err)
	txs = []*types.Transaction{types.NewTx(l1InfoTx)}

	b := types.NewBlock(&types.Header{
		Number:     big.NewInt(1),
		ParentHash: a.Hash(),
	}, txs, nil, nil, trie.NewStackTrie(nil))

	err = m.AddL2Block(a)
	require.NoError(t, err)

	txdata, err := m.TxData(eth.BlockID{})
	require.NoError(t, err)

	m.TxConfirmed(txdata.ID(), eth.BlockID{})

	_, err = m.TxData(eth.BlockID{})
	require.ErrorIs(t, err, io.EOF)

	err = m.Close()
	require.NoError(t, err)

	err = m.AddL2Block(b)
	require.NoError(t, err)

	_, err = m.TxData(eth.BlockID{})
	require.ErrorIs(t, err, io.EOF)
}

// TestChannelManagerCloseNoPendingChannel ensures that the channel manager
// can gracefully close with a pending channel, and will not produce any
// new channel frames after this point.
func TestChannelManagerClosePendingChannel(t *testing.T) {
	log := testlog.Logger(t, log.LvlCrit)
	m := batcher.NewChannelManager(log, batcher.ChannelConfig{
		TargetNumFrames:  100,
		TargetFrameSize:  1,
		MaxFrameSize:     1,
		ApproxComprRatio: 1.0,
		ChannelTimeout:   1000,
	})
	lBlock := types.NewBlock(&types.Header{
		BaseFee:    big.NewInt(10),
		Difficulty: common.Big0,
		Number:     big.NewInt(100),
	}, nil, nil, nil, trie.NewStackTrie(nil))
	l1InfoTx, err := derive.L1InfoDeposit(0, lBlock, eth.SystemConfig{}, false)
	require.NoError(t, err)
	txs := []*types.Transaction{types.NewTx(l1InfoTx)}

	a := types.NewBlock(&types.Header{
		Number: big.NewInt(0),
	}, txs, nil, nil, trie.NewStackTrie(nil))

	l1InfoTx, err = derive.L1InfoDeposit(1, lBlock, eth.SystemConfig{}, false)
	require.NoError(t, err)
	txs = []*types.Transaction{types.NewTx(l1InfoTx)}

	b := types.NewBlock(&types.Header{
		Number:     big.NewInt(1),
		ParentHash: a.Hash(),
	}, txs, nil, nil, trie.NewStackTrie(nil))

	err = m.AddL2Block(a)
	require.NoError(t, err)

	txdata, err := m.TxData(eth.BlockID{})
	require.NoError(t, err)

	m.TxConfirmed(txdata.ID(), eth.BlockID{})

	err = m.Close()
	require.NoError(t, err)

	txdata, err = m.TxData(eth.BlockID{})
	require.NoError(t, err)

	m.TxConfirmed(txdata.ID(), eth.BlockID{})

	err = m.AddL2Block(b)
	require.NoError(t, err)

	_, err = m.TxData(eth.BlockID{})
	require.ErrorIs(t, err, io.EOF)
}

// TestChannelManagerCloseAllTxsFailed ensures that the channel manager
// can gracefully close after producing transaction frames if none of these
// have successfully landed on chain.
func TestChannelManagerCloseAllTxsFailed(t *testing.T) {
	log := testlog.Logger(t, log.LvlCrit)
	m := batcher.NewChannelManager(log, batcher.ChannelConfig{
		TargetFrameSize:  0,
		MaxFrameSize:     100,
		ApproxComprRatio: 1.0,
		ChannelTimeout:   1000,
	})
	lBlock := types.NewBlock(&types.Header{
		BaseFee:    big.NewInt(10),
		Difficulty: common.Big0,
		Number:     big.NewInt(100),
	}, nil, nil, nil, trie.NewStackTrie(nil))
	l1InfoTx, err := derive.L1InfoDeposit(0, lBlock, eth.SystemConfig{}, false)
	require.NoError(t, err)
	txs := []*types.Transaction{types.NewTx(l1InfoTx)}

	a := types.NewBlock(&types.Header{
		Number: big.NewInt(0),
	}, txs, nil, nil, trie.NewStackTrie(nil))

	err = m.AddL2Block(a)
	require.NoError(t, err)

	txdata, err := m.TxData(eth.BlockID{})
	require.NoError(t, err)

	m.TxFailed(txdata.ID())

	// Show that this data will continue to be emitted as long as the transaction
	// fails and the channel manager is not closed
	txdata, err = m.TxData(eth.BlockID{})
	require.NoError(t, err)

	m.TxFailed(txdata.ID())

	err = m.Close()
	require.NoError(t, err)

	_, err = m.TxData(eth.BlockID{})
	require.ErrorIs(t, err, io.EOF)
}
