package client

import (
	"strings"
	"time"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/urfave/cli"
)

const (
	L1NodeAddrName         = "l1"
	L1TrustRPCName         = "l1.trustrpc"
	L1RPCProviderKindName  = "l1.rpckind"
	L1RPCRateLimitName     = "l1.rpc-rate-limit"
	L1RPCMaxBatchSizeName  = "l1.rpc-max-batch-size"
	L1HTTPPollIntervalName = "l1.http-poll-interval"
)

func RequiredCLIFlags(envPrefix string) []cli.Flag {
	return []cli.Flag{
		cli.StringFlag{
			Name:     L1NodeAddrName,
			Usage:    "Address of L1 User JSON-RPC endpoint to use (eth namespace required)",
			Value:    "http://127.0.0.1:8545",
			EnvVar:   opservice.PrefixEnvVar(envPrefix, "L1_ETH_RPC"),
			Required: true,
		},
	}
}

func OptionalCLIFlags(envPrefix string) []cli.Flag {
	return []cli.Flag{
		cli.BoolFlag{
			Name:   L1TrustRPCName,
			Usage:  "Trust the L1 RPC, sync faster at risk of malicious/buggy RPC providing bad or inconsistent L1 data",
			EnvVar: opservice.PrefixEnvVar(envPrefix, "L1_TRUST_RPC"),
		},
		cli.GenericFlag{
			Name: L1RPCProviderKindName,
			Usage: "The kind of RPC provider, used to inform optimal transactions receipts fetching, and thus reduce costs. Valid options: " +
				opservice.EnumString[RPCProviderKind](RPCProviderKinds),
			EnvVar: opservice.PrefixEnvVar(envPrefix, "L1_RPC_KIND"),
			Value: func() *RPCProviderKind {
				out := RPCKindBasic
				return &out
			}(),
		},
		cli.Float64Flag{
			Name:   L1RPCRateLimitName,
			Usage:  "Optional self-imposed global rate-limit on L1 RPC requests, specified in requests / second. Disabled if set to 0.",
			EnvVar: opservice.PrefixEnvVar(envPrefix, "L1_RPC_RATE_LIMIT"),
			Value:  0,
		},
		cli.IntFlag{
			Name:   L1RPCMaxBatchSizeName,
			Usage:  "Maximum number of RPC requests to bundle, e.g. during L1 blocks receipt fetching. The L1 RPC rate limit counts this as N items, but allows it to burst at once.",
			EnvVar: opservice.PrefixEnvVar(envPrefix, "L1_RPC_MAX_BATCH_SIZE"),
			Value:  20,
		},
		cli.DurationFlag{
			Name:   L1HTTPPollIntervalName,
			Usage:  "Polling interval for latest-block subscription when using an HTTP RPC provider. Ignored for other types of RPC endpoints.",
			EnvVar: opservice.PrefixEnvVar(envPrefix, "L1_HTTP_POLL_INTERVAL"),
			Value:  time.Second * 12,
		},
	}
}

func NewL1EndpointConfig(ctx *cli.Context) *L1EndpointConfig {
	return &L1EndpointConfig{
		NodeAddr:         ctx.GlobalString(L1NodeAddrName),
		TrustRPC:         ctx.GlobalBool(L1TrustRPCName),
		RPCKind:          RPCProviderKind(strings.ToLower(ctx.GlobalString(L1RPCProviderKindName))),
		RateLimit:        ctx.GlobalFloat64(L1RPCRateLimitName),
		BatchSize:        ctx.GlobalInt(L1RPCMaxBatchSizeName),
		HttpPollInterval: ctx.Duration(L1HTTPPollIntervalName),
	}
}
