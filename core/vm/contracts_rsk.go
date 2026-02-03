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
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/binary"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

var (
	// ErrRSKPrecompileFailed is returned when an RSK precompile execution fails.
	ErrRSKPrecompileFailed = errors.New("rsk precompile execution failed")
)

// RSK precompiled contract addresses
var (
	RSKBridgePrecompileAddr       = common.HexToAddress("0x0000000000000000000000000000000001000001")
	RSKRemascPrecompileAddr       = common.HexToAddress("0x0000000000000000000000000000000001000002")
	RSKHDWalletUtilsPrecompileAddr = common.HexToAddress("0x0000000000000000000000000000000001000006")
	RSKBlockHashPrecompileAddr    = common.HexToAddress("0x0000000000000000000000000000000001000008")
)

// PrecompiledContractsRSK contains the RSK-specific precompiled contracts.
// These are added to the base precompiles when running on RSK chains.
var PrecompiledContractsRSK = PrecompiledContracts{
	RSKBridgePrecompileAddr:        &rskBridge{},
	RSKRemascPrecompileAddr:        &rskRemasc{},
	RSKHDWalletUtilsPrecompileAddr: &rskHDWalletUtils{},
	RSKBlockHashPrecompileAddr:     &rskBlockHashQuery{},
}

// PrecompiledAddressesRSK contains the addresses of RSK-specific precompiles.
var PrecompiledAddressesRSK []common.Address

func init() {
	for k := range PrecompiledContractsRSK {
		PrecompiledAddressesRSK = append(PrecompiledAddressesRSK, k)
	}
}

// RSKBridgeState provides access to bridge state for the precompile.
type RSKBridgeState interface {
	GetBTCBlockchainBestChainHeight() uint64
	GetFederationAddress() string
	GetFederationThreshold() int
	GetFederationSize() int
	GetMinimumPegInValue() uint64
	GetMinimumPegOutValue() uint64
}

// rskBridge implements the RSK Bridge precompile for Bitcoin peg.
type rskBridge struct {
	state RSKBridgeState
}

// SetBridgeState sets the bridge state provider.
func (b *rskBridge) SetBridgeState(state RSKBridgeState) {
	b.state = state
}

// Bridge function selectors (keccak256 of function signature, first 4 bytes)
var (
	bridgeSelectorGetBtcBlockchainBestChainHeight    = []byte{0x0b, 0x88, 0x42, 0x7e}
	bridgeSelectorGetStateForBtcReleaseClient        = []byte{0x12, 0x3e, 0xc3, 0x44}
	bridgeSelectorGetStateForDebugging               = []byte{0x1d, 0x6c, 0x9e, 0x50}
	bridgeSelectorGetBtcBlockchainInitialBlockHeight = []byte{0x6f, 0x6c, 0x5b, 0x37}
	bridgeSelectorGetFederationAddress               = []byte{0x6b, 0xf7, 0x66, 0xb2}
	bridgeSelectorGetFederationSize                  = []byte{0x9e, 0x52, 0x37, 0x28}
	bridgeSelectorGetFederationThreshold             = []byte{0x98, 0x04, 0x6e, 0xac}
	bridgeSelectorGetMinimumPegInValue               = []byte{0x35, 0x8d, 0xdd, 0x60}
	bridgeSelectorGetMinimumPegOutValue              = []byte{0x4f, 0xa7, 0x73, 0xef}
)

func (b *rskBridge) RequiredGas(input []byte) uint64 {
	// Bridge operations have varying gas costs based on function
	if len(input) < 4 {
		return params.RSKBridgeBaseGas
	}

	selector := input[:4]

	// Read operations are cheaper
	if matchRSKSelector(selector, bridgeSelectorGetBtcBlockchainBestChainHeight) ||
		matchRSKSelector(selector, bridgeSelectorGetFederationAddress) ||
		matchRSKSelector(selector, bridgeSelectorGetFederationSize) ||
		matchRSKSelector(selector, bridgeSelectorGetFederationThreshold) ||
		matchRSKSelector(selector, bridgeSelectorGetMinimumPegInValue) ||
		matchRSKSelector(selector, bridgeSelectorGetMinimumPegOutValue) ||
		matchRSKSelector(selector, bridgeSelectorGetBtcBlockchainInitialBlockHeight) {
		return params.RSKBridgeReadGas
	}

	// State query operations
	if matchRSKSelector(selector, bridgeSelectorGetStateForBtcReleaseClient) ||
		matchRSKSelector(selector, bridgeSelectorGetStateForDebugging) {
		return params.RSKBridgeStateQueryGas
	}

	return params.RSKBridgeBaseGas
}

