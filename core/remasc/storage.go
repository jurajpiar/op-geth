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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// Storage key constants for REMASC state
var (
	// RewardBalanceKey is the storage key for the reward balance
	RewardBalanceKey = crypto.Keccak256Hash([]byte("remasc.rewardBalance"))

	// BurnedBalanceKey is the storage key for the burned balance
	BurnedBalanceKey = crypto.Keccak256Hash([]byte("remasc.burnedBalance"))

	// SiblingsKeyPrefix is the prefix for siblings storage keys
	SiblingsKeyPrefix = []byte("remasc.siblings.")

	// ProcessedBlocksKeyPrefix is the prefix for processed blocks storage keys
	ProcessedBlocksKeyPrefix = []byte("remasc.processed.")
)

// Storage provides persistent storage for REMASC state.
type Storage struct {
	stateDB StateDBStorage
}

// StateDBStorage is the interface for state storage operations.
type StateDBStorage interface {
	GetState(addr common.Address, key common.Hash) common.Hash
	SetState(addr common.Address, key common.Hash, value common.Hash)
}

// NewStorage creates a new Storage instance.
func NewStorage(stateDB StateDBStorage) *Storage {
	return &Storage{
		stateDB: stateDB,
	}
}

// GetRewardBalance retrieves the reward balance from storage.
func (s *Storage) GetRewardBalance() *big.Int {
	hash := s.stateDB.GetState(RemascAddress, RewardBalanceKey)
	return new(big.Int).SetBytes(hash.Bytes())
}

// SetRewardBalance stores the reward balance.
func (s *Storage) SetRewardBalance(balance *big.Int) {
	var hash common.Hash
	if balance != nil && balance.Sign() > 0 {
		hash = common.BigToHash(balance)
	}
	s.stateDB.SetState(RemascAddress, RewardBalanceKey, hash)
}

// GetBurnedBalance retrieves the burned balance from storage.
func (s *Storage) GetBurnedBalance() *big.Int {
	hash := s.stateDB.GetState(RemascAddress, BurnedBalanceKey)
	return new(big.Int).SetBytes(hash.Bytes())
}

// SetBurnedBalance stores the burned balance.
func (s *Storage) SetBurnedBalance(balance *big.Int) {
	var hash common.Hash
	if balance != nil && balance.Sign() > 0 {
		hash = common.BigToHash(balance)
	}
	s.stateDB.SetState(RemascAddress, BurnedBalanceKey, hash)
}

// siblingKey generates a storage key for a sibling at a given block number and index.
func siblingKey(blockNum uint64, index uint64) common.Hash {
	data := append(SiblingsKeyPrefix, common.BigToHash(new(big.Int).SetUint64(blockNum)).Bytes()...)
	data = append(data, common.BigToHash(new(big.Int).SetUint64(index)).Bytes()...)
	return crypto.Keccak256Hash(data)
}

// processedBlockKey generates a storage key for a processed block number.
func processedBlockKey(blockNum uint64) common.Hash {
	data := append(ProcessedBlocksKeyPrefix, common.BigToHash(new(big.Int).SetUint64(blockNum)).Bytes()...)
	return crypto.Keccak256Hash(data)
}

// IsBlockProcessed checks if a block has been processed.
func (s *Storage) IsBlockProcessed(blockNum uint64) bool {
	key := processedBlockKey(blockNum)
	hash := s.stateDB.GetState(RemascAddress, key)
	return hash != (common.Hash{})
}

// MarkBlockProcessed marks a block as processed.
func (s *Storage) MarkBlockProcessed(blockNum uint64) {
	key := processedBlockKey(blockNum)
	s.stateDB.SetState(RemascAddress, key, common.BigToHash(big.NewInt(1)))
}

// PersistentRemasc wraps Remasc with persistent storage support.
type PersistentRemasc struct {
	*Remasc
	storage *Storage
}

// NewPersistentRemasc creates a new PersistentRemasc instance.
func NewPersistentRemasc(config *Config, stateDB StateDBStorage) *PersistentRemasc {
	storage := NewStorage(stateDB)
	remasc := New(config)

	// Load persisted state
	remasc.rewardBalance = storage.GetRewardBalance()
	remasc.burnedBalance = storage.GetBurnedBalance()

	return &PersistentRemasc{
		Remasc:  remasc,
		storage: storage,
	}
}

// Commit persists the current REMASC state to storage.
func (p *PersistentRemasc) Commit() {
	p.storage.SetRewardBalance(p.rewardBalance)
	p.storage.SetBurnedBalance(p.burnedBalance)
}
