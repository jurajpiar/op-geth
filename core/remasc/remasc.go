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

package remasc

import (
	"errors"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

var (
	// ErrInvalidBlockReward is returned when block reward is invalid.
	ErrInvalidBlockReward = errors.New("invalid block reward")

	// ErrRewardAlreadyProcessed is returned when reward was already processed.
	ErrRewardAlreadyProcessed = errors.New("reward already processed")

	// ErrInvalidSibling is returned when a sibling (uncle) is invalid.
	ErrInvalidSibling = errors.New("invalid sibling")

	// ErrInvalidConfig is returned when the configuration is invalid.
	ErrInvalidConfig = errors.New("invalid remasc configuration")
)

// StateDB is the interface for accessing and modifying the world state.
type StateDB interface {
	GetBalance(addr common.Address) *big.Int
	AddBalance(addr common.Address, amount *big.Int)
	SubBalance(addr common.Address, amount *big.Int)
}

// Sibling represents a sibling (uncle) block that was included in the chain.
type Sibling struct {
	// Header is the uncle block header
	Header *types.Header

	// Coinbase is the miner address of the uncle block
	Coinbase common.Address

	// PaidFees are the fees collected in the uncle block
	PaidFees *big.Int

	// IncludedInBlock is the block number where this uncle was included
	IncludedInBlock uint64

	// PublishedBy is the address of the miner who included this uncle
	PublishedBy common.Address
}

// Remasc is the Reward Manager Smart Contract.
// It handles the distribution of mining rewards in RSK.
type Remasc struct {
	config *Config

	// Reward tracking
	rewardBalance *big.Int // Total unclaimed rewards in the pool
	burnedBalance *big.Int // Total burned rewards

	// Sibling tracking: block number -> siblings registered for that block
	siblings   map[uint64][]*Sibling
	siblingsMu sync.RWMutex

	// Processed blocks tracking
	processedBlocks map[uint64]bool
	processedMu     sync.RWMutex

	// Block coinbase mapping for reward payment
	blockCoinbases   map[uint64]common.Address
	blockCoinbasesMu sync.RWMutex
}

// New creates a new REMASC instance with the given configuration.
func New(config *Config) *Remasc {
	if config == nil {
		config = DefaultConfig()
	}
	return &Remasc{
		config:          config,
		rewardBalance:   new(big.Int),
		burnedBalance:   new(big.Int),
		siblings:        make(map[uint64][]*Sibling),
		processedBlocks: make(map[uint64]bool),
		blockCoinbases:  make(map[uint64]common.Address),
	}
}

// ProcessBlock processes rewards for a block.
// This should be called at the end of block execution.
func (r *Remasc) ProcessBlock(stateDB StateDB, header *types.Header, uncles []*types.Header, paidFees *big.Int) error {
	blockNum := header.Number.Uint64()
	coinbase := header.Coinbase

	// Check if already processed
	r.processedMu.Lock()
	if r.processedBlocks[blockNum] {
		r.processedMu.Unlock()
		return ErrRewardAlreadyProcessed
	}
	r.processedBlocks[blockNum] = true
	r.processedMu.Unlock()

	// Store the coinbase for this block (needed for paying mature rewards)
	r.blockCoinbasesMu.Lock()
	r.blockCoinbases[blockNum] = coinbase
	r.blockCoinbasesMu.Unlock()

	// Add paid fees to reward pool
	r.addToRewardPool(paidFees)

	// Register uncles (siblings) for reward tracking
	for _, uncle := range uncles {
		if err := r.registerSibling(header, uncle); err != nil {
			// Log error but continue - don't fail block processing
			continue
		}
	}

	// Pay mature rewards if we've passed the maturity period
	if blockNum >= r.config.MaturityPeriod {
		matureBlockNum := blockNum - r.config.MaturityPeriod
		if err := r.payMatureRewards(stateDB, matureBlockNum); err != nil {
			return err
		}
	}

	return nil
}

// addToRewardPool adds fees to the reward pool.
func (r *Remasc) addToRewardPool(fees *big.Int) {
	if fees == nil || fees.Sign() <= 0 {
		return
	}
	r.rewardBalance.Add(r.rewardBalance, fees)
}

// registerSibling registers a sibling (uncle) block for reward tracking.
func (r *Remasc) registerSibling(mainHeader *types.Header, uncle *types.Header) error {
	r.siblingsMu.Lock()
	defer r.siblingsMu.Unlock()

	sibling := &Sibling{
		Header:          uncle,
		Coinbase:        uncle.Coinbase,
		PaidFees:        new(big.Int), // Uncle fees are not directly accessible from header
		IncludedInBlock: mainHeader.Number.Uint64(),
		PublishedBy:     mainHeader.Coinbase,
	}

	uncleNum := uncle.Number.Uint64()
	r.siblings[uncleNum] = append(r.siblings[uncleNum], sibling)

	return nil
}

// payMatureRewards pays out mature rewards for a block.
func (r *Remasc) payMatureRewards(stateDB StateDB, blockNum uint64) error {
	// Calculate reward for this block
	reward := r.calculateBlockReward(blockNum)
	if reward == nil || reward.Sign() == 0 {
		return nil
	}

	// Get the coinbase for the mature block
	r.blockCoinbasesMu.RLock()
	matureCoinbase, ok := r.blockCoinbases[blockNum]
	r.blockCoinbasesMu.RUnlock()

	if !ok {
		// No coinbase recorded for this block
		return nil
	}

	// Get siblings for this block
	r.siblingsMu.RLock()
	siblings := r.siblings[blockNum]
	r.siblingsMu.RUnlock()

	// Calculate reward distribution
	minerReward, siblingRewards, rskLabsReward, federationReward, burnAmount := r.calculateDistribution(reward, siblings)

	// Pay miner of the mature block
	if minerReward.Sign() > 0 {
		stateDB.AddBalance(matureCoinbase, minerReward)
	}

	// Pay sibling miners and publishers
	for i, sibling := range siblings {
		if i < len(siblingRewards) {
			siblingReward := siblingRewards[i]
			if siblingReward.Sign() > 0 {
				// Calculate miner portion (after publisher cut)
				minerCut := new(big.Int).Mul(siblingReward, big.NewInt(int64(10000-r.config.PublishersCut)))
				minerCut.Div(minerCut, big.NewInt(10000))

				// Pay sibling miner
				stateDB.AddBalance(sibling.Coinbase, minerCut)

				// Pay publisher (miner who included the uncle)
				publisherCut := new(big.Int).Sub(siblingReward, minerCut)
				if publisherCut.Sign() > 0 {
					stateDB.AddBalance(sibling.PublishedBy, publisherCut)
				}
			}
		}
	}

	// Pay RSK Labs
	if rskLabsReward.Sign() > 0 {
		stateDB.AddBalance(RSKLabsAddress, rskLabsReward)
	}

	// Pay Federation
	if federationReward.Sign() > 0 && r.config.FederationAddress != (common.Address{}) {
		stateDB.AddBalance(r.config.FederationAddress, federationReward)
	}

	// Track burned amount
	if burnAmount.Sign() > 0 {
		r.burnedBalance.Add(r.burnedBalance, burnAmount)
	}

	// Subtract from reward pool
	totalPaid := new(big.Int).Add(minerReward, rskLabsReward)
	totalPaid.Add(totalPaid, federationReward)
	totalPaid.Add(totalPaid, burnAmount)
	for _, sr := range siblingRewards {
		totalPaid.Add(totalPaid, sr)
	}
	r.rewardBalance.Sub(r.rewardBalance, totalPaid)

	// Clean up siblings for this block
	r.siblingsMu.Lock()
	delete(r.siblings, blockNum)
	r.siblingsMu.Unlock()

	// Clean up coinbase tracking
	r.blockCoinbasesMu.Lock()
	delete(r.blockCoinbases, blockNum)
	r.blockCoinbasesMu.Unlock()

	return nil
}

// calculateBlockReward calculates the reward for a specific block.
// In RSK, rewards come from transaction fees, not block subsidy.
func (r *Remasc) calculateBlockReward(blockNum uint64) *big.Int {
	if r.rewardBalance.Sign() == 0 {
		return new(big.Int)
	}

	// Distribute reward pool over synthetic span
	// This creates a smoothing effect on miner rewards
	reward := new(big.Int).Div(r.rewardBalance, big.NewInt(int64(r.config.SyntheticSpan)))

	return reward
}

// calculateDistribution calculates how rewards are distributed among participants.
func (r *Remasc) calculateDistribution(totalReward *big.Int, siblings []*Sibling) (
	minerReward *big.Int,
	siblingRewards []*big.Int,
	rskLabsReward *big.Int,
	federationReward *big.Int,
	burnAmount *big.Int,
) {
	// Start with total reward
	remaining := new(big.Int).Set(totalReward)

	// Calculate RSK Labs cut
	rskLabsReward = new(big.Int).Mul(totalReward, big.NewInt(int64(r.config.RSKLabsCut)))
	rskLabsReward.Div(rskLabsReward, big.NewInt(10000))
	remaining.Sub(remaining, rskLabsReward)

	// Calculate federation cut
	federationReward = new(big.Int).Mul(totalReward, big.NewInt(int64(r.config.FederationCut)))
	federationReward.Div(federationReward, big.NewInt(10000))
	remaining.Sub(remaining, federationReward)

	// Calculate burn cut
	burnAmount = new(big.Int).Mul(totalReward, big.NewInt(int64(r.config.BurnedCut)))
	burnAmount.Div(burnAmount, big.NewInt(10000))
	remaining.Sub(remaining, burnAmount)

	// Calculate sibling rewards
	siblingRewards = make([]*big.Int, len(siblings))
	siblingTotal := new(big.Int)

	for i, sibling := range siblings {
		siblingReward := r.calculateSiblingReward(remaining, sibling)
		siblingRewards[i] = siblingReward
		siblingTotal.Add(siblingTotal, siblingReward)
	}
	remaining.Sub(remaining, siblingTotal)

	// Miner gets the remaining amount
	minerReward = remaining

	return
}

// calculateSiblingReward calculates the reward for a sibling (uncle) block.
// Rewards decrease with distance from the main chain.
func (r *Remasc) calculateSiblingReward(poolReward *big.Int, sibling *Sibling) *big.Int {
	siblingNum := sibling.Header.Number.Uint64()
	includedNum := sibling.IncludedInBlock
	distance := includedNum - siblingNum

	// Maximum distance for uncle inclusion is typically 7 blocks
	if distance > 7 {
		return new(big.Int)
	}

	// Base sibling reward: 1/32 of pool per sibling
	baseReward := new(big.Int).Div(poolReward, big.NewInt(32))

	// Adjust by distance (closer = more reward)
	// distance 1 = 7/8, distance 2 = 6/8, etc.
	distanceFactor := new(big.Int).SetUint64(8 - distance)
	adjustedReward := new(big.Int).Mul(baseReward, distanceFactor)
	adjustedReward.Div(adjustedReward, big.NewInt(8))

	return adjustedReward
}

// GetRewardBalance returns the current reward balance in the pool.
func (r *Remasc) GetRewardBalance() *big.Int {
	return new(big.Int).Set(r.rewardBalance)
}

// GetBurnedBalance returns the total burned balance.
func (r *Remasc) GetBurnedBalance() *big.Int {
	return new(big.Int).Set(r.burnedBalance)
}

// BrokenSelectionRule represents a violation of the selection rule.
// If a miner mines on a block that isn't the best block (by selection rule),
// they may be penalized.
type BrokenSelectionRule struct {
	BlockNumber uint64
	Miner       common.Address
	Penalty     *big.Int
}

// CheckSelectionRule checks if the selection rule was broken.
// The selection rule says miners should mine on the best block by difficulty.
func (r *Remasc) CheckSelectionRule(header *types.Header, parentHeader *types.Header, siblings []*Sibling) *BrokenSelectionRule {
	blockDiff := header.Difficulty

	for _, sibling := range siblings {
		siblingDiff := sibling.Header.Difficulty

		// If sibling had higher or equal difficulty and was valid,
		// the miner may have broken the selection rule
		if siblingDiff.Cmp(blockDiff) >= 0 {
			// Calculate penalty (portion of reward)
			penalty := new(big.Int).Div(new(big.Int).SetUint64(header.GasUsed), big.NewInt(10))

			return &BrokenSelectionRule{
				BlockNumber: header.Number.Uint64(),
				Miner:       header.Coinbase,
				Penalty:     penalty,
			}
		}
	}

	return nil
}

// ApplyPenalty applies a penalty for broken selection rule.
func (r *Remasc) ApplyPenalty(penalty *BrokenSelectionRule) {
	if penalty == nil || penalty.Penalty.Sign() == 0 {
		return
	}

	// Deduct from miner's reward and add to burn
	r.burnedBalance.Add(r.burnedBalance, penalty.Penalty)
}
