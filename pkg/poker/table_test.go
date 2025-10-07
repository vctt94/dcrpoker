package poker

import (
	"context"
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
		ID:             "tbl-test",
		Log:            createTestLogger(),
		GameLog:        createTestLogger(),
		HostID:         "host",
		BuyIn:          startingChips,
		MinPlayers:     minPlayers,
		MaxPlayers:     maxPlayers,
		SmallBlind:     sb,
		BigBlind:       bb,
		MinBalance:     0,
		StartingChips:  startingChips,
		TimeBank:       50 * time.Millisecond,
		AutoStartDelay: 0,
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

func TestAllInExactCall(t *testing.T) {
	tbl := newTestTable(t, 2, 2, 5, 10, 10)
	_, _ = tbl.AddNewUser("sb", "SB", 0, 0)
	_, _ = tbl.AddNewUser("bb", "BB", 0, 1)
	_ = tbl.SetPlayerReady("sb", true)
	_ = tbl.SetPlayerReady("bb", true)
	require.True(t, tbl.CheckAllPlayersReady())
	require.NoError(t, tbl.StartGame())

	// Wait until blinds are posted: expect one player at 5 (SB) and one at 10 (BB).
	require.Eventually(t, func() bool {
		g := tbl.GetGame()
		if g == nil {
			return false
		}
		ps := g.GetPlayers()
		if len(ps) != 2 {
			return false
		}
		// find SB by currentBet==5 and balance==5 (since stacks are 10)
		sbOK, bbOK := false, false
		for _, p := range ps {
			if p.CurrentBet() == 5 && p.Balance() == 5 {
				sbOK = true
			}
			if p.CurrentBet() == 10 {
				bbOK = true
			}
		}
		return sbOK && bbOK
	}, 300*time.Millisecond, 10*time.Millisecond, "blinds not posted")

	// Identify SB explicitly (currentBet==5).
	g := tbl.GetGame()
	require.NotNil(t, g)
	var sb *Player
	for _, p := range g.GetPlayers() {
		if p.CurrentBet() == 5 && p.Balance() == 5 {
			sb = p
			break
		}
	}
	require.NotNil(t, sb, "could not find SB")

	// Wait for SB to be in IN_GAME state (not AT_TABLE) so evReeval can work
	require.Eventually(t, func() bool {
		return sb.GetCurrentStateString() == "IN_GAME"
	}, 300*time.Millisecond, 10*time.Millisecond, "SB did not reach IN_GAME state")

	// SB calls exact remainder (5) -> should be all-in with currentBet=10.
	require.NoError(t, tbl.HandleCall(sb.ID()))

	// Wait for ALL_IN on SB.
	require.Eventually(t, func() bool {
		return sb.Balance() == 0 && sb.CurrentBet() == 10 && sb.GetCurrentStateString() == "ALL_IN"
	}, 300*time.Millisecond, 10*time.Millisecond, "SB did not go ALL_IN after exact call")

	assert.Equal(t, int64(0), sb.Balance())
	assert.Equal(t, int64(10), sb.CurrentBet())
	assert.Equal(t, "ALL_IN", sb.GetCurrentStateString())
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
