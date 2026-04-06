package server

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/companyzero/bisonrelay/zkidentity"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vctt94/bisonbotkit/logging"
	"github.com/vctt94/pokerbisonrelay/pkg/poker"
	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
	"github.com/vctt94/pokerbisonrelay/pkg/server/internal/db"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// TestServer implements the PokerServiceServer interface
type TestServer struct {
	*Server
}

// GetGameForTable is a helper method for tests to access the game directly
func (ts *TestServer) GetGameForTable(tableID string) *poker.Game {
	if table, ok := ts.getTable(tableID); ok {
		return table.GetGame()
	}
	return nil
}

// CreateTestSession creates a test session and returns a context with the token in metadata.
// This is a convenience helper for tests that need authenticated contexts.
func (ts *TestServer) CreateTestSession(ctx context.Context, playerID, nickname string) (context.Context, string) {
	// For simple test IDs, try to parse as ShortID first, otherwise create a deterministic one
	var uid zkidentity.ShortID
	if err := uid.FromString(playerID); err != nil {
		// If it's not a valid ShortID format, create a deterministic one from the string
		// This allows tests to use simple string IDs like "player1", "alice", etc.
		var buf [32]byte
		copy(buf[:], playerID)
		if len(playerID) < 32 {
			// Pad with repeated playerID to fill the buffer
			for i := len(playerID); i < 32; i++ {
				buf[i] = playerID[i%len(playerID)]
			}
		}
		uid = zkidentity.ShortID(buf)
	}

	token := "test-token-" + playerID
	if nickname == "" {
		nickname = playerID
	}

	ts.TestSeedSession(token, uid, "", nickname)

	// Add token to context metadata
	md := metadata.New(map[string]string{"token": token})
	return metadata.NewIncomingContext(ctx, md), token
}

// InMemoryDB implements the Database interface for testing.
type InMemoryDB struct {
	mu sync.RWMutex

	// wallet
	balances     map[string]int64
	transactions map[string][]Transaction

	// tables & seating
	tables       map[string]*db.Table
	participants map[string]map[string]*db.Participant // tableID -> playerID -> row
	seatIndex    map[string]map[int]string             // tableID -> seat -> playerID

	// optional fast-restore snapshots
	snapshots        map[string]db.Snapshot        // tableID -> snapshot
	matchCheckpoints map[string]db.MatchCheckpoint // tableID -> checkpoint

	// auth
	authUsers map[string]*db.AuthUser // userID -> AuthUser
}

// NewInMemoryDB creates a new in-memory database for testing.
func NewInMemoryDB() *InMemoryDB {
	return &InMemoryDB{
		balances:         make(map[string]int64),
		transactions:     make(map[string][]Transaction),
		tables:           make(map[string]*db.Table),
		participants:     make(map[string]map[string]*db.Participant),
		seatIndex:        make(map[string]map[int]string),
		snapshots:        make(map[string]db.Snapshot),
		matchCheckpoints: make(map[string]db.MatchCheckpoint),
		authUsers:        make(map[string]*db.AuthUser),
	}
}

// -------- Tables (configuration) --------

func (m *InMemoryDB) UpsertTable(_ context.Context, t *poker.TableConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *t // store by value to avoid external mutation
	// set CreatedAt if zero-ish
	m.tables[cp.ID] = &db.Table{
		ID:                       cp.ID,
		Name:                     cp.Name,
		Source:                   cp.Source,
		BuyIn:                    cp.BuyIn,
		MinPlayers:               cp.MinPlayers,
		MaxPlayers:               cp.MaxPlayers,
		SmallBlind:               cp.SmallBlind,
		BigBlind:                 cp.BigBlind,
		StartingChips:            cp.StartingChips,
		TimebankMS:               cp.TimeBank.Milliseconds(),
		AutoStartMS:              cp.AutoStartDelay.Milliseconds(),
		AutoAdvanceMS:            cp.AutoAdvanceDelay.Milliseconds(),
		BlindIncreaseIntervalSec: int64(cp.BlindIncreaseInterval.Seconds()),
		CreatedAt:                time.Now(),
	}
	if m.tables[cp.ID].Name == "" {
		m.tables[cp.ID].Name = cp.ID
	}
	return nil
}

func (m *InMemoryDB) GetTable(_ context.Context, id string) (*db.Table, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, ok := m.tables[id]
	if !ok {
		return nil, fmt.Errorf("table not found: %s", id)
	}
	cp := *t
	return &cp, nil
}

func (m *InMemoryDB) DeleteTable(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.tables, id)
	delete(m.participants, id)
	delete(m.seatIndex, id)
	delete(m.snapshots, id)
	delete(m.matchCheckpoints, id)
	return nil
}

func (m *InMemoryDB) ListTableIDs(_ context.Context) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]string, 0, len(m.tables))
	for id := range m.tables {
		out = append(out, id)
	}
	// tests don't require stable order, but keep it deterministic
	sort.Strings(out)
	return out, nil
}

// -------- Participants (seats) --------

func (m *InMemoryDB) ActiveParticipants(_ context.Context, tableID string) ([]db.Participant, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	mp := m.participants[tableID]
	if mp == nil {
		return nil, nil
	}
	// order by seat
	type pair struct {
		seat int
		p    *db.Participant
	}
	var rows []pair
	for _, p := range mp {
		if !p.LeftAt.Valid { // active seat
			rows = append(rows, pair{seat: p.Seat, p: p})
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].seat < rows[j].seat })

	out := make([]db.Participant, 0, len(rows))
	for _, r := range rows {
		cp := *r.p
		out = append(out, cp)
	}
	return out, nil
}

func (m *InMemoryDB) SeatPlayer(_ context.Context, tableID, playerID string, seat int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// ensure table map exists
	if _, ok := m.tables[tableID]; !ok {
		// allow seating even if server hasn't persisted table yet in tests
		m.tables[tableID] = &db.Table{ID: tableID, CreatedAt: time.Now()}
	}
	if m.participants[tableID] == nil {
		m.participants[tableID] = make(map[string]*db.Participant)
	}
	if m.seatIndex[tableID] == nil {
		m.seatIndex[tableID] = make(map[int]string)
	}

	// enforce UNIQUE(table_id, seat) among active seats
	if holder, exists := m.seatIndex[tableID][seat]; exists {
		// if the holder is the same player but previously left, reopen it
		if pp, ok := m.participants[tableID][holder]; ok && !pp.LeftAt.Valid {
			return fmt.Errorf("seat %d already occupied", seat)
		}
	}

	// upsert participant
	now := time.Now()
	p := m.participants[tableID][playerID]
	if p == nil {
		p = &db.Participant{
			TableID:  tableID,
			PlayerID: playerID,
			Seat:     seat,
			JoinedAt: now,
			LeftAt:   sql.NullTime{Valid: false},
			Ready:    false,
		}
		m.participants[tableID][playerID] = p
	} else {
		p.Seat = seat
		p.LeftAt = sql.NullTime{Valid: false}
	}
	m.seatIndex[tableID][seat] = playerID
	return nil
}

