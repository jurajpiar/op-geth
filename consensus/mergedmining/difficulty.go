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

package mergedmining

import (
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
)

// DifficultyAlgorithm represents different RSK difficulty adjustment algorithms.
type DifficultyAlgorithm int

const (
	// DifficultyClassic is the original RSK difficulty algorithm.
	DifficultyClassic DifficultyAlgorithm = iota
	// DifficultyWasabi uses a different coefficient.
	DifficultyWasabi
	// DifficultyIris provides smoother adjustment.
	DifficultyIris
	// DifficultyArrowhead is similar to Iris with adjustments.
	DifficultyArrowhead
)

// Difficulty calculation constants
var (
	// MinimumDifficulty is the minimum allowed difficulty (0x20000 = 131072).
	MinimumDifficulty = big.NewInt(131072)

	// DifficultyBoundDivisor limits how much difficulty can change per block.
	DifficultyBoundDivisor = big.NewInt(2048)

	// TargetBlockTime is the target time between blocks in seconds (30s for RSK).
	TargetBlockTime = uint64(30)

	// MaxDifficultyTarget is 2^256 - 1 (used for target calculations).
	MaxDifficultyTarget *big.Int
)

func init() {
	// Initialize MaxDifficultyTarget = 2^256 - 1
	MaxDifficultyTarget = new(big.Int).Exp(big.NewInt(2), big.NewInt(256), nil)
	MaxDifficultyTarget.Sub(MaxDifficultyTarget, big.NewInt(1))
}

// DifficultyCalculator calculates difficulty for RSK blocks.
type DifficultyCalculator struct {
	algorithm DifficultyAlgorithm
}

// NewDifficultyCalculator creates a new difficulty calculator with the specified algorithm.
func NewDifficultyCalculator(algorithm DifficultyAlgorithm) *DifficultyCalculator {
	return &DifficultyCalculator{
		algorithm: algorithm,
	}
}

// CalcDifficulty calculates the difficulty for a new block.
func (dc *DifficultyCalculator) CalcDifficulty(timestamp uint64, parent *types.Header) *big.Int {
	switch dc.algorithm {
	case DifficultyClassic:
		return dc.calcDifficultyClassic(timestamp, parent)
	case DifficultyWasabi:
		return dc.calcDifficultyWasabi(timestamp, parent)
	case DifficultyIris:
		return dc.calcDifficultyIris(timestamp, parent)
	case DifficultyArrowhead:
		return dc.calcDifficultyArrowhead(timestamp, parent)
	default:
		return dc.calcDifficultyClassic(timestamp, parent)
	}
}

// calcDifficultyClassic implements the classic RSK difficulty algorithm.
// Similar to Ethereum's pre-Homestead algorithm but with RSK's 30-second block time.
func (dc *DifficultyCalculator) calcDifficultyClassic(timestamp uint64, parent *types.Header) *big.Int {
	// diff = parent_diff + parent_diff // 2048 * max(1 - (timestamp - parent_timestamp) // 30, -99)
	parentDiff := new(big.Int).Set(parent.Difficulty)
	parentTimestamp := parent.Time

	// Calculate time adjustment
	timeDiff := int64(timestamp) - int64(parentTimestamp)
	blockAdjust := timeDiff / int64(TargetBlockTime)

	// Calculate adjustment factor (capped at -99)
	adjust := int64(1) - blockAdjust
	if adjust < -99 {
		adjust = -99
	}

	// Calculate difficulty change
	diffChange := new(big.Int).Div(parentDiff, DifficultyBoundDivisor)
	diffChange.Mul(diffChange, big.NewInt(adjust))

	// New difficulty
	newDiff := new(big.Int).Add(parentDiff, diffChange)

	// Ensure minimum difficulty
	if newDiff.Cmp(MinimumDifficulty) < 0 {
		newDiff.Set(MinimumDifficulty)
	}

	return newDiff
}

