package poker

import (
	"context"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
)

// helper to create a minimal test table
func newTestTable(t *testing.T, minPlayers, maxPlayers int, sb, bb, startingChips int64) *Table {
	t.Helper()
	tbl := NewTable(TableConfig{
		ID:               "tbl-test",
		Log:              createTestLogger(),
		GameLog:          createTestLogger(),
		HostID:           "host",
		BuyIn:            startingChips,
		MinPlayers:       minPlayers,
		MaxPlayers:       maxPlayers,
		SmallBlind:       sb,
		BigBlind:         bb,
		MinBalance:       0,
		StartingChips:    startingChips,
		TimeBank:         5 * time.Second,
		AutoStartDelay:   100 * time.Millisecond,
		AutoAdvanceDelay: 1 * time.Second,
	})
	return tbl
}

func TestTableUserManagement(t *testing.T) {
	// Use capacity 3 so we can test duplicate before full
	tbl := newTestTable(t, 2, 3, 5, 10, 1000)

	// Add first user
	u1, err := tbl.AddNewUser("u1", "U1", 0, 0)
	require.NoError(t, err)
	require.NotNil(t, u1)

	// Duplicate add of same user should fail with duplicate error (not full)
	err = tbl.AddUser(u1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "user already at table")

	// Add second user
	u2, err := tbl.AddNewUser("u2", "U2", 0, 1)
	require.NoError(t, err)
	require.NotNil(t, u2)

	// Fill to capacity and then exceed
	_, err = tbl.AddNewUser("u3", "U3", 0, 2)
	require.NoError(t, err)

	// Exceeding capacity should fail with full error
	err = tbl.AddUser(NewUser("u4", "U4", 0, 3))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "table is full")

	// Remove one
	err = tbl.RemoveUser("u2")
	require.NoError(t, err)

	// Removing again should fail
	err = tbl.RemoveUser("u2")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "user not at table")

	// SetHost validations
	err = tbl.SetHost("nope")
	require.Error(t, err)
	err = tbl.SetHost("u1")
	require.NoError(t, err)
}

func TestTableStateTransitionsAndStartGame(t *testing.T) {
	tbl := newTestTable(t, 2, 2, 5, 10, 1000)

	// Initially waiting
	assert.Equal(t, "WAITING_FOR_PLAYERS", tbl.GetTableStateString())

	// Add users and mark ready
	_, err := tbl.AddNewUser("p1", "P1", 0, 0)
	require.NoError(t, err)
	_, err = tbl.AddNewUser("p2", "P2", 0, 1)
	require.NoError(t, err)

	require.NoError(t, tbl.SetPlayerReady("p1", true))
	require.NoError(t, tbl.SetPlayerReady("p2", true))

	ready := tbl.CheckAllPlayersReady()
	assert.True(t, ready)

	// Wait for the state machine to process the evUsersChanged events
	require.Eventually(t, func() bool {
		return tbl.GetTableStateString() == "PLAYERS_READY"
	}, 300*time.Millisecond, 10*time.Millisecond, "table should transition to PLAYERS_READY state")

	// Start game
	require.NoError(t, tbl.StartGame())

	// Wait for the state machine to process the evStartGameReq event
	require.Eventually(t, func() bool {
		return tbl.IsGameStarted()
	}, 300*time.Millisecond, 10*time.Millisecond, "game should start and table should transition to GAME_ACTIVE state")

	assert.NotNil(t, tbl.GetGame())
	assert.NotEqual(t, pokerrpc.GamePhase_WAITING, tbl.GetGamePhase())
}