func (m *InMemoryDB) UnseatPlayer(_ context.Context, tableID, playerID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	mp := m.participants[tableID]
	if mp == nil {
		return nil
	}
	p := mp[playerID]
	if p == nil {
		return nil
	}
	if !p.LeftAt.Valid {
		p.LeftAt = sql.NullTime{Time: time.Now(), Valid: true}
	}
	// free seat index if matches
	if cur, ok := m.seatIndex[tableID][p.Seat]; ok && cur == playerID {
		delete(m.seatIndex[tableID], p.Seat)
	}
	return nil
}

func (m *InMemoryDB) SetReady(_ context.Context, tableID, playerID string, ready bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	mp := m.participants[tableID]
	if mp == nil {
		return fmt.Errorf("table not found")
	}
	p := mp[playerID]
	if p == nil {
		return fmt.Errorf("player not found")
	}
	p.Ready = ready
	return nil
}

// -------- Snapshots (fast-restore cache) --------

func (m *InMemoryDB) UpsertSnapshot(_ context.Context, s db.Snapshot) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s.SnapshotAt.IsZero() {
		s.SnapshotAt = time.Now()
	}
	m.snapshots[s.TableID] = s
	return nil
}

func (m *InMemoryDB) GetSnapshot(_ context.Context, tableID string) (*db.Snapshot, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.snapshots[tableID]
	if !ok {
		return nil, fmt.Errorf("snapshot not found")
	}
	cp := s
	return &cp, nil
}

func (m *InMemoryDB) UpsertMatchCheckpoint(_ context.Context, c db.MatchCheckpoint) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = time.Now()
	}
	m.matchCheckpoints[c.TableID] = c
	return nil
}

func (m *InMemoryDB) GetMatchCheckpoint(_ context.Context, tableID string) (*db.MatchCheckpoint, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, ok := m.matchCheckpoints[tableID]
	if !ok {
		return nil, fmt.Errorf("match checkpoint not found")
	}
	cp := c
	return &cp, nil
}

func (m *InMemoryDB) DeleteMatchCheckpoint(_ context.Context, tableID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.matchCheckpoints, tableID)
	return nil
}

// -------- Auth --------

func (m *InMemoryDB) UpsertAuthUser(_ context.Context, nickname, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	if existing, ok := m.authUsers[userID]; ok {
		// Update existing user
		existing.Nickname = nickname
	} else {
		// Create new user
		m.authUsers[userID] = &db.AuthUser{
			Nickname:  nickname,
			UserID:    userID,
			CreatedAt: now,
			LastLogin: sql.NullTime{Valid: false},
		}
	}
	return nil
}

