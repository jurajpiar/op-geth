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

// Package remasc implements the Reward Manager Smart Contract for RSK.
// REMASC handles the distribution of mining rewards including synthetic mining rewards,
// sibling (uncle) rewards, RSK Labs cut, and federation cuts.
package remasc

import (
	"github.com/ethereum/go-ethereum/common"
)

// RemascAddress is the address of the REMASC precompiled contract.
var RemascAddress = common.HexToAddress("0x0000000000000000000000000000000001000002")

// RSKLabsAddress is the address that receives RSK Labs' cut of the rewards.
var RSKLabsAddress = common.HexToAddress("0x0000000000000000000000000000000001000003")

// Config contains REMASC configuration parameters.
type Config struct {
	// MaturityPeriod is the number of blocks before rewards mature and can be paid out.
	// Default: 4000 blocks (~33 hours at 30s block time)
	MaturityPeriod uint64

	// SyntheticSpan is the span over which synthetic mining rewards are calculated.
	// Default: 4000 blocks
	SyntheticSpan uint64

	// RSKLabsCut is the percentage cut for RSK Labs in basis points (1/10000).
	// Default: 1000 (10%)
	RSKLabsCut uint64

	// FederationCut is the percentage cut for the federation in basis points.
	// Default: 0 (historically used, now 0%)
	FederationCut uint64

	// PublishersCut is the percentage cut for uncle publishers in basis points.
	// Default: 1000 (10%)
	PublishersCut uint64

	// BurnedCut is the percentage that gets burned in basis points.
	// Default: 0 (0%)
	BurnedCut uint64

	// FederationAddress is the address that receives the federation's cut.
	FederationAddress common.Address
}

// DefaultConfig returns the default REMASC configuration for RSK mainnet.
func DefaultConfig() *Config {
	return &Config{
		MaturityPeriod:    4000, // ~4000 blocks maturity (~33 hours at 30s block time)
		SyntheticSpan:     4000, // Reward span
		RSKLabsCut:        1000, // 10%
		FederationCut:     0,    // 0% (was used historically)
		PublishersCut:     1000, // 10% for uncle publishers
		BurnedCut:         0,    // 0%
		FederationAddress: common.Address{},
	}
}

// TestnetConfig returns the default REMASC configuration for RSK testnet.
func TestnetConfig() *Config {
	return &Config{
		MaturityPeriod:    4000,
		SyntheticSpan:     4000,
		RSKLabsCut:        1000,
		FederationCut:     0,
		PublishersCut:     1000,
		BurnedCut:         0,
		FederationAddress: common.Address{},
	}
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	// Total cuts should not exceed 10000 (100%)
	totalCuts := c.RSKLabsCut + c.FederationCut + c.BurnedCut
	if totalCuts > 10000 {
		return ErrInvalidConfig
	}
	return nil
}
