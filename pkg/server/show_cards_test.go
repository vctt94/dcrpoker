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
	updates := gsh.buildGameStatesFromSnapshot(snap, []string{"p2"})
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

// Auto-revealed cards during all-in auto-advance should be visible to everyone
// before showdown begins (countdown period). Today they stay hidden.
func TestAutoRevealHiddenBeforeShowdown(t *testing.T) {
	s := newBareServer()
	table := buildActiveHeadsUpTable(t, "auto-reveal-hidden")
	s.tables.Store(table.GetConfig().ID, table)

	require.Eventually(t, func() bool {
		return table.GetGamePhase() == pokerrpc.GamePhase_PRE_FLOP
	}, 2*time.Second, 10*time.Millisecond, "game should reach PRE_FLOP before forcing all-in")

	g := table.GetGame()
	require.NotNil(t, g, "game should exist after start")

	players := g.GetPlayers()
	require.Len(t, players, 2, "heads-up table should have two players")

	// Force both players all-in immediately to enable auto-advance + auto-reveal.
	firstActor := g.GetCurrentPlayerObject()
	require.NotNil(t, firstActor, "current player should be set preflop")
	require.Equal(t, players[0].ID(), firstActor.ID(), "SB/dealer should act first in heads-up")

	allInBet := players[0].CurrentBet() + players[0].Balance()
	require.NoError(t, g.HandlePlayerBet(players[0].ID(), allInBet))
	require.NoError(t, g.HandlePlayerCall(players[1].ID()))

	// Wait until the game marks both hands as revealed but before showdown happens.
	require.Eventually(t, func() bool {
		snap := g.GetStateSnapshot()
		if snap.Phase == pokerrpc.GamePhase_SHOWDOWN {
			return false
		}
		revealed := 0
		for _, ps := range snap.Players {
			if ps.ID != "" && ps.CardsRevealed {
				revealed++
			}
		}
		return revealed >= 2
	}, time.Second, 10*time.Millisecond, "auto-reveal should flip both players before showdown")

	snap, err := s.collectTableSnapshot(table.GetConfig().ID)
	require.NoError(t, err)

	gsh := NewGameStateHandler(s)
	updates := gsh.buildGameStatesFromSnapshot(snap, []string{"p1"})
	upd := updates["p1"]
	require.NotNil(t, upd, "p1 should receive a game update")

	var p2View *pokerrpc.Player
	for _, pl := range upd.Players {
		if pl != nil && pl.Id == "p2" {
			p2View = pl
			break
		}
	}
	require.NotNil(t, p2View, "update should include p2")

	assert.NotEqual(t, pokerrpc.GamePhase_SHOWDOWN, upd.Phase, "snapshot should be taken before showdown")
	assert.NotEmpty(t, p2View.Hand, "p1 should see p2's hole cards after auto-reveal during auto-advance")
}
