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
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// mockStateDB implements the StateDB interface for testing.
type mockStateDB struct {
	balances map[common.Address]*big.Int
}

func newMockStateDB() *mockStateDB {
	return &mockStateDB{
		balances: make(map[common.Address]*big.Int),
	}
}

func (m *mockStateDB) GetBalance(addr common.Address) *big.Int {
	if bal, ok := m.balances[addr]; ok {
		return new(big.Int).Set(bal)
	}
	return new(big.Int)
}

func (m *mockStateDB) AddBalance(addr common.Address, amount *big.Int) {
	if amount == nil || amount.Sign() <= 0 {
		return
	}
	if _, ok := m.balances[addr]; !ok {
		m.balances[addr] = new(big.Int)
	}
	m.balances[addr].Add(m.balances[addr], amount)
}

func (m *mockStateDB) SubBalance(addr common.Address, amount *big.Int) {
	if amount == nil || amount.Sign() <= 0 {
		return
	}
	if _, ok := m.balances[addr]; ok {
		m.balances[addr].Sub(m.balances[addr], amount)
	}
}

// TestNewRemasc tests REMASC creation.
func TestNewRemasc(t *testing.T) {
	// Test with nil config
	r := New(nil)
	if r == nil {
		t.Fatal("New(nil) returned nil")
	}
	if r.config == nil {
		t.Error("config should be set to default")
	}

	// Test with custom config
	config := &Config{
		MaturityPeriod: 5000,
		SyntheticSpan:  5000,
		RSKLabsCut:     500, // 5%
	}
	r = New(config)
	if r.config.MaturityPeriod != 5000 {
		t.Errorf("MaturityPeriod = %d, want 5000", r.config.MaturityPeriod)
	}
}

// TestRemascRewardBalance tests reward balance tracking.
func TestRemascRewardBalance(t *testing.T) {
	r := New(nil)

	// Initial balance should be zero
	balance := r.GetRewardBalance()
	if balance.Sign() != 0 {
		t.Errorf("initial reward balance = %v, want 0", balance)
	}

	// Add some rewards
	r.addToRewardPool(big.NewInt(1000))
	balance = r.GetRewardBalance()
	if balance.Cmp(big.NewInt(1000)) != 0 {
		t.Errorf("reward balance = %v, want 1000", balance)
	}

	// Add more rewards
	r.addToRewardPool(big.NewInt(500))
	balance = r.GetRewardBalance()
	if balance.Cmp(big.NewInt(1500)) != 0 {
		t.Errorf("reward balance = %v, want 1500", balance)
	}
}

// TestRemascBurnedBalance tests burned balance tracking.
func TestRemascBurnedBalance(t *testing.T) {
	r := New(nil)

	// Initial burned balance should be zero
	burned := r.GetBurnedBalance()
	if burned.Sign() != 0 {
		t.Errorf("initial burned balance = %v, want 0", burned)
	}
}

// TestRemascCalculateDistribution tests reward distribution calculation.
func TestRemascCalculateDistribution(t *testing.T) {
	config := &Config{
		MaturityPeriod: 4000,
		SyntheticSpan:  4000,
		RSKLabsCut:     1000, // 10%
		FederationCut:  0,
		PublishersCut:  1000, // 10%
		BurnedCut:      0,
	}
	r := New(config)

	totalReward := big.NewInt(10000)
	minerReward, siblingRewards, rskLabsReward, fedReward, burned := r.calculateDistribution(totalReward, nil)

	// RSK Labs should get 10%
	expectedRSKLabs := big.NewInt(1000)
	if rskLabsReward.Cmp(expectedRSKLabs) != 0 {
		t.Errorf("RSK Labs reward = %v, want %v", rskLabsReward, expectedRSKLabs)
	}

	// Federation should get 0%
	if fedReward.Sign() != 0 {
		t.Errorf("Federation reward = %v, want 0", fedReward)
	}

	// Burned should be 0
	if burned.Sign() != 0 {
		t.Errorf("burned = %v, want 0", burned)
	}

	// No siblings, so sibling rewards should be empty
	if len(siblingRewards) != 0 {
		t.Errorf("sibling rewards length = %d, want 0", len(siblingRewards))
	}

	// Miner should get the rest (90%)
	expectedMiner := big.NewInt(9000)
	if minerReward.Cmp(expectedMiner) != 0 {
		t.Errorf("miner reward = %v, want %v", minerReward, expectedMiner)
	}
}

