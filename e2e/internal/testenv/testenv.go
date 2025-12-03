package testenv

import (
	"context"
	"net"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/companyzero/bisonrelay/zkidentity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vctt94/bisonbotkit/logging"
	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
	"github.com/vctt94/pokerbisonrelay/pkg/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
)

// Env holds the runtime components that make up a fully functional instance of
// the poker server backed by a real SQLite database. Each E2E test spins-up
// its own env so tests are completely isolated and can run in parallel.
type Env struct {
	t           *testing.T
	DB          server.Database
	PokerSrv    *server.Server
	GRPCSrv     *grpc.Server
	Conn        *grpc.ClientConn
	LobbyClient pokerrpc.LobbyServiceClient
	PokerClient pokerrpc.PokerServiceClient

	dialOpts   []grpc.DialOption
	dialTarget string

	// Test session management
	sessionsMu sync.Mutex
	sessions   map[string]string // playerID -> token
}

// NewLogBackend creates a LogBackend for testing.
func NewLogBackend() *logging.LogBackend {
	logBackend, err := logging.NewLogBackend(logging.LogConfig{
		LogFile:        "",
		DebugLevel:     "debug",
		MaxLogFiles:    1,
		MaxBufferLines: 100,
	})
	if err != nil {
		return &logging.LogBackend{}
	}
	return logBackend
}

// New creates an isolated server + gRPC client connection for tests.
func New(t *testing.T) *Env {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "poker.sqlite")
	database, err := server.NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("db: %v", err)
	}

	logBackend := NewLogBackend()
	pokerSrv, err := server.NewTestServer(database, logBackend)
	if err != nil {
		t.Fatalf("server: %v", err)
	}
	lis := bufconn.Listen(1024 * 1024)

	grpcSrv := grpc.NewServer()
	pokerrpc.RegisterLobbyServiceServer(grpcSrv, pokerSrv)
	pokerrpc.RegisterPokerServiceServer(grpcSrv, pokerSrv)
	pokerrpc.RegisterAuthServiceServer(grpcSrv, pokerSrv)
	pokerrpc.RegisterPokerRefereeServer(grpcSrv, pokerSrv)
	go func() { _ = grpcSrv.Serve(lis) }()

	dialer := func(ctx context.Context, _ string) (net.Conn, error) { return lis.Dial() }
	dialTarget := "bufnet"
	dialOpts := []grpc.DialOption{
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	conn, err := grpc.DialContext(context.Background(), dialTarget, dialOpts...)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}

	return &Env{
		t:           t,
		DB:          database,
		PokerSrv:    pokerSrv,
		GRPCSrv:     grpcSrv,
		Conn:        conn,
		LobbyClient: pokerrpc.NewLobbyServiceClient(conn),
		PokerClient: pokerrpc.NewPokerServiceClient(conn),
		dialOpts:    dialOpts,
		dialTarget:  dialTarget,
		sessions:    make(map[string]string),
	}
}

// Close gracefully shuts down all resources.
func (e *Env) Close() {
	_ = e.Conn.Close()
	e.PokerSrv.Stop()
	e.GRPCSrv.Stop()
	_ = e.DB.Close()
}

// SetBalance ensures the player has exactly the specified balance by accessing
// the player through the server's internal state. This only works if the player
// is currently in an active game.
func (e *Env) SetBalance(ctx context.Context, playerID string, balance int64) {
	// Find the player in any active game
	tables := e.PokerSrv.GetAllTables()
	for _, table := range tables {
		game := table.GetGame()
		if game == nil {
			continue
		}
		players := game.GetPlayers()
		for _, player := range players {
			if player.ID() == playerID {
				player.SetBalance(balance)
				return
			}
		}
	}
	// Player not found in any active game - this is expected for players not yet in a game
	// In that case, balance will be set when they join a game
}

// WaitForGameStart polls GetGameState until GameStarted==true or the timeout
// expires (in which case the test fails).
func (e *Env) WaitForGameStart(ctx context.Context, tableID string, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	for {
		resp, err := e.PokerClient.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
		if err == nil && resp.GameState.GetGameStarted() {
			return
		}
		select {
		case <-ctx.Done():
			e.t.Fatalf("game did not start within %s", timeout)
		case <-time.After(50 * time.Millisecond):
		}
	}
}

// WaitForGamePhase polls GetGameState until the given phase is reached or the timeout expires.
func (e *Env) WaitForGamePhase(ctx context.Context, tableID string, phase pokerrpc.GamePhase, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	for {
		resp, err := e.PokerClient.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
		if err == nil && resp.GameState.GetPhase() == phase {
			return
		}
		select {
		case <-ctx.Done():
			e.t.Fatalf("game did not reach phase %s within %s", phase, timeout)
		case <-time.After(50 * time.Millisecond):
		}
	}
}

