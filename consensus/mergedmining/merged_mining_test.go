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
	"bytes"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// TestParseBitcoinHeader tests Bitcoin header parsing.
func TestParseBitcoinHeader(t *testing.T) {
	// Create a mock 80-byte Bitcoin header
	data := make([]byte, BitcoinHeaderSize)

	// Set version (4 bytes, little-endian)
	data[0] = 0x01
	data[1] = 0x00
	data[2] = 0x00
	data[3] = 0x00

	// Set timestamp (at offset 68, 4 bytes)
	data[68] = 0x01
	data[69] = 0x02
	data[70] = 0x03
	data[71] = 0x04

	// Set bits (at offset 72, 4 bytes)
	data[72] = 0xff
	data[73] = 0xff
	data[74] = 0x00
	data[75] = 0x1d

	// Set nonce (at offset 76, 4 bytes)
	data[76] = 0x00
	data[77] = 0x01
	data[78] = 0x02
	data[79] = 0x03

	header, err := ParseBitcoinHeader(data)
	if err != nil {
		t.Fatalf("ParseBitcoinHeader failed: %v", err)
	}

	if header.Version != 1 {
		t.Errorf("Version = %d, want 1", header.Version)
	}

	// Test with too short data
	_, err = ParseBitcoinHeader(data[:79])
	if err != ErrInvalidBitcoinHeader {
		t.Errorf("expected ErrInvalidBitcoinHeader for short data, got %v", err)
	}
}

// TestBitcoinHeaderSerialize tests Bitcoin header serialization.
func TestBitcoinHeaderSerialize(t *testing.T) {
	header := &BitcoinHeader{
		Version:   1,
		Timestamp: 0x04030201,
		Bits:      0x1d00ffff,
		Nonce:     0x03020100,
	}

	data := header.Serialize()
	if len(data) != BitcoinHeaderSize {
		t.Errorf("serialized length = %d, want %d", len(data), BitcoinHeaderSize)
	}

	// Parse it back
	parsed, err := ParseBitcoinHeader(data)
	if err != nil {
		t.Fatalf("ParseBitcoinHeader failed: %v", err)
	}

	if parsed.Version != header.Version {
		t.Errorf("Version = %d, want %d", parsed.Version, header.Version)
	}
	if parsed.Timestamp != header.Timestamp {
		t.Errorf("Timestamp = %d, want %d", parsed.Timestamp, header.Timestamp)
	}
	if parsed.Bits != header.Bits {
		t.Errorf("Bits = %d, want %d", parsed.Bits, header.Bits)
	}
	if parsed.Nonce != header.Nonce {
		t.Errorf("Nonce = %d, want %d", parsed.Nonce, header.Nonce)
	}
}

// TestDoubleSHA256 tests double SHA256 hashing.
func TestDoubleSHA256(t *testing.T) {
	data := []byte("test data")
	hash := DoubleSHA256(data)

	// Hash should be 32 bytes
	if len(hash) != 32 {
		t.Errorf("hash length = %d, want 32", len(hash))
	}

	// Same data should produce same hash
	hash2 := DoubleSHA256(data)
	if hash != hash2 {
		t.Error("DoubleSHA256 should be deterministic")
	}

	// Different data should produce different hash
	hash3 := DoubleSHA256([]byte("different data"))
	if hash == hash3 {
		t.Error("DoubleSHA256 should produce different hashes for different data")
	}
}

// TestCompactToBig tests compact target to big.Int conversion.
func TestCompactToBig(t *testing.T) {
	tests := []struct {
		name    string
		compact uint32
		want    *big.Int
	}{
		{
			name:    "bitcoin genesis difficulty",
			compact: 0x1d00ffff,
			want:    new(big.Int).Lsh(big.NewInt(0xffff), 8*(0x1d-3)),
		},
		{
			name:    "zero",
			compact: 0,
			want:    big.NewInt(0),
		},
		{
			name:    "small value",
			compact: 0x03123456,
			want:    big.NewInt(0x123456),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CompactToBig(tt.compact)
			if got.Cmp(tt.want) != 0 {
				t.Errorf("CompactToBig(%x) = %v, want %v", tt.compact, got, tt.want)
			}
		})
	}
}