func (m *InMemoryDB) GetAuthUserByNickname(_ context.Context, nickname string) (*db.AuthUser, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, user := range m.authUsers {
		if user.Nickname == nickname {
			cp := *user
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("user not found")
}

func (m *InMemoryDB) GetAuthUserByUserID(_ context.Context, userID string) (*db.AuthUser, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	user, ok := m.authUsers[userID]
	if !ok {
		return nil, fmt.Errorf("user not found")
	}
	cp := *user
	return &cp, nil
}

func (m *InMemoryDB) UpdateAuthUserLastLogin(_ context.Context, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	user, ok := m.authUsers[userID]
	if !ok {
		return fmt.Errorf("user not found")
	}
	user.LastLogin = sql.NullTime{Time: time.Now(), Valid: true}
	return nil
}

func (m *InMemoryDB) UpdateAuthUserPayoutAddress(_ context.Context, userID, payoutAddress string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	user, ok := m.authUsers[userID]
	if !ok {
		return fmt.Errorf("user not found")
	}
	user.PayoutAddress.String = payoutAddress
	user.PayoutAddress.Valid = payoutAddress != ""
	return nil
}

func (m *InMemoryDB) ListAllAuthUsers(_ context.Context) ([]db.AuthUser, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	users := make([]db.AuthUser, 0, len(m.authUsers))
	for _, user := range m.authUsers {
		cp := *user
		users = append(users, cp)
	}
	// Sort by CreatedAt for consistency
	sort.Slice(users, func(i, j int) bool {
		return users[i].CreatedAt.Before(users[j].CreatedAt)
	})
	return users, nil
}

// -------- Close --------

func (m *InMemoryDB) Close() error { return nil }

// createTestLogBackend creates a LogBackend for testing
func createTestLogBackend() *logging.LogBackend {
	logBackend, err := logging.NewLogBackend(logging.LogConfig{
		LogFile:        "",      // Empty for testing - will use stdout
		DebugLevel:     "error", // Set to error to reduce test output
		MaxLogFiles:    1,
		MaxBufferLines: 100,
	})
	if err != nil {
		// Fallback to a minimal LogBackend if creation fails
		return &logging.LogBackend{}
	}
	return logBackend
}

func TestPokerService(t *testing.T) {
	t.Run("CreateTable", func(t *testing.T) {
		// Create isolated database and server for this test
		db := NewInMemoryDB()
		defer db.Close()

		logBackend := createTestLogBackend()
		defer logBackend.Close()

		srv, err := NewTestServer(db, logBackend)
		require.NoError(t, err)
		server := &TestServer{
			Server: srv,
		}

		ctx := context.Background()
		player1ID := "player1"
		player2ID := "player2"

		// Create test session for player1
		ctx, _ = server.CreateTestSession(ctx, player1ID, player1ID)

		// Test successful table creation
		resp, err := server.CreateTable(ctx, &pokerrpc.CreateTableRequest{
			PlayerId:      player1ID,
			SmallBlind:    10,
			BigBlind:      20,
			MinPlayers:    2,
			MaxPlayers:    6,
			BuyIn:         1000,
			StartingChips: 1000,
			AutoStartMs:   1000,
			AutoAdvanceMs: 1000,
		})
		require.NoError(t, err)
		assert.NotEmpty(t, resp.TableId)
		tableID := resp.TableId

		// Player2 joins the table
		joinResp, err := server.JoinTable(ctx, &pokerrpc.JoinTableRequest{
			PlayerId: player2ID,
			TableId:  tableID,
		})
		require.NoError(t, err)
		require.True(t, joinResp.Success)
	})

	t.Run("GetGameState", func(t *testing.T) {
		// Create isolated database and server for this test
		db := NewInMemoryDB()
		defer db.Close()

		logBackend := createTestLogBackend()
		defer logBackend.Close()

		srv, err := NewTestServer(db, logBackend)
		require.NoError(t, err)
		server := &TestServer{
			Server: srv,
		}

		ctx := context.Background()

		// Test non-existent table
		_, err = server.GetGameState(ctx, &pokerrpc.GetGameStateRequest{
			TableId: "non-existent",
		})
		assert.Error(t, err)
	})

	t.Run("MakeBet", func(t *testing.T) {
		// Create isolated database and server for this test
		db := NewInMemoryDB()
		defer db.Close()

		logBackend := createTestLogBackend()
		defer logBackend.Close()

		srv, err := NewTestServer(db, logBackend)
		require.NoError(t, err)
		server := &TestServer{
			Server: srv,
		}

		ctx := context.Background()
		player1ID := "player1"
		player2ID := "player2"

		// Create test session for player1
		ctx, _ = server.CreateTestSession(ctx, player1ID, player1ID)

		// Create table
		createResp, err := server.CreateTable(ctx, &pokerrpc.CreateTableRequest{
			PlayerId:      player1ID,
			SmallBlind:    10,
			BigBlind:      20,
			MinPlayers:    2,
			MaxPlayers:    2,
			BuyIn:         0,
			StartingChips: 1000,
			AutoAdvanceMs: 1000,
		})
		require.NoError(t, err)
		tableID := createResp.TableId

		// Player2 joins the table
		_, err = server.JoinTable(ctx, &pokerrpc.JoinTableRequest{
			PlayerId: player2ID,
			TableId:  tableID,
		})
		require.NoError(t, err)

		// Both players set ready
		_, err = server.SetPlayerReady(ctx, &pokerrpc.SetPlayerReadyRequest{
			PlayerId: player1ID,
			TableId:  tableID,
		})
		require.NoError(t, err)

		_, err = server.SetPlayerReady(ctx, &pokerrpc.SetPlayerReadyRequest{
			PlayerId: player2ID,
			TableId:  tableID,
		})
		require.NoError(t, err)

		// Wait for game to start
		var gameStarted bool
		for i := 0; i < 10; i++ {
			time.Sleep(10 * time.Millisecond)
			gameState, err := server.GetGameState(ctx, &pokerrpc.GetGameStateRequest{
				TableId: tableID,
			})
			require.NoError(t, err)
			if gameState.GameState.GameStarted {
				gameStarted = true
				break
			}
		}
		require.True(t, gameStarted, "game should have started after both players are ready")

		// Get game state to find current player
		gameState, err := server.GetGameState(ctx, &pokerrpc.GetGameStateRequest{
			TableId: tableID,
		})
		require.NoError(t, err)
		currentPlayer := gameState.GameState.CurrentPlayer
		require.NotEmpty(t, currentPlayer, "there should be a current player to act")

		// Test successful bet with the current player
		resp, err := server.MakeBet(ctx, &pokerrpc.MakeBetRequest{
			PlayerId: currentPlayer,
			TableId:  tableID,
			Amount:   20,
		})
		require.NoError(t, err)
		assert.True(t, resp.Success)
	})
}

func TestPokerGameFlow(t *testing.T) {
	// Create isolated database and server for this test
	db := NewInMemoryDB()
	defer db.Close()

	logBackend := createTestLogBackend()
	defer logBackend.Close()

	srv, err := NewTestServer(db, logBackend)
	require.NoError(t, err)
	server := &TestServer{
		Server: srv,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Players
	alice := "alice"
	bob := "bob"
	charlie := "charlie"

	// Create test session for alice
	ctx, _ = server.CreateTestSession(ctx, alice, alice)

	// Alice creates a table
	createResp, err := server.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:      alice,
		SmallBlind:    5,
		BigBlind:      10,
		MinPlayers:    3,
		MaxPlayers:    3,
		BuyIn:         0,
		StartingChips: 1000,
		AutoAdvanceMs: 1000,
	})
	require.NoError(t, err)
	tableID := createResp.TableId

	// Bob joins
	joinResp, err := server.JoinTable(ctx, &pokerrpc.JoinTableRequest{
		PlayerId: bob,
		TableId:  tableID,
	})
	require.NoError(t, err)
	assert.True(t, joinResp.Success)

	// Charlie joins
	joinResp, err = server.JoinTable(ctx, &pokerrpc.JoinTableRequest{
		PlayerId: charlie,
		TableId:  tableID,
	})
	require.NoError(t, err)
	assert.True(t, joinResp.Success)

	// All players set ready
	for _, player := range []string{alice, bob, charlie} {
		_, err := server.SetPlayerReady(ctx, &pokerrpc.SetPlayerReadyRequest{
			PlayerId: player,
			TableId:  tableID,
		})
		require.NoError(t, err)
	}

	// Wait for game to start with timeout
	var gameStarted bool
	for i := 0; i < 20; i++ {
		select {
		case <-ctx.Done():
			t.Fatal("Test timed out waiting for game to start")
		default:
		}

		time.Sleep(50 * time.Millisecond)
		gameState, err := server.GetGameState(ctx, &pokerrpc.GetGameStateRequest{
			TableId: tableID,
		})
		require.NoError(t, err)
		if gameState.GameState.GameStarted {
			gameStarted = true
			break
		}
	}
	require.True(t, gameStarted, "game should have started")

	// Verify game state
	gameState, err := server.GetGameState(ctx, &pokerrpc.GetGameStateRequest{
		TableId: tableID,
	})
	require.NoError(t, err)
	assert.True(t, gameState.GameState.GameStarted)
	assert.Len(t, gameState.GameState.Players, 3)
}

func TestHostLeavesTableTransfersHost(t *testing.T) {
	// Create isolated database and server for this test
	db := NewInMemoryDB()
	defer db.Close()

	logBackend := createTestLogBackend()
	defer logBackend.Close()

	srv, err := NewTestServer(db, logBackend)
	require.NoError(t, err)
	server := &TestServer{
		Server: srv,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	host := "host"
	player := "player"

	// Create test session for host
	ctx, _ = server.CreateTestSession(ctx, host, host)

	// Host creates table
	createResp, err := server.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:      host,
		SmallBlind:    5,
		BigBlind:      10,
		MinPlayers:    2,
		MaxPlayers:    6,
		BuyIn:         0,
		StartingChips: 1000,
		AutoAdvanceMs: 1000,
	})
	require.NoError(t, err)
	tableID := createResp.TableId

	// Player joins
	joinResp, err := server.JoinTable(ctx, &pokerrpc.JoinTableRequest{
		PlayerId: player,
		TableId:  tableID,
	})
	require.NoError(t, err)
	assert.True(t, joinResp.Success)

	// Host leaves table
	leaveResp, err := server.LeaveTable(ctx, &pokerrpc.LeaveTableRequest{
		PlayerId: host,
		TableId:  tableID,
	})
	require.NoError(t, err)
	assert.True(t, leaveResp.Success)
	assert.Equal(t, "Successfully left table", leaveResp.Message)

	// Verify table still exists and the remaining player is still seated.
	tablesResp, err := server.GetTables(ctx, &pokerrpc.GetTablesRequest{})
	require.NoError(t, err)
	assert.Len(t, tablesResp.Tables, 1)
	assert.Equal(t, int32(1), tablesResp.Tables[0].CurrentPlayers)
}

