package server

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
)

// Showing cards should not reveal to opponents until showdown. Today, calling
// ShowCards mid-hand exposes the caller's hole cards immediately.
func TestShowCardsRejectedDuringBetting(t *testing.T) {
	s := newBareServer()
	table := buildActiveHeadsUpTable(t, "show-mid-hand")
	s.tables.Store(table.GetConfig().ID, table)

	// Wait until the first hand reaches PRE_FLOP to ensure we're mid-hand.
	require.Eventually(t, func() bool {
		return table.GetGamePhase() == pokerrpc.GamePhase_PRE_FLOP
	}, 2*time.Second, 10*time.Millisecond, "game should reach PRE_FLOP before testing show-cards")
	require.NotEqual(t, pokerrpc.GamePhase_SHOWDOWN, table.GetGamePhase(), "hand should not be at showdown yet")

	_, err := s.ShowCards(context.Background(), &pokerrpc.ShowCardsRequest{
		TableId:  table.GetConfig().ID,
		PlayerId: "p1",
	})

	require.NoError(t, err, "showing cards mid-hand currently succeeds (bug path)")

	// Build updates as p2 would see them; p1's hand should NOT be visible yet.
	snap, snapErr := s.collectTableSnapshot(table.GetConfig().ID)
	require.NoError(t, snapErr)
	gsh := NewGameStateHandler(s)
	updates := gsh.buildGameStatesFromSnapshot(snap)
	upd := updates["p2"]
	require.NotNil(t, upd, "p2 should receive a game update")

	var p1View *pokerrpc.Player
	for _, pl := range upd.Players {
		if pl != nil && pl.Id == "p1" {
			p1View = pl
			break
		}
	}
	require.NotNil(t, p1View, "p2's update should include p1")
	assert.Empty(t, p1View.Hand, "p1's hand should stay hidden during betting even after ShowCards")
	assert.Empty(t, p1View.HandDescription, "hand description should stay hidden during betting")
}
