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

package params

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// RSK Chain IDs
const (
	RSKMainnetChainID = 30
	RSKTestnetChainID = 31
	RSKRegtestChainID = 33
)

// RSK Genesis hashes
var (
	RSKMainnetGenesisHash = common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000") // TODO: Set actual RSK mainnet genesis hash
	RSKTestnetGenesisHash = common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000") // TODO: Set actual RSK testnet genesis hash
)

// RSK Precompile addresses
var (
	RSKBridgeAddress       = common.HexToAddress("0x0000000000000000000000000000000001000001")
	RSKRemascAddress       = common.HexToAddress("0x0000000000000000000000000000000001000002")
	RSKLabsAddress         = common.HexToAddress("0x0000000000000000000000000000000001000003")
	RSKHDWalletUtilsAddress = common.HexToAddress("0x0000000000000000000000000000000001000006")
	RSKBlockHashAddress    = common.HexToAddress("0x0000000000000000000000000000000001000008")
)

// RSKConfig contains RSK-specific chain configuration.
type RSKConfig struct {
	// REMASC configuration
	RSKLabsCut     uint64 `json:"rskLabsCut"`     // RSK Labs cut in basis points (1000 = 10%)
	FederationCut  uint64 `json:"federationCut"`  // Federation cut in basis points
	PublishersCut  uint64 `json:"publishersCut"`  // Uncle publisher cut in basis points
	BurnedCut      uint64 `json:"burnedCut"`      // Burned cut in basis points
	MaturityPeriod uint64 `json:"maturityPeriod"` // Block maturity period for rewards (4000 blocks)
	SyntheticSpan  uint64 `json:"syntheticSpan"`  // Synthetic mining reward span

	// Merged mining
	MinimumDifficulty *big.Int `json:"minimumDifficulty,omitempty"` // Minimum difficulty (0x20000)

	// Federation address for bridge operations
	FederationAddress common.Address `json:"federationAddress,omitempty"`
}

// DefaultRSKConfig returns the default RSK configuration.
func DefaultRSKConfig() *RSKConfig {
	return &RSKConfig{
		RSKLabsCut:        1000, // 10%
		FederationCut:     0,    // 0%
		PublishersCut:     1000, // 10%
		BurnedCut:         0,    // 0%
		MaturityPeriod:    4000, // ~4000 blocks maturity
		SyntheticSpan:     4000,
		MinimumDifficulty: big.NewInt(131072), // 0x20000
	}
}

var (
	// RSKMainnetChainConfig is the chain parameters for RSK mainnet.
	RSKMainnetChainConfig = &ChainConfig{
		ChainID:             big.NewInt(RSKMainnetChainID),
		HomesteadBlock:      big.NewInt(0),
		DAOForkBlock:        nil,
		DAOForkSupport:      false,
		EIP150Block:         big.NewInt(0),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
		PetersburgBlock:     big.NewInt(0),
		IstanbulBlock:       big.NewInt(0),
		MuirGlacierBlock:    nil,
		BerlinBlock:         big.NewInt(0),
		LondonBlock:         big.NewInt(0),
		ArrowGlacierBlock:   nil,
		GrayGlacierBlock:    nil,

		// RSK fork activations (RSKIPs) - block numbers
		RSK: &RSKConfig{
			RSKLabsCut:        1000,
			FederationCut:     0,
			PublishersCut:     1000,
			BurnedCut:         0,
			MaturityPeriod:    4000,
			SyntheticSpan:     4000,
			MinimumDifficulty: big.NewInt(131072),
		},

		// RSK uses merged mining, not Ethash or Clique
		MergedMining: &MergedMiningConfig{},
	}

	// RSKTestnetChainConfig is the chain parameters for RSK testnet.
	RSKTestnetChainConfig = &ChainConfig{
		ChainID:             big.NewInt(RSKTestnetChainID),
		HomesteadBlock:      big.NewInt(0),
		DAOForkBlock:        nil,
		DAOForkSupport:      false,
		EIP150Block:         big.NewInt(0),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
		PetersburgBlock:     big.NewInt(0),
		IstanbulBlock:       big.NewInt(0),
		MuirGlacierBlock:    nil,
		BerlinBlock:         big.NewInt(0),
		LondonBlock:         big.NewInt(0),
		ArrowGlacierBlock:   nil,
		GrayGlacierBlock:    nil,

		// RSK fork activations (RSKIPs)
		RSK: &RSKConfig{
			RSKLabsCut:        1000,
			FederationCut:     0,
			PublishersCut:     1000,
			BurnedCut:         0,
			MaturityPeriod:    4000,
			SyntheticSpan:     4000,
			MinimumDifficulty: big.NewInt(131072),
		},

		// RSK uses merged mining
		MergedMining: &MergedMiningConfig{},
	}
)

// MergedMiningConfig is the consensus engine config for Bitcoin merged mining.
type MergedMiningConfig struct {
	// TargetBlockTime is the target time between blocks in seconds (30 for RSK).
	TargetBlockTime uint64 `json:"targetBlockTime,omitempty"`

	// DifficultyBoundDivisor is the bound divisor for difficulty adjustment (2048 for RSK).
	DifficultyBoundDivisor uint64 `json:"difficultyBoundDivisor,omitempty"`
}

// String implements the stringer interface for MergedMiningConfig.
func (c MergedMiningConfig) String() string {
	return "mergedmining"
}

// RSK RSKIP fork activation blocks for mainnet
var (
	// RSKIP91 - STATICCALL opcode
	RSKIP91MainnetBlock = big.NewInt(729000)

	// RSKIP120 - Shift opcodes (SHL, SHR, SAR)
	RSKIP120MainnetBlock = big.NewInt(1591000)

	// RSKIP125 - CREATE2 opcode
	RSKIP125MainnetBlock = big.NewInt(1591000)

	// RSKIP140 - EXTCODEHASH opcode
	RSKIP140MainnetBlock = big.NewInt(2392700)

	// RSKIP151 - SELFBALANCE opcode
	RSKIP151MainnetBlock = big.NewInt(3614800)

	// RSKIP152 - CHAINID opcode
	RSKIP152MainnetBlock = big.NewInt(3614800)

	// RSKIP191 - Disable TXINDEX/DUPN/SWAPN opcodes
	RSKIP191MainnetBlock = big.NewInt(3614800)

	// RSKIP398 - PUSH0 opcode
	RSKIP398MainnetBlock = big.NewInt(4598500)

	// RSKIP412 - BASEFEE opcode
	RSKIP412MainnetBlock = big.NewInt(5468000)

	// RSKIP445 - MCOPY opcode
	RSKIP445MainnetBlock = big.NewInt(6223000)

	// RSKIP446 - TLOAD/TSTORE opcodes
	RSKIP446MainnetBlock = big.NewInt(6223000)
)