// GetBalance fetches a player's current balance by accessing the player through
// the server's internal state. This only works if the player is currently in an active game.
// Returns 0 if the player is not found in any active game.
func (e *Env) GetBalance(ctx context.Context, playerID string) int64 {
	// Find the player in any active game
	tables := e.PokerSrv.GetAllTables()
	for _, table := range tables {
		game := table.GetGame()
		if game == nil {
			continue
		}
		players := game.GetPlayers()
		for _, player := range players {
			if player.ID() == playerID {
				return player.Balance()
			}
		}
	}
	// Player not found in any active game - return 0
	return 0
}

// GetGameState gets the current game state for a table.
func (e *Env) GetGameState(ctx context.Context, tableID string) *pokerrpc.GameUpdate {
	resp, err := e.PokerClient.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
	require.NoError(e.t, err)
	return resp.GameState
}

// GetGameStateAllowNotFound returns the game state and a flag indicating the
// table was already removed. Any other error fails the test.
func (e *Env) GetGameStateAllowNotFound(ctx context.Context, tableID string) (*pokerrpc.GameUpdate, bool) {
	resp, err := e.PokerClient.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, true
		}
		require.NoError(e.t, err)
	}
	return resp.GameState, false
}

// WaitForShowdownOrRemoval waits until the game reaches SHOWDOWN or the table
// is removed (NotFound), to accommodate fast game-over flows where the table is
// closed immediately after settlement.
func (e *Env) WaitForShowdownOrRemoval(ctx context.Context, tableID string, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	for {
		resp, err := e.PokerClient.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
		if err == nil {
			if resp.GameState.GetPhase() == pokerrpc.GamePhase_SHOWDOWN {
				return
			}
		} else if status.Code(err) == codes.NotFound {
			return
		}

		select {
		case <-ctx.Done():
			e.t.Fatalf("game did not reach SHOWDOWN (or table removed) within %s", timeout)
		case <-time.After(50 * time.Millisecond):
		}
	}
}

// CreateStandardTable creates a table with standard settings for testing.
// Uses BuyIn: 0 to avoid escrow requirement in tests.
func (e *Env) CreateStandardTable(ctx context.Context, creatorID string, minPlayers, maxPlayers int) string {
	return e.CreateTableWithBuyIn(ctx, creatorID, minPlayers, maxPlayers, 0)
}

// CreateTableWithBuyIn creates a table with the provided buy-in/stack values.
// It automatically creates a test session for the creator if one doesn't exist.
// The creatorID parameter can be a simple string like "alice"; it will be converted
// to the ShortID string representation for the request.
func (e *Env) CreateTableWithBuyIn(ctx context.Context, creatorID string, minPlayers, maxPlayers int, buyIn int64) string {
	// Ensure test session exists for the creator
	e.EnsureTestSession(ctx, creatorID, creatorID)

	// Create context with token metadata
	ctx = e.ContextWithToken(ctx, creatorID)

	// Convert playerID to ShortID string representation for the request
	// The server validates that req.PlayerId matches sess.userID.String()
	playerIDStr := PlayerIDToShortIDString(creatorID)

	createResp, err := e.LobbyClient.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:      playerIDStr,
		SmallBlind:    10,
		BigBlind:      20,
		MinPlayers:    int32(minPlayers),
		MaxPlayers:    int32(maxPlayers),
		BuyIn:         buyIn,
		MinBalance:    buyIn,
		StartingChips: buyIn,
		AutoStartMs:   100,
		AutoAdvanceMs: 1000,
	})
	require.NoError(e.t, err)
	assert.NotEmpty(e.t, createResp.TableId)
	return createResp.TableId
}

// CreateTable is a helper that wraps LobbyClient.CreateTable with automatic token
// authentication and ShortID conversion. The playerID parameter can be a simple
// string like "alice"; it will be converted to the ShortID string representation.
// This is a convenience wrapper for tests that need custom table parameters.
func (e *Env) CreateTable(ctx context.Context, req *pokerrpc.CreateTableRequest) (*pokerrpc.CreateTableResponse, error) {
	// Ensure test session exists
	e.EnsureTestSession(ctx, req.PlayerId, req.PlayerId)

	// Create context with token metadata
	ctx = e.ContextWithToken(ctx, req.PlayerId)

	// Convert playerID to ShortID string representation
	playerIDStr := PlayerIDToShortIDString(req.PlayerId)

	// Clone the request to avoid copying the mutex
	reqCopy := proto.Clone(req).(*pokerrpc.CreateTableRequest)
	reqCopy.PlayerId = playerIDStr

	return e.LobbyClient.CreateTable(ctx, reqCopy)
}