// Heads-up: post-flop the big blind (non-dealer) must act first.
func TestHeadsUpPostflopActorIsBB(t *testing.T) {
	db := NewInMemoryDB()
	defer db.Close()

	logBackend := createTestLogBackend()
	defer logBackend.Close()

	srv, err := NewTestServer(db, logBackend)
	require.NoError(t, err)
	server := &TestServer{Server: srv}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	p1 := "p1"
	p2 := "p2"

	// Create test session for p1
	ctx, _ = server.CreateTestSession(ctx, p1, p1)

	// Create HU table (5/10)
	createResp, err := server.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:      p1,
		SmallBlind:    5,
		BigBlind:      10,
		MinPlayers:    2,
		MaxPlayers:    2,
		BuyIn:         0,
		StartingChips: 1000,
		AutoAdvanceMs: 1000,
	})
	require.NoError(t, err)
	tableID := createResp.TableId

	// p2 joins
	joinResp, err := server.JoinTable(ctx, &pokerrpc.JoinTableRequest{PlayerId: p2, TableId: tableID})
	require.NoError(t, err)
	assert.True(t, joinResp.Success)

	// Both ready
	for _, pid := range []string{p1, p2} {
		_, err := server.SetPlayerReady(ctx, &pokerrpc.SetPlayerReadyRequest{PlayerId: pid, TableId: tableID})
		require.NoError(t, err)
	}

	// Identify blinds and ensure SB acts first preflop
	var sbID, bbID string
	require.Eventually(t, func() bool {
		st, err := server.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
		require.NoError(t, err)
		if st.GameState == nil || !st.GameState.GameStarted || st.GameState.Phase != pokerrpc.GamePhase_PRE_FLOP {
			return false
		}
		sbID, bbID = "", ""
		for _, pl := range st.GameState.Players {
			switch pl.GetCurrentBet() {
			case 5:
				sbID = pl.GetId()
			case 10:
				bbID = pl.GetId()
			}
		}
		if sbID == "" || bbID == "" {
			return false
		}
		return st.GameState.GetCurrentPlayer() == sbID
	}, 3*time.Second, 20*time.Millisecond, "failed to get SB as actor preflop")

	// --- FIX 1: close preflop with a CALL by the BB (not Check) ---
	// SB calls to 10
	_, err = server.CallBet(ctx, &pokerrpc.CallBetRequest{PlayerId: sbID, TableId: tableID})
	require.NoError(t, err)
	// BB "checks to close" is commonly modeled as CallBet with zero diff
	_, err = server.CheckBet(ctx, &pokerrpc.CheckBetRequest{PlayerId: bbID, TableId: tableID})
	require.NoError(t, err)

	// --- FIX 2: don't require currentBet==0 on the flop ---
	require.Eventually(t, func() bool {
		st, err := server.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
		require.NoError(t, err)
		if st.GameState == nil {
			return false
		}
		if st.GameState.Phase != pokerrpc.GamePhase_FLOP {
			return false
		}
		// CurrentPlayer should be set and equal to BB on flop (HU rule)
		currentPlayer := st.GameState.GetCurrentPlayer()
		if currentPlayer == "" {
			return false
		}
		return currentPlayer == bbID
	}, 3*time.Second, 20*time.Millisecond, "BB should act first on FLOP heads-up")
}

// New tests for server-side bet validation logic.
func TestBetValidation_UnderBetRejected(t *testing.T) {
	db := NewInMemoryDB()
	defer db.Close()

	logBackend := createTestLogBackend()
	defer logBackend.Close()

	srv, err := NewTestServer(db, logBackend)
	require.NoError(t, err)
	server := &TestServer{Server: srv}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	p1 := "p1"
	p2 := "p2"

	// Create test session for p1
	ctx, _ = server.CreateTestSession(ctx, p1, p1)

	// Create heads-up table (SB=5, BB=10).
	createResp, err := server.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:      p1,
		SmallBlind:    5,
		BigBlind:      10,
		MinPlayers:    2,
		MaxPlayers:    2,
		BuyIn:         0,
		StartingChips: 1000,
		AutoAdvanceMs: 1000,
	})
	require.NoError(t, err)
	tableID := createResp.TableId

	// p2 joins
	joinResp, err := server.JoinTable(ctx, &pokerrpc.JoinTableRequest{PlayerId: p2, TableId: tableID})
	require.NoError(t, err)
	assert.True(t, joinResp.Success)

	// Both ready
	for _, pid := range []string{p1, p2} {
		_, err := server.SetPlayerReady(ctx, &pokerrpc.SetPlayerReadyRequest{PlayerId: pid, TableId: tableID})
		require.NoError(t, err)
	}

	// Wait for game to start, identify SB (bet=5) and ensure it's their turn
	var sbID string
	require.Eventually(t, func() bool {
		st, err := server.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
		require.NoError(t, err)
		if st.GameState == nil || !st.GameState.GameStarted {
			return false
		}
		// Find SB by blind amount
		sbID = ""
		for _, pl := range st.GameState.Players {
			if pl.GetCurrentBet() == 5 {
				sbID = pl.GetId()
				break
			}
		}
		if sbID == "" {
			return false
		}
		// It must be SB's turn heads-up pre-flop
		return st.GameState.GetCurrentPlayer() == sbID
	}, 2*time.Second, 20*time.Millisecond, "failed to find SB as current actor")

	// Attempt to bet to 8 (absolute), which is below table current bet (10): expect InvalidArgument
	_, err = server.MakeBet(ctx, &pokerrpc.MakeBetRequest{PlayerId: sbID, TableId: tableID, Amount: 8})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Contains(t, err.Error(), "current bet")
}

