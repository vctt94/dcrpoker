package server

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/decred/slog"
	"github.com/vctt94/pokerbisonrelay/pkg/poker"
	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
	sdb "github.com/vctt94/pokerbisonrelay/pkg/server/internal/db"
)

// failingDB implements Database and forces UpsertSnapshot to fail to validate server error paths.
type failingDB struct{}

func (f failingDB) GetPlayerBalance(ctx context.Context, playerID string) (int64, error) {
	return 0, nil
}
func (f failingDB) UpdatePlayerBalance(ctx context.Context, playerID string, amount int64, typ, desc string) error {
	return nil
}

// Tables (only what the Server may call during this test)
func (f failingDB) GetTable(ctx context.Context, id string) (*sdb.Table, error) { return nil, nil }
func (f failingDB) DeleteTable(ctx context.Context, id string) error            { return nil }
func (f failingDB) ListTableIDs(ctx context.Context) ([]string, error)          { return nil, nil }

// Participants
func (f failingDB) ActiveParticipants(ctx context.Context, tableID string) ([]sdb.Participant, error) {
	return nil, nil
}
func (f failingDB) SeatPlayer(ctx context.Context, tableID, playerID string, seat int) error {
	return nil
}
func (f failingDB) UnseatPlayer(ctx context.Context, tableID, playerID string) error { return nil }

// Snapshots — force failure
func (f failingDB) UpsertSnapshot(ctx context.Context, s sdb.Snapshot) error {
	return errors.New("forced snapshot error")
}

func (f failingDB) GetSnapshot(ctx context.Context, tableID string) (*sdb.Snapshot, error) {
	return nil, fmt.Errorf("snapshot not found")
}

// Auth
func (f failingDB) UpsertAuthUser(ctx context.Context, _, _ string) error { return nil }
func (f failingDB) GetAuthUserByNickname(ctx context.Context, _ string) (*sdb.AuthUser, error) {
	return nil, fmt.Errorf("user not found")
}
func (f failingDB) GetAuthUserByUserID(ctx context.Context, _ string) (*sdb.AuthUser, error) {
	return nil, fmt.Errorf("user not found")
}
func (f failingDB) UpdateAuthUserLastLogin(ctx context.Context, _ string) error  { return nil }
func (f failingDB) ListAllAuthUsers(ctx context.Context) ([]sdb.AuthUser, error) { return nil, nil }

func (f failingDB) Close() error { return nil }

func TestEventProcessorQueueFullDrop(t *testing.T) {
	s := newBareServer()
	ep := NewEventProcessor(s, 1, 0) // small queue, no workers to process
	ep.Start()
	defer ep.Stop()

	ep.PublishEvent(&GameEvent{Type: pokerrpc.NotificationType_PLAYER_READY, TableID: "tid"})
	if got := len(ep.queue); got != 1 {
		t.Fatalf("expected queue length 1, got %d", got)
	}
	// Publish another; should drop due to full queue, not panic
	ep.PublishEvent(&GameEvent{Type: pokerrpc.NotificationType_BET_MADE, TableID: "tid"})
	if got := len(ep.queue); got != 1 {
		t.Fatalf("queue length changed unexpectedly, got %d", got)
	}
}

func (f failingDB) UpsertTable(_ context.Context, _ *poker.TableConfig) error { return nil }

func (f failingDB) DeleteTableState(string) error { return nil }

func (f failingDB) DeletePlayerState(string, string) error { return nil }
func (f failingDB) GetAllTableIDs() ([]string, error)      { return nil, nil }

func TestSaveTableState_SnapshotFailure(t *testing.T) {
	s := &Server{log: slog.Disabled, db: failingDB{}}

	// Build and register a minimal active table so snapshot has content
	cfg := poker.TableConfig{
		ID:               "tid",
		Log:              slog.Disabled,
		GameLog:          slog.Disabled,
		HostID:           "h",
		MinPlayers:       2,
		MaxPlayers:       2,
		SmallBlind:       5,
		BigBlind:         10,
		StartingChips:    1000,
		TimeBank:         time.Second,
		AutoAdvanceDelay: 1 * time.Second,
	}
	tbl := poker.NewTable(cfg)
	user1, err := tbl.AddNewUser("p1", nil)
	if err != nil {
		t.Fatalf("add p1: %v", err)
	}
	user2, err := tbl.AddNewUser("p2", nil)
	if err != nil {
		t.Fatalf("add p2: %v", err)
	}
	_ = user1.SendReady()
	_ = user2.SendReady()
	if !tbl.CheckAllPlayersReady() {
		t.Fatal("players should be ready")
	}
	if err := tbl.StartGame(); err != nil {
		t.Fatalf("start game: %v", err)
	}
	s.tables.Store("tid", tbl)

	// saveTableState should bubble up the forced snapshot error
	if err := s.saveTableState("tid"); err == nil {
		t.Fatalf("expected error from UpsertSnapshot, got nil")
	}
}