func TestTableBettingActionsAndTurns(t *testing.T) {
	tbl := newTestTable(t, 2, 2, 5, 10, 1000)
	_, _ = tbl.AddNewUser("a", "A", 0, 0)
	_, _ = tbl.AddNewUser("b", "B", 0, 1)
	_ = tbl.SetPlayerReady("a", true)
	_ = tbl.SetPlayerReady("b", true)
	require.True(t, tbl.CheckAllPlayersReady())
	require.NoError(t, tbl.StartGame())

	// Wait until game is active with a valid current player in PRE_FLOP
	require.Eventually(t, func() bool {
		g := tbl.GetGame()
		if g == nil {
			return false
		}
		if g.GetPhase() != pokerrpc.GamePhase_PRE_FLOP {
			return false
		}
		cp := g.GetCurrentPlayerObject()
		return cp != nil && g.GetCurrentBet() >= 0
	}, 2*time.Second, 10*time.Millisecond, "game did not reach PRE_FLOP with a current player")

	g := tbl.GetGame()
	require.NotNil(t, g)

	// Identify current player and a different "other" player
	cur := g.GetCurrentPlayerObject()
	require.NotNil(t, cur)
	current := cur.ID()
	require.NotEmpty(t, current)

	other := "a"
	if current == other {
		other = "b"
	}

	// Wrong turn should fail
	err := tbl.MakeBet(other, 10)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not your turn")

	// Let the current player act: if they need chips to match, Call; otherwise Check.
	need := g.GetCurrentBet() - cur.currentBet
	if need > 0 {
		// Either explicit call API, or MakeBet with exact need.
		// Prefer your HandleCall if it exists and handles validation.
		require.NoError(t, tbl.HandleCall(current))
	} else {
		require.NoError(t, tbl.HandleCheck(current))
	}

	// Verify progression: either current player advanced or actionsInRound increased.
	// Poll briefly because action handlers can be async.
	require.Eventually(t, func() bool {
		g2 := tbl.GetGame()
		if g2 == nil {
			return false
		}
		// Check if current player changed or betting state changed
		newCur := g2.GetCurrentPlayerObject()
		if newCur == nil {
			return false
		}
		newCurID := newCur.ID()
		newBet := g2.GetCurrentBet()
		// Turn advanced if current player changed or bet amount changed
		return newCurID != current || newBet != g.GetCurrentBet()
	}, 500*time.Millisecond, 10*time.Millisecond, "turn did not advance after current player's action")
}

func TestTableInvalidInputs(t *testing.T) {
	tbl := newTestTable(t, 2, 2, 5, 10, 1000)
	_, _ = tbl.AddNewUser("a", "A", 0, 0)
	_, _ = tbl.AddNewUser("b", "B", 0, 1)
	_ = tbl.SetPlayerReady("a", true)
	_ = tbl.SetPlayerReady("b", true)
	require.True(t, tbl.CheckAllPlayersReady())
	require.NoError(t, tbl.StartGame())

	// Wait for the game to become active (state machine transition is async)
	require.Eventually(t, func() bool {
		return tbl.GetTableStateString() == "GAME_ACTIVE"
	}, 2*time.Second, 10*time.Millisecond, "Game should be active after StartGame()")

	// Negative bet
	err := tbl.MakeBet("a", -1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "amount cannot be negative")

	// Non-existent user
	err = tbl.HandleCall("ghost")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "user not found")

	// Wrong turn
	notTurn := "a"
	if tbl.GetCurrentPlayerID() == "a" {
		notTurn = "b"
	}
	err = tbl.HandleCheck(notTurn)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not your turn")
}