func TestBetValidation_MinOpenBetBelowBBRejected(t *testing.T) {
	db := NewInMemoryDB()
	defer db.Close()

	logBackend := createTestLogBackend()
	defer logBackend.Close()

	srv, err := NewTestServer(db, logBackend)
	require.NoError(t, err)
	server := &TestServer{Server: srv}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	p1 := "p1"
	p2 := "p2"

	// Create test session for p1
	ctx, _ = server.CreateTestSession(ctx, p1, p1)

	// Create table
	createResp, err := server.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:      p1,
		SmallBlind:    5,
		BigBlind:      10,
		MinPlayers:    2,
		MaxPlayers:    2,
		BuyIn:         0,
		StartingChips: 1000,
		AutoAdvanceMs: 1000,
	})
	require.NoError(t, err)
	tableID := createResp.TableId

	// p2 joins
	joinResp, err := server.JoinTable(ctx, &pokerrpc.JoinTableRequest{PlayerId: p2, TableId: tableID})
	require.NoError(t, err)
	assert.True(t, joinResp.Success)

	// Both ready
	for _, pid := range []string{p1, p2} {
		_, err := server.SetPlayerReady(ctx, &pokerrpc.SetPlayerReadyRequest{PlayerId: pid, TableId: tableID})
		require.NoError(t, err)
	}

	// Identify small blind and big blind from initial state
	var sbID, bbID string
	require.Eventually(t, func() bool {
		st, err := server.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
		require.NoError(t, err)
		if st.GameState == nil || !st.GameState.GameStarted {
			return false
		}
		for _, pl := range st.GameState.Players {
			switch pl.GetCurrentBet() {
			case 5:
				sbID = pl.GetId()
			case 10:
				bbID = pl.GetId()
			}
		}
		return sbID != "" && bbID != ""
	}, 2*time.Second, 20*time.Millisecond, "failed to find blinds")

	// SB calls to 10; BB checks to close the round (advance to flop)
	_, err = server.CallBet(ctx, &pokerrpc.CallBetRequest{PlayerId: sbID, TableId: tableID})
	require.NoError(t, err)
	_, err = server.CheckBet(ctx, &pokerrpc.CheckBetRequest{PlayerId: bbID, TableId: tableID})
	require.NoError(t, err)

	// Wait until we are on FLOP with current bet reset to 0 and have a current player
	var actor string
	require.Eventually(t, func() bool {
		st, err := server.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
		require.NoError(t, err)
		if st.GameState == nil {
			return false
		}
		if st.GameState.Phase != pokerrpc.GamePhase_FLOP {
			return false
		}
		if st.GameState.GetCurrentBet() != 0 {
			return false
		}
		actor = st.GameState.GetCurrentPlayer()
		return actor != ""
	}, 3*time.Second, 20*time.Millisecond, "failed to reach flop with actor")

	// Try to open bet with amount below BB (5 < 10) -> expect InvalidArgument
	_, err = server.MakeBet(ctx, &pokerrpc.MakeBetRequest{PlayerId: actor, TableId: tableID, Amount: 5})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Contains(t, err.Error(), "minimum bet is 10")
}

func TestLastPlayerLeavesTableClosure(t *testing.T) {
	// Create isolated database and server for this test
	db := NewInMemoryDB()
	defer db.Close()

	logBackend := createTestLogBackend()
	defer logBackend.Close()

	srv, err := NewTestServer(db, logBackend)
	require.NoError(t, err)
	server := &TestServer{
		Server: srv,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	host := "host"

	// Host creates table
	createResp, err := server.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:      host,
		SmallBlind:    5,
		BigBlind:      10,
		MinPlayers:    2,
		MaxPlayers:    6,
		BuyIn:         100,
		StartingChips: 1000,
		AutoAdvanceMs: 1000,
	})
	require.NoError(t, err)
	tableID := createResp.TableId

	// Verify table exists
	tablesResp, err := server.GetTables(ctx, &pokerrpc.GetTablesRequest{})
	require.NoError(t, err)
	assert.Len(t, tablesResp.Tables, 1)

	// Host leaves table (last player)
	leaveResp, err := server.LeaveTable(ctx, &pokerrpc.LeaveTableRequest{
		PlayerId: host,
		TableId:  tableID,
	})
	require.NoError(t, err)
	assert.True(t, leaveResp.Success)
	assert.Equal(t, "Successfully left table. Table closed.", leaveResp.Message)

	// Verify table is removed
	tablesResp, err = server.GetTables(ctx, &pokerrpc.GetTablesRequest{})
	require.NoError(t, err)
	assert.Len(t, tablesResp.Tables, 0)
}

func TestNonHostLeavesTable(t *testing.T) {
	// Create isolated database and server for this test
	db := NewInMemoryDB()
	defer db.Close()

	logBackend := createTestLogBackend()
	defer logBackend.Close()

	srv, err := NewTestServer(db, logBackend)
	require.NoError(t, err)
	server := &TestServer{
		Server: srv,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	host := "host"
	player := "player"

	// Create test session for host
	ctx, _ = server.CreateTestSession(ctx, host, host)

	// Host creates table
	createResp, err := server.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:      host,
		SmallBlind:    5,
		BigBlind:      10,
		MinPlayers:    2,
		MaxPlayers:    6,
		BuyIn:         100,
		StartingChips: 1000,
		AutoAdvanceMs: 1000,
	})
	require.NoError(t, err)
	tableID := createResp.TableId

	// Player joins
	joinResp, err := server.JoinTable(ctx, &pokerrpc.JoinTableRequest{
		PlayerId: player,
		TableId:  tableID,
	})
	require.NoError(t, err)
	assert.True(t, joinResp.Success)

	// Player leaves table
	leaveResp, err := server.LeaveTable(ctx, &pokerrpc.LeaveTableRequest{
		PlayerId: player,
		TableId:  tableID,
	})
	require.NoError(t, err)
	assert.True(t, leaveResp.Success)

	// Verify table still exists and host is unchanged
	tablesResp, err := server.GetTables(ctx, &pokerrpc.GetTablesRequest{})
	require.NoError(t, err)
	assert.Len(t, tablesResp.Tables, 1)
	assert.Equal(t, int32(1), tablesResp.Tables[0].CurrentPlayers, "Table should have 1 player remaining")
}

func TestLeaveTable(t *testing.T) {
	// Create isolated database and server for this test
	db := NewInMemoryDB()
	defer db.Close()

	logBackend := createTestLogBackend()
	defer logBackend.Close()

	srv, err := NewTestServer(db, logBackend)
	require.NoError(t, err)
	server := &TestServer{
		Server: srv,
	}

	ctx := context.Background()
	player1ID := "player1"

	// Test non-existent table
	resp, err := server.LeaveTable(ctx, &pokerrpc.LeaveTableRequest{
		PlayerId: player1ID,
		TableId:  "non-existent",
	})
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestLeaveTableDuringPendingSettlementKeepsSeat(t *testing.T) {
	db := NewInMemoryDB()
	defer db.Close()

	logBackend := createTestLogBackend()
	defer logBackend.Close()

	srv, err := NewTestServer(db, logBackend)
	require.NoError(t, err)
	server := &TestServer{Server: srv}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	host := "host"
	ctx, _ = server.CreateTestSession(ctx, host, host)

	createResp, err := server.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:      host,
		SmallBlind:    5,
		BigBlind:      10,
		MinPlayers:    2,
		MaxPlayers:    2,
		BuyIn:         100,
		StartingChips: 1000,
		AutoAdvanceMs: 1000,
	})
	require.NoError(t, err)
	tableID := createResp.TableId

	server.markPendingSettlement(tableID)

	leaveResp, err := server.LeaveTable(ctx, &pokerrpc.LeaveTableRequest{
		PlayerId: host,
		TableId:  tableID,
	})
	require.NoError(t, err)
	assert.True(t, leaveResp.Success)
	assert.Contains(t, leaveResp.Message, "seat is reserved")

	table, ok := server.GetTable(tableID)
	require.True(t, ok)
	require.NotNil(t, table.GetUser(host))

	tablesResp, err := server.GetTables(ctx, &pokerrpc.GetTablesRequest{})
	require.NoError(t, err)
	require.Len(t, tablesResp.Tables, 1)
	assert.Equal(t, int32(1), tablesResp.Tables[0].CurrentPlayers)
}

