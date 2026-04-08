package server

import (
	"context"
	"strings"
	"time"

	"github.com/vctt94/pokerbisonrelay/pkg/poker"
	"github.com/vctt94/pokerbisonrelay/pkg/server/internal/db"
)

// metricsDB decorates a Database with basic latency + busy error metrics.
type metricsDB struct {
	inner Database
}

func (m *metricsDB) Close() error { return m.inner.Close() }

func (m *metricsDB) UpsertSnapshot(ctx context.Context, s db.Snapshot) error {
	start := time.Now()
	err := m.inner.UpsertSnapshot(ctx, s)
	m.observe(start, err)
	return err
}

func (m *metricsDB) GetSnapshot(ctx context.Context, tableID string) (*db.Snapshot, error) {
	start := time.Now()
	v, err := m.inner.GetSnapshot(ctx, tableID)
	m.observe(start, err)
	return v, err
}

func (m *metricsDB) UpsertMatchCheckpoint(ctx context.Context, c db.MatchCheckpoint) error {
	start := time.Now()
	err := m.inner.UpsertMatchCheckpoint(ctx, c)
	m.observe(start, err)
	return err
}

func (m *metricsDB) GetMatchCheckpoint(ctx context.Context, tableID string) (*db.MatchCheckpoint, error) {
	start := time.Now()
	v, err := m.inner.GetMatchCheckpoint(ctx, tableID)
	m.observe(start, err)
	return v, err
}

func (m *metricsDB) DeleteMatchCheckpoint(ctx context.Context, tableID string) error {
	start := time.Now()
	err := m.inner.DeleteMatchCheckpoint(ctx, tableID)
	m.observe(start, err)
	return err
}

func (m *metricsDB) ReplaceSettlementEscrows(ctx context.Context, matchID string, seats map[uint32]string) error {
	start := time.Now()
	err := m.inner.ReplaceSettlementEscrows(ctx, matchID, seats)
	m.observe(start, err)
	return err
}

func (m *metricsDB) ListSettlementEscrows(ctx context.Context) ([]db.SettlementEscrow, error) {
	start := time.Now()
	v, err := m.inner.ListSettlementEscrows(ctx)
	m.observe(start, err)
	return v, err
}

func (m *metricsDB) DeleteSettlementEscrows(ctx context.Context, matchID string) error {
	start := time.Now()
	err := m.inner.DeleteSettlementEscrows(ctx, matchID)
	m.observe(start, err)
	return err
}

func (m *metricsDB) UpsertRefereeEscrow(ctx context.Context, row db.RefereeEscrow) error {
	start := time.Now()
	err := m.inner.UpsertRefereeEscrow(ctx, row)
	m.observe(start, err)
	return err
}

func (m *metricsDB) ListRefereeEscrows(ctx context.Context) ([]db.RefereeEscrow, error) {
	start := time.Now()
	v, err := m.inner.ListRefereeEscrows(ctx)
	m.observe(start, err)
	return v, err
}

func (m *metricsDB) DeleteRefereeEscrow(ctx context.Context, escrowID string) error {
	start := time.Now()
	err := m.inner.DeleteRefereeEscrow(ctx, escrowID)
	m.observe(start, err)
	return err
}

func (m *metricsDB) UpsertRefereeBranchGamma(ctx context.Context, row db.RefereeBranchGamma) error {
	start := time.Now()
	err := m.inner.UpsertRefereeBranchGamma(ctx, row)
	m.observe(start, err)
	return err
}

func (m *metricsDB) ListRefereeBranchGammas(ctx context.Context) ([]db.RefereeBranchGamma, error) {
	start := time.Now()
	v, err := m.inner.ListRefereeBranchGammas(ctx)
	m.observe(start, err)
	return v, err
}

func (m *metricsDB) DeleteRefereeBranchGammas(ctx context.Context, matchID string) error {
	start := time.Now()
	err := m.inner.DeleteRefereeBranchGammas(ctx, matchID)
	m.observe(start, err)
	return err
}

func (m *metricsDB) UpsertRefereePresign(ctx context.Context, row db.RefereePresign) error {
	start := time.Now()
	err := m.inner.UpsertRefereePresign(ctx, row)
	m.observe(start, err)
	return err
}

func (m *metricsDB) ListRefereePresigns(ctx context.Context) ([]db.RefereePresign, error) {
	start := time.Now()
	v, err := m.inner.ListRefereePresigns(ctx)
	m.observe(start, err)
	return v, err
}