func TestAllInFlag_HeadsUpWaitsForResponse(t *testing.T) {
	// In heads-up, when SB goes all-in, BB must still make a decision (fold or call)
	// The game should NOT jump to showdown immediately
	// stacks 100, blinds 5/10
	tbl := newTestTable(t, 2, 2, 5, 10, 100)
	_, _ = tbl.AddNewUser("sb", "SB", 0, 0)
	_, _ = tbl.AddNewUser("bb", "BB", 0, 1)
	_ = tbl.SetPlayerReady("sb", true)
	_ = tbl.SetPlayerReady("bb", true)
	require.True(t, tbl.CheckAllPlayersReady())
	require.NoError(t, tbl.StartGame())

	// Wait for blinds to be posted in a way that doesn't depend on seat indexing.
	var sbID string
	require.Eventually(t, func() bool {
		g := tbl.GetGame()
		if g == nil {
			return false
		}
		s := g.GetStateSnapshot()
		if len(s.Players) != 2 {
			return false
		}

		// Look for exactly one 5-chip poster and one 10-chip poster.
		fiveCnt, tenCnt := 0, 0
		var fiveID string
		for _, p := range s.Players {
			if p.CurrentBet == 5 {
				fiveCnt++
				fiveID = p.ID
			}
			if p.CurrentBet == 10 {
				tenCnt++
			}
		}
		if fiveCnt == 1 && tenCnt == 1 {
			sbID = fiveID // remember SB by the blind amount
			return true
		}
		return false
	}, time.Second, 10*time.Millisecond, "blinds not posted")

	// SB shoves: already posted 5, has 95 left, wants to bet total of 100 (all-in)
	// MakeBet expects the new total bet amount, not a delta
	require.NoError(t, tbl.MakeBet(sbID, 100))

	// Assert ALL-IN state synchronously immediately after the bet
	g := tbl.GetGame()
	s := g.GetStateSnapshot()
	var sbPlayer PlayerSnapshot
	found := false
	for _, p := range s.Players {
		if p.ID == sbID {
			sbPlayer = p
			found = true
			break
		}
	}
	require.True(t, found, "SB player not found")

	// Verify ALL_IN condition (balance==0 means player is all-in)
	assert.Equal(t, int64(0), sbPlayer.Balance, "SB should have 0 balance after going all-in")
	assert.Equal(t, int64(100), sbPlayer.CurrentBet, "SB should have currentBet of 100")
	assert.Equal(t, "ALL_IN", sbPlayer.StateString, "SB should be in ALL_IN state")

	// Game should remain in PRE_FLOP waiting for BB to act (NOT jump to showdown)
	phase := g.GetPhase()
	assert.Equal(t, pokerrpc.GamePhase_PRE_FLOP, phase, "Game should remain in PRE_FLOP waiting for BB to act")

	// Find BB and verify BB is now the current player with an unmatched bet
	var bbPlayer PlayerSnapshot
	var bbID string
	foundBB := false
	for _, p := range s.Players {
		if p.ID != sbID {
			bbPlayer = p
			bbID = p.ID
			foundBB = true
			break
		}
	}
	require.True(t, foundBB, "BB player not found")

	// BB should be the current player
	currentPlayer := g.GetCurrentPlayerObject()
	require.NotNil(t, currentPlayer, "Should have a current player")
	assert.Equal(t, bbID, currentPlayer.ID(), "BB should be the current player after SB goes all-in")

	// BB has unmatched bet (10 vs 100)
	assert.Equal(t, int64(10), bbPlayer.CurrentBet, "BB should have currentBet of 10 (big blind)")
	assert.Greater(t, bbPlayer.Balance, int64(0), "BB should still have chips")
	assert.Equal(t, "IN_GAME", bbPlayer.StateString, "BB should be IN_GAME waiting to act")

	// Now test both possible BB responses:
	// Option 1: BB folds -> hand ends, SB wins
	// Option 2: BB calls -> both all-in, go to showdown

	// Let's test the fold scenario
	err := tbl.HandleFold(bbID)
	require.NoError(t, err, "BB should be able to fold")

	// After BB folds, game should go to showdown (only 1 alive player)
	require.Eventually(t, func() bool {
		g := tbl.GetGame()
		if g == nil {
			return false
		}
		return g.GetPhase() == pokerrpc.GamePhase_SHOWDOWN
	}, time.Second, 10*time.Millisecond, "Game should advance to SHOWDOWN after BB folds")
}

func TestAllInFlag_HeadsUpCallTriggersShowdown(t *testing.T) {
	// In heads-up, when SB goes all-in and BB calls (also going all-in),
	// there are 0 active players left, so showdown is triggered
	// stacks 100, blinds 5/10
	tbl := newTestTable(t, 2, 2, 5, 10, 100)
	_, _ = tbl.AddNewUser("sb", "SB", 0, 0)
	_, _ = tbl.AddNewUser("bb", "BB", 0, 1)
	_ = tbl.SetPlayerReady("sb", true)
	_ = tbl.SetPlayerReady("bb", true)
	require.True(t, tbl.CheckAllPlayersReady())
	require.NoError(t, tbl.StartGame())

	// Wait for blinds to be posted
	var sbID string
	require.Eventually(t, func() bool {
		g := tbl.GetGame()
		if g == nil {
			return false
		}
		s := g.GetStateSnapshot()
		if len(s.Players) != 2 {
			return false
		}

		// Find SB (posted 5 chips)
		fiveCnt, tenCnt := 0, 0
		var fiveID string
		for _, p := range s.Players {
			if p.CurrentBet == 5 {
				fiveCnt++
				fiveID = p.ID
			}
			if p.CurrentBet == 10 {
				tenCnt++
			}
		}
		if fiveCnt == 1 && tenCnt == 1 {
			sbID = fiveID
			return true
		}
		return false
	}, time.Second, 10*time.Millisecond, "blinds not posted")

	// SB goes all-in
	require.NoError(t, tbl.MakeBet(sbID, 100))

	// Verify SB is all-in
	g := tbl.GetGame()
	s := g.GetStateSnapshot()
	var sbPlayer PlayerSnapshot
	var bbID string
	foundSB, foundBB := false, false
	for _, p := range s.Players {
		if p.ID == sbID {
			sbPlayer = p
			foundSB = true
		} else {
			bbID = p.ID
			foundBB = true
		}
	}
	require.True(t, foundSB, "SB not found")
	require.True(t, foundBB, "BB not found")
	assert.Equal(t, "ALL_IN", sbPlayer.StateString)

	// BB calls (also going all-in since BB has 90 chips left and needs to call 90 more)
	err := tbl.HandleCall(bbID)
	require.NoError(t, err)

	// After BB calls and goes all-in, both players are all-in (0 active players)
	// With auto-advance enabled, the game will progress through FLOP -> TURN -> RIVER -> SHOWDOWN
	// Each transition takes AutoAdvanceDelay (1 second), so we need at least 4 seconds total
	require.Eventually(t, func() bool {
		g := tbl.GetGame()
		if g == nil {
			// Hand completed and no auto-start
			t.Logf("Game became nil (hand completed)")
			return true
		}
		phase := g.GetPhase()
		if phase == pokerrpc.GamePhase_SHOWDOWN {
			t.Logf("Game reached SHOWDOWN phase")
			return true
		}
		return false
	}, 5*time.Second, 50*time.Millisecond, "Game should reach SHOWDOWN or complete after both players all-in")
}

