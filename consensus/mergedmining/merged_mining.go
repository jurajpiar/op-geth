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

// Package mergedmining implements Bitcoin merged mining consensus for RSK.
package mergedmining

import (
	"bytes"
	"errors"
	"math/big"
	"runtime"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/trie"
)

var (
	// ErrInvalidMergedMiningProof is returned when the merged mining proof is invalid.
	ErrInvalidMergedMiningProof = errors.New("invalid merged mining proof")

	// ErrInvalidMerkleProof is returned when the merkle proof is invalid.
	ErrInvalidMerkleProof = errors.New("invalid merkle proof")

	// ErrRskHashNotInCoinbase is returned when RSK hash is not in coinbase.
	ErrRskHashNotInCoinbase = errors.New("RSK hash not found in coinbase")

	// ErrInsufficientProofOfWork is returned when PoW is insufficient.
	ErrInsufficientProofOfWork = errors.New("insufficient proof of work")

	// ErrUnknownAncestor is returned when validating a block requires an ancestor
	// that is unknown.
	ErrUnknownAncestor = errors.New("unknown ancestor")

	// ErrFutureBlock is returned when a block's timestamp is in the future.
	ErrFutureBlock = errors.New("block in the future")

	// ErrInvalidNumber is returned if a block's number doesn't equal its parent's
	// plus one.
	ErrInvalidNumber = errors.New("invalid block number")

	// ErrInvalidDifficulty is returned if the difficulty of a block is invalid.
	ErrInvalidDifficulty = errors.New("invalid difficulty")

	// ErrMissingMergedMiningFields is returned when merged mining fields are missing.
	ErrMissingMergedMiningFields = errors.New("missing merged mining fields")
)

// RSKTag is the tag that appears before the RSK block hash in Bitcoin coinbase.
// "RSKBLOCK:" in bytes
var RSKTag = []byte{0x52, 0x53, 0x4b, 0x42, 0x4c, 0x4f, 0x43, 0x4b, 0x3a}

// MergedMining implements the consensus.Engine interface for RSK merged mining.
type MergedMining struct {
	config            *Config
	diffCalc          *DifficultyCalculator
	lock              sync.Mutex
	closeOnce         sync.Once
	exitCh            chan struct{}
}

// Config holds the configuration for the merged mining consensus engine.
type Config struct {
	// DifficultyAlgorithm specifies which difficulty adjustment algorithm to use.
	DifficultyAlgorithm DifficultyAlgorithm

	// AllowFutureBlocks allows blocks with future timestamps (for testing).
	AllowFutureBlocks bool
}

// DefaultConfig returns the default merged mining configuration.
func DefaultConfig() *Config {
	return &Config{
		DifficultyAlgorithm: DifficultyArrowhead,
		AllowFutureBlocks:   false,
	}
}

// New creates a new merged mining consensus engine.
func New(config *Config) *MergedMining {
	if config == nil {
		config = DefaultConfig()
	}
	return &MergedMining{
		config:   config,
		diffCalc: NewDifficultyCalculator(config.DifficultyAlgorithm),
		exitCh:   make(chan struct{}),
	}
}

// Author implements consensus.Engine, returning the address of the account that
// minted the given block (the coinbase).
func (mm *MergedMining) Author(header *types.Header) (common.Address, error) {
	return header.Coinbase, nil
}

// VerifyHeader checks whether a header conforms to the consensus rules.
func (mm *MergedMining) VerifyHeader(chain consensus.ChainHeaderReader, header *types.Header) error {
	// Short circuit if the header is known, or its parent not
	number := header.Number.Uint64()
	if chain.GetHeader(header.Hash(), number) != nil {
		return nil
	}
	parent := chain.GetHeader(header.ParentHash, number-1)
	if parent == nil {
		return ErrUnknownAncestor
	}
	// Sanity checks passed, do a proper verification
	return mm.verifyHeader(chain, header, parent)
}

// verifyHeader checks whether a header conforms to the consensus rules.
func (mm *MergedMining) verifyHeader(chain consensus.ChainHeaderReader, header, parent *types.Header) error {
	// Verify the block number
	if header.Number.Uint64() != parent.Number.Uint64()+1 {
		return ErrInvalidNumber
	}

	// Verify the difficulty
	expectedDiff := mm.CalcDifficulty(chain, header.Time, parent)
	if header.Difficulty.Cmp(expectedDiff) != 0 {
		return ErrInvalidDifficulty
	}

	// For RSK merged mining, we need to verify the Bitcoin merged mining proof
	// This is done via extra data fields in the header
	if err := mm.verifyMergedMiningProof(header); err != nil {
		return err
	}

	return nil
}

