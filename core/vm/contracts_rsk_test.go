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

package vm

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// TestRSKBridgePrecompile tests the RSK Bridge precompile.
func TestRSKBridgePrecompile(t *testing.T) {
	bridge := &rskBridge{}

	tests := []struct {
		name    string
		input   []byte
		wantErr bool
	}{
		{
			name:    "empty input",
			input:   []byte{},
			wantErr: true,
		},
		{
			name:    "too short input",
			input:   []byte{0x01, 0x02, 0x03},
			wantErr: true,
		},
		{
			name:  "getBtcBlockchainBestChainHeight selector",
			input: bridgeSelectorGetBtcBlockchainBestChainHeight,
		},
		{
			name:  "getFederationAddress selector",
			input: bridgeSelectorGetFederationAddress,
		},
		{
			name:  "getFederationSize selector",
			input: bridgeSelectorGetFederationSize,
		},
		{
			name:  "getFederationThreshold selector",
			input: bridgeSelectorGetFederationThreshold,
		},
		{
			name:  "getMinimumPegInValue selector",
			input: bridgeSelectorGetMinimumPegInValue,
		},
		{
			name:  "getMinimumPegOutValue selector",
			input: bridgeSelectorGetMinimumPegOutValue,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := bridge.Run(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("rskBridge.Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestRSKBridgeGas tests gas calculation for Bridge precompile.
func TestRSKBridgeGas(t *testing.T) {
	bridge := &rskBridge{}

	tests := []struct {
		name     string
		input    []byte
		expected uint64
	}{
		{
			name:     "empty input - base gas",
			input:    []byte{},
			expected: 100000, // RSKBridgeBaseGas
		},
		{
			name:     "read operation - getBtcBlockchainBestChainHeight",
			input:    bridgeSelectorGetBtcBlockchainBestChainHeight,
			expected: 10000, // RSKBridgeReadGas
		},
		{
			name:     "state query operation",
			input:    bridgeSelectorGetStateForBtcReleaseClient,
			expected: 50000, // RSKBridgeStateQueryGas
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := bridge.RequiredGas(tt.input)
			if got != tt.expected {
				t.Errorf("rskBridge.RequiredGas() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestRSKRemascPrecompile tests the RSK REMASC precompile.
func TestRSKRemascPrecompile(t *testing.T) {
	remasc := &rskRemasc{}

	tests := []struct {
		name    string
		input   []byte
		wantErr bool
	}{
		{
			name:  "empty input - end of block call",
			input: []byte{},
		},
		{
			name:  "getRewardBalance selector",
			input: remascSelectorGetRewardBalance,
		},
		{
			name:  "getBurnedBalance selector",
			input: remascSelectorGetBurnedBalance,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := remasc.Run(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("rskRemasc.Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestRSKHDWalletUtilsPrecompile tests the RSK HD Wallet Utils precompile.
func TestRSKHDWalletUtilsPrecompile(t *testing.T) {
	hdWallet := &rskHDWalletUtils{}

	tests := []struct {
		name    string
		input   []byte
		wantErr bool
	}{
		{
			name:    "empty input",
			input:   []byte{},
			wantErr: true,
		},
		{
			name:    "too short input",
			input:   []byte{0x01, 0x02, 0x03},
			wantErr: true,
		},
		{
			name:    "derivation with insufficient data",
			input:   append(hdWalletSelectorDeriveExtendedPubKey, make([]byte, 10)...),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := hdWallet.Run(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("rskHDWalletUtils.Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestRSKBlockHashQueryPrecompile tests the RSK Block Hash Query precompile.
func TestRSKBlockHashQueryPrecompile(t *testing.T) {
	blockHash := &rskBlockHashQuery{}

	tests := []struct {
		name    string
		input   []byte
		wantErr bool
	}{
		{
			name:    "empty input",
			input:   []byte{},
			wantErr: true,
		},
		{
			name:  "block number as 32 bytes (legacy format)",
			input: common.LeftPadBytes(big.NewInt(100).Bytes(), 32),
		},
		{
			name:  "getBlockHash selector with block number",
			input: append(blockHashSelectorGetBlockHash, common.LeftPadBytes(big.NewInt(100).Bytes(), 32)...),
		},
		{
			name:  "getDepth selector",
			input: blockHashSelectorGetDepth,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := blockHash.Run(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("rskBlockHashQuery.Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestMatchRSKSelector tests the selector matching function.
func TestMatchRSKSelector(t *testing.T) {
	tests := []struct {
		name     string
		a        []byte
		b        []byte
		expected bool
	}{
		{
			name:     "matching selectors",
			a:        []byte{0x01, 0x02, 0x03, 0x04},
			b:        []byte{0x01, 0x02, 0x03, 0x04},
			expected: true,
		},
		{
			name:     "non-matching selectors",
			a:        []byte{0x01, 0x02, 0x03, 0x04},
			b:        []byte{0x05, 0x06, 0x07, 0x08},
			expected: false,
		},
		{
			name:     "short selector a",
			a:        []byte{0x01, 0x02},
			b:        []byte{0x01, 0x02, 0x03, 0x04},
			expected: false,
		},
		{
			name:     "short selector b",
			a:        []byte{0x01, 0x02, 0x03, 0x04},
			b:        []byte{0x01, 0x02},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchRSKSelector(tt.a, tt.b)
			if got != tt.expected {
				t.Errorf("matchRSKSelector() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestBase58Encode tests Base58 encoding.
func TestBase58Encode(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name:     "empty input",
			input:    []byte{},
			expected: "",
		},
		{
			name:     "single byte zero",
			input:    []byte{0x00},
			expected: "1",
		},
		{
			name:     "hello world",
			input:    []byte("hello"),
			expected: "Cn8eVZg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(base58Encode(tt.input))
			if got != tt.expected {
				t.Errorf("base58Encode() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestRSKPrecompiledAddresses tests that RSK precompiled addresses are correctly set.
func TestRSKPrecompiledAddresses(t *testing.T) {
	expectedAddresses := map[common.Address]string{
		RSKBridgePrecompileAddr:        "RSK_BRIDGE",
		RSKRemascPrecompileAddr:        "RSK_REMASC",
		RSKHDWalletUtilsPrecompileAddr: "RSK_HD_WALLET_UTILS",
		RSKBlockHashPrecompileAddr:     "RSK_BLOCK_HASH_QUERY",
	}

	for addr, name := range expectedAddresses {
		contract, ok := PrecompiledContractsRSK[addr]
		if !ok {
			t.Errorf("RSK precompile %s not found at address %s", name, addr.Hex())
			continue
		}
		if contract.Name() != name {
			t.Errorf("RSK precompile at %s has name %s, want %s", addr.Hex(), contract.Name(), name)
		}
	}
}

// TestHDWalletCompressPublicKey tests public key compression.
func TestHDWalletCompressPublicKey(t *testing.T) {
	hdWallet := &rskHDWalletUtils{}

	// Test with a sample uncompressed public key (64 bytes)
	// x coordinate: 32 bytes, y coordinate: 32 bytes
	pubKey := make([]byte, 64)
	// Set some test values
	pubKey[0] = 0x01  // x coordinate first byte
	pubKey[31] = 0x02 // x coordinate last byte
	pubKey[32] = 0x03 // y coordinate first byte
	pubKey[63] = 0x00 // y coordinate last byte (even)

	compressed := hdWallet.compressPublicKey(pubKey)
	if compressed == nil {
		t.Fatal("compressPublicKey returned nil")
	}

	if len(compressed) != 33 {
		t.Errorf("compressed key length = %d, want 33", len(compressed))
	}

	// Since y is even (ends with 0x00), prefix should be 0x02
	if compressed[0] != 0x02 {
		t.Errorf("compressed key prefix = %02x, want 0x02", compressed[0])
	}

	// Test with odd y coordinate
	pubKey[63] = 0x01 // y coordinate last byte (odd)
	compressed = hdWallet.compressPublicKey(pubKey)

	// Since y is odd, prefix should be 0x03
	if compressed[0] != 0x03 {
		t.Errorf("compressed key prefix = %02x, want 0x03", compressed[0])
	}
}

// TestHMacSHA512 tests HMAC-SHA512 function.
func TestHMacSHA512(t *testing.T) {
	key := []byte("test key")
	data := []byte("test data")

	result := hmacSHA512(key, data)
	if len(result) != 64 {
		t.Errorf("hmacSHA512 result length = %d, want 64", len(result))
	}

	// Same key and data should produce same result
	result2 := hmacSHA512(key, data)
	if !bytes.Equal(result, result2) {
		t.Error("hmacSHA512 should be deterministic")
	}

	// Different data should produce different result
	result3 := hmacSHA512(key, []byte("different data"))
	if bytes.Equal(result, result3) {
		t.Error("hmacSHA512 should produce different results for different data")
	}
}
