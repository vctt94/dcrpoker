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
func (f failingDB) SetReady(ctx context.Context, tableID, playerID string, ready bool) error {
	return nil
}

// Snapshots — force failure
func (f failingDB) UpsertSnapshot(ctx context.Context, s sdb.Snapshot) error {
	return errors.New("forced snapshot error")
}

func (f failingDB) UpsertMatchCheckpoint(ctx context.Context, c sdb.MatchCheckpoint) error {
	return nil
}

func (f failingDB) GetSnapshot(ctx context.Context, tableID string) (*sdb.Snapshot, error) {
	return nil, fmt.Errorf("snapshot not found")
}

func (f failingDB) GetMatchCheckpoint(ctx context.Context, tableID string) (*sdb.MatchCheckpoint, error) {
	return nil, fmt.Errorf("match checkpoint not found")
}

func (f failingDB) DeleteMatchCheckpoint(ctx context.Context, tableID string) error { return nil }
func (f failingDB) ReplaceSettlementEscrows(ctx context.Context, matchID string, seats map[uint32]string) error {
	return nil
}
func (f failingDB) ListSettlementEscrows(ctx context.Context) ([]sdb.SettlementEscrow, error) {
	return nil, nil
}
func (f failingDB) DeleteSettlementEscrows(ctx context.Context, matchID string) error { return nil }
func (f failingDB) UpsertRefereeEscrow(ctx context.Context, row sdb.RefereeEscrow) error {
	return nil
}
func (f failingDB) ListRefereeEscrows(ctx context.Context) ([]sdb.RefereeEscrow, error) {
	return nil, nil
}
func (f failingDB) DeleteRefereeEscrow(ctx context.Context, escrowID string) error { return nil }
func (f failingDB) UpsertRefereeBranchGamma(ctx context.Context, row sdb.RefereeBranchGamma) error {
	return nil
}
func (f failingDB) ListRefereeBranchGammas(ctx context.Context) ([]sdb.RefereeBranchGamma, error) {
	return nil, nil
}
func (f failingDB) DeleteRefereeBranchGammas(ctx context.Context, matchID string) error { return nil }
func (f failingDB) UpsertRefereePresign(ctx context.Context, row sdb.RefereePresign) error {
	return nil
}
func (f failingDB) ListRefereePresigns(ctx context.Context) ([]sdb.RefereePresign, error) {
	return nil, nil
}
func (f failingDB) DeleteRefereePresigns(ctx context.Context, matchID string) error { return nil }
func (f failingDB) UpsertPendingSettlement(ctx context.Context, row sdb.PendingSettlement) error {
	return nil
}
func (f failingDB) ListPendingSettlements(ctx context.Context) ([]sdb.PendingSettlement, error) {
	return nil, nil
}
func (f failingDB) DeletePendingSettlement(ctx context.Context, matchID string) error { return nil }

// Auth
func (f failingDB) UpsertAuthUser(ctx context.Context, _, _ string) error { return nil }
func (f failingDB) GetAuthUserByNickname(ctx context.Context, _ string) (*sdb.AuthUser, error) {
	return nil, fmt.Errorf("user not found")
}
func (f failingDB) GetAuthUserByUserID(ctx context.Context, _ string) (*sdb.AuthUser, error) {
	return nil, fmt.Errorf("user not found")
}
func (f failingDB) UpdateAuthUserLastLogin(ctx context.Context, _ string) error { return nil }
func (f failingDB) UpdateAuthUserPayoutAddress(ctx context.Context, _, _ string) error {
	return nil
}
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