func matchRSKSelector(a, b []byte) bool {
	if len(a) < 4 || len(b) < 4 {
		return false
	}
	return a[0] == b[0] && a[1] == b[1] && a[2] == b[2] && a[3] == b[3]
}

func (b *rskBridge) Run(input []byte) ([]byte, error) {
	if len(input) < 4 {
		return nil, ErrRSKPrecompileFailed
	}

	// Function selector (first 4 bytes)
	selector := input[:4]

	// Handle different bridge functions
	if matchRSKSelector(selector, bridgeSelectorGetBtcBlockchainBestChainHeight) {
		return b.getBtcBlockchainBestChainHeight()
	}
	if matchRSKSelector(selector, bridgeSelectorGetStateForBtcReleaseClient) {
		return b.getStateForBtcReleaseClient()
	}
	if matchRSKSelector(selector, bridgeSelectorGetStateForDebugging) {
		return b.getStateForDebugging()
	}
	if matchRSKSelector(selector, bridgeSelectorGetBtcBlockchainInitialBlockHeight) {
		return b.getBtcBlockchainInitialBlockHeight()
	}
	if matchRSKSelector(selector, bridgeSelectorGetFederationAddress) {
		return b.getFederationAddress()
	}
	if matchRSKSelector(selector, bridgeSelectorGetFederationSize) {
		return b.getFederationSize()
	}
	if matchRSKSelector(selector, bridgeSelectorGetFederationThreshold) {
		return b.getFederationThreshold()
	}
	if matchRSKSelector(selector, bridgeSelectorGetMinimumPegInValue) {
		return b.getMinimumPegInValue()
	}
	if matchRSKSelector(selector, bridgeSelectorGetMinimumPegOutValue) {
		return b.getMinimumPegOutValue()
	}

	return nil, ErrRSKPrecompileFailed
}

func (b *rskBridge) Name() string {
	return "RSK_BRIDGE"
}

func (b *rskBridge) getBtcBlockchainBestChainHeight() ([]byte, error) {
	if b.state == nil {
		// Default value when bridge state not set
		return common.LeftPadBytes(big.NewInt(0).Bytes(), 32), nil
	}
	height := b.state.GetBTCBlockchainBestChainHeight()
	return common.LeftPadBytes(big.NewInt(int64(height)).Bytes(), 32), nil
}

func (b *rskBridge) getStateForBtcReleaseClient() ([]byte, error) {
	// Returns ABI-encoded state for BTC release client
	if b.state == nil {
		return make([]byte, 64), nil
	}

	result := make([]byte, 64)
	height := b.state.GetBTCBlockchainBestChainHeight()
	copy(result[24:32], big.NewInt(int64(height)).Bytes())
	return result, nil
}

func (b *rskBridge) getStateForDebugging() ([]byte, error) {
	if b.state == nil {
		return make([]byte, 128), nil
	}

	result := make([]byte, 128)
	// Height
	height := b.state.GetBTCBlockchainBestChainHeight()
	copy(result[24:32], big.NewInt(int64(height)).Bytes())
	// Federation size
	fedSize := b.state.GetFederationSize()
	copy(result[56:64], big.NewInt(int64(fedSize)).Bytes())
	// Federation threshold
	fedThreshold := b.state.GetFederationThreshold()
	copy(result[88:96], big.NewInt(int64(fedThreshold)).Bytes())

	return result, nil
}

func (b *rskBridge) getBtcBlockchainInitialBlockHeight() ([]byte, error) {
	// RSK mainnet started tracking from BTC block ~478558
	initialHeight := big.NewInt(478558)
	return common.LeftPadBytes(initialHeight.Bytes(), 32), nil
}

func (b *rskBridge) getFederationAddress() ([]byte, error) {
	if b.state == nil {
		return make([]byte, 32), nil
	}
	addr := b.state.GetFederationAddress()
	return common.LeftPadBytes([]byte(addr), 32), nil
}

