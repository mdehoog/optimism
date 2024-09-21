package batcher

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTxID_String(t *testing.T) {
	for _, test := range []struct {
		desc   string
		id     TxID
		expStr string
	}{
		{
			desc:   "empty",
			id:     txID{},
			expStr: "",
		},
		{
			desc:   "nil",
			id:     nil,
			expStr: "",
		},
		{
			desc: "single",
			id: txID{{
				ChID:        [16]byte{0: 0xca, 15: 0xaf},
				FrameNumber: 42,
			}},
			expStr: "ca0000000000000000000000000000af:42",
		},
		{
			desc: "multi",
			id: txID{
				{
					ChID:        [16]byte{0: 0xca, 15: 0xaf},
					FrameNumber: 42,
				},
				{
					ChID:        [16]byte{0: 0xca, 15: 0xaf},
					FrameNumber: 33,
				},
				{
					ChID:        [16]byte{0: 0xbe, 15: 0xef},
					FrameNumber: 0,
				},
				{
					ChID:        [16]byte{0: 0xbe, 15: 0xef},
					FrameNumber: 128,
				},
			},
			expStr: "ca0000000000000000000000000000af:42+33|be0000000000000000000000000000ef:0+128",
		},
	} {
		t.Run(test.desc, func(t *testing.T) {
			require.Equal(t, test.expStr, test.id.String())
		})
	}
}