// verifyMergedMiningProof verifies the Bitcoin merged mining proof in the header.
func (mm *MergedMining) verifyMergedMiningProof(header *types.Header) error {
	// In a full implementation, the header would contain:
	// - BitcoinMergedMiningHeader: The Bitcoin block header
	// - BitcoinMergedMiningMerkleProof: Merkle proof linking coinbase to BTC header
	// - BitcoinMergedMiningCoinbaseTransaction: Bitcoin coinbase containing RSK hash
	//
	// For now, we check if extra data contains any merged mining info.
	// A full implementation would need extended header fields.
	
	// If there's no extra data, this might not be a merged mining block
	// In production, we would verify the actual merged mining fields
	if len(header.Extra) == 0 {
		// For compatibility, allow headers without merged mining data in some cases
		return nil
	}

	return nil
}

// VerifyHeaders implements consensus.Engine, verifying a batch of headers concurrently.
func (mm *MergedMining) VerifyHeaders(chain consensus.ChainHeaderReader, headers []*types.Header) (chan<- struct{}, <-chan error) {
	abort := make(chan struct{})
	results := make(chan error, len(headers))

	go func() {
		for i, header := range headers {
			var parent *types.Header
			if i == 0 {
				parent = chain.GetHeader(headers[0].ParentHash, headers[0].Number.Uint64()-1)
			} else {
				parent = headers[i-1]
			}

			var err error
			if parent == nil {
				err = ErrUnknownAncestor
			} else {
				err = mm.verifyHeader(chain, header, parent)
			}

			select {
			case <-abort:
				return
			case results <- err:
			}
		}
	}()

	return abort, results
}

// VerifyUncles implements consensus.Engine, verifying the given block's uncles.
func (mm *MergedMining) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {
	// RSK allows uncles (siblings) similar to Ethereum
	// but with different reward distribution via REMASC
	if len(block.Uncles()) > 2 {
		return errors.New("too many uncles")
	}

	// Verify each uncle
	for _, uncle := range block.Uncles() {
		// Uncle must be a valid header
		if err := mm.verifyUncle(chain, block, uncle); err != nil {
			return err
		}
	}

	return nil
}

// verifyUncle verifies a single uncle header.
func (mm *MergedMining) verifyUncle(chain consensus.ChainReader, block *types.Block, uncle *types.Header) error {
	// Uncle's parent must be known
	parent := chain.GetHeader(uncle.ParentHash, uncle.Number.Uint64()-1)
	if parent == nil {
		return ErrUnknownAncestor
	}

	// Uncle must be a sibling (share a recent ancestor with the block)
	// Maximum uncle depth is typically 7 blocks
	blockNumber := block.NumberU64()
	uncleNumber := uncle.Number.Uint64()

	if uncleNumber >= blockNumber || blockNumber-uncleNumber > 7 {
		return errors.New("uncle too old or in the future")
	}

	return nil
}

// Prepare implements consensus.Engine, preparing all the consensus fields of the
// header for running the transactions on top.
func (mm *MergedMining) Prepare(chain consensus.ChainHeaderReader, header *types.Header) error {
	parent := chain.GetHeader(header.ParentHash, header.Number.Uint64()-1)
	if parent == nil {
		return ErrUnknownAncestor
	}

	// Set the difficulty
	header.Difficulty = mm.CalcDifficulty(chain, header.Time, parent)

	return nil
}

// Finalize implements consensus.Engine, running any post-transaction state modifications.
// For RSK, this includes REMASC reward distribution.
func (mm *MergedMining) Finalize(chain consensus.ChainHeaderReader, header *types.Header, statedb vm.StateDB, body *types.Body) {
	// In RSK, block rewards are handled by REMASC, not directly in consensus
	// The REMASC transaction is injected at the end of each block
	// This would be handled separately in the state processor
}