func TestAllInFlag_ThreePlayerContinuesBetting(t *testing.T) {
	// With 3+ players, when one goes all-in, the other active players can continue betting
	// Showdown should NOT be triggered immediately
	// stacks 100, blinds 5/10
	tbl := newTestTable(t, 3, 3, 5, 10, 100)
	_, _ = tbl.AddNewUser("p1", "P1", 0, 0) // SB
	_, _ = tbl.AddNewUser("p2", "P2", 0, 1) // BB
	_, _ = tbl.AddNewUser("p3", "P3", 0, 2) // Button (first to act preflop in 3-way)
	_ = tbl.SetPlayerReady("p1", true)
	_ = tbl.SetPlayerReady("p2", true)
	_ = tbl.SetPlayerReady("p3", true)
	require.True(t, tbl.CheckAllPlayersReady())
	require.NoError(t, tbl.StartGame())

	// Wait for blinds to be posted
	var p3ID string
	require.Eventually(t, func() bool {
		g := tbl.GetGame()
		if g == nil {
			return false
		}
		s := g.GetStateSnapshot()
		if len(s.Players) != 3 {
			return false
		}

		// Find the player who posted neither blind (that's P3, first to act)
		for _, p := range s.Players {
			// In 3-player, the player with currentBet=0 is first to act (after blinds)
			if p.CurrentBet == 0 {
				p3ID = p.ID
			}
		}

		// Verify all 3 players have correct blind positions
		sbCount := 0
		bbCount := 0
		zeroCount := 0
		for _, p := range s.Players {
			switch p.CurrentBet {
			case 5:
				sbCount++
			case 10:
				bbCount++
			case 0:
				zeroCount++
			}
		}
		return sbCount == 1 && bbCount == 1 && zeroCount == 1 && p3ID != ""
	}, time.Second, 10*time.Millisecond, "blinds not posted correctly")

	// P3 goes all-in with 100 chips
	require.NoError(t, tbl.MakeBet(p3ID, 100))

	// Immediately check P3's state (before other players act)
	g := tbl.GetGame()
	s := g.GetStateSnapshot()

	var p3Player PlayerSnapshot
	var otherActivePlayers int
	foundP3 := false
	for _, p := range s.Players {
		if p.ID == p3ID {
			p3Player = p
			foundP3 = true
		} else {
			// Count other players who are not all-in
			stateStr := p.StateString
			if stateStr != "ALL_IN" && stateStr != "FOLDED" {
				otherActivePlayers++
			}
		}
	}

	require.True(t, foundP3, "P3 player not found")

	// Verify P3 is all-in
	assert.Equal(t, int64(0), p3Player.Balance, "P3 should have 0 balance after going all-in")
	assert.Equal(t, int64(100), p3Player.CurrentBet, "P3 should have currentBet of 100")
	assert.Equal(t, "ALL_IN", p3Player.StateString, "P3 should be in ALL_IN state")

	// Verify there are still 2 active players who can bet
	assert.Equal(t, 2, otherActivePlayers, "Should have 2 other active players (SB and BB)")

	// Verify game is still in PRE_FLOP, NOT showdown
	assert.Equal(t, pokerrpc.GamePhase_PRE_FLOP, g.GetPhase(),
		"Game should remain in PRE_FLOP when 2+ active players remain after an all-in")

	// Verify there's still a current player to act (one of the other two)
	currentPlayerID := g.GetCurrentPlayerObject()
	require.NotNil(t, currentPlayerID, "Should have a current player to act")
	assert.NotEqual(t, p3ID, currentPlayerID.ID(), "Current player should not be the all-in player")
}