// JoinTable is a helper that wraps LobbyClient.JoinTable with automatic token
// authentication and ShortID conversion. The playerID parameter can be a simple
// string like "alice"; it will be converted to the ShortID string representation.
func (e *Env) JoinTable(ctx context.Context, playerID, tableID string) (*pokerrpc.JoinTableResponse, error) {
	// Ensure test session exists
	e.EnsureTestSession(ctx, playerID, playerID)

	// Create context with token metadata
	ctx = e.ContextWithToken(ctx, playerID)

	// Convert playerID to ShortID string representation
	playerIDStr := PlayerIDToShortIDString(playerID)

	return e.LobbyClient.JoinTable(ctx, &pokerrpc.JoinTableRequest{
		PlayerId: playerIDStr,
		TableId:  tableID,
	})
}

// SetPlayerReady is a helper that wraps LobbyClient.SetPlayerReady with automatic
// token authentication and ShortID conversion. The playerID parameter can be a
// simple string like "alice"; it will be converted to the ShortID string representation.
func (e *Env) SetPlayerReady(ctx context.Context, playerID, tableID string) (*pokerrpc.SetPlayerReadyResponse, error) {
	// Ensure test session exists
	e.EnsureTestSession(ctx, playerID, playerID)

	// Create context with token metadata
	ctx = e.ContextWithToken(ctx, playerID)

	// Convert playerID to ShortID string representation
	playerIDStr := PlayerIDToShortIDString(playerID)

	return e.LobbyClient.SetPlayerReady(ctx, &pokerrpc.SetPlayerReadyRequest{
		PlayerId: playerIDStr,
		TableId:  tableID,
	})
}

// DialOptions returns dial options for creating additional clients that share
// the in-memory gRPC server used by this env.
func (e *Env) DialOptions() []grpc.DialOption {
	return append([]grpc.DialOption{}, e.dialOpts...)
}

// DialTarget returns the dial target to use with DialOptions.
func (e *Env) DialTarget() string {
	return e.dialTarget
}

// RegisterTestUser registers a test user directly in the auth_users table.
// This is required for operations that have foreign key constraints on auth_users.
// For simple test IDs (like "alice", "player1"), use the same string for both userID and nickname.
// This bypasses normal registration validation to allow any string ID for testing.
func (e *Env) RegisterTestUser(ctx context.Context, userID, nickname string) error {
	return e.DB.UpsertAuthUser(ctx, nickname, userID)
}

// PlayerIDToShortIDString converts a playerID string to its ShortID string representation.
// This uses the same deterministic logic as EnsureTestSession to ensure consistency.
// Tests should use this when making RPC calls that require a PlayerId matching the session's userID.
func PlayerIDToShortIDString(playerID string) string {
	var uid zkidentity.ShortID
	if err := uid.FromString(playerID); err != nil {
		// If not a valid ShortID, create a deterministic one
		var buf [32]byte
		copy(buf[:], playerID)
		if len(playerID) < 32 {
			for i := len(playerID); i < 32; i++ {
				buf[i] = playerID[i%len(playerID)]
			}
		}
		uid = zkidentity.ShortID(buf)
	}
	return uid.String()
}

// EnsureTestSession creates a test session for a player if one doesn't exist.
func (e *Env) EnsureTestSession(ctx context.Context, playerID, nickname string) string {
	e.sessionsMu.Lock()
	defer e.sessionsMu.Unlock()

	if token, exists := e.sessions[playerID]; exists {
		return token
	}

	// Create a deterministic ShortID from the playerID
	var uid zkidentity.ShortID
	if err := uid.FromString(playerID); err != nil {
		// If not a valid ShortID, create a deterministic one
		var buf [32]byte
		copy(buf[:], playerID)
		if len(playerID) < 32 {
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

	e.PokerSrv.TestSeedSession(token, uid, "", nickname)
	e.sessions[playerID] = token
	return token
}

// ContextWithToken returns a context with the token for the given playerID in metadata.
func (e *Env) ContextWithToken(ctx context.Context, playerID string) context.Context {
	e.sessionsMu.Lock()
	token, exists := e.sessions[playerID]
	e.sessionsMu.Unlock()

	if !exists {
		// Auto-create session if it doesn't exist
		token = e.EnsureTestSession(ctx, playerID, playerID)
	}

	return metadata.AppendToOutgoingContext(ctx, "token", token)
}
