package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	testenv "github.com/vctt94/pokerbisonrelay/e2e/internal/testenv"
	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
	"google.golang.org/protobuf/proto"
)

// Regression-style E2E: when a player loses all chips, the game roster should
// prune that player (GameEventPlayerLost flow). We run real server + gRPC, play
// with 3 players, have two shove until one busts, and assert the busted player
// disappears from the game state roster.
func TestPlayerLostPrunesGameRoster_E2E(t *testing.T) {
	env := testenv.New(t)
	defer env.Close()

	ctx := context.Background()

	players := []string{"p1", "p2", "p3"}
	shortIDs := map[string]string{}
	for _, p := range players {
		shortIDs[p] = testenv.PlayerIDToShortIDString(p)
	}
	aliasForID := func(id string) string {
		for alias, short := range shortIDs {
			if short == id {
				return alias
			}
		}
		return ""
	}

	// Create a fast table: small stacks + short auto timings to reach bust quickly.
	hostCtx := env.ContextWithToken(ctx, "p1")
	createResp, err := env.CreateTable(hostCtx, &pokerrpc.CreateTableRequest{
		PlayerId:      "p1",
		SmallBlind:    5,
		BigBlind:      10,
		MinPlayers:    2, // allow continued play after one busts
		MaxPlayers:    3,
		BuyIn:         0,
		StartingChips: 100, // small stack to bust quickly
		AutoStartMs:   200, // disable auto-start so we can inspect post-showdown roster
		AutoAdvanceMs: 500,
	})
	require.NoError(t, err)
	tableID := createResp.TableId

	// Seat remaining players and mark everyone ready.
	for _, p := range []string{"p2", "p3"} {
		_, err := env.JoinTable(ctx, p, tableID)
		require.NoError(t, err)
	}
	for _, p := range players {
		_, err := env.SetPlayerReady(ctx, p, tableID)
		require.NoError(t, err)
	}

	env.WaitForGameStart(ctx, tableID, 3*time.Second)
	env.WaitForGamePhase(ctx, tableID, pokerrpc.GamePhase_PRE_FLOP, 3*time.Second)

	eliminated := ""
	const maxHands = 6
	var capturedShowdown *pokerrpc.GameUpdate

	for hand := 1; hand <= maxHands && eliminated == ""; hand++ {
		// Ensure we start at PRE_FLOP for this hand (may already be there).
		env.WaitForGamePhase(ctx, tableID, pokerrpc.GamePhase_PRE_FLOP, 3*time.Second)

		// Drive betting: p3 always folds; p1 and p2 shove (bet balance+blind).
		handDeadline := time.Now().Add(6 * time.Second)
		for {
			state, removed := env.GetGameStateAllowNotFound(ctx, tableID)
			require.False(t, removed, "table should exist during hand")

			if state.Phase == pokerrpc.GamePhase_SHOWDOWN {
				break
			}
			switch state.Phase {
			case pokerrpc.GamePhase_PRE_FLOP, pokerrpc.GamePhase_FLOP, pokerrpc.GamePhase_TURN, pokerrpc.GamePhase_RIVER:
			default:
				// Wait for betting street
				if time.Now().After(handDeadline) {
					t.Fatalf("hand %d stuck in non-betting phase %v", hand, state.Phase)
				}
				time.Sleep(10 * time.Millisecond)
				continue
			}

			currID := state.CurrentPlayer
			if currID == "" {
				if time.Now().After(handDeadline) {
					t.Fatalf("hand %d has no current player in phase %v", hand, state.Phase)
				}
				time.Sleep(10 * time.Millisecond)
				continue
			}

			alias := aliasForID(currID)
			require.NotEmpty(t, alias, "current player alias should resolve")

			var self *pokerrpc.Player
			for _, p := range state.Players {
				if p != nil && p.Id == currID {
					self = p
					break
				}
			}
			if self == nil || !self.IsTurn {
				if time.Now().After(handDeadline) {
					t.Fatalf("hand %d: missing IsTurn snapshot for %s", hand, currID)
				}
				time.Sleep(10 * time.Millisecond)
				continue
			}

			var actErr error
			ctxWithToken := env.ContextWithToken(ctx, alias)
			if alias == "p3" {
				_, actErr = env.PokerClient.FoldBet(ctxWithToken, &pokerrpc.FoldBetRequest{
					PlayerId: currID,
					TableId:  tableID,
				})
			} else {
				allIn := self.CurrentBet + self.Balance
				_, actErr = env.PokerClient.MakeBet(ctxWithToken, &pokerrpc.MakeBetRequest{
					PlayerId: currID,
					TableId:  tableID,
					Amount:   allIn,
				})
			}

			if actErr != nil {
				if isTransientTurnError(actErr) && time.Now().Before(handDeadline) {
					time.Sleep(10 * time.Millisecond)
					continue
				}
				require.NoError(t, actErr, "action failed for %s", alias)
			}
		}

		// Wait for showdown settlement or table removal.
		env.WaitForShowdownOrRemoval(ctx, tableID, 5*time.Second)

		// Cache the first SHOWDOWN snapshot before table/game teardown.
		if capturedShowdown == nil {
			require.Eventually(t, func() bool {
				ctxPoll, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
				defer cancel()
				resp, err := env.PokerClient.GetGameState(ctxPoll, &pokerrpc.GetGameStateRequest{TableId: tableID})
				if err != nil {
					return false
				}
				if resp.GameState.GetPhase() != pokerrpc.GamePhase_SHOWDOWN {
					return false
				}
				capturedShowdown = proto.Clone(resp.GameState).(*pokerrpc.GameUpdate)
				var dbg string
				for _, p := range capturedShowdown.Players {
					if p != nil {
						dbg += p.Id + fmt.Sprintf(" bal=%d bet=%d folded=%t | ", p.Balance, p.CurrentBet, p.Folded)
					}
				}
				t.Logf("captured showdown snapshot: pot=%d currentBet=%d players=[%s]", capturedShowdown.Pot, capturedShowdown.CurrentBet, dbg)
				return true
			}, 2*time.Second, 50*time.Millisecond, "expected to capture SHOWDOWN snapshot before teardown")
		}

		// Identify the busted player (balance==0) from the post-showdown snapshot,
		// then ensure *that* player ID is pruned from subsequent game state.
		// After PLAYER_LOST, the next hand should start with only the survivors.
		// Detect which original player disappeared from the roster without relying on balances.
		require.Eventually(t, func() bool {
			ctxPoll, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
			defer cancel()

			resp, err := env.PokerClient.GetGameState(ctxPoll, &pokerrpc.GetGameStateRequest{TableId: tableID})
			if err != nil {
				return false
			}

			// Debug: log current balances to understand bust/over conditions.
			var dbg string
			for _, p := range resp.GameState.Players {
				if p != nil {
					dbg += p.Id + fmt.Sprintf(" bal=%d bet=%d folded=%t | ", p.Balance, p.CurrentBet, p.Folded)
				}
			}
			t.Logf("post-showdown snapshot: pot=%d currentBet=%d players=[%s]", resp.GameState.Pot, resp.GameState.CurrentBet, dbg)

			present := make(map[string]bool)
			for _, p := range resp.GameState.Players {
				if p != nil {
					present[p.Id] = true
				}
			}

			missingIDs := []string{}
			for _, short := range shortIDs {
				if !present[short] {
					missingIDs = append(missingIDs, short)
				}
			}

			if len(missingIDs) != 1 {
				return false
			}
			eliminated = missingIDs[0]
			return true
		}, 3*time.Second, 50*time.Millisecond, "expected exactly one player to disappear from roster after bust")

		require.NotEmpty(t, eliminated, "expected to identify which player was pruned")

		// Cross-check internal game roster to ensure the Game.players slice was pruned.
		table, ok := env.PokerSrv.GetTable(tableID)
		require.True(t, ok, "table should still exist")
		game := table.GetGame()
		require.NotNil(t, game, "game should still be active")
		livePlayers := game.GetPlayers()
		require.Len(t, livePlayers, 2, "game roster should have 2 survivors")
		for _, p := range livePlayers {
			require.NotNil(t, p, "survivor player should not be nil")
			require.NotEqual(t, eliminated, p.ID(), "busted player should not remain in game roster")
		}
	}

	require.NotEmpty(t, eliminated, "expected to eliminate one player within %d hands", maxHands)
	t.Logf("✓ Eliminated player pruned from game roster: %s", eliminated)
}
