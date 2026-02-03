// Copyright 2024 The op-rskgo Authors
// This file is part of the op-rskgo library.
//
// The op-rskgo library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The op-rskgo library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the op-rskgo library. If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/remasc"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

var (
	// ErrGasPriceBelowMinimum is returned when gas price is below RSK minimum.
	ErrGasPriceBelowMinimum = errors.New("gas price below minimum")

	// ErrRemascFailed is returned when REMASC transaction fails.
	ErrRemascFailed = errors.New("REMASC transaction failed")
)

// RSKStateTransitionHooks provides hooks for RSK-specific state transition logic.
type RSKStateTransitionHooks struct {
	config       *params.ChainConfig
	remasc       *remasc.Remasc
	paidFeesPool *big.Int // Accumulated fees in the current block
	blockNumber  uint64
	coinbase     common.Address
}

// NewRSKStateTransitionHooks creates new RSK state transition hooks.
func NewRSKStateTransitionHooks(config *params.ChainConfig, blockNumber uint64, coinbase common.Address) *RSKStateTransitionHooks {
	var remascInstance *remasc.Remasc
	if config.IsRSK() && config.RSK != nil {
		remascConfig := &remasc.Config{
			MaturityPeriod:    config.RSK.MaturityPeriod,
			SyntheticSpan:     config.RSK.SyntheticSpan,
			RSKLabsCut:        config.RSK.RSKLabsCut,
			FederationCut:     config.RSK.FederationCut,
			PublishersCut:     config.RSK.PublishersCut,
			BurnedCut:         config.RSK.BurnedCut,
			FederationAddress: config.RSK.FederationAddress,
		}
		remascInstance = remasc.New(remascConfig)
	}

	return &RSKStateTransitionHooks{
		config:       config,
		remasc:       remascInstance,
		paidFeesPool: new(big.Int),
		blockNumber:  blockNumber,
		coinbase:     coinbase,
	}
}

// ValidateGasPrice validates that the gas price meets RSK minimum requirements.
func (h *RSKStateTransitionHooks) ValidateGasPrice(gasPrice *big.Int, minimumGasPrice *big.Int) error {
	if !h.config.IsRSK() {
		return nil
	}

	// If minimum gas price is set and transaction gas price is below it
	if minimumGasPrice != nil && minimumGasPrice.Sign() > 0 {
		if gasPrice.Cmp(minimumGasPrice) < 0 {
			return ErrGasPriceBelowMinimum
		}
	}

	return nil
}

// AccumulateFees adds transaction fees to the block's fee pool.
// In RSK, fees are not paid directly to miners but accumulated for REMASC distribution.
func (h *RSKStateTransitionHooks) AccumulateFees(fee *big.Int) {
	if !h.config.IsRSK() || fee == nil || fee.Sign() <= 0 {
		return
	}
	h.paidFeesPool.Add(h.paidFeesPool, fee)
}

// GetPaidFees returns the total fees paid in the current block.
func (h *RSKStateTransitionHooks) GetPaidFees() *big.Int {
	return new(big.Int).Set(h.paidFeesPool)
}

// ProcessEndOfBlock processes end-of-block operations for RSK.
// This includes calling REMASC to handle reward distribution.
func (h *RSKStateTransitionHooks) ProcessEndOfBlock(
	stateDB vm.StateDB,
	header *types.Header,
	uncles []*types.Header,
) error {
	if !h.config.IsRSK() {
		return nil
	}

	// Create REMASC-compatible StateDB wrapper
	remascStateDB := &remascStateDBWrapper{stateDB: stateDB}

	// Process block rewards through REMASC
	if h.remasc != nil {
		if err := h.remasc.ProcessBlock(remascStateDB, header, uncles, h.paidFeesPool); err != nil {
			return err
		}
	}

	return nil
}

// InjectRemascTransaction creates the synthetic REMASC transaction for end of block.
// In RSK, each block contains a final "REMASC" transaction that triggers reward distribution.
func (h *RSKStateTransitionHooks) InjectRemascTransaction() *types.Transaction {
	if !h.config.IsRSK() {
		return nil
	}

	// Create REMASC call data
	// The REMASC precompile at 0x1000002 is called with minimal data
	remascAddr := params.RSKRemascAddress

	// Create a transaction calling the REMASC precompile
	// This is a synthetic transaction that doesn't require signing
	txData := &types.LegacyTx{
		Nonce:    0, // REMASC tx always has nonce 0
		GasPrice: big.NewInt(0),
		Gas:      uint64(1000000), // High gas limit for REMASC processing
		To:       &remascAddr,
		Value:    big.NewInt(0),
		Data:     []byte{}, // Empty data - REMASC handles based on block context
	}

	return types.NewTx(txData)
}

