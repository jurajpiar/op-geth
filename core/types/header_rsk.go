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

package types

import (
	"math/big"
)

// RSKHeaderExtension contains RSK-specific header fields.
// These fields are stored in the header's Extra field for RSK blocks.
type RSKHeaderExtension struct {
	// PaidFees is the total fees paid in this block
	PaidFees *big.Int

	// MinimumGasPrice is the minimum gas price for transactions in this block
	MinimumGasPrice *big.Int

	// UncleCount is the number of uncles included in this block
	UncleCount uint32

	// Bitcoin merged mining fields
	BitcoinMergedMiningHeader              []byte // 80-byte Bitcoin block header
	BitcoinMergedMiningMerkleProof         []byte // Merkle proof linking coinbase to header
	BitcoinMergedMiningCoinbaseTransaction []byte // Bitcoin coinbase containing RSK hash

	// Optional fields
	UmmRoot                  []byte  // UMM root (RSKIP-92)
	TxExecutionSublistsEdges []int16 // RSKIP-144 parallel execution edges

	// Header version (RSKIP-351: 0 for V0, 1 for V1)
	Version byte

	// UseRskip92Encoding indicates whether to use compressed encoding
	UseRskip92Encoding bool
}

// NewRSKHeaderExtension creates a new RSK header extension with default values.
func NewRSKHeaderExtension() *RSKHeaderExtension {
	return &RSKHeaderExtension{
		PaidFees:        new(big.Int),
		MinimumGasPrice: new(big.Int),
		UncleCount:      0,
		Version:         0,
	}
}

// RSKHeader wraps a standard Header with RSK-specific extensions.
type RSKHeader struct {
	*Header
	RSK *RSKHeaderExtension
}

// NewRSKHeader creates a new RSK header from a standard header.
func NewRSKHeader(h *Header) *RSKHeader {
	return &RSKHeader{
		Header: h,
		RSK:    NewRSKHeaderExtension(),
	}
}

// HasMergedMiningFields returns true if this header has Bitcoin merged mining data.
func (ext *RSKHeaderExtension) HasMergedMiningFields() bool {
	return len(ext.BitcoinMergedMiningCoinbaseTransaction) > 0 ||
		len(ext.BitcoinMergedMiningHeader) > 0 ||
		len(ext.BitcoinMergedMiningMerkleProof) > 0
}

// SetPaidFees sets the total paid fees for the block.
func (ext *RSKHeaderExtension) SetPaidFees(fees *big.Int) {
	if fees != nil {
		ext.PaidFees = new(big.Int).Set(fees)
	} else {
		ext.PaidFees = new(big.Int)
	}
}

// SetMinimumGasPrice sets the minimum gas price for the block.
func (ext *RSKHeaderExtension) SetMinimumGasPrice(price *big.Int) {
	if price != nil {
		ext.MinimumGasPrice = new(big.Int).Set(price)
	} else {
		ext.MinimumGasPrice = new(big.Int)
	}
}

// SetMergedMiningData sets the Bitcoin merged mining fields.
func (ext *RSKHeaderExtension) SetMergedMiningData(header, proof, coinbase []byte) {
	if header != nil {
		ext.BitcoinMergedMiningHeader = make([]byte, len(header))
		copy(ext.BitcoinMergedMiningHeader, header)
	}
	if proof != nil {
		ext.BitcoinMergedMiningMerkleProof = make([]byte, len(proof))
		copy(ext.BitcoinMergedMiningMerkleProof, proof)
	}
	if coinbase != nil {
		ext.BitcoinMergedMiningCoinbaseTransaction = make([]byte, len(coinbase))
		copy(ext.BitcoinMergedMiningCoinbaseTransaction, coinbase)
	}
}