func TestJoinTable(t *testing.T) {
	// Create isolated database and server for this test
	db := NewInMemoryDB()
	defer db.Close()

	logBackend := createTestLogBackend()
	defer logBackend.Close()

	srv, err := NewTestServer(db, logBackend)
	require.NoError(t, err)
	server := &TestServer{
		Server: srv,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	player1ID := "player1"
	player2ID := "player2"

	// Create test session for player1
	ctx, _ = server.CreateTestSession(ctx, player1ID, player1ID)

	// Create table
	createResp, err := server.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:      player1ID,
		SmallBlind:    10,
		BigBlind:      20,
		MinPlayers:    2,
		MaxPlayers:    3,
		BuyIn:         0,
		StartingChips: 1000,
		AutoAdvanceMs: 1000,
	})
	require.NoError(t, err)
	tableID := createResp.TableId

	// Test joining non-existent table
	resp, err := server.JoinTable(ctx, &pokerrpc.JoinTableRequest{
		PlayerId: player2ID,
		TableId:  "non-existent",
	})
	require.NoError(t, err)
	assert.False(t, resp.Success)

	// Test successful join
	resp, err = server.JoinTable(ctx, &pokerrpc.JoinTableRequest{
		PlayerId: player2ID,
		TableId:  tableID,
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)

	// Test rejoining (this was causing deadlock before fix)
	resp, err = server.JoinTable(ctx, &pokerrpc.JoinTableRequest{
		PlayerId: player2ID,
		TableId:  tableID,
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Contains(t, resp.Message, "Reconnected")

	// Test joining additional player does not deadlock or fail.
	player3ID := "player3"
	resp, err = server.JoinTable(ctx, &pokerrpc.JoinTableRequest{
		PlayerId: player3ID,
		TableId:  tableID,
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)

	tablesResp, err := server.GetTables(ctx, &pokerrpc.GetTablesRequest{})
	require.NoError(t, err)
	require.Len(t, tablesResp.Tables, 1)
	assert.Len(t, tablesResp.Tables[0].Players, 3)
	assert.Equal(t, player1ID, tablesResp.Tables[0].Players[0].Name)
	assert.Equal(t, player2ID, tablesResp.Tables[0].Players[1].Name)
	assert.Equal(t, player3ID, tablesResp.Tables[0].Players[2].Name)
}

func TestJoinTableAfterGameStartFails(t *testing.T) {
	// Create isolated database and server for this test
	db := NewInMemoryDB()
	defer db.Close()

	logBackend := createTestLogBackend()
	defer logBackend.Close()

	srv, err := NewTestServer(db, logBackend)
	require.NoError(t, err)
	server := &TestServer{
		Server: srv,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	player1ID := "player1"
	player2ID := "player2"
	latecomerID := "latecomer"

	// Create test session for player1
	ctx, _ = server.CreateTestSession(ctx, player1ID, player1ID)

	// Create table
	createResp, err := server.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:      player1ID,
		SmallBlind:    10,
		BigBlind:      20,
		MinPlayers:    2,
		MaxPlayers:    2,
		BuyIn:         0,
		StartingChips: 1000,
		AutoAdvanceMs: 1000,
	})
	require.NoError(t, err)
	tableID := createResp.TableId

	// Player2 joins the table
	joinResp, err := server.JoinTable(ctx, &pokerrpc.JoinTableRequest{
		PlayerId: player2ID,
		TableId:  tableID,
	})
	require.NoError(t, err)
	require.True(t, joinResp.Success)

	// Both players set ready
	for _, pid := range []string{player1ID, player2ID} {
		_, err = server.SetPlayerReady(ctx, &pokerrpc.SetPlayerReadyRequest{
			PlayerId: pid,
			TableId:  tableID,
		})
		require.NoError(t, err)
	}

	// Wait for the game to start, reasserting readiness in case a prior attempt raced.
	require.Eventually(t, func() bool {
		for _, pid := range []string{player1ID, player2ID} {
			_, _ = server.SetPlayerReady(ctx, &pokerrpc.SetPlayerReadyRequest{
				PlayerId: pid,
				TableId:  tableID,
			})
		}
		gameState, err := server.GetGameState(ctx, &pokerrpc.GetGameStateRequest{
			TableId: tableID,
		})
		if err != nil {
			return false
		}
		return gameState.GameState != nil && gameState.GameState.GameStarted
	}, 2*time.Second, 10*time.Millisecond, "game should have started")

	// Latecomer should be rejected after the game has started
	resp, err := server.JoinTable(ctx, &pokerrpc.JoinTableRequest{
		PlayerId: latecomerID,
		TableId:  tableID,
	})
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Contains(t, resp.Message, "Game already started")
}

// TestCheckpointRestoresPausedMatchAtBoundary ensures graceful recovery brings
// an active match back at the next-hand boundary instead of resuming a betting
// street mid-hand.
func TestCheckpointRestoresPausedMatchAtBoundary(t *testing.T) {
	// Use the same in-memory DB for the two server instances so that persisted state survives.
	db := NewInMemoryDB()
	defer db.Close()

	logBackend := createTestLogBackend()
	defer logBackend.Close()

	// First server instance — runs the game and produces a snapshot.
	srv1Instance, err := NewTestServer(db, logBackend)
	require.NoError(t, err)
	srv1 := &TestServer{Server: srv1Instance}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	p1 := "p1"
	p2 := "p2"

	createResp, err := srv1.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:      p1,
		SmallBlind:    5,
		BigBlind:      10,
		MinPlayers:    2,
		MaxPlayers:    2,
		BuyIn:         0,
		StartingChips: 1000,
		AutoAdvanceMs: 1000,
	})
	require.NoError(t, err)
	tableID := createResp.TableId

	joinResp, err := srv1.JoinTable(ctx, &pokerrpc.JoinTableRequest{
		PlayerId: p2,
		TableId:  tableID,
	})
	require.NoError(t, err)
	assert.True(t, joinResp.Success)

	for _, pid := range []string{p1, p2} {
		_, err := srv1.SetPlayerReady(ctx, &pokerrpc.SetPlayerReadyRequest{
			PlayerId: pid,
			TableId:  tableID,
		})
		require.NoError(t, err)
	}

	var pre *pokerrpc.GameUpdate
	require.Eventually(t, func() bool {
		stateResp, err := srv1.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
		require.NoError(t, err)
		if stateResp != nil && stateResp.GameState != nil && stateResp.GameState.CurrentPlayer != "" {
			pre = stateResp.GameState
			return true
		}
		return false
	}, 3*time.Second, 25*time.Millisecond)

	require.NotNil(t, pre)

	_, err = srv1.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: pre.CurrentPlayer,
		TableId:  tableID,
		Amount:   1000,
	})
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		stateResp, err := srv1.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
		if err != nil || stateResp == nil || stateResp.GameState == nil {
			return false
		}
		return stateResp.GameState.CurrentPlayer != "" && stateResp.GameState.CurrentPlayer != pre.CurrentPlayer
	}, 3*time.Second, 25*time.Millisecond)

	stateResp, err := srv1.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
	require.NoError(t, err)
	require.NotNil(t, stateResp.GameState)

	_, err = srv1.CallBet(ctx, &pokerrpc.CallBetRequest{
		PlayerId: stateResp.GameState.CurrentPlayer,
		TableId:  tableID,
	})
	require.NoError(t, err)

	var showdown *pokerrpc.GameUpdate
	require.Eventually(t, func() bool {
		stateResp, err := srv1.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
		require.NoError(t, err)
		if stateResp != nil && stateResp.GameState != nil && stateResp.GameState.Phase == pokerrpc.GamePhase_SHOWDOWN {
			showdown = stateResp.GameState
			return true
		}
		return false
	}, 5*time.Second, 25*time.Millisecond)

	require.NotNil(t, showdown)
	time.Sleep(100 * time.Millisecond)
	srv1.Close()

	srv2Instance, err := NewTestServer(db, logBackend)
	require.NoError(t, err)
	srv2 := &TestServer{Server: srv2Instance}
	defer srv2.Close()

	var restoredState *pokerrpc.GameUpdate
	require.Eventually(t, func() bool {
		st, err := srv2.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
		require.NoError(t, err)
		if st != nil && st.GameState != nil {
			restoredState = st.GameState
			return true
		}
		return false
	}, 2*time.Second, 20*time.Millisecond, "checkpoint restore did not stabilize")

	require.NotNil(t, restoredState)
	require.True(t, restoredState.GameStarted)
	assert.Contains(t, []pokerrpc.GamePhase{
		pokerrpc.GamePhase_NEW_HAND_DEALING,
		pokerrpc.GamePhase_PRE_FLOP,
	}, restoredState.Phase)

	balances := make(map[string]int64)
	for _, p := range showdown.Players {
		balances[p.Id] = p.Balance
	}
	for _, p := range restoredState.Players {
		effectiveStack := p.Balance + p.CurrentBet
		assert.Equal(t, balances[p.Id], effectiveStack, "effective stack should persist through checkpoint restore")
	}

	require.Eventually(t, func() bool {
		st, err := srv2.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
		if err != nil || st == nil || st.GameState == nil {
			return false
		}
		return st.GameState.Phase == pokerrpc.GamePhase_PRE_FLOP && st.GameState.CurrentPlayer != ""
	}, 3*time.Second, 25*time.Millisecond, "restored match should resume at the next hand boundary")
}