func (b *rskBridge) getFederationSize() ([]byte, error) {
	if b.state == nil {
		return common.LeftPadBytes(big.NewInt(0).Bytes(), 32), nil
	}
	size := b.state.GetFederationSize()
	return common.LeftPadBytes(big.NewInt(int64(size)).Bytes(), 32), nil
}

func (b *rskBridge) getFederationThreshold() ([]byte, error) {
	if b.state == nil {
		return common.LeftPadBytes(big.NewInt(0).Bytes(), 32), nil
	}
	threshold := b.state.GetFederationThreshold()
	return common.LeftPadBytes(big.NewInt(int64(threshold)).Bytes(), 32), nil
}

func (b *rskBridge) getMinimumPegInValue() ([]byte, error) {
	if b.state == nil {
		// Default minimum: 0.005 BTC in satoshi
		return common.LeftPadBytes(big.NewInt(500000).Bytes(), 32), nil
	}
	minValue := b.state.GetMinimumPegInValue()
	return common.LeftPadBytes(big.NewInt(int64(minValue)).Bytes(), 32), nil
}

func (b *rskBridge) getMinimumPegOutValue() ([]byte, error) {
	if b.state == nil {
		// Default minimum: 0.008 BTC in satoshi
		return common.LeftPadBytes(big.NewInt(800000).Bytes(), 32), nil
	}
	minValue := b.state.GetMinimumPegOutValue()
	return common.LeftPadBytes(big.NewInt(int64(minValue)).Bytes(), 32), nil
}

// RSKRemascState provides access to REMASC state for the precompile.
type RSKRemascState interface {
	GetRewardBalance() *big.Int
	GetBurnedBalance() *big.Int
}

// RSKRemascProcessor handles block reward processing.
type RSKRemascProcessor interface {
	ProcessBlockRewards(coinbase common.Address, blockNumber uint64, paidFees *big.Int) error
}

// rskRemasc implements the REMASC precompile for reward distribution.
type rskRemasc struct {
	state     RSKRemascState
	processor RSKRemascProcessor
}

// SetRemascState sets the REMASC state provider.
func (r *rskRemasc) SetRemascState(state RSKRemascState) {
	r.state = state
}

// SetRemascProcessor sets the REMASC processor for block rewards.
func (r *rskRemasc) SetRemascProcessor(processor RSKRemascProcessor) {
	r.processor = processor
}

// REMASC function selectors
var (
	remascSelectorGetRewardBalance = []byte{0x45, 0x19, 0x8c, 0x83}
	remascSelectorGetBurnedBalance = []byte{0x8d, 0x87, 0x0e, 0xb8}
	remascSelectorProcessRewards   = []byte{0xf0, 0xf7, 0x3a, 0x04}
)

func (r *rskRemasc) RequiredGas(input []byte) uint64 {
	if len(input) < 4 {
		return params.RSKRemascBaseGas
	}

	selector := input[:4]

	// Read operations are cheaper
	if matchRSKSelector(selector, remascSelectorGetRewardBalance) ||
		matchRSKSelector(selector, remascSelectorGetBurnedBalance) {
		return params.RSKRemascReadGas
	}

	// Process rewards is expensive
	if matchRSKSelector(selector, remascSelectorProcessRewards) {
		return params.RSKRemascProcessGas
	}

	return params.RSKRemascBaseGas
}

func (r *rskRemasc) Run(input []byte) ([]byte, error) {
	// REMASC is called at the end of each block to process rewards
	// It can also be called to query state

	if len(input) < 4 {
		// No selector - this is the end-of-block call
		// In real RSK, this triggers reward processing
		return nil, nil
	}

	selector := input[:4]

	if matchRSKSelector(selector, remascSelectorGetRewardBalance) {
		return r.getRewardBalance()
	}
	if matchRSKSelector(selector, remascSelectorGetBurnedBalance) {
		return r.getBurnedBalance()
	}
	if matchRSKSelector(selector, remascSelectorProcessRewards) {
		// This would be called by the block executor
		// Input format: coinbase (20 bytes) + blockNumber (32 bytes) + paidFees (32 bytes)
		if len(input) < 4+20+32+32 {
			return nil, ErrRSKPrecompileFailed
		}
		return r.processRewards(input[4:])
	}

	// Default: return empty (end of block call)
	return nil, nil
}

