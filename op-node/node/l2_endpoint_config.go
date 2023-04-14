package node

import (
	"context"
	"errors"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/sources"
	"github.com/ethereum-optimism/optimism/op-service/client"

	"github.com/ethereum/go-ethereum/log"
	gn "github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
)

type L2EndpointSetup interface {
	// Setup a RPC client to a L2 execution engine to process rollup blocks with.
	Setup(ctx context.Context, log log.Logger, rollupCfg *rollup.Config) (cl client.RPC, rpcCfg *sources.EngineClientConfig, err error)
	Check() error
}

type L2SyncEndpointSetup interface {
	// Setup a RPC client to another L2 node to sync L2 blocks from.
	// It may return a nil client with nil error if RPC based sync is not enabled.
	Setup(ctx context.Context, log log.Logger, rollupCfg *rollup.Config) (cl client.RPC, rpcCfg *sources.SyncClientConfig, err error)
	Check() error
}

type L2EndpointConfig struct {
	L2EngineAddr string // Address of L2 Engine JSON-RPC endpoint to use (engine and eth namespace required)

	// JWT secrets for L2 Engine API authentication during HTTP or initial Websocket communication.
	// Any value for an IPC connection.
	L2EngineJWTSecret [32]byte
}

var _ L2EndpointSetup = (*L2EndpointConfig)(nil)

func (cfg *L2EndpointConfig) Check() error {
	if cfg.L2EngineAddr == "" {
		return errors.New("empty L2 Engine Address")
	}

	return nil
}

func (cfg *L2EndpointConfig) Setup(ctx context.Context, log log.Logger, rollupCfg *rollup.Config) (client.RPC, *sources.EngineClientConfig, error) {
	if err := cfg.Check(); err != nil {
		return nil, nil, err
	}
	auth := rpc.WithHTTPAuth(gn.NewJWTAuth(cfg.L2EngineJWTSecret))
	l2Node, err := client.NewRPC(ctx, log, cfg.L2EngineAddr, client.WithGethRPCOptions(auth))
	if err != nil {
		return nil, nil, err
	}

	return l2Node, sources.EngineClientDefaultConfig(rollupCfg), nil
}

// PreparedL2Endpoints enables testing with in-process pre-setup RPC connections to L2 engines
type PreparedL2Endpoints struct {
	Client client.RPC
}

func (p *PreparedL2Endpoints) Check() error {
	if p.Client == nil {
		return errors.New("client cannot be nil")
	}
	return nil
}

var _ L2EndpointSetup = (*PreparedL2Endpoints)(nil)

func (p *PreparedL2Endpoints) Setup(ctx context.Context, log log.Logger, rollupCfg *rollup.Config) (client.RPC, *sources.EngineClientConfig, error) {
	return p.Client, sources.EngineClientDefaultConfig(rollupCfg), nil
}

// L2SyncEndpointConfig contains configuration for the fallback sync endpoint
type L2SyncEndpointConfig struct {
	// Address of the L2 RPC to use for backup sync, may be empty if RPC alt-sync is disabled.
	L2NodeAddr string
	TrustRPC   bool
}

var _ L2SyncEndpointSetup = (*L2SyncEndpointConfig)(nil)

// Setup creates an RPC client to sync from.
// It will return nil without error if no sync method is configured.
func (cfg *L2SyncEndpointConfig) Setup(ctx context.Context, log log.Logger, rollupCfg *rollup.Config) (client.RPC, *sources.SyncClientConfig, error) {
	if cfg.L2NodeAddr == "" {
		return nil, nil, nil
	}
	l2Node, err := client.NewRPC(ctx, log, cfg.L2NodeAddr)
	if err != nil {
		return nil, nil, err
	}

	return l2Node, sources.SyncClientDefaultConfig(rollupCfg, cfg.TrustRPC), nil
}

func (cfg *L2SyncEndpointConfig) Check() error {
	// empty addr is valid, as it is optional.
	return nil
}

type PreparedL2SyncEndpoint struct {
	// RPC endpoint to use for syncing, may be nil if RPC alt-sync is disabled.
	Client   client.RPC
	TrustRPC bool
}

var _ L2SyncEndpointSetup = (*PreparedL2SyncEndpoint)(nil)

func (cfg *PreparedL2SyncEndpoint) Setup(ctx context.Context, log log.Logger, rollupCfg *rollup.Config) (client.RPC, *sources.SyncClientConfig, error) {
	return cfg.Client, sources.SyncClientDefaultConfig(rollupCfg, cfg.TrustRPC), nil
}

func (cfg *PreparedL2SyncEndpoint) Check() error {
	return nil
}
