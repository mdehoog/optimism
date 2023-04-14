package client

import "fmt"

// Cost break-down sources:
// Alchemy: https://docs.alchemy.com/reference/compute-units
// QuickNode: https://www.quicknode.com/docs/ethereum/api_credits
// Infura: no pricing table available.
//
// Receipts are encoded the same everywhere:
//
//     blockHash, blockNumber, transactionIndex, transactionHash, from, to, cumulativeGasUsed, gasUsed,
//     contractAddress, logs, logsBloom, status, effectiveGasPrice, type.
//
// Note that Alchemy/Geth still have a "root" field for legacy reasons,
// but ethereum does not compute state-roots per tx anymore, so quicknode and others do not serve this data.

// RPCProviderKind identifies an RPC provider, used to hint at the optimal receipt fetching approach.
type RPCProviderKind string

const (
	RPCKindAlchemy    RPCProviderKind = "alchemy"
	RPCKindQuickNode  RPCProviderKind = "quicknode"
	RPCKindInfura     RPCProviderKind = "infura"
	RPCKindParity     RPCProviderKind = "parity"
	RPCKindNethermind RPCProviderKind = "nethermind"
	RPCKindDebugGeth  RPCProviderKind = "debug_geth"
	RPCKindErigon     RPCProviderKind = "erigon"
	RPCKindBasic      RPCProviderKind = "basic" // try only the standard most basic receipt fetching
	RPCKindAny        RPCProviderKind = "any"   // try any method available
)

var RPCProviderKinds = []RPCProviderKind{
	RPCKindAlchemy,
	RPCKindQuickNode,
	RPCKindInfura,
	RPCKindParity,
	RPCKindNethermind,
	RPCKindDebugGeth,
	RPCKindErigon,
	RPCKindBasic,
	RPCKindAny,
}

func (kind RPCProviderKind) String() string {
	return string(kind)
}

func (kind *RPCProviderKind) Set(value string) error {
	if !ValidRPCProviderKind(RPCProviderKind(value)) {
		return fmt.Errorf("unknown rpc kind: %q", value)
	}
	*kind = RPCProviderKind(value)
	return nil
}

func ValidRPCProviderKind(value RPCProviderKind) bool {
	for _, k := range RPCProviderKinds {
		if k == value {
			return true
		}
	}
	return false
}
