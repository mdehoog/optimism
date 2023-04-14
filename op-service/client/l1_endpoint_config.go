package client

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

type L1EndpointConfig struct {
	NodeAddr string // Address of L1 User JSON-RPC endpoint to use (eth namespace required)

	// TrustRPC: if we trust the L1 RPC we do not have to validate L1 response contents like headers
	// against block hashes, or cached transaction sender addresses.
	// Thus we can sync faster at the risk of the source RPC being wrong.
	TrustRPC bool

	// RPCKind identifies the RPC provider kind that serves the RPC,
	// to inform the optimal usage of the RPC for transaction receipts fetching.
	RPCKind RPCProviderKind

	// RateLimit specifies a self-imposed rate-limit on L1 requests. 0 is no rate-limit.
	RateLimit float64

	// BatchSize specifies the maximum batch-size, which also applies as L1 rate-limit burst amount (if set).
	BatchSize int

	// HttpPollInterval specifies the interval between polling for the latest L1 block,
	// when the RPC is detected to be an HTTP type.
	// It is recommended to use websockets or IPC for efficient following of the changing block.
	// Setting this to 0 disables polling.
	HttpPollInterval time.Duration
}

func (cfg *L1EndpointConfig) Check() error {
	if cfg.BatchSize < 1 || cfg.BatchSize > 500 {
		return fmt.Errorf("batch size is invalid or unreasonable: %d", cfg.BatchSize)
	}
	if cfg.RateLimit < 0 {
		return fmt.Errorf("rate limit cannot be negative")
	}
	return nil
}

func (cfg *L1EndpointConfig) Setup(ctx context.Context, log log.Logger) (RPC, error) {
	opts := []RPCOption{
		WithHttpPollInterval(cfg.HttpPollInterval),
		WithDialBackoff(10),
	}
	if cfg.RateLimit != 0 {
		opts = append(opts, WithRateLimit(cfg.RateLimit, cfg.BatchSize))
	}

	l1Node, err := NewRPC(ctx, log, cfg.NodeAddr, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to dial L1 address (%s): %w", cfg.NodeAddr, err)
	}
	return l1Node, nil
}

// PreparedL1Endpoint enables testing with an in-process pre-setup RPC connection to L1
type PreparedL1Endpoint struct {
	Client   RPC
	TrustRPC bool
	RPCKind  RPCProviderKind
}

func (p *PreparedL1Endpoint) Setup(ctx context.Context, log log.Logger) (RPC, error) {
	return p.Client, nil
}

func (p *PreparedL1Endpoint) Check() error {
	if p.Client == nil {
		return errors.New("rpc client cannot be nil")
	}

	return nil
}