// TestRemascCalculateSiblingReward tests sibling reward calculation.
func TestRemascCalculateSiblingReward(t *testing.T) {
	r := New(nil)

	poolReward := big.NewInt(32000) // 32 * 1000 for easy calculation

	tests := []struct {
		name        string
		siblingNum  uint64
		includedNum uint64
		expectZero  bool
	}{
		{
			name:        "distance 1 (closest)",
			siblingNum:  100,
			includedNum: 101,
		},
		{
			name:        "distance 7 (max)",
			siblingNum:  100,
			includedNum: 107,
		},
		{
			name:        "distance 8 (too far)",
			siblingNum:  100,
			includedNum: 108,
			expectZero:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sibling := &Sibling{
				Header: &types.Header{
					Number: big.NewInt(int64(tt.siblingNum)),
				},
				IncludedInBlock: tt.includedNum,
			}

			reward := r.calculateSiblingReward(poolReward, sibling)

			if tt.expectZero && reward.Sign() != 0 {
				t.Errorf("expected zero reward, got %v", reward)
			}
			if !tt.expectZero && reward.Sign() == 0 {
				t.Errorf("expected non-zero reward, got 0")
			}
		})
	}
}

// TestRemascProcessBlockDuplicate tests duplicate block processing prevention.
func TestRemascProcessBlockDuplicate(t *testing.T) {
	r := New(nil)
	stateDB := newMockStateDB()

	header := &types.Header{
		Number:   big.NewInt(100),
		Coinbase: common.HexToAddress("0x1234"),
		Time:     1000,
	}

	// First call should succeed
	err := r.ProcessBlock(stateDB, header, nil, big.NewInt(1000))
	if err != nil {
		t.Errorf("first ProcessBlock error = %v", err)
	}

	// Second call with same block should fail
	err = r.ProcessBlock(stateDB, header, nil, big.NewInt(1000))
	if err != ErrRewardAlreadyProcessed {
		t.Errorf("duplicate ProcessBlock error = %v, want %v", err, ErrRewardAlreadyProcessed)
	}
}

// TestRemascConfig tests configuration validation.
func TestRemascConfig(t *testing.T) {
	// Valid config
	config := &Config{
		RSKLabsCut:    1000,
		FederationCut: 500,
		BurnedCut:     500,
	}
	if err := config.Validate(); err != nil {
		t.Errorf("valid config returned error: %v", err)
	}

	// Invalid config (total > 100%)
	config = &Config{
		RSKLabsCut:    5000,
		FederationCut: 5000,
		BurnedCut:     1000, // Total = 11000 > 10000
	}
	if err := config.Validate(); err != ErrInvalidConfig {
		t.Errorf("invalid config error = %v, want %v", err, ErrInvalidConfig)
	}
}

// TestDefaultConfig tests default configuration.
func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.MaturityPeriod != 4000 {
		t.Errorf("MaturityPeriod = %d, want 4000", config.MaturityPeriod)
	}
	if config.SyntheticSpan != 4000 {
		t.Errorf("SyntheticSpan = %d, want 4000", config.SyntheticSpan)
	}
	if config.RSKLabsCut != 1000 {
		t.Errorf("RSKLabsCut = %d, want 1000", config.RSKLabsCut)
	}
	if config.PublishersCut != 1000 {
		t.Errorf("PublishersCut = %d, want 1000", config.PublishersCut)
	}
}

// TestRemascCalculateBlockReward tests block reward calculation.
func TestRemascCalculateBlockReward(t *testing.T) {
	config := &Config{
		SyntheticSpan: 4000,
	}
	r := New(config)

	// With zero balance, reward should be zero
	reward := r.calculateBlockReward(100)
	if reward.Sign() != 0 {
		t.Errorf("reward with zero balance = %v, want 0", reward)
	}

	// Add some balance
	r.rewardBalance = big.NewInt(4000000) // 4M

	// Reward should be balance / syntheticSpan = 4M / 4000 = 1000
	reward = r.calculateBlockReward(100)
	expectedReward := big.NewInt(1000)
	if reward.Cmp(expectedReward) != 0 {
		t.Errorf("reward = %v, want %v", reward, expectedReward)
	}
}
