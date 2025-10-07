package server

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vctt94/bisonbotkit/logging"
	"github.com/vctt94/pokerbisonrelay/pkg/poker"
	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
	"github.com/vctt94/pokerbisonrelay/pkg/server/internal/db"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestServer implements the PokerServiceServer interface
type TestServer struct {
	*Server
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
	snapshots map[string]db.Snapshot // tableID -> snapshot
}

// NewInMemoryDB creates a new in-memory database for testing.
func NewInMemoryDB() *InMemoryDB {
	return &InMemoryDB{
		balances:     make(map[string]int64),
		transactions: make(map[string][]Transaction),
		tables:       make(map[string]*db.Table),
		participants: make(map[string]map[string]*db.Participant),
		seatIndex:    make(map[string]map[int]string),
		snapshots:    make(map[string]db.Snapshot),
	}
}

// -------- Players / Wallet --------

func (m *InMemoryDB) GetPlayerBalance(_ context.Context, playerID string) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	bal, ok := m.balances[playerID]
	if !ok {
		return 0, fmt.Errorf("player not found")
	}
	return bal, nil
}

func (m *InMemoryDB) UpdatePlayerBalance(_ context.Context, playerID string, amount int64, typ, description string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// create player lazily
	if _, ok := m.balances[playerID]; !ok {
		m.balances[playerID] = 0
	}

	m.balances[playerID] += amount
	tx := Transaction{
		ID:          int64(len(m.transactions[playerID]) + 1),
		PlayerID:    playerID,
		Amount:      amount,
		Type:        typ,
		Description: description,
		CreatedAt:   time.Now().Format(time.RFC3339),
	}
	m.transactions[playerID] = append(m.transactions[playerID], tx)
	return nil
}

// -------- Tables (configuration) --------

