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
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

// RSK Genesis timestamps
const (
	// RSKMainnetGenesisTimestamp is the timestamp of RSK mainnet genesis.
	// RSK mainnet launched on January 4, 2018.
	RSKMainnetGenesisTimestamp uint64 = 1525284684

	// RSKTestnetGenesisTimestamp is the timestamp of RSK testnet genesis.
	RSKTestnetGenesisTimestamp uint64 = 1521183519
)

// DefaultRSKMainnetGenesisBlock returns the RSK mainnet genesis block.
func DefaultRSKMainnetGenesisBlock() *Genesis {
	return &Genesis{
		Config:     params.RSKMainnetChainConfig,
		Nonce:      0,
		Timestamp:  RSKMainnetGenesisTimestamp,
		ExtraData:  []byte{},
		GasLimit:   6800000,
		Difficulty: params.MinimumDifficulty,
		Mixhash:    common.Hash{},
		Coinbase:   common.Address{},
		Alloc:      rskMainnetGenesisAlloc(),
	}
}

// DefaultRSKTestnetGenesisBlock returns the RSK testnet genesis block.
func DefaultRSKTestnetGenesisBlock() *Genesis {
	return &Genesis{
		Config:     params.RSKTestnetChainConfig,
		Nonce:      0,
		Timestamp:  RSKTestnetGenesisTimestamp,
		ExtraData:  []byte{},
		GasLimit:   6800000,
		Difficulty: params.MinimumDifficulty,
		Mixhash:    common.Hash{},
		Coinbase:   common.Address{},
		Alloc:      rskTestnetGenesisAlloc(),
	}
}

// rskMainnetGenesisAlloc returns the genesis allocation for RSK mainnet.
// This includes precompile accounts and initial balances.
func rskMainnetGenesisAlloc() types.GenesisAlloc {
	alloc := make(types.GenesisAlloc)

	// RSK Bridge precompile - requires code and storage
	alloc[params.RSKBridgeAddress] = types.Account{
		Balance: new(big.Int),
		Code:    []byte{}, // Precompile code is handled natively
		Nonce:   0,
	}

	// RSK REMASC precompile - requires code and initial state
	alloc[params.RSKRemascAddress] = types.Account{
		Balance: new(big.Int),
		Code:    []byte{},
		Nonce:   0,
	}

	// RSK Labs address - receives portion of rewards
	alloc[params.RSKLabsAddress] = types.Account{
		Balance: new(big.Int),
		Nonce:   0,
	}

	// RSK HD Wallet Utils precompile
	alloc[params.RSKHDWalletUtilsAddress] = types.Account{
		Balance: new(big.Int),
		Code:    []byte{},
		Nonce:   0,
	}

	// RSK Block Hash Query precompile
	alloc[params.RSKBlockHashAddress] = types.Account{
		Balance: new(big.Int),
		Code:    []byte{},
		Nonce:   0,
	}

	// Add standard precompile accounts (addresses 0x01 to 0x09)
	for i := 1; i <= 9; i++ {
		addr := common.BytesToAddress([]byte{byte(i)})
		alloc[addr] = types.Account{
			Balance: new(big.Int),
			Nonce:   0,
		}
	}

	return alloc
}

// rskTestnetGenesisAlloc returns the genesis allocation for RSK testnet.
func rskTestnetGenesisAlloc() types.GenesisAlloc {
	// Testnet has similar allocations to mainnet
	alloc := rskMainnetGenesisAlloc()

	// Add some testnet-specific allocations if needed
	// (e.g., faucet addresses with initial balance)

	return alloc
}

// RSKGenesisInfo contains information about RSK genesis blocks.
type RSKGenesisInfo struct {
	ChainID         uint64
	GenesisHash     common.Hash
	GenesisTime     uint64
	BridgeAddress   common.Address
	RemascAddress   common.Address
	RSKLabsAddress  common.Address
}

