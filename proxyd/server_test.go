package proxyd

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/eth/filters"
	"github.com/stretchr/testify/require"
)

func TestIsAllowedBlockRange(t *testing.T) {
	newReq := func(method string, filter filters.FilterCriteria) *RPCReq {
		type output struct {
			BlockHash *common.Hash `json:"blockHash"`
			FromBlock *string      `json:"fromBlock"`
			ToBlock   *string      `json:"toBlock"`
		}
		text := func(i *big.Int) *string {
			if i == nil {
				return nil
			}
			f := "0x" + i.Text(16)
			return &f
		}
		msg, err := json.Marshal([]interface{}{output{
			BlockHash: filter.BlockHash,
			FromBlock: text(filter.FromBlock),
			ToBlock:   text(filter.ToBlock),
		}})
		require.NoError(t, err)
		return &RPCReq{
			Method: method,
			Params: msg,
		}
	}
	tests := []struct {
		name          string
		maxBlockRange uint64
		latestBlock   uint64
		req           *RPCReq
		result        bool
	}{
		{
			name:          "non range request",
			maxBlockRange: 2000,
			latestBlock:   123456,
			req:           newReq("eth_chainId", filters.FilterCriteria{}),
			result:        true,
		},
		{
			name:          "block range disabled",
			maxBlockRange: 0,
			latestBlock:   123456,
			req:           newReq("eth_getLogs", filters.FilterCriteria{FromBlock: big.NewInt(0)}),
			result:        true,
		},
		{
			name:          "block hash set",
			maxBlockRange: 2000,
			latestBlock:   123456,
			req:           newReq("eth_getLogs", filters.FilterCriteria{BlockHash: &common.Hash{1}}),
			result:        true,
		},
		{
			name:          "within range",
			maxBlockRange: 2000,
			latestBlock:   123456,
			req:           newReq("eth_getLogs", filters.FilterCriteria{FromBlock: big.NewInt(100), ToBlock: big.NewInt(1500)}),
			result:        true,
		},
		{
			name:          "outside range",
			maxBlockRange: 2000,
			latestBlock:   123456,
			req:           newReq("eth_getLogs", filters.FilterCriteria{FromBlock: big.NewInt(100), ToBlock: big.NewInt(3000)}),
			result:        false,
		},
		{
			name:          "invalid request",
			maxBlockRange: 2000,
			latestBlock:   123456,
			req:           &RPCReq{Method: "eth_getLogs"},
			result:        true,
		},
		{
			name:          "default from",
			maxBlockRange: 2000,
			latestBlock:   123456,
			req:           newReq("eth_getLogs", filters.FilterCriteria{ToBlock: big.NewInt(3000)}),
			result:        true,
		},
		{
			name:          "default to",
			maxBlockRange: 2000,
			latestBlock:   123456,
			req:           newReq("eth_getLogs", filters.FilterCriteria{FromBlock: big.NewInt(100)}),
			result:        false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			server := &Server{
				maxBlockRange:     tt.maxBlockRange,
				latestBlockPoller: &LatestBlockPoller{},
			}
			server.latestBlockPoller.bn.Store(tt.latestBlock)
			got := server.isAllowedBlockRange(tt.req)
			require.Equal(t, tt.result, got)
		})
	}
}
