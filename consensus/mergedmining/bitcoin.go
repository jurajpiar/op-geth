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
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"math/big"
)

const (
	// BitcoinHeaderSize is the size of a Bitcoin block header in bytes.
	BitcoinHeaderSize = 80
)

var (
	// ErrInvalidBitcoinHeader is returned when the Bitcoin header is invalid.
	ErrInvalidBitcoinHeader = errors.New("invalid Bitcoin header")
)

// BitcoinHeader represents a Bitcoin block header.
type BitcoinHeader struct {
	Version       int32    // Block version
	PrevBlockHash [32]byte // Hash of the previous block
	MerkleRoot    [32]byte // Merkle root of transactions
	Timestamp     uint32   // Block timestamp
	Bits          uint32   // Difficulty target in compact form
	Nonce         uint32   // Nonce used for mining
}

// ParseBitcoinHeader parses a Bitcoin block header from bytes.
// Bitcoin headers are exactly 80 bytes and encoded in little-endian.
func ParseBitcoinHeader(data []byte) (*BitcoinHeader, error) {
	if len(data) < BitcoinHeaderSize {
		return nil, ErrInvalidBitcoinHeader
	}

	header := &BitcoinHeader{}
	r := bytes.NewReader(data)

	if err := binary.Read(r, binary.LittleEndian, &header.Version); err != nil {
		return nil, err
	}
	if _, err := r.Read(header.PrevBlockHash[:]); err != nil {
		return nil, err
	}
	if _, err := r.Read(header.MerkleRoot[:]); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &header.Timestamp); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &header.Bits); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &header.Nonce); err != nil {
		return nil, err
	}

	return header, nil
}

// Serialize serializes the Bitcoin header to bytes.
func (h *BitcoinHeader) Serialize() []byte {
	buf := make([]byte, BitcoinHeaderSize)
	binary.LittleEndian.PutUint32(buf[0:4], uint32(h.Version))
	copy(buf[4:36], h.PrevBlockHash[:])
	copy(buf[36:68], h.MerkleRoot[:])
	binary.LittleEndian.PutUint32(buf[68:72], h.Timestamp)
	binary.LittleEndian.PutUint32(buf[72:76], h.Bits)
	binary.LittleEndian.PutUint32(buf[76:80], h.Nonce)
	return buf
}

// Hash calculates the Bitcoin block hash using double SHA256.
// Bitcoin uses double SHA256 for block hashing.
func (h *BitcoinHeader) Hash() [32]byte {
	data := h.Serialize()
	return DoubleSHA256(data)
}

// Target calculates the target value from the compact "bits" representation.
// The compact format stores the target as: mantissa * 256^(exponent-3)
func (h *BitcoinHeader) Target() *big.Int {
	return CompactToBig(h.Bits)
}

// SHA256Hash calculates a single SHA256 hash.
func SHA256Hash(data []byte) [32]byte {
	return sha256.Sum256(data)
}

// DoubleSHA256 calculates double SHA256 (SHA256(SHA256(data))).
// This is the standard hash function used in Bitcoin.
func DoubleSHA256(data []byte) [32]byte {
	hash1 := sha256.Sum256(data)
	return sha256.Sum256(hash1[:])
}

// ReverseBytes reverses a byte slice.
// Used for converting between Bitcoin's little-endian format and big-endian.
func ReverseBytes(b []byte) []byte {
	result := make([]byte, len(b))
	for i := 0; i < len(b); i++ {
		result[i] = b[len(b)-1-i]
	}
	return result
}

// CompactToBig converts a compact representation to a big.Int.
// The compact format is: 0xAABBCCDD where AA is the size and BBCCDD is the mantissa.
func CompactToBig(compact uint32) *big.Int {
	// Extract the size (exponent) from the first byte
	size := compact >> 24
	// Extract the mantissa from the remaining bytes
	mantissa := compact & 0x007fffff

	var target *big.Int
	if size <= 3 {
		// For small sizes, shift mantissa right
		mantissa >>= 8 * (3 - size)
		target = big.NewInt(int64(mantissa))
	} else {
		// For larger sizes, shift mantissa left
		target = big.NewInt(int64(mantissa))
		target.Lsh(target, 8*(uint(size)-3))
	}

	// Handle negative bit (bit 0x00800000 indicates negative)
	if compact&0x00800000 != 0 {
		target.Neg(target)
	}

	return target
}

// BigToCompact converts a big.Int to compact representation.
func BigToCompact(n *big.Int) uint32 {
	if n.Sign() == 0 {
		return 0
	}

	// Get the number of bytes needed
	nBytes := n.Bytes()
	nSize := uint32(len(nBytes))

	var compact uint32
	if nSize <= 3 {
		compact = uint32(n.Uint64()) << (8 * (3 - nSize))
	} else {
		// For larger numbers, take the top 3 bytes
		mantissa := new(big.Int).Rsh(n, uint(8*(nSize-3)))
		compact = uint32(mantissa.Uint64())
	}

	// Set the size byte
	if compact&0x00800000 != 0 {
		compact >>= 8
		nSize++
	}

	return compact | (nSize << 24)
}

// VerifyMerkleProof verifies a Merkle proof for a transaction in a Bitcoin block.
// The proof consists of sibling hashes that, combined with the transaction hash,
// should produce the Merkle root.
func VerifyMerkleProof(txHash [32]byte, proof []byte, merkleRoot [32]byte) error {
	currentHash := txHash
	proofReader := bytes.NewReader(proof)

	for proofReader.Len() > 0 {
		var sibling [32]byte
		var position byte

		if _, err := proofReader.Read(sibling[:]); err != nil {
			return ErrInvalidMerkleProof
		}

		// Read position byte if available
		if err := binary.Read(proofReader, binary.LittleEndian, &position); err != nil {
			// If we can't read position, assume end of proof
			break
		}

		if position == 0 {
			// Current hash is on the left
			currentHash = DoubleSHA256(append(currentHash[:], sibling[:]...))
		} else {
			// Current hash is on the right
			currentHash = DoubleSHA256(append(sibling[:], currentHash[:]...))
		}
	}

	if currentHash != merkleRoot {
		return ErrInvalidMerkleProof
	}

	return nil
}