func (r *rskRemasc) Name() string {
	return "RSK_REMASC"
}

func (r *rskRemasc) getRewardBalance() ([]byte, error) {
	if r.state == nil {
		return make([]byte, 32), nil
	}
	balance := r.state.GetRewardBalance()
	if balance == nil {
		return make([]byte, 32), nil
	}
	return common.LeftPadBytes(balance.Bytes(), 32), nil
}

func (r *rskRemasc) getBurnedBalance() ([]byte, error) {
	if r.state == nil {
		return make([]byte, 32), nil
	}
	balance := r.state.GetBurnedBalance()
	if balance == nil {
		return make([]byte, 32), nil
	}
	return common.LeftPadBytes(balance.Bytes(), 32), nil
}

func (r *rskRemasc) processRewards(data []byte) ([]byte, error) {
	if r.processor == nil {
		// No processor set, skip reward processing
		return nil, nil
	}

	// Parse input
	var coinbase common.Address
	copy(coinbase[:], data[:20])

	blockNumber := new(big.Int).SetBytes(data[20:52])
	paidFees := new(big.Int).SetBytes(data[52:84])

	// Process rewards
	if err := r.processor.ProcessBlockRewards(coinbase, blockNumber.Uint64(), paidFees); err != nil {
		return nil, err
	}

	return nil, nil
}

// rskHDWalletUtils implements HD Wallet utilities precompile.
// Supports BIP32 HD key derivation operations.
type rskHDWalletUtils struct{}

// HD Wallet function selectors
var (
	hdWalletSelectorDeriveExtendedPubKey = []byte{0x8d, 0xa5, 0xcb, 0x5b}
	hdWalletSelectorToBase58Check        = []byte{0x41, 0x7b, 0xc0, 0x33}
)

func (h *rskHDWalletUtils) RequiredGas(input []byte) uint64 {
	if len(input) < 4 {
		return params.RSKHDWalletUtilsBaseGas
	}

	// Derivation is more expensive than encoding
	selector := input[:4]
	if matchRSKSelector(selector, hdWalletSelectorDeriveExtendedPubKey) {
		// Cost depends on derivation path length
		pathLen := 0
		if len(input) > 68 { // 4 (selector) + 64 (pubkey)
			pathLen = (len(input) - 68) / 4 // Each path component is 4 bytes
		}
		return uint64(params.RSKHDWalletUtilsBaseGas + params.RSKHDWalletUtilsDerivationGas*uint64(pathLen))
	}

	return params.RSKHDWalletUtilsBaseGas
}

func (h *rskHDWalletUtils) Run(input []byte) ([]byte, error) {
	if len(input) < 4 {
		return nil, ErrRSKPrecompileFailed
	}

	selector := input[:4]
	data := input[4:]

	if matchRSKSelector(selector, hdWalletSelectorDeriveExtendedPubKey) {
		return h.deriveExtendedPublicKey(data)
	}
	if matchRSKSelector(selector, hdWalletSelectorToBase58Check) {
		return h.toBase58Check(data)
	}

	return nil, ErrRSKPrecompileFailed
}

func (h *rskHDWalletUtils) Name() string {
	return "RSK_HD_WALLET_UTILS"
}

// deriveExtendedPublicKey derives a child extended public key using BIP32.
// Input format: xpub (64 bytes) + chainCode (32 bytes) + path components (4 bytes each)
func (h *rskHDWalletUtils) deriveExtendedPublicKey(data []byte) ([]byte, error) {
	if len(data) < 96 { // Need at least pubkey (64) + chainCode (32)
		return nil, ErrRSKPrecompileFailed
	}

	// Extract parent public key (64 bytes: x and y coordinates)
	parentPubKey := data[:64]
	// Extract chain code (32 bytes)
	chainCode := data[64:96]

	// Extract derivation path
	pathData := data[96:]
	if len(pathData)%4 != 0 {
		return nil, ErrRSKPrecompileFailed
	}

	// Current key and chain code
	currentPubKey := make([]byte, 64)
	copy(currentPubKey, parentPubKey)
	currentChainCode := make([]byte, 32)
	copy(currentChainCode, chainCode)

	// Derive each path component
	for i := 0; i < len(pathData); i += 4 {
		childIndex := binary.BigEndian.Uint32(pathData[i : i+4])

		// Check if hardened derivation requested (not allowed for public key derivation)
		if childIndex >= 0x80000000 {
			return nil, errors.New("cannot derive hardened child from public key")
		}

		// Derive child key using BIP32 algorithm
		newPubKey, newChainCode, err := h.deriveChildPublicKey(currentPubKey, currentChainCode, childIndex)
		if err != nil {
			return nil, err
		}

		currentPubKey = newPubKey
		currentChainCode = newChainCode
	}

	// Return derived public key (64 bytes) + chain code (32 bytes)
	result := make([]byte, 96)
	copy(result[:64], currentPubKey)
	copy(result[64:], currentChainCode)

	return result, nil
}

