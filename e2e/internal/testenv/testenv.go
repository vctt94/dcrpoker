package testenv

import (
	"context"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vctt94/bisonbotkit/logging"
	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
	"github.com/vctt94/pokerbisonrelay/pkg/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
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
	}
}

// Close gracefully shuts down all resources.
func (e *Env) Close() {
	_ = e.Conn.Close()
	e.PokerSrv.Stop()
	e.GRPCSrv.Stop()
	_ = e.DB.Close()
}

// SetBalance ensures the player has exactly the specified balance by calculating
// the delta against the current stored balance and issuing a single
// UpdateBalance call.
func (e *Env) SetBalance(ctx context.Context, playerID string, balance int64) {
	var currBal int64
	if resp, err := e.LobbyClient.GetBalance(ctx, &pokerrpc.GetBalanceRequest{PlayerId: playerID}); err == nil {
		currBal = resp.GetBalance()
	}
	delta := balance - currBal
	if delta == 0 {
		return
	}
	_, err := e.LobbyClient.UpdateBalance(ctx, &pokerrpc.UpdateBalanceRequest{
		PlayerId:    playerID,
		Amount:      delta,
		Description: "seed balance",
	})
	require.NoError(e.t, err)
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

// GetBalance fetches a player's current balance.
func (e *Env) GetBalance(ctx context.Context, playerID string) int64 {
	resp, err := e.LobbyClient.GetBalance(ctx, &pokerrpc.GetBalanceRequest{PlayerId: playerID})
	require.NoError(e.t, err)
	return resp.Balance
}

// GetGameState gets the current game state for a table.
func (e *Env) GetGameState(ctx context.Context, tableID string) *pokerrpc.GameUpdate {
	resp, err := e.PokerClient.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
	require.NoError(e.t, err)
	return resp.GameState
}

// CreateStandardTable creates a table with standard settings for testing.
func (e *Env) CreateStandardTable(ctx context.Context, creatorID string, minPlayers, maxPlayers int) string {
	return e.CreateTableWithBuyIn(ctx, creatorID, minPlayers, maxPlayers, 1_000)
}

// CreateTableWithBuyIn creates a table with the provided buy-in/stack values.
func (e *Env) CreateTableWithBuyIn(ctx context.Context, creatorID string, minPlayers, maxPlayers int, buyIn int64) string {
	createResp, err := e.LobbyClient.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:      creatorID,
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

// DialOptions returns dial options for creating additional clients that share
// the in-memory gRPC server used by this env.
func (e *Env) DialOptions() []grpc.DialOption {
	return append([]grpc.DialOption{}, e.dialOpts...)
}

// DialTarget returns the dial target to use with DialOptions.
func (e *Env) DialTarget() string {
	return e.dialTarget
}