// Copy creates a deep copy of the RSK header extension.
func (ext *RSKHeaderExtension) Copy() *RSKHeaderExtension {
	cpy := &RSKHeaderExtension{
		UncleCount:         ext.UncleCount,
		Version:            ext.Version,
		UseRskip92Encoding: ext.UseRskip92Encoding,
	}

	if ext.PaidFees != nil {
		cpy.PaidFees = new(big.Int).Set(ext.PaidFees)
	}
	if ext.MinimumGasPrice != nil {
		cpy.MinimumGasPrice = new(big.Int).Set(ext.MinimumGasPrice)
	}
	if len(ext.BitcoinMergedMiningHeader) > 0 {
		cpy.BitcoinMergedMiningHeader = make([]byte, len(ext.BitcoinMergedMiningHeader))
		copy(cpy.BitcoinMergedMiningHeader, ext.BitcoinMergedMiningHeader)
	}
	if len(ext.BitcoinMergedMiningMerkleProof) > 0 {
		cpy.BitcoinMergedMiningMerkleProof = make([]byte, len(ext.BitcoinMergedMiningMerkleProof))
		copy(cpy.BitcoinMergedMiningMerkleProof, ext.BitcoinMergedMiningMerkleProof)
	}
	if len(ext.BitcoinMergedMiningCoinbaseTransaction) > 0 {
		cpy.BitcoinMergedMiningCoinbaseTransaction = make([]byte, len(ext.BitcoinMergedMiningCoinbaseTransaction))
		copy(cpy.BitcoinMergedMiningCoinbaseTransaction, ext.BitcoinMergedMiningCoinbaseTransaction)
	}
	if len(ext.UmmRoot) > 0 {
		cpy.UmmRoot = make([]byte, len(ext.UmmRoot))
		copy(cpy.UmmRoot, ext.UmmRoot)
	}
	if len(ext.TxExecutionSublistsEdges) > 0 {
		cpy.TxExecutionSublistsEdges = make([]int16, len(ext.TxExecutionSublistsEdges))
		copy(cpy.TxExecutionSublistsEdges, ext.TxExecutionSublistsEdges)
	}

	return cpy
}

// RSKBlockData contains RSK-specific block data beyond the header.
type RSKBlockData struct {
	// Siblings (uncles) in RSK contribute to REMASC rewards
	Siblings []*Header
}

// NewRSKBlockData creates a new RSK block data container.
func NewRSKBlockData() *RSKBlockData {
	return &RSKBlockData{
		Siblings: make([]*Header, 0),
	}
}

// AddSibling adds a sibling (uncle) to the block data.
func (bd *RSKBlockData) AddSibling(sibling *Header) {
	bd.Siblings = append(bd.Siblings, sibling)
}

// RSKExtraDataCodec provides encoding/decoding for RSK extra data.
type RSKExtraDataCodec struct{}

// EncodeRSKExtension encodes RSK header extension to bytes for storage in Extra field.
// This is a simplified encoding - a full implementation would use RLP.
func (c *RSKExtraDataCodec) EncodeRSKExtension(ext *RSKHeaderExtension) []byte {
	if ext == nil {
		return nil
	}

	// Simple encoding format:
	// [version:1][paidFees:32][minGasPrice:32][uncleCount:4][mmHeaderLen:4][mmHeader:N]...
	// For a full implementation, use proper RLP encoding

	var data []byte

	// Version byte
	data = append(data, ext.Version)

	// Paid fees (32 bytes, big-endian)
	feeBytes := make([]byte, 32)
	if ext.PaidFees != nil {
		ext.PaidFees.FillBytes(feeBytes)
	}
	data = append(data, feeBytes...)

	// Minimum gas price (32 bytes, big-endian)
	priceBytes := make([]byte, 32)
	if ext.MinimumGasPrice != nil {
		ext.MinimumGasPrice.FillBytes(priceBytes)
	}
	data = append(data, priceBytes...)

	// Uncle count (4 bytes)
	uncleBytes := make([]byte, 4)
	uncleBytes[0] = byte(ext.UncleCount >> 24)
	uncleBytes[1] = byte(ext.UncleCount >> 16)
	uncleBytes[2] = byte(ext.UncleCount >> 8)
	uncleBytes[3] = byte(ext.UncleCount)
	data = append(data, uncleBytes...)

	// Bitcoin merged mining header (length-prefixed)
	mmHeaderLen := make([]byte, 4)
	mmHeaderLen[0] = byte(len(ext.BitcoinMergedMiningHeader) >> 24)
	mmHeaderLen[1] = byte(len(ext.BitcoinMergedMiningHeader) >> 16)
	mmHeaderLen[2] = byte(len(ext.BitcoinMergedMiningHeader) >> 8)
	mmHeaderLen[3] = byte(len(ext.BitcoinMergedMiningHeader))
	data = append(data, mmHeaderLen...)
	data = append(data, ext.BitcoinMergedMiningHeader...)

	return data
}