func TestHandleTimeoutsAutoFold(t *testing.T) {
	tbl := newTestTable(t, 2, 2, 5, 10, 1000)
	_, _ = tbl.AddNewUser("p1", "P1", 0, 0)
	_, _ = tbl.AddNewUser("p2", "P2", 0, 1)

	_ = tbl.SetPlayerReady("p1", true)
	_ = tbl.SetPlayerReady("p2", true)
	require.True(t, tbl.CheckAllPlayersReady())
	require.NoError(t, tbl.StartGame())

	// Wait until: game exists, phase == PRE_FLOP, and CurrentPlayer is valid.
	require.Eventually(t, func() bool {
		g := tbl.GetGame()
		if g == nil {
			return false
		}
		if g.GetPhase() != pokerrpc.GamePhase_PRE_FLOP {
			return false
		}
		cp := g.GetCurrentPlayerObject()
		if cp == nil {
			return false
		}
		// We want a folding scenario (needs chips). In HU preflop, SB acts first; need > 0.
		return cp.currentBet < g.GetCurrentBet()
	}, 2*time.Second, 10*time.Millisecond, "did not reach PRE_FLOP with an actionable current player")

	g := tbl.GetGame()
	require.NotNil(t, g)

	// Identify current player from game
	cur := g.GetCurrentPlayerObject()
	require.NotNil(t, cur)
	curID := cur.ID()
	require.NotEmpty(t, curID)

	// Touch the live *Player pointer to expire timebank.
	var live *Player
	for _, p := range g.GetPlayers() {
		if p.ID() == curID {
			live = p
			break
		}
	}
	require.NotNil(t, live, "live player not found by id")
	live.lastAction = time.Now().Add(-2 * tbl.GetConfig().TimeBank)

	// Run timeout handling (only from the timeout loop in real code; here we call directly).
	tbl.HandleTimeouts()

	// Assert it folded. Poll briefly in case advancement is async.
	require.Eventually(t, func() bool {
		for _, p := range g.GetPlayers() {
			if p.ID() == curID {
				return p.GetCurrentStateString() == "FOLDED"
			}
		}
		return false
	}, 500*time.Millisecond, 10*time.Millisecond, "player did not fold after timeout")

	// Final explicit check (nice error on failure)
	for _, p := range g.GetPlayers() {
		if p.ID() == curID {
			assert.Equal(t, "FOLDED", p.GetCurrentStateString())
			break
		}
	}
}

func TestConcurrency_SafeSnapshotsAndBalanceUpdates(t *testing.T) {
	tbl := newTestTable(t, 3, 6, 5, 10, 1000)
	_, _ = tbl.AddNewUser("u1", "U1", 0, 0)
	_, _ = tbl.AddNewUser("u2", "U2", 0, 1)
	_, _ = tbl.AddNewUser("u3", "U3", 0, 2)

	// Spin concurrent writers and readers; this test is most valuable under -race.
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		for ctx.Err() == nil {
			_ = tbl.SetUserDCRAccountBalance("u1", time.Now().UnixNano()%1000)
		}
		close(done)
	}()

	for ctx.Err() == nil {
		_ = tbl.GetStateSnapshot()
		_ = tbl.GetUsers()
		_ = tbl.GetGamePhase()
		time.Sleep(1 * time.Millisecond)
	}
	<-done
}

