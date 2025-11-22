package server

import (
	"context"
	"testing"
	"time"

	"github.com/decred/slog"
	"github.com/stretchr/testify/require"
	"github.com/vctt94/pokerbisonrelay/pkg/poker"
	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
	"github.com/vctt94/pokerbisonrelay/pkg/server/internal/db"
)

// ---------- Test scaffolding ---------- //

// stubDB is a minimal in-memory implementation of the Database interface used only for these unit tests.
type stubDB struct{}

// --- Players / wallet ---
func (stubDB) GetPlayerBalance(ctx context.Context, _ string) (int64, error) { return 0, nil }
func (stubDB) UpdatePlayerBalance(ctx context.Context, _ string, _ int64, _ string, _ string) error {
	return nil
}

func (stubDB) GetSnapshot(ctx context.Context, _ string) (*db.Snapshot, error) {
	return nil, nil
}

// --- Tables (only what tests may touch indirectly) ---
func (stubDB) GetTable(ctx context.Context, id string) (*db.Table, error) {
	return &db.Table{ID: id}, nil
}
func (stubDB) DeleteTable(ctx context.Context, _ string) error    { return nil }
func (stubDB) ListTableIDs(ctx context.Context) ([]string, error) { return nil, nil }

// --- Participants ---
func (stubDB) ActiveParticipants(ctx context.Context, _ string) ([]db.Participant, error) {
	return nil, nil
}
func (stubDB) SeatPlayer(ctx context.Context, _ string, _ string, _ int) error { return nil }
func (stubDB) UnseatPlayer(ctx context.Context, _ string, _ string) error      { return nil }

// --- Snapshots (fast-restore cache) ---
func (stubDB) UpsertSnapshot(ctx context.Context, _ db.Snapshot) error   { return nil }
func (stubDB) UpsertTable(_ context.Context, _ *poker.TableConfig) error { return nil }

// --- Auth ---
func (stubDB) UpsertAuthUser(ctx context.Context, _, _ string) error { return nil }
func (stubDB) GetAuthUserByNickname(ctx context.Context, _ string) (*db.AuthUser, error) {
	return nil, nil
}
func (stubDB) GetAuthUserByUserID(ctx context.Context, _ string) (*db.AuthUser, error) {
	return nil, nil
}
func (stubDB) UpdateAuthUserLastLogin(ctx context.Context, _ string) error { return nil }
func (stubDB) ListAllAuthUsers(ctx context.Context) ([]db.AuthUser, error) { return nil, nil }

// --- Close ---
func (stubDB) Close() error { return nil }

// newBareServer returns a minimal Server suitable for snapshot tests.
func newBareServer() *Server {
	return &Server{
		log: slog.Disabled,
		db:  stubDB{},
	}
}

// helper to build a 2-player table already in GAME_ACTIVE phase.
func buildActiveHeadsUpTable(t *testing.T, id string) *poker.Table {
	cfg := poker.TableConfig{
		ID:               id,
		Log:              slog.Disabled,
		HostID:           "p1",
		BuyIn:            0,
		MinPlayers:       2,
		MaxPlayers:       2,
		SmallBlind:       10,
		BigBlind:         20,
		StartingChips:    1000,
		TimeBank:         30 * time.Second,
		AutoAdvanceDelay: 1 * time.Second,
	}

	table := poker.NewTable(cfg)

	if _, err := table.AddNewUser("p1", "p1", 1000, 0); err != nil {
		t.Fatalf("add user p1: %v", err)
	}
	if _, err := table.AddNewUser("p2", "p2", 1000, 1); err != nil {
		t.Fatalf("add user p2: %v", err)
	}
	if err := table.SetPlayerReady("p1", true); err != nil {
		t.Fatalf("ready p1: %v", err)
	}
	if err := table.SetPlayerReady("p2", true); err != nil {
		t.Fatalf("ready p2: %v", err)
	}
	// advance state machine
	if !table.CheckAllPlayersReady() {
		t.Fatal("table should report PLAYERS_READY")
	}
	if err := table.StartGame(); err != nil {
		t.Fatalf("start game: %v", err)
	}
	return table
}

// ---------- Tests ---------- //

// TestTableSnapshotCurrentBet confirms CurrentBet in snapshot equals table BigBlind right after blinds.
func TestTableSnapshotCurrentBet(t *testing.T) {
	s := newBareServer()
	table := buildActiveHeadsUpTable(t, "table_test")
	s.tables.Store(table.GetConfig().ID, table)

	// Wait until PRE_FLOP with blinds posted.
	require.Eventually(t, func() bool {
		g := table.GetGame()
		if g == nil {
			return false
		}
		if g.GetPhase() != pokerrpc.GamePhase_PRE_FLOP {
			return false
		}
		snap := g.GetStateSnapshot()
		return snap.CurrentBet == table.GetConfig().BigBlind
	}, 2*time.Second, 10*time.Millisecond, "game did not reach PRE_FLOP with blinds posted")

	snap, err := s.collectTableSnapshot(table.GetConfig().ID)
	require.NoError(t, err)
	require.NotNil(t, snap.GameSnapshot)

	got, want := snap.GameSnapshot.CurrentBet, table.GetConfig().BigBlind
	if got != want {
		t.Fatalf("CurrentBet mismatch: got %d want %d", got, want)
	}
}

// TestCollectTableSnapshotMissingTable ensures an error is returned when trying
// to snapshot a non-existent table.
func TestCollectTableSnapshotMissingTable(t *testing.T) {
	s := newBareServer()
	_, err := s.collectTableSnapshot("unknown")
	if err == nil {
		t.Fatalf("expected error when table is missing, got nil")
	}
}