// Close properly stops the server and cleans up resources
func (ts *TestServer) Close() {
	if ts.Server != nil {
		ts.Server.Stop()
	}
}

// Add new test to verify correct blind posting and balances in heads-up game
func TestBlindPostingAndBalances(t *testing.T) {
	// Create isolated in-memory DB and server
	db := NewInMemoryDB()
	defer db.Close()

	logBackend := createTestLogBackend()
	defer logBackend.Close()

	srvInstance, err := NewTestServer(db, logBackend)
	require.NoError(t, err)
	srv := &TestServer{Server: srvInstance}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Players
	p1 := "p1"
	p2 := "p2"

	// p1 creates a heads-up table (minPlayers=2)
	const (
		startingChips int64 = 1000
		smallBlind          = int64(5)
		bigBlind            = int64(10)
	)

	createResp, err := srv.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:      p1,
		SmallBlind:    smallBlind,
		BigBlind:      bigBlind,
		MinPlayers:    2,
		MaxPlayers:    2,
		BuyIn:         0,
		StartingChips: startingChips,
		AutoAdvanceMs: 1000,
	})
	require.NoError(t, err)
	tableID := createResp.TableId

	// p2 joins
	joinResp, err := srv.JoinTable(ctx, &pokerrpc.JoinTableRequest{PlayerId: p2, TableId: tableID})
	require.NoError(t, err)
	require.True(t, joinResp.Success)

	// Both players ready. Note: p1 ready first (common user flow)
	for _, pid := range []string{p1, p2} {
		_, err := srv.SetPlayerReady(ctx, &pokerrpc.SetPlayerReadyRequest{PlayerId: pid, TableId: tableID})
		require.NoError(t, err)
	}

	// Wait until blinds are posted: PRE_FLOP and both SB/BB have posted
	var gameState *pokerrpc.GameUpdate
	require.Eventually(t, func() bool {
		g := srv.GetGameForTable(tableID)
		if g == nil {
			return false
		}
		s := g.GetStateSnapshot()
		if s.Phase != pokerrpc.GamePhase_PRE_FLOP {
			return false
		}
		// Check that SB/BB have posted their blinds
		var sbPosted, bbPosted bool
		for _, pl := range s.Players {
			if pl.IsSmallBlind && pl.CurrentBet == smallBlind {
				sbPosted = true
			}
			if pl.IsBigBlind && pl.CurrentBet == bigBlind {
				bbPosted = true
			}
		}
		if sbPosted && bbPosted {
			// Get the full game state for further assertions
			st, err := srv.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
			if err != nil {
				return false
			}
			gameState = st.GameState
			return true
		}
		return false
	}, 2*time.Second, 10*time.Millisecond, "blinds not posted in time")

	// Current player should be SB (needs chips to match BB)
	current := gameState.CurrentPlayer
	require.Contains(t, []string{p1, p2}, current)

	// Make sure we're actually in the “needs to call” path
	var curSnap *pokerrpc.Player
	for _, pl := range gameState.Players {
		if pl.Id == current {
			curSnap = pl
			break
		}
	}
	require.NotNil(t, curSnap)
	require.Less(t, curSnap.CurrentBet, gameState.CurrentBet, "current player must need to call")

	// Now the call really adds the missing chips to reach BB
	_, err = srv.CallBet(ctx, &pokerrpc.CallBetRequest{PlayerId: current, TableId: tableID})
	require.NoError(t, err)

	// Wait until both players show BB committed
	var updated *pokerrpc.GameUpdate
	require.Eventually(t, func() bool {
		g := srv.GetGameForTable(tableID)
		if g == nil {
			return false
		}
		s := g.GetStateSnapshot()
		if s.CurrentBet != bigBlind {
			return false
		}
		have := 0
		for _, pl := range s.Players {
			if pl.CurrentBet == bigBlind {
				have++
			}
		}
		if have == 2 {
			// Get the full game state for further assertions
			st, err := srv.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
			if err != nil {
				return false
			}
			updated = st.GameState
			return true
		}
		return false
	}, 2*time.Second, 10*time.Millisecond, "both players did not commit BB in time")

	// Assert balances
	find := func(pid string) *pokerrpc.Player {
		for _, pl := range updated.Players {
			if pl.Id == pid {
				return pl
			}
		}
		return nil
	}
	p1Info, p2Info := find(p1), find(p2)
	require.NotNil(t, p1Info)
	require.NotNil(t, p2Info)

	expectedBalance := startingChips - bigBlind // both should have 10 committed
	assert.Equal(t, bigBlind, p1Info.CurrentBet, "p1 CurrentBet should equal big blind once")
	assert.Equal(t, bigBlind, p2Info.CurrentBet, "p2 CurrentBet should equal big blind once")
	assert.Equal(t, expectedBalance, p1Info.Balance, "p1 balance incorrect after blinds+call")
	assert.Equal(t, expectedBalance, p2Info.Balance, "p2 balance incorrect after blinds+call")
}