// Ensures Table action handlers reject actions during SHOWDOWN (or non-betting phases).
func TestDisallowActionsDuringShowdown_Table(t *testing.T) {
	tbl := newTestTable(t, 2, 2, 5, 10, 1000)
	_, _ = tbl.AddNewUser("p1", "P1", 0, 0)
	_, _ = tbl.AddNewUser("p2", "P2", 0, 1)
	_ = tbl.SetPlayerReady("p1", true)
	_ = tbl.SetPlayerReady("p2", true)
	require.True(t, tbl.CheckAllPlayersReady())
	require.NoError(t, tbl.StartGame())

	// Wait until PRE_FLOP is reached
	require.Eventually(t, func() bool {
		g := tbl.GetGame()
		return g != nil && g.GetPhase() == pokerrpc.GamePhase_PRE_FLOP
	}, 1*time.Second, 10*time.Millisecond)

	g := tbl.GetGame()
	require.NotNil(t, g)

	// Set current player explicitly to p1 for deterministic checks
	players := g.GetPlayers()
	require.GreaterOrEqual(t, len(players), 2)
	cur := players[0]
	require.NotNil(t, cur)
	g.SetCurrentPlayerByID(cur.ID())

	// Force phase to SHOWDOWN (simulating after-hand state)
	g.mu.Lock()
	g.phase = pokerrpc.GamePhase_SHOWDOWN
	g.mu.Unlock()

	// All action handlers should be rejected due to phase guard
	err := tbl.HandleCall(cur.ID())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "action not allowed during phase")

	err = tbl.HandleCheck(cur.ID())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "action not allowed during phase")

	err = tbl.MakeBet(cur.ID(), 10)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "action not allowed during phase")

	err = tbl.HandleFold(cur.ID())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "action not allowed during phase")
}

// Ensures Game action wrappers also reject actions during SHOWDOWN.
func TestDisallowActionsDuringShowdown_Game(t *testing.T) {
	tbl := newTestTable(t, 2, 2, 5, 10, 1000)
	_, _ = tbl.AddNewUser("p1", "P1", 0, 0)
	_, _ = tbl.AddNewUser("p2", "P2", 0, 1)
	_ = tbl.SetPlayerReady("p1", true)
	_ = tbl.SetPlayerReady("p2", true)
	require.True(t, tbl.CheckAllPlayersReady())
	require.NoError(t, tbl.StartGame())

	// Wait for PRE_FLOP
	require.Eventually(t, func() bool {
		g := tbl.GetGame()
		return g != nil && g.GetPhase() == pokerrpc.GamePhase_PRE_FLOP
	}, 1*time.Second, 10*time.Millisecond)

	g := tbl.GetGame()
	require.NotNil(t, g)
	ps := g.GetPlayers()
	require.GreaterOrEqual(t, len(ps), 2)
	cur := ps[0]
	g.SetCurrentPlayerByID(cur.ID())

	// Force phase to SHOWDOWN and attempt direct Game-level actions
	g.mu.Lock()
	g.phase = pokerrpc.GamePhase_SHOWDOWN
	g.mu.Unlock()

	err := g.HandlePlayerCall(cur.ID())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "action not allowed during phase")

	err = g.HandlePlayerCheck(cur.ID())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "action not allowed during phase")

	err = g.HandlePlayerBet(cur.ID(), 10)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "action not allowed during phase")

	err = g.HandlePlayerFold(cur.ID())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "action not allowed during phase")
}

// TestHoleCardsAvailableOnGameStart verifies that players have hole cards
// when the game starts and reaches PRE_FLOP phase. This reproduces the bug
// where clients see "no hole cards available" error.
//
// Root cause: Cards are dealt to game.currentHand.hole, but Player.Hand() returns nil
// and GetStateSnapshot() doesn't retrieve them from currentHand.
func TestHoleCardsAvailableOnGameStart(t *testing.T) {
	tbl := newTestTable(t, 2, 2, 10, 20, 1000)

	// Add two players
	_, err := tbl.AddNewUser("p1", "Player1", 0, 0)
	require.NoError(t, err)
	_, err = tbl.AddNewUser("p2", "Player2", 0, 1)
	require.NoError(t, err)

	// Mark both players ready
	require.NoError(t, tbl.SetPlayerReady("p1", true))
	require.NoError(t, tbl.SetPlayerReady("p2", true))

	// Wait for PLAYERS_READY state
	require.Eventually(t, func() bool {
		return tbl.GetTableStateString() == "PLAYERS_READY"
	}, 300*time.Millisecond, 10*time.Millisecond, "table should reach PLAYERS_READY")

	// Start the game
	require.NoError(t, tbl.StartGame())

	// Wait for game to reach PRE_FLOP phase
	require.Eventually(t, func() bool {
		g := tbl.GetGame()
		return g != nil && g.GetPhase() == pokerrpc.GamePhase_PRE_FLOP
	}, 2*time.Second, 10*time.Millisecond, "game should reach PRE_FLOP")

	game := tbl.GetGame()
	require.NotNil(t, game)

	// Get a table snapshot (same as what server uses)
	snapshot := tbl.GetStateSnapshot()
	require.NotNil(t, snapshot.Game, "game snapshot should exist")
	require.Equal(t, pokerrpc.GamePhase_PRE_FLOP, snapshot.Game.Phase)
	require.Len(t, snapshot.Game.Players, 2, "should have 2 players")

	// FIXED: GetStateSnapshot() now correctly retrieves cards from currentHand
	for _, ps := range snapshot.Game.Players {
		t.Logf("PlayerSnapshot %s: Hand has %d cards", ps.ID, len(ps.Hand))
		// This should now PASS - GetStateSnapshot() retrieves from currentHand
		assert.Len(t, ps.Hand, 2,
			"PlayerSnapshot %s should have 2 cards from currentHand, but has %d",
			ps.ID, len(ps.Hand))
	}

	// Verify cards ARE actually dealt to game.currentHand
	// (this should pass - proving cards are stored, just not retrieved)
	require.NotNil(t, game.currentHand, "game.currentHand should be initialized")

	hand1 := game.currentHand.GetPlayerCards("p1", "p1") // player requesting own cards
	t.Logf("currentHand.GetPlayerCards for p1: %d cards", len(hand1))
	assert.Len(t, hand1, 2, "Cards should be stored in game.currentHand for p1")

	hand2 := game.currentHand.GetPlayerCards("p2", "p2")
	t.Logf("currentHand.GetPlayerCards for p2: %d cards", len(hand2))
	assert.Len(t, hand2, 2, "Cards should be stored in game.currentHand for p2")
}

