package proxyd

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

type roundTripperReturn struct {
	r *RPCRes
	e error
}

func TestLatestBlockPoller(t *testing.T) {
	tests := []struct {
		name         string
		res          []roundTripperReturn
		blockNumbers []uint64
	}{
		{
			name: "success",
			res: []roundTripperReturn{
				{r: &RPCRes{Result: hexutil.EncodeUint64(10)}},
				{r: &RPCRes{Result: hexutil.EncodeUint64(12)}},
			},
			blockNumbers: []uint64{10, 12},
		},
		{
			name: "errors",
			res: []roundTripperReturn{
				{e: errors.New("error")},
				{r: &RPCRes{Result: hexutil.EncodeUint64(10)}},
				{e: errors.New("error")},
				{r: &RPCRes{Result: hexutil.EncodeUint64(12)}},
			},
			blockNumbers: []uint64{0, 10, 10, 12},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			i := -1
			rt := func(ctx context.Context, req json.RawMessage) (*RPCRes, error) {
				i++
				return tt.res[i].r, tt.res[i].e
			}
			pollingInterval := 2 * time.Second
			bp := NewLatestBlockPoller(pollingInterval, rt)
			time.Sleep(50 * time.Millisecond)
			for i, bn := range tt.blockNumbers {
				if i != 0 {
					time.Sleep(pollingInterval + 50*time.Millisecond)
				}
				got := bp.Get()
				require.Equal(t, bn, got)
			}
			bp.Shutdown()
		})
	}

}