// deriveChildPublicKey derives a child public key using BIP32.
func (h *rskHDWalletUtils) deriveChildPublicKey(parentPubKey, chainCode []byte, index uint32) ([]byte, []byte, error) {
	// Serialize public key in compressed form for HMAC
	compressedPubKey := h.compressPublicKey(parentPubKey)

	// Create HMAC-SHA512 data: serialized public key || index
	hmacData := make([]byte, 33+4)
	copy(hmacData[:33], compressedPubKey)
	binary.BigEndian.PutUint32(hmacData[33:], index)

	// HMAC-SHA512
	hmacResult := hmacSHA512(chainCode, hmacData)

	// Split result: first 32 bytes = tweak, last 32 bytes = new chain code
	tweak := hmacResult[:32]
	newChainCode := hmacResult[32:]

	// Add tweak to parent public key (EC point addition)
	childPubKey, err := h.addTweakToPublicKey(parentPubKey, tweak)
	if err != nil {
		return nil, nil, err
	}

	return childPubKey, newChainCode, nil
}

// compressPublicKey compresses a 64-byte public key to 33 bytes.
func (h *rskHDWalletUtils) compressPublicKey(pubKey []byte) []byte {
	if len(pubKey) != 64 {
		return nil
	}

	compressed := make([]byte, 33)
	// Prefix: 0x02 if y is even, 0x03 if y is odd
	if pubKey[63]&1 == 0 {
		compressed[0] = 0x02
	} else {
		compressed[0] = 0x03
	}
	copy(compressed[1:], pubKey[:32]) // x coordinate

	return compressed
}

// addTweakToPublicKey adds a scalar tweak to a public key.
func (h *rskHDWalletUtils) addTweakToPublicKey(pubKey, tweak []byte) ([]byte, error) {
	// Check for zero tweak
	tweakScalar := new(big.Int).SetBytes(tweak)
	if tweakScalar.Sign() == 0 {
		// Zero tweak - return parent key unchanged
		return pubKey, nil
	}

	// Check tweak is valid (not >= curve order)
	curveOrder, _ := new(big.Int).SetString("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141", 16)
	if tweakScalar.Cmp(curveOrder) >= 0 {
		return nil, errors.New("invalid tweak value")
	}

	// Parse parent public key coordinates
	parentX := new(big.Int).SetBytes(pubKey[:32])
	parentY := new(big.Int).SetBytes(pubKey[32:64])

	// Get the secp256k1 curve
	curve := crypto.S256()

	// Compute tweak * G (generator point)
	tweakGx, tweakGy := curve.ScalarBaseMult(tweak)

	// Add to parent point: childPubKey = parentPubKey + tweak * G
	childX, childY := curve.Add(parentX, parentY, tweakGx, tweakGy)

	// Check result is valid (not point at infinity)
	if childX.Sign() == 0 && childY.Sign() == 0 {
		return nil, errors.New("derived key is point at infinity")
	}

	// Convert back to bytes
	childPubKey := make([]byte, 64)
	childX.FillBytes(childPubKey[:32])
	childY.FillBytes(childPubKey[32:])

	return childPubKey, nil
}