// CalculateMinimumGasPrice calculates the minimum gas price for the next block.
// RSK uses a dynamic minimum gas price based on block fullness.
func (h *RSKStateTransitionHooks) CalculateMinimumGasPrice(
	parentMinGasPrice *big.Int,
	parentGasUsed uint64,
	parentGasLimit uint64,
) *big.Int {
	if !h.config.IsRSK() {
		return nil
	}

	if parentMinGasPrice == nil {
		// Default minimum gas price
		return big.NewInt(59240000) // ~0.059 Gwei
	}

	// RSK's minimum gas price algorithm
	// Increases if blocks are full, decreases if blocks are empty

	// Calculate fill ratio (0-100)
	fillRatio := uint64(0)
	if parentGasLimit > 0 {
		fillRatio = (parentGasUsed * 100) / parentGasLimit
	}

	minGasPrice := new(big.Int).Set(parentMinGasPrice)

	// Adjust based on block fullness
	if fillRatio > 67 {
		// Block is >67% full, increase minimum gas price by 1%
		increase := new(big.Int).Div(minGasPrice, big.NewInt(100))
		minGasPrice.Add(minGasPrice, increase)
	} else if fillRatio < 33 {
		// Block is <33% full, decrease minimum gas price by 1%
		decrease := new(big.Int).Div(minGasPrice, big.NewInt(100))
		minGasPrice.Sub(minGasPrice, decrease)
	}

	// Ensure minimum doesn't go below absolute floor
	absoluteFloor := big.NewInt(1000000) // 0.001 Gwei
	if minGasPrice.Cmp(absoluteFloor) < 0 {
		minGasPrice.Set(absoluteFloor)
	}

	return minGasPrice
}

// IsRemascTransaction checks if a transaction is a REMASC synthetic transaction.
func IsRemascTransaction(tx *types.Transaction) bool {
	if tx == nil || tx.To() == nil {
		return false
	}
	return *tx.To() == params.RSKRemascAddress && tx.GasPrice().Sign() == 0
}

// RSKFeeDistribution handles the distribution of fees for RSK.
type RSKFeeDistribution struct {
	MinerReward     *big.Int // Reward to miner (after maturity)
	RSKLabsReward   *big.Int // RSK Labs cut
	FederationReward *big.Int // Federation cut (if any)
	BurnedAmount    *big.Int // Burned amount
	PooledAmount    *big.Int // Amount added to reward pool
}

// CalculateFeeDistribution calculates how fees should be distributed.
func (h *RSKStateTransitionHooks) CalculateFeeDistribution(totalFees *big.Int) *RSKFeeDistribution {
	if !h.config.IsRSK() || h.config.RSK == nil {
		return nil
	}

	distribution := &RSKFeeDistribution{
		MinerReward:     new(big.Int),
		RSKLabsReward:   new(big.Int),
		FederationReward: new(big.Int),
		BurnedAmount:    new(big.Int),
		PooledAmount:    new(big.Int),
	}

	if totalFees == nil || totalFees.Sign() <= 0 {
		return distribution
	}

	// RSK Labs cut
	rskLabsCut := new(big.Int).Mul(totalFees, big.NewInt(int64(h.config.RSK.RSKLabsCut)))
	rskLabsCut.Div(rskLabsCut, big.NewInt(10000))
	distribution.RSKLabsReward = rskLabsCut

	// Federation cut
	fedCut := new(big.Int).Mul(totalFees, big.NewInt(int64(h.config.RSK.FederationCut)))
	fedCut.Div(fedCut, big.NewInt(10000))
	distribution.FederationReward = fedCut

	// Burned amount
	burned := new(big.Int).Mul(totalFees, big.NewInt(int64(h.config.RSK.BurnedCut)))
	burned.Div(burned, big.NewInt(10000))
	distribution.BurnedAmount = burned

	// Remainder goes to pool for miner rewards (after maturity)
	pooled := new(big.Int).Set(totalFees)
	pooled.Sub(pooled, rskLabsCut)
	pooled.Sub(pooled, fedCut)
	pooled.Sub(pooled, burned)
	distribution.PooledAmount = pooled

	return distribution
}

// remascStateDBWrapper wraps vm.StateDB to implement remasc.StateDB interface.
type remascStateDBWrapper struct {
	stateDB vm.StateDB
}

func (w *remascStateDBWrapper) GetBalance(addr common.Address) *big.Int {
	return w.stateDB.GetBalance(addr).ToBig()
}

func (w *remascStateDBWrapper) AddBalance(addr common.Address, amount *big.Int) {
	if amount != nil && amount.Sign() > 0 {
		u256Amount, _ := uint256.FromBig(amount)
		w.stateDB.AddBalance(addr, u256Amount, tracing.BalanceChangeUnspecified)
	}
}

func (w *remascStateDBWrapper) SubBalance(addr common.Address, amount *big.Int) {
	if amount != nil && amount.Sign() > 0 {
		u256Amount, _ := uint256.FromBig(amount)
		w.stateDB.SubBalance(addr, u256Amount, tracing.BalanceChangeUnspecified)
	}
}