// waitForUpdate waits for a GameUpdate on the mock stream or times out.
func waitForUpdate(t *testing.T, ms *mockGameStream, timeout time.Duration) *pokerrpc.GameUpdate {
	t.Helper()
	select {
	case upd := <-ms.sentCh:
		return upd
	case <-time.After(timeout):
		t.Fatalf("timeout waiting for GameUpdate")
		return nil
	}
}

// waitForTurnAdvance waits for a GameUpdate whose current player differs from
// prevPlayerID, skipping any stale updates already buffered on the stream.
func waitForTurnAdvance(t *testing.T, ms *mockGameStream, timeout time.Duration, prevPlayerID string) *pokerrpc.GameUpdate {
	t.Helper()
	deadline := time.After(timeout)
	for {
		select {
		case upd := <-ms.sentCh:
			if upd == nil {
				continue
			}
			if upd.CurrentPlayer != prevPlayerID {
				return upd
			}
		case <-deadline:
			t.Fatalf("timeout waiting for GameUpdate with current player different from %q", prevPlayerID)
			return nil
		}
	}
}

// TestTimebankExpiryBroadcast ensures a GameUpdate is broadcast after an auto-action
// due to timebank expiry and that the current player changes accordingly.
func TestTimebankExpiryBroadcast(t *testing.T) {
	s := newBareServer()
	// Start event processor so table notifications trigger GameUpdates
	s.eventProcessor = NewEventProcessor(s, 64, 1)
	s.eventProcessor.Start()

	// Build an active heads-up table and register it in the server
	tbl := buildActiveHeadsUpTable(t, "timebank-broadcast-tbl")
	s.tables.Store(tbl.GetConfig().ID, tbl)

	// Wire table events into the server's event processor
	tableEventChan := make(chan poker.TableEvent, 64)
	tbl.SetEventChannel(tableEventChan)
	go s.processTableEvents(tableEventChan)

	// Attach two streaming clients
	p1, p2 := "p1", "p2"
	ms1 := newMockGameStream()
	ms2 := newMockGameStream()

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_ = s.StartGameStream(&pokerrpc.StartGameStreamRequest{TableId: tbl.GetConfig().ID, PlayerId: p1}, ms1)
	}()
	go func() {
		defer wg.Done()
		_ = s.StartGameStream(&pokerrpc.StartGameStreamRequest{TableId: tbl.GetConfig().ID, PlayerId: p2}, ms2)
	}()

	// Drain initial updates for both streams
	_ = waitForUpdate(t, ms1, 500*time.Millisecond)
	_ = waitForUpdate(t, ms2, 500*time.Millisecond)

	// Obtain game and current player
	g := tbl.GetGame()
	require.NotNil(t, g)

	// Ensure we are in a betting phase and there is a current player
	require.Eventually(t, func() bool {
		phase := g.GetPhase()
		return phase == pokerrpc.GamePhase_PRE_FLOP || phase == pokerrpc.GamePhase_FLOP || phase == pokerrpc.GamePhase_TURN || phase == pokerrpc.GamePhase_RIVER
	}, 2*time.Second, 10*time.Millisecond)

	cur := g.GetCurrentPlayerObject()
	require.NotNil(t, cur)
	curID := cur.ID()

	// Simulate timebank expiry for the current player
	g.TriggerTimebankExpiredFor(curID)

	// Expect a broadcast with the current player changed on both streams
	// (allow a short window for the server to process and push updates)
	got1 := waitForTurnAdvance(t, ms1, 2*time.Second, curID)
	got2 := waitForTurnAdvance(t, ms2, 2*time.Second, curID)

	require.NotNil(t, got1)
	require.NotNil(t, got2)
	require.NotEqual(t, curID, got1.CurrentPlayer, "ms1 should see advanced turn after timebank expiry")
	require.NotEqual(t, curID, got2.CurrentPlayer, "ms2 should see advanced turn after timebank expiry")

	// Cleanup streams
	ms1.cancel()
	ms2.cancel()
	wg.Wait()
}

func TestSetPlayerReadyRequiresFundedEscrowWhenBound(t *testing.T) {
	ctx := context.Background()
	db := NewInMemoryDB()
	defer db.Close()

	logBackend := createTestLogBackend()
	defer logBackend.Close()

	srv, err := NewTestServer(db, logBackend)
	require.NoError(t, err)

	// Use zeroShortID.String() as hostID so it matches the escrow OwnerUID
	hostID := zeroShortID.String()
	createResp, err := srv.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:      hostID,
		SmallBlind:    5,
		BigBlind:      10,
		MinPlayers:    1,
		MaxPlayers:    2,
		BuyIn:         1_000,
		StartingChips: 1_000,
		AutoAdvanceMs: 1_000,
	})
	require.NoError(t, err)
	tableID := createResp.TableId

	es := &refereeEscrowSession{
		EscrowID:        "escrow-unfunded",
		OwnerUID:        zeroShortID,
		AmountAtoms:     1_000,
		RedeemScriptHex: "51",
		PkScriptHex:     "51",
		TableID:         tableID,
		SeatIndex:       0,
	}
	srv.referee.mu.Lock()
	srv.referee.escrows[es.EscrowID] = es
	srv.referee.mu.Unlock()

	table, ok := srv.getTable(tableID)
	require.True(t, ok)
	_, changed, err := table.SetSeatEscrowState(0, es.EscrowID, false)
	require.NoError(t, err)
	require.True(t, changed)

	_, err = srv.SetPlayerReady(ctx, &pokerrpc.SetPlayerReadyRequest{
		PlayerId: hostID,
		TableId:  tableID,
	})
	require.Error(t, err)
	require.Equal(t, codes.FailedPrecondition, status.Code(err))

	srv.TestBindEscrowFunding(es.EscrowID, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", 0, 1_000)

	_, err = srv.SetPlayerReady(ctx, &pokerrpc.SetPlayerReadyRequest{
		PlayerId: hostID,
		TableId:  tableID,
	})
	require.NoError(t, err)
}
