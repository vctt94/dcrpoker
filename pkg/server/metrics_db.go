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
