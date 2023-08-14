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

func NewLatestBlockPoller(rt RoundTripper) *LatestBlockPoller {
	p := &LatestBlockPoller{
		rt: rt,
	}
	p.updateLatestBlockNumber()
	go p.pollLatestBlockNumber()
	return p
}

func (p *LatestBlockPoller) LatestBlockNumber() uint64 {
	return p.latestBlockNumber.Load()
}

func (p *LatestBlockPoller) Shutdown() {
	p.srvMu.Lock()
	defer p.srvMu.Unlock()
	p.shutdown = true
}

func (p *LatestBlockPoller) pollLatestBlockNumber() {
	ticker := time.NewTicker(2 * time.Second)
	for range ticker.C {
		p.srvMu.Lock()
		if p.shutdown {
			ticker.Stop()
			p.srvMu.Unlock()
			return
		}
		p.updateLatestBlockNumber()
		p.srvMu.Unlock()
	}
}

func (p *LatestBlockPoller) updateLatestBlockNumber() {
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