// DecodeRSKExtension decodes RSK header extension from bytes.
func (c *RSKExtraDataCodec) DecodeRSKExtension(data []byte) (*RSKHeaderExtension, error) {
	if len(data) < 69 { // Minimum: 1 + 32 + 32 + 4
		return NewRSKHeaderExtension(), nil
	}

	ext := NewRSKHeaderExtension()

	// Version byte
	ext.Version = data[0]

	// Paid fees
	ext.PaidFees = new(big.Int).SetBytes(data[1:33])

	// Minimum gas price
	ext.MinimumGasPrice = new(big.Int).SetBytes(data[33:65])

	// Uncle count
	ext.UncleCount = uint32(data[65])<<24 | uint32(data[66])<<16 | uint32(data[67])<<8 | uint32(data[68])

	// Bitcoin merged mining header (if present)
	if len(data) >= 73 {
		mmHeaderLen := uint32(data[69])<<24 | uint32(data[70])<<16 | uint32(data[71])<<8 | uint32(data[72])
		if len(data) >= int(73+mmHeaderLen) {
			ext.BitcoinMergedMiningHeader = make([]byte, mmHeaderLen)
			copy(ext.BitcoinMergedMiningHeader, data[73:73+mmHeaderLen])
		}
	}

	return ext, nil
}

// IsRSKBlock checks if a header is an RSK block based on chain ID or other markers.
// This is a helper function to determine if RSK-specific processing is needed.
func IsRSKBlock(header *Header, chainID *big.Int) bool {
	if chainID == nil {
		return false
	}
	// RSK mainnet chainID = 30, testnet = 31
	id := chainID.Uint64()
	return id == 30 || id == 31 || id == 33 // mainnet, testnet, regtest
}

// GetRSKExtension attempts to decode RSK extension from a header's Extra field.
func GetRSKExtension(header *Header) *RSKHeaderExtension {
	if header == nil || len(header.Extra) == 0 {
		return nil
	}

	codec := &RSKExtraDataCodec{}
	ext, err := codec.DecodeRSKExtension(header.Extra)
	if err != nil {
		return nil
	}
	return ext
}

// SetRSKExtension encodes and sets RSK extension data in a header's Extra field.
func SetRSKExtension(header *Header, ext *RSKHeaderExtension) {
	if header == nil || ext == nil {
		return
	}

	codec := &RSKExtraDataCodec{}
	header.Extra = codec.EncodeRSKExtension(ext)
}

// RSKBlock wraps a standard Block with RSK-specific functionality.
type RSKBlock struct {
	*Block
	rskHeader *RSKHeader
	rskData   *RSKBlockData
}

// NewRSKBlock creates a new RSK block wrapper.
func NewRSKBlock(block *Block) *RSKBlock {
	rskBlock := &RSKBlock{
		Block:   block,
		rskData: NewRSKBlockData(),
	}

	if block != nil && block.Header() != nil {
		rskBlock.rskHeader = NewRSKHeader(block.Header())
		// Try to decode RSK extension from extra data
		if ext := GetRSKExtension(block.Header()); ext != nil {
			rskBlock.rskHeader.RSK = ext
		}
	}

	return rskBlock
}

// RSKHeader returns the RSK header wrapper.
func (b *RSKBlock) RSKHeader() *RSKHeader {
	return b.rskHeader
}

// RSKData returns the RSK block data.
func (b *RSKBlock) RSKData() *RSKBlockData {
	return b.rskData
}

// PaidFees returns the total fees paid in this block.
func (b *RSKBlock) PaidFees() *big.Int {
	if b.rskHeader != nil && b.rskHeader.RSK != nil && b.rskHeader.RSK.PaidFees != nil {
		return new(big.Int).Set(b.rskHeader.RSK.PaidFees)
	}
	return new(big.Int)
}

// MinimumGasPrice returns the minimum gas price for this block.
func (b *RSKBlock) MinimumGasPrice() *big.Int {
	if b.rskHeader != nil && b.rskHeader.RSK != nil && b.rskHeader.RSK.MinimumGasPrice != nil {
		return new(big.Int).Set(b.rskHeader.RSK.MinimumGasPrice)
	}
	return new(big.Int)
}
