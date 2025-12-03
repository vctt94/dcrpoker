package server

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vctt94/pokerbisonrelay/pkg/chainwatcher"
)

func TestClassifyEscrowFundingState(t *testing.T) {
	amount := uint64(1_000_000)
	tests := []struct {
		name           string
		hadFunding     bool
		csvBlocks      uint32
		requiredConfs  uint32
		update         chainwatcher.DepositUpdate
		wantState      string
		wantReasonHint string
	}{
		{
			name:           "unfunded when no utxo and never funded",
			update:         chainwatcher.DepositUpdate{UTXOCount: 0},
			wantState:      escrowStateUnfunded,
			wantReasonHint: "",
		},
		{
			name:           "spent when utxo count zero but had funding before",
			hadFunding:     true,
			update:         chainwatcher.DepositUpdate{UTXOCount: 0},
			wantState:      escrowStateSpent,
			wantReasonHint: "spent",
		},
		{
			name:           "invalid multi utxo",
			update:         chainwatcher.DepositUpdate{UTXOCount: 2, UTXOs: []*chainwatcher.EscrowUTXO{{Txid: "a", Vout: 0}, {Txid: "b", Vout: 1}}},
			wantState:      escrowStateInvalid,
			wantReasonHint: "single funding",
		},
		{
			name:           "invalid missing utxo detail",
			update:         chainwatcher.DepositUpdate{UTXOCount: 1},
			wantState:      escrowStateInvalid,
			wantReasonHint: "missing UTXO",
		},
		{
			name:           "invalid amount mismatch",
			update:         chainwatcher.DepositUpdate{UTXOCount: 1, UTXOs: []*chainwatcher.EscrowUTXO{{Value: amount + 1}}},
			wantState:      escrowStateInvalid,
			wantReasonHint: "expected funding",
		},
		{
			name:      "mempool when conf zero",
			update:    chainwatcher.DepositUpdate{UTXOCount: 1, UTXOs: []*chainwatcher.EscrowUTXO{{Value: amount}}, Confs: 0},
			wantState: escrowStateMempool,
			csvBlocks: 10,
		},
		{
			name:          "confirming when conf below required",
			update:        chainwatcher.DepositUpdate{UTXOCount: 1, UTXOs: []*chainwatcher.EscrowUTXO{{Value: amount}}, Confs: 1},
			csvBlocks:     0,
			requiredConfs: 2,
			wantState:     escrowStateConfirming,
		},
		{
			name:      "ready when conf meets required and below csv",
			update:    chainwatcher.DepositUpdate{UTXOCount: 1, UTXOs: []*chainwatcher.EscrowUTXO{{Value: amount}}, Confs: 1},
			wantState: escrowStateReady,
			csvBlocks: 10,
		},
		{
			name:      "csv matured when conf crosses csv",
			update:    chainwatcher.DepositUpdate{UTXOCount: 1, UTXOs: []*chainwatcher.EscrowUTXO{{Value: amount}}, Confs: 12},
			wantState: escrowStateCsvMatured,
			csvBlocks: 8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			required := tt.requiredConfs
			if required == 0 {
				required = escrowRequiredConfirmations
			}
			decision := classifyEscrowFundingState(amount, tt.csvBlocks, required, tt.hadFunding, tt.update)
			require.Equal(t, tt.wantState, decision.state)
			if tt.wantReasonHint != "" {
				require.Contains(t, decision.reason, tt.wantReasonHint)
			}
			if tt.wantState == escrowStateReady || tt.wantState == escrowStateCsvMatured {
				require.NotNil(t, decision.bound)
			}
			if tt.wantState == escrowStateInvalid {
				require.Nil(t, decision.bound)
			}
		})
	}
}