func (m *InMemoryDB) UpsertTable(_ context.Context, t *poker.TableConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *t // store by value to avoid external mutation
	// set CreatedAt if zero-ish
	m.tables[cp.ID] = &db.Table{
		ID:         cp.ID,
		HostID:     cp.HostID,
		BuyIn:      cp.BuyIn,
		MinPlayers: cp.MinPlayers,
		MaxPlayers: cp.MaxPlayers,
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
	t.Run("GetBalance", func(t *testing.T) {
		// Create isolated database and server for this test
		db := NewInMemoryDB()
		defer db.Close()

		logBackend := createTestLogBackend()
		defer logBackend.Close()

		server := &TestServer{
			Server: NewServer(db, logBackend),
		}

		ctx := context.Background()
		playerID := "player1"

		// Test non-existent player
		_, err := server.GetBalance(ctx, &pokerrpc.GetBalanceRequest{PlayerId: "non-existent"})
		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, st.Code())
		assert.Contains(t, st.Message(), "player not found")

		// Create player first
		_, err = server.UpdateBalance(ctx, &pokerrpc.UpdateBalanceRequest{
			PlayerId:    playerID,
			Amount:      0,
			Description: "initial balance",
		})
		require.NoError(t, err)

		// Test existing player
		resp, err := server.GetBalance(ctx, &pokerrpc.GetBalanceRequest{PlayerId: playerID})
		require.NoError(t, err)
		assert.Equal(t, int64(0), resp.Balance)
	})

	t.Run("UpdateBalance", func(t *testing.T) {
		// Create isolated database and server for this test
		db := NewInMemoryDB()
		defer db.Close()

		logBackend := createTestLogBackend()
		defer logBackend.Close()

		server := &TestServer{
			Server: NewServer(db, logBackend),
		}

		ctx := context.Background()
		playerID := "player1"

		// Test deposit
		resp, err := server.UpdateBalance(ctx, &pokerrpc.UpdateBalanceRequest{
			PlayerId:    playerID,
			Amount:      1000,
			Description: "initial deposit",
		})
		require.NoError(t, err)
		assert.Equal(t, int64(1000), resp.NewBalance)

		// Test withdrawal
		resp, err = server.UpdateBalance(ctx, &pokerrpc.UpdateBalanceRequest{
			PlayerId:    playerID,
			Amount:      -500,
			Description: "withdrawal",
		})
		require.NoError(t, err)
		assert.Equal(t, int64(500), resp.NewBalance)
	})

	t.Run("CreateTable", func(t *testing.T) {
		// Create isolated database and server for this test
		db := NewInMemoryDB()
		defer db.Close()

		logBackend := createTestLogBackend()
		defer logBackend.Close()

		server := &TestServer{
			Server: NewServer(db, logBackend),
		}

		ctx := context.Background()
		player1ID := "player1"
		player2ID := "player2"

		// Set up initial balances
		_, err := server.UpdateBalance(ctx, &pokerrpc.UpdateBalanceRequest{
			PlayerId:    player1ID,
			Amount:      2500,
			Description: "initial deposit",
		})
		require.NoError(t, err)

		_, err = server.UpdateBalance(ctx, &pokerrpc.UpdateBalanceRequest{
			PlayerId:    player2ID,
			Amount:      1000,
			Description: "initial deposit",
		})
		require.NoError(t, err)

		// Test successful table creation
		resp, err := server.CreateTable(ctx, &pokerrpc.CreateTableRequest{
			PlayerId:      player1ID,
			SmallBlind:    10,
			BigBlind:      20,
			MinPlayers:    2,
			MaxPlayers:    6,
			BuyIn:         1000,
			StartingChips: 1000,
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

		server := &TestServer{
			Server: NewServer(db, logBackend),
		}

		ctx := context.Background()

		// Test non-existent table
		_, err := server.GetGameState(ctx, &pokerrpc.GetGameStateRequest{
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

		server := &TestServer{
			Server: NewServer(db, logBackend),
		}

		ctx := context.Background()
		player1ID := "player1"
		player2ID := "player2"

		// Set up initial balances
		_, err := server.UpdateBalance(ctx, &pokerrpc.UpdateBalanceRequest{
			PlayerId:    player1ID,
			Amount:      2500,
			Description: "initial deposit",
		})
		require.NoError(t, err)

		_, err = server.UpdateBalance(ctx, &pokerrpc.UpdateBalanceRequest{
			PlayerId:    player2ID,
			Amount:      1000,
			Description: "initial deposit",
		})
		require.NoError(t, err)

		// Create table
		createResp, err := server.CreateTable(ctx, &pokerrpc.CreateTableRequest{
			PlayerId:      player1ID,
			SmallBlind:    10,
			BigBlind:      20,
			MinPlayers:    2,
			MaxPlayers:    6,
			BuyIn:         1000,
			StartingChips: 1000,
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

	server := &TestServer{
		Server: NewServer(db, logBackend),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Players
	alice := "alice"
	bob := "bob"
	charlie := "charlie"

	// Give players initial balance
	for _, player := range []string{alice, bob, charlie} {
		_, err := server.UpdateBalance(ctx, &pokerrpc.UpdateBalanceRequest{
			PlayerId:    player,
			Amount:      5000,
			Description: "initial balance",
		})
		require.NoError(t, err)
	}

	// Alice creates a table
	createResp, err := server.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:      alice,
		SmallBlind:    5,
		BigBlind:      10,
		MinPlayers:    3,
		MaxPlayers:    6,
		BuyIn:         100,
		StartingChips: 1000,
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

	server := &TestServer{
		Server: NewServer(db, logBackend),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	host := "host"
	player := "player"

	// Give players initial balance
	for _, p := range []string{host, player} {
		_, err := server.UpdateBalance(ctx, &pokerrpc.UpdateBalanceRequest{
			PlayerId:    p,
			Amount:      5000,
			Description: "initial balance",
		})
		require.NoError(t, err)
	}

	// Host creates table
	createResp, err := server.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:      host,
		SmallBlind:    5,
		BigBlind:      10,
		MinPlayers:    2,
		MaxPlayers:    6,
		BuyIn:         100,
		StartingChips: 1000,
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
	assert.Contains(t, leaveResp.Message, "Host transferred")

	// Verify table still exists and player is now host
	tablesResp, err := server.GetTables(ctx, &pokerrpc.GetTablesRequest{})
	require.NoError(t, err)
	assert.Len(t, tablesResp.Tables, 1)
	assert.Equal(t, player, tablesResp.Tables[0].HostId)
}

// Heads-up: post-flop the big blind (non-dealer) must act first.
func TestHeadsUpPostflopActorIsBB(t *testing.T) {
	db := NewInMemoryDB()
	defer db.Close()

	logBackend := createTestLogBackend()
	defer logBackend.Close()

	server := &TestServer{Server: NewServer(db, logBackend)}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	p1 := "p1"
	p2 := "p2"

	// Fund players
	for _, pid := range []string{p1, p2} {
		_, err := server.UpdateBalance(ctx, &pokerrpc.UpdateBalanceRequest{
			PlayerId: pid, Amount: 5000, Description: "init",
		})
		require.NoError(t, err)
	}

	// Create HU table (5/10)
	createResp, err := server.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:      p1,
		SmallBlind:    5,
		BigBlind:      10,
		MinPlayers:    2,
		MaxPlayers:    2,
		BuyIn:         100,
		StartingChips: 1000,
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

	server := &TestServer{Server: NewServer(db, logBackend)}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	p1 := "p1"
	p2 := "p2"

	// Fund players
	for _, pid := range []string{p1, p2} {
		_, err := server.UpdateBalance(ctx, &pokerrpc.UpdateBalanceRequest{PlayerId: pid, Amount: 5000, Description: "init"})
		require.NoError(t, err)
	}

	// Create heads-up table (SB=5, BB=10).
	createResp, err := server.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:      p1,
		SmallBlind:    5,
		BigBlind:      10,
		MinPlayers:    2,
		MaxPlayers:    2,
		BuyIn:         100,
		StartingChips: 1000,
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

	server := &TestServer{Server: NewServer(db, logBackend)}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	p1 := "p1"
	p2 := "p2"

	// Fund players
	for _, pid := range []string{p1, p2} {
		_, err := server.UpdateBalance(ctx, &pokerrpc.UpdateBalanceRequest{PlayerId: pid, Amount: 5000, Description: "init"})
		require.NoError(t, err)
	}

	// Create table
	createResp, err := server.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:      p1,
		SmallBlind:    5,
		BigBlind:      10,
		MinPlayers:    2,
		MaxPlayers:    2,
		BuyIn:         100,
		StartingChips: 1000,
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
	assert.Contains(t, err.Error(), "big blind")
}

func TestLastPlayerLeavesTableClosure(t *testing.T) {
	// Create isolated database and server for this test
	db := NewInMemoryDB()
	defer db.Close()

	logBackend := createTestLogBackend()
	defer logBackend.Close()

	server := &TestServer{
		Server: NewServer(db, logBackend),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	host := "host"

	// Give host initial balance
	_, err := server.UpdateBalance(ctx, &pokerrpc.UpdateBalanceRequest{
		PlayerId:    host,
		Amount:      5000,
		Description: "initial balance",
	})
	require.NoError(t, err)

	// Host creates table
	createResp, err := server.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:      host,
		SmallBlind:    5,
		BigBlind:      10,
		MinPlayers:    2,
		MaxPlayers:    6,
		BuyIn:         100,
		StartingChips: 1000,
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
	assert.Contains(t, leaveResp.Message, "table closed")

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

	server := &TestServer{
		Server: NewServer(db, logBackend),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	host := "host"
	player := "player"

	// Give players initial balance
	for _, p := range []string{host, player} {
		_, err := server.UpdateBalance(ctx, &pokerrpc.UpdateBalanceRequest{
			PlayerId:    p,
			Amount:      5000,
			Description: "initial balance",
		})
		require.NoError(t, err)
	}

	// Host creates table
	createResp, err := server.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:      host,
		SmallBlind:    5,
		BigBlind:      10,
		MinPlayers:    2,
		MaxPlayers:    6,
		BuyIn:         100,
		StartingChips: 1000,
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
	assert.Equal(t, host, tablesResp.Tables[0].HostId)
	assert.Equal(t, int32(1), tablesResp.Tables[0].CurrentPlayers, "Table should have 1 player remaining")
}

func TestLeaveTable(t *testing.T) {
	// Create isolated database and server for this test
	db := NewInMemoryDB()
	defer db.Close()

	logBackend := createTestLogBackend()
	defer logBackend.Close()

	server := &TestServer{
		Server: NewServer(db, logBackend),
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

func TestJoinTable(t *testing.T) {
	// Create isolated database and server for this test
	db := NewInMemoryDB()
	defer db.Close()

	logBackend := createTestLogBackend()
	defer logBackend.Close()

	server := &TestServer{
		Server: NewServer(db, logBackend),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	player1ID := "player1"
	player2ID := "player2"

	// Set up initial balances
	_, err := server.UpdateBalance(ctx, &pokerrpc.UpdateBalanceRequest{
		PlayerId:    player1ID,
		Amount:      2500,
		Description: "initial deposit",
	})
	require.NoError(t, err)

	_, err = server.UpdateBalance(ctx, &pokerrpc.UpdateBalanceRequest{
		PlayerId:    player2ID,
		Amount:      1000,
		Description: "initial deposit",
	})
	require.NoError(t, err)

	// Create table
	createResp, err := server.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:      player1ID,
		SmallBlind:    10,
		BigBlind:      20,
		MinPlayers:    2,
		MaxPlayers:    6,
		BuyIn:         1000,
		StartingChips: 1000,
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

	// Test joining with insufficient balance
	player3ID := "player3"
	_, err = server.UpdateBalance(ctx, &pokerrpc.UpdateBalanceRequest{
		PlayerId:    player3ID,
		Amount:      500, // Not enough for 1000 buy-in
		Description: "insufficient balance",
	})
	require.NoError(t, err)

	resp, err = server.JoinTable(ctx, &pokerrpc.JoinTableRequest{
		PlayerId: player3ID,
		TableId:  tableID,
	})
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Contains(t, resp.Message, "Insufficient DCR balance")
}

// TestSnapshotRestoresCurrentPlayer ensures that when a snapshot is taken while it is a particular
// player's turn (and that player subsequently disconnects), restoring the table from the persisted
// snapshot correctly identifies the same player as the current player to act.
func TestSnapshotRestoresCurrentPlayer(t *testing.T) {
	// Use the same in-memory DB for the two server instances so that persisted state survives.
	db := NewInMemoryDB()
	defer db.Close()

	logBackend := createTestLogBackend()
	defer logBackend.Close()

	// First server instance — runs the game and produces a snapshot.
	srv1 := &TestServer{Server: NewServer(db, logBackend)}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Players
	p1 := "p1"
	p2 := "p2"
	p3 := "p3"

	// Fund players
	for _, pid := range []string{p1, p2, p3} {
		_, err := srv1.UpdateBalance(ctx, &pokerrpc.UpdateBalanceRequest{
			PlayerId:    pid,
			Amount:      5000,
			Description: "initial",
		})
		require.NoError(t, err)
	}

	// p1 creates table
	createResp, err := srv1.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:      p1,
		SmallBlind:    5,
		BigBlind:      10,
		MinPlayers:    3,
		MaxPlayers:    6,
		BuyIn:         100,
		StartingChips: 1000,
	})
	require.NoError(t, err)
	tableID := createResp.TableId

	// p2 and p3 join
	for _, pid := range []string{p2, p3} {
		joinResp, err := srv1.JoinTable(ctx, &pokerrpc.JoinTableRequest{
			PlayerId: pid,
			TableId:  tableID,
		})
		require.NoError(t, err)
		assert.True(t, joinResp.Success)
	}

	// Everyone ready
	for _, pid := range []string{p1, p2, p3} {
		_, err := srv1.SetPlayerReady(ctx, &pokerrpc.SetPlayerReadyRequest{
			PlayerId: pid,
			TableId:  tableID,
		})
		require.NoError(t, err)
	}

	// Wait for game to start
	var currentPlayer string
	for i := 0; i < 20; i++ {
		time.Sleep(25 * time.Millisecond)
		stateResp, err := srv1.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
		require.NoError(t, err)

		if stateResp.GameState.GameStarted && stateResp.GameState.CurrentPlayer != "" {
			currentPlayer = stateResp.GameState.CurrentPlayer
			break
		}
	}
	require.NotEmpty(t, currentPlayer, "failed to retrieve current player")

	// Give the async persistence some time to complete safely.
	time.Sleep(50 * time.Millisecond)

	// Second server instance — loads the previously saved snapshot.
	srv2 := &TestServer{Server: NewServer(db, logBackend)}

	// Wait until the table FSM transitions to GAME_ACTIVE and a current player is available.
	var restoredState *pokerrpc.GameUpdate
	require.Eventually(t, func() bool {
		st, err := srv2.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
		require.NoError(t, err)
		if st != nil {
			restoredState = st.GameState
			if restoredState != nil && restoredState.GameStarted && restoredState.CurrentPlayer != "" {
				return true
			}
		}
		return false
	}, 2*time.Second, 20*time.Millisecond, "restore did not stabilize")

	// For any follow-up assertions below, ensure we captured the stabilized state.
	require.NotNil(t, restoredState)
	require.True(t, restoredState.GameStarted)
	require.NotEmpty(t, restoredState.CurrentPlayer)

	// Debug: Print the restored state to understand what's happening
	t.Logf("Restored state: GameStarted=%v, CurrentPlayer=%s, CurrentBet=%d, Phase=%v",
		restoredState.GameStarted, restoredState.CurrentPlayer,
		restoredState.CurrentBet, restoredState.Phase)

	// Verify the game state is consistent
	assert.True(t, restoredState.GameStarted, "game should still be started after restoration")
	assert.NotEmpty(t, restoredState.CurrentPlayer, "current player should be calculated after restoration")
	assert.Greater(t, restoredState.CurrentBet, int64(0), "current bet should be positive after restoration")
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

	srv := &TestServer{Server: NewServer(db, logBackend)}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Players
	p1 := "p1"
	p2 := "p2"

	// Fund players with sufficient DCR balance (atoms)
	for _, pid := range []string{p1, p2} {
		_, err := srv.UpdateBalance(ctx, &pokerrpc.UpdateBalanceRequest{
			PlayerId:    pid,
			Amount:      5000,
			Description: "initial deposit",
		})
		require.NoError(t, err)
	}

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
		BuyIn:         100,
		StartingChips: startingChips,
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

	/// Wait until blinds are posted: PRE_FLOP and CurrentBet == bigBlind
	waitFor := func(cond func(*pokerrpc.GameUpdate) bool) *pokerrpc.GameUpdate {
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			st, err := srv.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
			require.NoError(t, err)
			if cond(st.GameState) {
				return st.GameState
			}
			time.Sleep(10 * time.Millisecond)
		}
		t.Fatal("condition not satisfied in time")
		return nil
	}

	gameState := waitFor(func(g *pokerrpc.GameUpdate) bool {
		return g.GameStarted && g.Phase == pokerrpc.GamePhase_PRE_FLOP && g.CurrentBet == bigBlind
	})

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
	updated := waitFor(func(g *pokerrpc.GameUpdate) bool {
		if g.CurrentBet != bigBlind {
			return false
		}
		have := 0
		for _, pl := range g.Players {
			if pl.CurrentBet == bigBlind {
				have++
			}
		}
		return have == 2
	})

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