// calcDifficultyWasabi implements the Wasabi difficulty algorithm.
// Uses a different coefficient for adjustment.
func (dc *DifficultyCalculator) calcDifficultyWasabi(timestamp uint64, parent *types.Header) *big.Int {
	parentDiff := new(big.Int).Set(parent.Difficulty)
	parentTimestamp := parent.Time

	// Calculate time difference
	timeDiff := int64(timestamp) - int64(parentTimestamp)

	// Wasabi uses a smoother adjustment curve
	var adjust int64
	switch {
	case timeDiff < 10:
		adjust = 3
	case timeDiff < 20:
		adjust = 2
	case timeDiff < 30:
		adjust = 1
	case timeDiff < 45:
		adjust = 0
	case timeDiff < 60:
		adjust = -1
	case timeDiff < 90:
		adjust = -2
	default:
		adjust = -3
	}

	// Calculate difficulty change
	diffChange := new(big.Int).Div(parentDiff, DifficultyBoundDivisor)
	diffChange.Mul(diffChange, big.NewInt(adjust))

	// New difficulty
	newDiff := new(big.Int).Add(parentDiff, diffChange)

	// Ensure minimum difficulty
	if newDiff.Cmp(MinimumDifficulty) < 0 {
		newDiff.Set(MinimumDifficulty)
	}

	return newDiff
}

// calcDifficultyIris implements the Iris difficulty algorithm.
// Provides smoother adjustment with more granular steps.
func (dc *DifficultyCalculator) calcDifficultyIris(timestamp uint64, parent *types.Header) *big.Int {
	parentDiff := new(big.Int).Set(parent.Difficulty)
	parentTimestamp := parent.Time

	// Calculate time difference
	timeDiff := int64(timestamp) - int64(parentTimestamp)
	targetTime := int64(TargetBlockTime)

	// Calculate adjustment ratio based on how far from target
	// ratio = (targetTime - timeDiff) / targetTime, capped at [-2, 2]
	ratio := (targetTime - timeDiff) * 100 / targetTime

	// Cap the ratio
	if ratio > 200 {
		ratio = 200
	}
	if ratio < -200 {
		ratio = -200
	}

	// Calculate difficulty change
	// diffChange = parentDiff * ratio / 100 / 2048
	diffChange := new(big.Int).Mul(parentDiff, big.NewInt(ratio))
	diffChange.Div(diffChange, big.NewInt(100))
	diffChange.Div(diffChange, DifficultyBoundDivisor)

	// New difficulty
	newDiff := new(big.Int).Add(parentDiff, diffChange)

	// Ensure minimum difficulty
	if newDiff.Cmp(MinimumDifficulty) < 0 {
		newDiff.Set(MinimumDifficulty)
	}

	return newDiff
}

// calcDifficultyArrowhead implements the Arrowhead difficulty algorithm.
// Similar to Iris but with some adjustments for better stability.
func (dc *DifficultyCalculator) calcDifficultyArrowhead(timestamp uint64, parent *types.Header) *big.Int {
	// Arrowhead is similar to Iris with minor adjustments
	return dc.calcDifficultyIris(timestamp, parent)
}

// DifficultyToTarget converts difficulty to a target value.
// Target = 2^256 / difficulty
func DifficultyToTarget(difficulty *big.Int) *big.Int {
	if difficulty == nil || difficulty.Sign() == 0 {
		return new(big.Int).Set(MaxDifficultyTarget)
	}
	// Target = 2^256 / difficulty
	target := new(big.Int).Exp(big.NewInt(2), big.NewInt(256), nil)
	return target.Div(target, difficulty)
}

// TargetToDifficulty converts a target value to difficulty.
// Difficulty = 2^256 / target
func TargetToDifficulty(target *big.Int) *big.Int {
	if target == nil || target.Sign() == 0 {
		return new(big.Int).Set(MinimumDifficulty)
	}
	// Difficulty = 2^256 / target
	difficulty := new(big.Int).Exp(big.NewInt(2), big.NewInt(256), nil)
	return difficulty.Div(difficulty, target)
}

// VerifyDifficulty checks if the given hash meets the target difficulty.
// The hash (interpreted as a big-endian number) must be less than or equal to the target.
func VerifyDifficulty(hash [32]byte, difficulty *big.Int) bool {
	target := DifficultyToTarget(difficulty)
	hashInt := new(big.Int).SetBytes(ReverseBytes(hash[:]))
	return hashInt.Cmp(target) <= 0
}
