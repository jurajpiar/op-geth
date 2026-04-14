package rsk

import (
	"gorsk/rskblocks"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// CalculateReceiptsRoot calculates the RSK receipt trie root using gorsk's Unitrie implementation.
// This should be used instead of types.DeriveSha for RSK chains.
func CalculateReceiptsRoot(receipts types.Receipts) common.Hash {
	rskReceipts := make([]*rskblocks.TransactionReceipt, len(receipts))
	for i, r := range receipts {
		rskReceipts[i] = ConvertReceipt(r)
	}

	root := rskblocks.CalculateReceiptsTrieRoot(rskReceipts)
	return common.BytesToHash(root)
}

// ConvertReceipt converts a go-ethereum Receipt to an RSK TransactionReceipt.
// RSK receipts have different RLP encoding:
// - Gas values are encoded as byte arrays (big-endian), not uint64
// - Field order: [postTxState, cumulativeGas, bloom, logs, gasUsed, status]
func ConvertReceipt(r *types.Receipt) *rskblocks.TransactionReceipt {
	if r == nil {
		return nil
	}

	// Convert logs
	logs := make([]*rskblocks.Log, len(r.Logs))
	for i, log := range r.Logs {
		logs[i] = &rskblocks.Log{
			Address: log.Address,
			Topics:  log.Topics,
			Data:    log.Data,
		}
	}

	// Handle PostState vs Status (EIP-658)
	// In RSK:
	// - If PostState (root) is present, use it directly
	// - If PostState is empty but Status is set, put status bytes in BOTH PostState AND Status
	var postState []byte
	var status []byte

	if len(r.PostState) > 0 {
		// Pre-Byzantium style: use the state root
		postState = r.PostState
	} else {
		// Post-Byzantium style: put status in PostState for RSK
		// RSK encodes 0x01 for success, empty for failure
		if r.Status == types.ReceiptStatusSuccessful {
			postState = []byte{0x01}
		}
		// Note: for failed transactions, postState remains nil (empty bytes in RLP)
	}

	// Always set Status field based on receipt status
	if r.Status == types.ReceiptStatusSuccessful {
		status = []byte{0x01}
	}
	// Note: for failed transactions, status remains nil

	return &rskblocks.TransactionReceipt{
		PostState:         postState,
		CumulativeGasUsed: r.CumulativeGasUsed,
		Bloom:             r.Bloom,
		Logs:              logs,
		TxHash:            r.TxHash,
		ContractAddress:   r.ContractAddress,
		GasUsed:           r.GasUsed,
		Status:            status,
	}
}