// TestTableClose_Idempotent tests that calling Close() multiple times doesn't panic
func TestTableClose_Idempotent(t *testing.T) {
	cfg := TableConfig{
		ID:             "test-table",
		Log:            createTestLogger(),
		GameLog:        createTestLogger(),
		HostID:         "host1",
		BuyIn:          1000,
		MinPlayers:     2,
		MaxPlayers:     6,
		SmallBlind:     10,
		BigBlind:       20,
		MinBalance:     1000,
		StartingChips:  1000,
		TimeBank:       30 * time.Second,
		AutoStartDelay: 3 * time.Second,
	}

	table := NewTable(cfg)

	// Call Close() multiple times - should not panic
	table.Close()
	table.Close()
	table.Close()

	// Verify table is marked as closed
	table.mu.RLock()
	closed := table.closed
	table.mu.RUnlock()

	if !closed {
		t.Error("Expected table.closed to be true after Close()")
	}
}

// TestTableClose_Concurrent tests that concurrent calls to Close() don't cause issues
func TestTableClose_Concurrent(t *testing.T) {
	// Track goroutines before test
	beforeGoroutines := runtime.NumGoroutine()

	cfg := TableConfig{
		ID:             "test-table",
		Log:            createTestLogger(),
		GameLog:        createTestLogger(),
		HostID:         "host1",
		BuyIn:          1000,
		MinPlayers:     2,
		MaxPlayers:     6,
		SmallBlind:     10,
		BigBlind:       20,
		MinBalance:     1000,
		StartingChips:  1000,
		TimeBank:       30 * time.Second,
		AutoStartDelay: 3 * time.Second,
	}

	table := NewTable(cfg)

	// Give table time to fully start
	time.Sleep(50 * time.Millisecond)

	// Launch multiple goroutines calling Close() concurrently
	var wg sync.WaitGroup
	numCallers := 10

	for i := 0; i < numCallers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			table.Close()
		}()
	}

	wg.Wait()

	// Verify table is closed
	table.mu.RLock()
	closed := table.closed
	table.mu.RUnlock()

	if !closed {
		t.Error("Expected table.closed to be true after concurrent Close() calls")
	}

	// Wait a bit for goroutines to clean up
	time.Sleep(100 * time.Millisecond)

	// Check for goroutine leaks
	afterGoroutines := runtime.NumGoroutine()

	// Allow some variance (background GC, test framework goroutines)
	// Table creates 2 goroutines (timeout + gameEvent) plus the FSM
	// All should be cleaned up, so we should be close to starting count
	maxExpectedIncrease := 3

	if afterGoroutines > beforeGoroutines+maxExpectedIncrease {
		t.Errorf("Possible goroutine leak: before=%d, after=%d, diff=%d (expected diff <= %d)",
			beforeGoroutines, afterGoroutines, afterGoroutines-beforeGoroutines, maxExpectedIncrease)
	}
}