// TestReverseBytes tests byte reversal.
func TestReverseBytes(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected []byte
	}{
		{
			name:     "simple",
			input:    []byte{0x01, 0x02, 0x03, 0x04},
			expected: []byte{0x04, 0x03, 0x02, 0x01},
		},
		{
			name:     "empty",
			input:    []byte{},
			expected: []byte{},
		},
		{
			name:     "single byte",
			input:    []byte{0xff},
			expected: []byte{0xff},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ReverseBytes(tt.input)
			if !bytes.Equal(got, tt.expected) {
				t.Errorf("ReverseBytes() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestDifficultyCalculator tests difficulty calculation.
func TestDifficultyCalculator(t *testing.T) {
	dc := NewDifficultyCalculator(DifficultyClassic)

	parent := &types.Header{
		Difficulty: big.NewInt(1000000),
		Time:       1000,
		Number:     big.NewInt(100),
	}

	tests := []struct {
		name      string
		timestamp uint64
		wantInc   bool // true if difficulty should increase
	}{
		{
			name:      "fast block - should increase",
			timestamp: 1010, // 10 seconds
			wantInc:   true,
		},
		{
			name:      "normal block - minimal change",
			timestamp: 1030, // 30 seconds (target)
			wantInc:   false,
		},
		{
			name:      "slow block - should decrease",
			timestamp: 1090, // 90 seconds
			wantInc:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff := dc.CalcDifficulty(tt.timestamp, parent)

			if diff.Sign() <= 0 {
				t.Error("difficulty should be positive")
			}

			if tt.wantInc && diff.Cmp(parent.Difficulty) <= 0 {
				t.Errorf("difficulty should increase, got %v (parent: %v)", diff, parent.Difficulty)
			}
		})
	}
}

// TestDifficultyToTarget tests difficulty to target conversion.
func TestDifficultyToTarget(t *testing.T) {
	tests := []struct {
		name       string
		difficulty *big.Int
	}{
		{
			name:       "normal difficulty",
			difficulty: big.NewInt(1000000),
		},
		{
			name:       "high difficulty",
			difficulty: new(big.Int).Exp(big.NewInt(10), big.NewInt(20), nil),
		},
		{
			name:       "low difficulty",
			difficulty: big.NewInt(1),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := DifficultyToTarget(tt.difficulty)

			// Target should be inversely proportional to difficulty
			if target.Sign() <= 0 {
				t.Error("target should be positive")
			}

			// Convert back
			diff := TargetToDifficulty(target)

			// Due to integer division, the round-trip may not be exact
			// but should be close
			ratio := new(big.Float).Quo(
				new(big.Float).SetInt(diff),
				new(big.Float).SetInt(tt.difficulty),
			)
			ratioFloat, _ := ratio.Float64()

			// Allow 1% tolerance due to integer division
			if ratioFloat < 0.99 || ratioFloat > 1.01 {
				t.Errorf("round-trip ratio = %v, want ~1.0", ratioFloat)
			}
		})
	}
}

// TestVerifyDifficulty tests PoW verification.
func TestVerifyDifficulty(t *testing.T) {
	// High difficulty = low target = hard to find valid hash
	highDiff := new(big.Int).Exp(big.NewInt(10), big.NewInt(30), nil)

	// Low difficulty = high target = easy to find valid hash
	lowDiff := big.NewInt(1)

	// A hash of all zeros should pass low difficulty
	zeroHash := [32]byte{}
	if !VerifyDifficulty(zeroHash, lowDiff) {
		t.Error("zero hash should pass low difficulty")
	}

	// A hash of all 0xff should fail high difficulty
	maxHash := [32]byte{}
	for i := range maxHash {
		maxHash[i] = 0xff
	}
	if VerifyDifficulty(maxHash, highDiff) {
		t.Error("max hash should fail high difficulty")
	}
}

// TestMergedMiningNew tests merged mining engine creation.
func TestMergedMiningNew(t *testing.T) {
	// Test with nil config
	mm := New(nil)
	if mm == nil {
		t.Fatal("New(nil) returned nil")
	}
	if mm.config == nil {
		t.Error("config should be set to default")
	}

	// Test with custom config
	config := &Config{
		DifficultyAlgorithm: DifficultyIris,
	}
	mm = New(config)
	if mm.config.DifficultyAlgorithm != DifficultyIris {
		t.Error("custom config not applied")
	}
}

// TestMergedMiningAuthor tests Author method.
func TestMergedMiningAuthor(t *testing.T) {
	mm := New(nil)

	coinbase := common.HexToAddress("0x1234567890123456789012345678901234567890")
	header := &types.Header{
		Coinbase: coinbase,
	}

	author, err := mm.Author(header)
	if err != nil {
		t.Errorf("Author() error = %v", err)
	}
	if author != coinbase {
		t.Errorf("Author() = %v, want %v", author, coinbase)
	}
}

// TestMergedMiningClose tests Close method.
func TestMergedMiningClose(t *testing.T) {
	mm := New(nil)

	err := mm.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Second close should also succeed
	err = mm.Close()
	if err != nil {
		t.Errorf("second Close() error = %v", err)
	}
}

// TestRSKTagPresence tests that RSKTag is correctly defined.
func TestRSKTagPresence(t *testing.T) {
	expectedTag := []byte("RSKBLOCK:")
	if !bytes.Equal(RSKTag, expectedTag) {
		t.Errorf("RSKTag = %v, want %v", RSKTag, expectedTag)
	}
}

// TestMergedMiningValidator tests the merged mining validator.
func TestMergedMiningValidator(t *testing.T) {
	validator := NewMergedMiningValidator()
	if validator == nil {
		t.Fatal("NewMergedMiningValidator returned nil")
	}
}

// TestVerifyRskHashInCoinbase tests RSK hash verification in coinbase.
func TestVerifyRskHashInCoinbase(t *testing.T) {
	validator := NewMergedMiningValidator()

	rskHash := common.HexToHash("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")

	// Create valid coinbase with RSK tag and hash
	validCoinbase := append([]byte("some data"), RSKTag...)
	validCoinbase = append(validCoinbase, rskHash.Bytes()...)
	validCoinbase = append(validCoinbase, []byte("more data")...)

	err := validator.verifyRskHashInCoinbase(validCoinbase, rskHash)
	if err != nil {
		t.Errorf("valid coinbase verification failed: %v", err)
	}

	// Coinbase without RSK tag should fail
	invalidCoinbase := []byte("no rsk tag here")
	err = validator.verifyRskHashInCoinbase(invalidCoinbase, rskHash)
	if err != ErrRskHashNotInCoinbase {
		t.Errorf("expected ErrRskHashNotInCoinbase, got %v", err)
	}

	// Coinbase with wrong hash should fail
	wrongHash := common.HexToHash("0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890")
	err = validator.verifyRskHashInCoinbase(validCoinbase, wrongHash)
	if err != ErrRskHashNotInCoinbase {
		t.Errorf("expected ErrRskHashNotInCoinbase for wrong hash, got %v", err)
	}
}
