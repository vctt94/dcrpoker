package db

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSettlementEscrowsRoundTrip(t *testing.T) {
	db, err := NewDB(filepath.Join(t.TempDir(), "poker.db"))
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()
	matchID := "table-1"

	err = db.ReplaceSettlementEscrows(ctx, matchID, map[uint32]string{
		2: "escrow-2",
		0: "escrow-0",
	})
	require.NoError(t, err)

	rows, err := db.ListSettlementEscrows(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.Equal(t, matchID, rows[0].MatchID)
	require.Equal(t, uint32(0), rows[0].Seat)
	require.Equal(t, "escrow-0", rows[0].EscrowID)
	require.Equal(t, uint32(2), rows[1].Seat)
	require.Equal(t, "escrow-2", rows[1].EscrowID)

	err = db.ReplaceSettlementEscrows(ctx, matchID, map[uint32]string{
		1: "escrow-1b",
	})
	require.NoError(t, err)

	rows, err = db.ListSettlementEscrows(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, uint32(1), rows[0].Seat)
	require.Equal(t, "escrow-1b", rows[0].EscrowID)

	err = db.DeleteSettlementEscrows(ctx, matchID)
	require.NoError(t, err)

	rows, err = db.ListSettlementEscrows(ctx)
	require.NoError(t, err)
	require.Empty(t, rows)
}

func TestRefereeRecoveryStateRoundTrip(t *testing.T) {
	db, err := NewDB(filepath.Join(t.TempDir(), "poker.db"))
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()

	escrowPayload := []byte(`{"escrow_id":"escrow-1"}`)
	err = db.UpsertRefereeEscrow(ctx, RefereeEscrow{
		EscrowID: "escrow-1",
		Payload:  escrowPayload,
	})
	require.NoError(t, err)

	presignPayload := []byte(`{"input_id":"tx:0"}`)
	err = db.UpsertRefereePresign(ctx, RefereePresign{
		MatchID: "match-1",
		Branch:  0,
		InputID: "tx:0",
		Payload: presignPayload,
	})
	require.NoError(t, err)

	err = db.UpsertRefereeBranchGamma(ctx, RefereeBranchGamma{
		MatchID:  "match-1",
		Branch:   0,
		GammaHex: "deadbeef",
	})
	require.NoError(t, err)

	err = db.UpsertPendingSettlement(ctx, PendingSettlement{
		MatchID:    "match-1",
		TableID:    "table-1",
		WinnerID:   "player-1",
		WinnerSeat: 1,
	})
	require.NoError(t, err)

	escrows, err := db.ListRefereeEscrows(ctx)
	require.NoError(t, err)
	require.Len(t, escrows, 1)
	require.Equal(t, "escrow-1", escrows[0].EscrowID)
	require.Equal(t, escrowPayload, escrows[0].Payload)

	presigns, err := db.ListRefereePresigns(ctx)
	require.NoError(t, err)
	require.Len(t, presigns, 1)
	require.Equal(t, "match-1", presigns[0].MatchID)
	require.Equal(t, presignPayload, presigns[0].Payload)

	gammas, err := db.ListRefereeBranchGammas(ctx)
	require.NoError(t, err)
	require.Len(t, gammas, 1)
	require.Equal(t, "deadbeef", gammas[0].GammaHex)

	pending, err := db.ListPendingSettlements(ctx)
	require.NoError(t, err)
	require.Len(t, pending, 1)
	require.Equal(t, int32(1), pending[0].WinnerSeat)

	require.NoError(t, db.DeleteRefereePresigns(ctx, "match-1"))
	require.NoError(t, db.DeleteRefereeBranchGammas(ctx, "match-1"))
	require.NoError(t, db.DeletePendingSettlement(ctx, "match-1"))
	require.NoError(t, db.DeleteRefereeEscrow(ctx, "escrow-1"))

	escrows, err = db.ListRefereeEscrows(ctx)
	require.NoError(t, err)
	require.Empty(t, escrows)
	presigns, err = db.ListRefereePresigns(ctx)
	require.NoError(t, err)
	require.Empty(t, presigns)
	gammas, err = db.ListRefereeBranchGammas(ctx)
	require.NoError(t, err)
	require.Empty(t, gammas)
	pending, err = db.ListPendingSettlements(ctx)
	require.NoError(t, err)
	require.Empty(t, pending)
}
