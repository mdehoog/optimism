package proxyd

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
)

type LatestBlockPoller struct {
	latestBlockNumber atomic.Uint64
	shutdown          bool
	rt                RoundTripper
	srvMu             sync.Mutex
}

type RoundTripper func(ctx context.Context, req json.RawMessage) (*RPCRes, error)

// NewLatestBlockPoller creates a new LatestBlockPoller and starts polling
// for the latest block number in a separate goroutine.
func NewLatestBlockPoller(rt RoundTripper) *LatestBlockPoller {
	p := &LatestBlockPoller{
		rt: rt,
	}
	p.update()
	go p.start()
	return p
}

// Get returns the latest block number.
func (p *LatestBlockPoller) Get() uint64 {
	return p.latestBlockNumber.Load()
}

// Shutdown stops the poller.
func (p *LatestBlockPoller) Shutdown() {
	p.srvMu.Lock()
	defer p.srvMu.Unlock()
	p.shutdown = true
}

func (p *LatestBlockPoller) start() {
	ticker := time.NewTicker(2 * time.Second)
	for range ticker.C {
		p.srvMu.Lock()
		if p.shutdown {
			ticker.Stop()
			p.srvMu.Unlock()
			return
		}
		p.update()
		p.srvMu.Unlock()
	}
}

func (p *LatestBlockPoller) update() {
	req := json.RawMessage("{\"id\":0,\"jsonrpc\":\"2.0\",\"method\":\"eth_blockNumber\"}")
	res, err := p.rt(context.Background(), req)
	if res != nil && res.IsError() {
		err = errors.New(res.Error.Error())
	}
	if err != nil {
		log.Error("error requesting latest block number", "err", err)
		return
	}
	bns := res.Result.(string)
	bn, err := hexutil.DecodeUint64(bns)
	if err != nil {
		log.Error("error decoding hex block number", "err", err)
		return
	}
	p.latestBlockNumber.Store(bn)
}
