package op_e2e

import (
	"io/fs"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
)

func TestGasPriceOracle(t *testing.T) {
	backend := backends.NewSimulatedBackend(map[common.Address]core.GenesisAccount{
		predeploys.GasPriceOracleAddr: {
			Code:    common.FromHex(bindings.GasPriceOracleDeployedBin),
			Balance: big.NewInt(0),
			Storage: map[common.Hash]common.Hash{
				common.HexToHash("0x0"): common.HexToHash("0x0101"), // isEcotone = true, isFjord = true
			},
		},
		predeploys.L1BlockAddr: {
			Code:    common.FromHex(bindings.L1BlockDeployedBin),
			Balance: big.NewInt(0),
		},
	}, math.MaxUint64)

	caller, err := bindings.NewGasPriceOracleCaller(predeploys.GasPriceOracleAddr, backend)
	assert.NoError(t, err)

	atLeastOnce := false
	err = filepath.WalkDir("../specs", func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		used, err := caller.GetL1Fee(&bind.CallOpts{}, b)
		if err != nil {
			return err
		}

		var (
			intercept          int64 = -27_321_890
			fastlzCoef         int64 = 1_031_462
			uncompressedTxCoef int64 = -88_664

			l1BaseFeeScalar uint64 = 11_111
			l1BlobFeeScalar uint64 = 1_250_000
		)

		l1BaseFee, err := caller.BaseFee(&bind.CallOpts{})

		if err != nil {
			return err
		}

		l1BaseFeeScaled := l1BaseFeeScalar * l1BaseFee.Uint64() * 16
		l1BlobBaseFee, err := caller.BlobBaseFee(&bind.CallOpts{})

		if err != nil {
			return err
		}
		l1BlobFeeScaled := l1BlobFeeScalar * l1BlobBaseFee.Uint64()
		l1FeeScaled := l1BaseFeeScaled + l1BlobFeeScaled
		fastLzLength := types.FlzCompressLen(b)
		expected := uint64(((intercept + fastlzCoef*int64(fastLzLength) + uncompressedTxCoef*int64(len(b)+64)) * int64(l1FeeScaled)) / 1e12)

		assert.Equal(t, used.Uint64(), uint64(expected), path)

		atLeastOnce = true
		return nil
	})
	assert.NoError(t, err)
	assert.True(t, atLeastOnce)
}
