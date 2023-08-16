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
	rt       RoundTripper
	bn       atomic.Uint64
	shutdown bool
	mutex    sync.Mutex
}

type RoundTripper func(ctx context.Context, req json.RawMessage) (*RPCRes, error)

// NewLatestBlockPoller creates a new LatestBlockPoller and starts polling
// for the latest block number in a separate goroutine.
func NewLatestBlockPoller(pollingInterval time.Duration, rt RoundTripper) *LatestBlockPoller {
	p := &LatestBlockPoller{
		rt: rt,
	}
	p.poll()
	go p.start(pollingInterval)
	return p
}

// Get returns the latest block number.
func (p *LatestBlockPoller) Get() uint64 {
	return p.bn.Load()
}

// Shutdown stops the poller.
func (p *LatestBlockPoller) Shutdown() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.shutdown = true
}

func (p *LatestBlockPoller) start(pollingInterval time.Duration) {
	ticker := time.NewTicker(pollingInterval)
	for range ticker.C {
		p.mutex.Lock()
		if p.shutdown {
			ticker.Stop()
			p.mutex.Unlock()
			return
		}
		p.poll()
		p.mutex.Unlock()
	}
}

func (p *LatestBlockPoller) poll() {
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
	p.bn.Store(bn)
}