// toBase58Check converts data to Base58Check encoding.
func (h *rskHDWalletUtils) toBase58Check(data []byte) ([]byte, error) {
	if len(data) < 1 {
		return nil, ErrRSKPrecompileFailed
	}

	// Add checksum (first 4 bytes of double SHA256)
	hash1 := sha256.Sum256(data)
	hash2 := sha256.Sum256(hash1[:])
	checksum := hash2[:4]

	// Append checksum
	withChecksum := append(data, checksum...)

	// Base58 encode
	result := base58Encode(withChecksum)

	return result, nil
}

// hmacSHA512 computes HMAC-SHA512 using the standard library.
func hmacSHA512(key, data []byte) []byte {
	h := hmac.New(sha512.New, key)
	h.Write(data)
	return h.Sum(nil)
}

// base58Encode encodes data in Base58.
func base58Encode(data []byte) []byte {
	const alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

	// Convert to big integer
	num := new(big.Int).SetBytes(data)
	zero := big.NewInt(0)
	base := big.NewInt(58)

	var result []byte
	mod := new(big.Int)

	for num.Cmp(zero) > 0 {
		num.DivMod(num, base, mod)
		result = append([]byte{alphabet[mod.Int64()]}, result...)
	}

	// Add leading zeros
	for _, b := range data {
		if b != 0 {
			break
		}
		result = append([]byte{'1'}, result...)
	}

	return result
}

// RSKBlockHashProvider provides block hash lookups for the precompile.
type RSKBlockHashProvider interface {
	GetBlockHash(blockNum uint64) common.Hash
	GetCurrentBlockNumber() uint64
}

// rskBlockHashQuery implements block hash query precompile.
// This extends the BLOCKHASH opcode to allow querying hashes beyond the 256 block limit.
type rskBlockHashQuery struct {
	hashProvider RSKBlockHashProvider
}

// SetBlockHashProvider sets the block hash provider.
func (b *rskBlockHashQuery) SetBlockHashProvider(provider RSKBlockHashProvider) {
	b.hashProvider = provider
}

// Block hash query function selectors
var (
	blockHashSelectorGetBlockHash = []byte{0xee, 0x82, 0xac, 0x5e}
	blockHashSelectorGetDepth     = []byte{0xfa, 0x78, 0x3e, 0x41}
)

func (b *rskBlockHashQuery) RequiredGas(input []byte) uint64 {
	return params.RSKBlockHashQueryGas
}

func (b *rskBlockHashQuery) Run(input []byte) ([]byte, error) {
	if len(input) < 4 {
		// Direct call with just block number (legacy format)
		if len(input) == 32 {
			return b.getBlockHash(input)
		}
		return nil, ErrRSKPrecompileFailed
	}

	selector := input[:4]

	if matchRSKSelector(selector, blockHashSelectorGetBlockHash) {
		if len(input) < 36 { // selector (4) + blockNum (32)
			return nil, ErrRSKPrecompileFailed
		}
		return b.getBlockHash(input[4:36])
	}

	if matchRSKSelector(selector, blockHashSelectorGetDepth) {
		return b.getDepth()
	}

	// Legacy format: just block number
	return b.getBlockHash(input[:32])
}

func (b *rskBlockHashQuery) Name() string {
	return "RSK_BLOCK_HASH_QUERY"
}

func (b *rskBlockHashQuery) getBlockHash(blockNumBytes []byte) ([]byte, error) {
	blockNum := new(big.Int).SetBytes(blockNumBytes)

	if b.hashProvider == nil {
		// Without provider, return zero hash
		return make([]byte, 32), nil
	}

	// Check if block number is within range
	currentBlock := b.hashProvider.GetCurrentBlockNumber()
	requestedBlock := blockNum.Uint64()

	// Cannot query future blocks
	if requestedBlock >= currentBlock {
		return make([]byte, 32), nil
	}

	// RSK's block hash precompile can query deeper than 256 blocks
	// (unlike the BLOCKHASH opcode)
	maxDepth := uint64(10000) // Allow up to 10000 blocks back
	if currentBlock-requestedBlock > maxDepth {
		return make([]byte, 32), nil
	}

	hash := b.hashProvider.GetBlockHash(requestedBlock)
	return hash[:], nil
}

func (b *rskBlockHashQuery) getDepth() ([]byte, error) {
	// Return the maximum depth that can be queried
	maxDepth := big.NewInt(10000)
	return common.LeftPadBytes(maxDepth.Bytes(), 32), nil
}