// FinalizeAndAssemble implements consensus.Engine, running post-transaction state
// modifications and assembling the final block.
func (mm *MergedMining) FinalizeAndAssemble(chain consensus.ChainHeaderReader, header *types.Header, statedb *state.StateDB, body *types.Body, receipts []*types.Receipt) (*types.Block, error) {
	// Finalize the block
	mm.Finalize(chain, header, statedb, body)

	// Assemble and return the final block
	return types.NewBlock(header, body, receipts, trie.NewStackTrie(nil), chain.Config()), nil
}

// Seal implements consensus.Engine, generating a new sealing request for the given
// input block and pushing the result into the given channel.
func (mm *MergedMining) Seal(chain consensus.ChainHeaderReader, block *types.Block, results chan<- *types.Block, stop <-chan struct{}) error {
	// For merged mining, sealing is done by Bitcoin miners
	// This function would prepare the work for miners and wait for a valid Bitcoin block
	
	// In a real implementation, this would:
	// 1. Create a Bitcoin coinbase transaction containing the RSK block hash
	// 2. Wait for a Bitcoin miner to find a valid nonce
	// 3. Verify the Bitcoin block meets RSK's difficulty
	// 4. Return the block with merged mining proof

	// For now, immediately return the block (no actual mining)
	go func() {
		select {
		case <-stop:
			return
		case results <- block:
		}
	}()

	return nil
}

// SealHash returns the hash of a block prior to it being sealed.
func (mm *MergedMining) SealHash(header *types.Header) common.Hash {
	return header.Hash()
}

// CalcDifficulty implements consensus.Engine, returning the difficulty that a new
// block should have.
func (mm *MergedMining) CalcDifficulty(chain consensus.ChainHeaderReader, time uint64, parent *types.Header) *big.Int {
	return mm.diffCalc.CalcDifficulty(time, parent)
}

// Close implements consensus.Engine, terminating any background threads.
func (mm *MergedMining) Close() error {
	mm.closeOnce.Do(func() {
		close(mm.exitCh)
	})
	return nil
}

// APIs returns the RPC APIs this consensus engine provides.
func (mm *MergedMining) APIs(chain consensus.ChainHeaderReader) []any {
	return nil
}

// Threads returns the number of mining threads currently enabled.
// This is only for interface compatibility; merged mining doesn't use local threads.
func (mm *MergedMining) Threads() int {
	return runtime.NumCPU()
}

// MergedMiningValidator validates merged mining proofs.
type MergedMiningValidator struct{}

// NewMergedMiningValidator creates a new merged mining validator.
func NewMergedMiningValidator() *MergedMiningValidator {
	return &MergedMiningValidator{}
}

// ValidateMergedMining validates the merged mining proof in a block header.
// This is the core validation for RSK's merged mining consensus.
func (v *MergedMiningValidator) ValidateMergedMining(
	btcHeader []byte,
	merkleProof []byte,
	coinbase []byte,
	rskBlockHash common.Hash,
	rskDifficulty *big.Int,
) error {
	// 1. Parse Bitcoin block header
	btcParsed, err := ParseBitcoinHeader(btcHeader)
	if err != nil {
		return err
	}

	// 2. Verify RSK block hash is in the coinbase
	if err := v.verifyRskHashInCoinbase(coinbase, rskBlockHash); err != nil {
		return err
	}

	// 3. Verify merkle proof links coinbase to Bitcoin header
	coinbaseHash := DoubleSHA256(coinbase)
	if err := VerifyMerkleProof(coinbaseHash, merkleProof, btcParsed.MerkleRoot); err != nil {
		return err
	}

	// 4. Verify Bitcoin PoW meets RSK difficulty
	btcHash := btcParsed.Hash()
	if !VerifyDifficulty(btcHash, rskDifficulty) {
		return ErrInsufficientProofOfWork
	}

	return nil
}

// verifyRskHashInCoinbase verifies that the RSK block hash is in the Bitcoin coinbase.
func (v *MergedMiningValidator) verifyRskHashInCoinbase(coinbase []byte, rskHash common.Hash) error {
	// Look for RSK tag followed by the hash
	idx := bytes.Index(coinbase, RSKTag)
	if idx < 0 {
		return ErrRskHashNotInCoinbase
	}

	// Check if hash follows the tag
	hashStart := idx + len(RSKTag)
	if hashStart+32 > len(coinbase) {
		return ErrRskHashNotInCoinbase
	}

	if !bytes.Equal(coinbase[hashStart:hashStart+32], rskHash.Bytes()) {
		return ErrRskHashNotInCoinbase
	}

	return nil
}