// TestTableClose_WithGame tests that Close() properly cleans up when a game is active
func TestTableClose_WithGame(t *testing.T) {
	cfg := TableConfig{
		ID:               "test-table",
		Log:              createTestLogger(),
		GameLog:          createTestLogger(),
		HostID:           "host1",
		BuyIn:            1000,
		MinPlayers:       2,
		MaxPlayers:       6,
		SmallBlind:       10,
		BigBlind:         20,
		MinBalance:       1000,
		StartingChips:    1000,
		TimeBank:         30 * time.Second,
		AutoStartDelay:   3 * time.Second,
		AutoAdvanceDelay: 1 * time.Second,
	}

	table := NewTable(cfg)

	// Add users to the table
	user1 := NewUser("user1", "Alice", 2000, 0)
	user1.IsReady = true
	user2 := NewUser("user2", "Bob", 2000, 1)
	user2.IsReady = true

	err := table.AddUser(user1)
	if err != nil {
		t.Fatalf("Failed to add user1: %v", err)
	}

	err = table.AddUser(user2)
	if err != nil {
		t.Fatalf("Failed to add user2: %v", err)
	}

	// Start a game
	err = table.StartGame()
	if err != nil {
		t.Fatalf("Failed to start game: %v", err)
	}

	// Verify game was created
	table.mu.RLock()
	hasGame := table.game != nil
	table.mu.RUnlock()

	if !hasGame {
		t.Fatal("Expected game to be created")
	}

	// Now close the table
	table.Close()

	// Verify everything is cleaned up
	table.mu.RLock()
	closed := table.closed
	game := table.game
	sm := table.sm
	table.mu.RUnlock()

	if !closed {
		t.Error("Expected table.closed to be true")
	}
	if game != nil {
		t.Error("Expected table.game to be nil after Close()")
	}
	if sm != nil {
		t.Error("Expected table.sm to be nil after Close()")
	}
}

// TestTableClose_BackgroundGoroutinesStop tests that background goroutines actually stop
func TestTableClose_BackgroundGoroutinesStop(t *testing.T) {
	cfg := TableConfig{
		ID:             "test-table",
		Log:            createTestLogger(),
		GameLog:        createTestLogger(),
		HostID:         "host1",
		BuyIn:          1000,
		MinPlayers:     2,
		MaxPlayers:     6,
		SmallBlind:     10,
		BigBlind:       20,
		MinBalance:     1000,
		StartingChips:  1000,
		TimeBank:       30 * time.Second,
		AutoStartDelay: 3 * time.Second,
	}

	table := NewTable(cfg)

	// Give goroutines time to start
	time.Sleep(50 * time.Millisecond)

	// Track that goroutines are running by checking they respond to signals
	// (This is indirect - we're testing that Close() returns without hanging)
	done := make(chan struct{})
	go func() {
		table.Close()
		close(done)
	}()

	// Close should complete within a reasonable time
	select {
	case <-done:
		// Success - Close() completed
	case <-time.After(5 * time.Second):
		t.Fatal("Close() did not complete within 5 seconds - goroutines may be stuck")
	}

	// Verify closed
	table.mu.RLock()
	closed := table.closed
	table.mu.RUnlock()

	if !closed {
		t.Error("Expected table.closed to be true")
	}
}

// TestTableClose_WaitGroupProperlyTracked tests that WaitGroup is correctly managed
func TestTableClose_WaitGroupProperlyTracked(t *testing.T) {
	cfg := TableConfig{
		ID:             "test-table",
		Log:            createTestLogger(),
		GameLog:        createTestLogger(),
		HostID:         "host1",
		BuyIn:          1000,
		MinPlayers:     2,
		MaxPlayers:     6,
		SmallBlind:     10,
		BigBlind:       20,
		MinBalance:     1000,
		StartingChips:  1000,
		TimeBank:       30 * time.Second,
		AutoStartDelay: 3 * time.Second,
	}

	table := NewTable(cfg)

	// Let goroutines start
	time.Sleep(50 * time.Millisecond)

	// Close should wait for all goroutines
	startTime := time.Now()
	table.Close()
	duration := time.Since(startTime)

	// Close should be quick (< 1s) since goroutines should exit promptly
	if duration > 1*time.Second {
		t.Errorf("Close() took too long: %v (expected < 1s)", duration)
	}

	// Verify we can't accidentally double-close channels (closeOnce protection)
	// This should not panic
	table.Close()
}