func (m *metricsDB) DeleteRefereePresigns(ctx context.Context, matchID string) error {
	start := time.Now()
	err := m.inner.DeleteRefereePresigns(ctx, matchID)
	m.observe(start, err)
	return err
}

func (m *metricsDB) UpsertPendingSettlement(ctx context.Context, row db.PendingSettlement) error {
	start := time.Now()
	err := m.inner.UpsertPendingSettlement(ctx, row)
	m.observe(start, err)
	return err
}

func (m *metricsDB) ListPendingSettlements(ctx context.Context) ([]db.PendingSettlement, error) {
	start := time.Now()
	v, err := m.inner.ListPendingSettlements(ctx)
	m.observe(start, err)
	return v, err
}

func (m *metricsDB) DeletePendingSettlement(ctx context.Context, matchID string) error {
	start := time.Now()
	err := m.inner.DeletePendingSettlement(ctx, matchID)
	m.observe(start, err)
	return err
}

func (m *metricsDB) UpsertTable(ctx context.Context, t *poker.TableConfig) error {
	start := time.Now()
	err := m.inner.UpsertTable(ctx, t)
	m.observe(start, err)
	return err
}

func (m *metricsDB) GetTable(ctx context.Context, id string) (*db.Table, error) {
	start := time.Now()
	v, err := m.inner.GetTable(ctx, id)
	m.observe(start, err)
	return v, err
}

func (m *metricsDB) DeleteTable(ctx context.Context, id string) error {
	start := time.Now()
	err := m.inner.DeleteTable(ctx, id)
	m.observe(start, err)
	return err
}

func (m *metricsDB) ListTableIDs(ctx context.Context) ([]string, error) {
	start := time.Now()
	v, err := m.inner.ListTableIDs(ctx)
	m.observe(start, err)
	return v, err
}

func (m *metricsDB) ActiveParticipants(ctx context.Context, tableID string) ([]db.Participant, error) {
	start := time.Now()
	v, err := m.inner.ActiveParticipants(ctx, tableID)
	m.observe(start, err)
	return v, err
}

func (m *metricsDB) SeatPlayer(ctx context.Context, tableID, playerID string, seat int) error {
	start := time.Now()
	err := m.inner.SeatPlayer(ctx, tableID, playerID, seat)
	m.observe(start, err)
	return err
}

func (m *metricsDB) UnseatPlayer(ctx context.Context, tableID, playerID string) error {
	start := time.Now()
	err := m.inner.UnseatPlayer(ctx, tableID, playerID)
	m.observe(start, err)
	return err
}

func (m *metricsDB) SetReady(ctx context.Context, tableID, playerID string, ready bool) error {
	start := time.Now()
	err := m.inner.SetReady(ctx, tableID, playerID, ready)
	m.observe(start, err)
	return err
}

func (m *metricsDB) UpsertAuthUser(ctx context.Context, nickname, userID string) error {
	start := time.Now()
	err := m.inner.UpsertAuthUser(ctx, nickname, userID)
	m.observe(start, err)
	return err
}

func (m *metricsDB) GetAuthUserByNickname(ctx context.Context, nickname string) (*db.AuthUser, error) {
	start := time.Now()
	user, err := m.inner.GetAuthUserByNickname(ctx, nickname)
	m.observe(start, err)
	return user, err
}

func (m *metricsDB) GetAuthUserByUserID(ctx context.Context, userID string) (*db.AuthUser, error) {
	start := time.Now()
	user, err := m.inner.GetAuthUserByUserID(ctx, userID)
	m.observe(start, err)
	return user, err
}

func (m *metricsDB) UpdateAuthUserLastLogin(ctx context.Context, userID string) error {
	start := time.Now()
	err := m.inner.UpdateAuthUserLastLogin(ctx, userID)
	m.observe(start, err)
	return err
}

func (m *metricsDB) UpdateAuthUserPayoutAddress(ctx context.Context, userID, payoutAddress string) error {
	start := time.Now()
	err := m.inner.UpdateAuthUserPayoutAddress(ctx, userID, payoutAddress)
	m.observe(start, err)
	return err
}

func (m *metricsDB) ListAllAuthUsers(ctx context.Context) ([]db.AuthUser, error) {
	start := time.Now()
	users, err := m.inner.ListAllAuthUsers(ctx)
	m.observe(start, err)
	return users, err
}

func (m *metricsDB) observe(start time.Time, err error) {
	GetMetrics().ObserveDBOp(time.Since(start))
	if err == nil {
		return
	}
	ls := strings.ToLower(err.Error())
	if strings.Contains(ls, "database is locked") || strings.Contains(ls, "busy") {
		GetMetrics().IncDBBusy()
	}
}