// GetRSKMainnetGenesisInfo returns information about RSK mainnet genesis.
func GetRSKMainnetGenesisInfo() *RSKGenesisInfo {
	return &RSKGenesisInfo{
		ChainID:        params.RSKMainnetChainID,
		GenesisHash:    params.RSKMainnetGenesisHash,
		GenesisTime:    RSKMainnetGenesisTimestamp,
		BridgeAddress:  params.RSKBridgeAddress,
		RemascAddress:  params.RSKRemascAddress,
		RSKLabsAddress: params.RSKLabsAddress,
	}
}

// GetRSKTestnetGenesisInfo returns information about RSK testnet genesis.
func GetRSKTestnetGenesisInfo() *RSKGenesisInfo {
	return &RSKGenesisInfo{
		ChainID:        params.RSKTestnetChainID,
		GenesisHash:    params.RSKTestnetGenesisHash,
		GenesisTime:    RSKTestnetGenesisTimestamp,
		BridgeAddress:  params.RSKBridgeAddress,
		RemascAddress:  params.RSKRemascAddress,
		RSKLabsAddress: params.RSKLabsAddress,
	}
}

// IsRSKGenesis checks if the given genesis configuration is for RSK.
func IsRSKGenesis(genesis *Genesis) bool {
	if genesis == nil || genesis.Config == nil {
		return false
	}
	return genesis.Config.IsRSK()
}

// ValidateRSKGenesis validates an RSK genesis configuration.
func ValidateRSKGenesis(genesis *Genesis) error {
	if !IsRSKGenesis(genesis) {
		return nil // Not an RSK genesis, nothing to validate
	}

	// Check that required precompile addresses are allocated
	if genesis.Alloc == nil {
		return ErrGenesisNoAlloc
	}

	// Check Bridge allocation
	if _, ok := genesis.Alloc[params.RSKBridgeAddress]; !ok {
		return ErrMissingBridgeAlloc
	}

	// Check REMASC allocation
	if _, ok := genesis.Alloc[params.RSKRemascAddress]; !ok {
		return ErrMissingRemascAlloc
	}

	return nil
}

// RSK genesis validation errors
var (
	ErrGenesisNoAlloc      = &rskGenesisError{"genesis has no allocation"}
	ErrMissingBridgeAlloc  = &rskGenesisError{"missing Bridge precompile allocation"}
	ErrMissingRemascAlloc  = &rskGenesisError{"missing REMASC precompile allocation"}
)

type rskGenesisError struct {
	msg string
}

func (e *rskGenesisError) Error() string {
	return e.msg
}

// SetupRSKPrecompileAccounts adds RSK precompile accounts to a genesis allocation.
func SetupRSKPrecompileAccounts(alloc types.GenesisAlloc) types.GenesisAlloc {
	if alloc == nil {
		alloc = make(types.GenesisAlloc)
	}

	// Add RSK precompile accounts if not present
	if _, ok := alloc[params.RSKBridgeAddress]; !ok {
		alloc[params.RSKBridgeAddress] = types.Account{
			Balance: new(big.Int),
			Nonce:   0,
		}
	}

	if _, ok := alloc[params.RSKRemascAddress]; !ok {
		alloc[params.RSKRemascAddress] = types.Account{
			Balance: new(big.Int),
			Nonce:   0,
		}
	}

	if _, ok := alloc[params.RSKLabsAddress]; !ok {
		alloc[params.RSKLabsAddress] = types.Account{
			Balance: new(big.Int),
			Nonce:   0,
		}
	}

	if _, ok := alloc[params.RSKHDWalletUtilsAddress]; !ok {
		alloc[params.RSKHDWalletUtilsAddress] = types.Account{
			Balance: new(big.Int),
			Nonce:   0,
		}
	}

	if _, ok := alloc[params.RSKBlockHashAddress]; !ok {
		alloc[params.RSKBlockHashAddress] = types.Account{
			Balance: new(big.Int),
			Nonce:   0,
		}
	}

	return alloc
}
