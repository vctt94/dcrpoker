package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	testenv "github.com/vctt94/pokerbisonrelay/e2e/internal/testenv"
	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Regression: when a game ends, the table should be removed from the server
// registry and subsequent RPCs should treat it as gone. Currently the table
// keeps emitting GAME_STATE updates after GAME_OVER/TABLE_REMOVED.
func TestTableRemovedAfterGameOver(t *testing.T) {
	t.Parallel()

	env := testenv.New(t)
	defer env.Close()

	ctx := context.Background()

	// Map the readable aliases to the ShortID strings returned by the API.
	playerIDs := map[string]string{
		"alice": testenv.PlayerIDToShortIDString("alice"),
		"bob":   testenv.PlayerIDToShortIDString("bob"),
	}
	aliasForID := func(id string) string {
		for alias, shortID := range playerIDs {
			if shortID == id {
				return alias
			}
		}
		return ""
	}
	// Create a fast table with tiny stacks to reach game over quickly.
	hostCtx := env.ContextWithToken(ctx, "alice")
	createResp, err := env.CreateTable(hostCtx, &pokerrpc.CreateTableRequest{
		PlayerId:      "alice",
		SmallBlind:    10,
		BigBlind:      20,
		MinPlayers:    2,
		MaxPlayers:    2,
		BuyIn:         0,
		StartingChips: 100,
		AutoStartMs:   500,
		AutoAdvanceMs: 500,
	})
	require.NoError(t, err)
	tableID := createResp.TableId

	// Place an all-in bet for the given player. This waits until it's
	// definitively their turn to avoid FSM races, then bets balance+currentBet
	// so blinds are included in the all-in amount.
	allIn := func(playerID string) {
		alias := aliasForID(playerID)
		require.NotEmpty(t, alias, "unknown alias for player %s", playerID)

		var betErr error
		ok := assert.Eventually(t, func() bool {
			st := env.GetGameState(ctx, tableID)
			if st.CurrentPlayer != playerID {
				return false
			}
			var self *pokerrpc.Player
			for _, p := range st.Players {
				if p != nil && p.Id == playerID {
					self = p
					break
				}
			}
			if self == nil || !self.IsTurn {
				return false
			}

			amount := self.CurrentBet + self.Balance
			_, betErr = env.PokerClient.MakeBet(env.ContextWithToken(ctx, alias), &pokerrpc.MakeBetRequest{
				PlayerId: playerID,
				TableId:  tableID,
				Amount:   amount,
			})
			if betErr != nil && isTransientTurnError(betErr) {
				return false
			}
			return betErr == nil
		}, 2*time.Second, 10*time.Millisecond, "failed to place all-in bet for %s", alias)

		if !ok {
			require.NoError(t, betErr)
		}
	}

	// Join second player and ready up both.
	_, err = env.JoinTable(ctx, "bob", tableID)
	require.NoError(t, err)
	for _, p := range []string{"alice", "bob"} {
		_, err := env.SetPlayerReady(ctx, p, tableID)
		require.NoError(t, err)
	}

	env.WaitForGameStart(ctx, tableID, 3*time.Second)
	env.WaitForGamePhase(ctx, tableID, pokerrpc.GamePhase_PRE_FLOP, 3*time.Second)

	// Play up to 3 all-in hands to guarantee someone busts (ties are rare but possible).
	winnerID := ""
	for hand := 1; hand <= 3 && winnerID == ""; hand++ {
		state, removed := env.GetGameStateAllowNotFound(ctx, tableID)
		if removed {
			// Table already gone; treat as terminal winner determined.
			winnerID = "table_removed"
			break
		}

		firstID := state.CurrentPlayer
		allIn(firstID)

		// Wait for turn to advance to the second player.
		require.Eventually(t, func() bool {
			st, removed := env.GetGameStateAllowNotFound(ctx, tableID)
			if removed {
				return true
			}
			return st.CurrentPlayer != firstID
		}, 2*time.Second, 25*time.Millisecond, "turn should advance after first all-in")

		state, removed = env.GetGameStateAllowNotFound(ctx, tableID)
		if removed {
			winnerID = "table_removed"
			break
		}
		secondID := state.CurrentPlayer
		allIn(secondID)

		// Wait for showdown to resolve the all-ins. Use GetLastWinners to
		// confirm settlement completed before the table potentially closes.
		require.Eventually(t, func() bool {
			resp, err := env.PokerClient.GetLastWinners(ctx, &pokerrpc.GetLastWinnersRequest{
				TableId: tableID,
			})
			return err == nil && len(resp.GetWinners()) > 0
		}, 5*time.Second, 25*time.Millisecond, "expected showdown winners to be available")

		finalState := env.GetGameState(ctx, tableID)
		var (
			playersWithChips int
			soleSurvivorID   string
		)
		for _, ps := range finalState.Players {
			if ps == nil {
				continue
			}
			if ps.Balance > 0 {
				playersWithChips++
				soleSurvivorID = ps.Id
			}
		}

		if playersWithChips == 1 {
			winnerID = soleSurvivorID
			break
		}

		// Tie/split pot: wait for the next hand to auto-start.
		time.Sleep(1 * time.Second)
		env.WaitForGamePhase(ctx, tableID, pokerrpc.GamePhase_PRE_FLOP, 5*time.Second)
	}
	require.NotEmpty(t, winnerID, "expected a single winner after all-in hands")

	// Wait a bit for GAME_ENDED event to be processed, which triggers table removal scheduling.
	// The removal has a 1 second grace period, then needs event processing time.
	time.Sleep(500 * time.Millisecond)

	// The table should disappear once the match is finished.
	// scheduleTableRemoval has a 1s grace period, plus we need time for:
	// - Event to be published and queued
	// - Event processor to process TABLE_REMOVED
	// - finalizeTableRemoval to complete
	require.Eventually(t, func() bool {
		_, ok := env.PokerSrv.GetTable(tableID)
		return !ok
	}, 5*time.Second, 100*time.Millisecond, "table should be removed after game over")

	_, err = env.PokerClient.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
	require.Error(t, err, "game state should not be available after table removal")
	assert.Equal(t, codes.NotFound, status.Code(err), "expected NotFound once table is removed")
}
