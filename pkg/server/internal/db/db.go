package db

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/vctt94/pokerbisonrelay/pkg/poker"
)

// DB wraps sql.
type DB struct {
	*sql.DB
}

// NewDB opens (or creates) a SQLite database, enables FKs, and creates schema.
func NewDB(path string) (*DB, error) {
	d, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	// SQLite recommended settings for server workload.
	// Restrict pool to a single connection; SQLite does not benefit from multiple writers.
	d.SetMaxOpenConns(1)
	d.SetMaxIdleConns(1)

	// Busy timeout to avoid immediate "database is locked" errors.
	if _, err := d.Exec(`PRAGMA busy_timeout = 5000;`); err != nil {
		_ = d.Close()
		return nil, fmt.Errorf("set busy_timeout: %w", err)
	}
	// WAL journaling for better concurrency.
	if _, err := d.Exec(`PRAGMA journal_mode = WAL;`); err != nil {
		_ = d.Close()
		return nil, fmt.Errorf("set journal_mode=WAL: %w", err)
	}
	// Foreign key enforcement.
	if _, err := d.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
		_ = d.Close()
		return nil, fmt.Errorf("enable foreign_keys: %w", err)
	}
	// Synchronous = NORMAL for throughput; consider FULL for maximum durability.
	if _, err := d.Exec(`PRAGMA synchronous = NORMAL;`); err != nil {
		_ = d.Close()
		return nil, fmt.Errorf("set synchronous=NORMAL: %w", err)
	}
	// Approx 64MB page cache (negative = KiB units).
	if _, err := d.Exec(`PRAGMA cache_size = -65536;`); err != nil {
		_ = d.Close()
		return nil, fmt.Errorf("set cache_size: %w", err)
	}
	// Temp objects in memory to reduce disk I/O for transient data.
	if _, err := d.Exec(`PRAGMA temp_store = MEMORY;`); err != nil {
		_ = d.Close()
		return nil, fmt.Errorf("set temp_store=MEMORY: %w", err)
	}
	// Cap mmap usage to 256MB.
	if _, err := d.Exec(`PRAGMA mmap_size = 268435456;`); err != nil {
		_ = d.Close()
		return nil, fmt.Errorf("set mmap_size: %w", err)
	}
	// Control WAL autocheckpointing.
	if _, err := d.Exec(`PRAGMA wal_autocheckpoint = 1000;`); err != nil {
		_ = d.Close()
		return nil, fmt.Errorf("set wal_autocheckpoint: %w", err)
	}
	if err := applyMigrations(d); err != nil {
		_ = d.Close()
		return nil, err
	}
	return &DB{d}, nil
}

// Close closes the underlying sql.DB to release all resources.
func (db *DB) Close() error { return db.DB.Close() }

// withTx runs fn in a transaction, committing on success.
func (db *DB) withTx(ctx context.Context, fn func(*sql.Tx) error) (err error) {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	if err = fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

// =========================
// ======= MIGRATIONS ======
// =========================

//go:embed migrations
var migrationFS embed.FS

type migration struct {
	version int
	name    string
	path    string
}

func applyMigrations(db *sql.DB) error {
	// Ensure the schema_migrations tracking table exists
	if _, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS schema_migrations (
            version     INTEGER PRIMARY KEY,
            name        TEXT NOT NULL,
            applied_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
        );
    `); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	// Discover migration files from embed FS
	entries, err := fs.ReadDir(migrationFS, "migrations")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	var migs []migration
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if filepath.Ext(name) != ".sql" {
			continue
		}
		// Parse version prefix: 0001_description.sql
		base := filepath.Base(name)
		parts := strings.SplitN(base, "_", 2)
		if len(parts) < 1 {
			continue
		}
		vstr := strings.TrimPrefix(parts[0], "")
		v, err := strconv.Atoi(strings.TrimLeft(vstr, "0"))
		if err != nil {
			// Fall back: if vstr is all zeros, treat as 0
			if vstr == "0000" || vstr == "0" {
				v = 0
			} else {
				return fmt.Errorf("invalid migration filename: %s", name)
			}
		}
		migs = append(migs, migration{version: v, name: name, path: filepath.Join("migrations", name)})
	}

	sort.Slice(migs, func(i, j int) bool { return migs[i].name < migs[j].name })

	// Load applied versions into a set
	applied := make(map[int]bool)
	rows, err := db.Query(`SELECT version FROM schema_migrations`)
	if err != nil {
		return fmt.Errorf("query schema_migrations: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return err
		}
		applied[v] = true
	}
	if err := rows.Err(); err != nil {
		return err
	}

	// Apply pending migrations
	for _, m := range migs {
		if applied[m.version] {
			continue
		}
		// Read SQL file
		b, err := migrationFS.ReadFile(m.path)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", m.name, err)
		}
		sqlText := string(b)

		// Execute statements one-by-one for better error control.
		// Split on ';' and trim; ignore empties.
		// Note: scripts in this repo are simple and do not embed semicolons in strings.
		stmts := splitSQLStatements(sqlText)
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		for _, s := range stmts {
			if s == "" {
				continue
			}
			if _, err := tx.Exec(s); err != nil {
				// Allow duplicate column on ADD COLUMN
				lerr := strings.ToLower(err.Error())
				if strings.Contains(lerr, "duplicate column") || strings.Contains(lerr, "duplicate column name") {
					continue
				}
				_ = tx.Rollback()
				return fmt.Errorf("migration %s failed: %w (stmt: %s)", m.name, err, s)
			}
		}
		if _, err := tx.Exec(`INSERT INTO schema_migrations(version, name) VALUES (?, ?)`, m.version, m.name); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %s: %w", m.name, err)
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

// splitSQLStatements splits a string on semicolons into independent statements.
func splitSQLStatements(s string) []string {
	out := make([]string, 0, 8)
	var b strings.Builder
	inSingle := false
	inDouble := false
	inLineComment := false
	inBlockComment := false
	rs := []rune(s)
	for i := 0; i < len(rs); i++ {
		r := rs[i]

		// Handle end of line comments
		if inLineComment {
			if r == '\n' || r == '\r' {
				inLineComment = false
				// Preserve newline so statement boundaries aren’t lost
				b.WriteRune(r)
			}
			continue
		}
		// Handle end of block comments
		if inBlockComment {
			if r == '*' && i+1 < len(rs) && rs[i+1] == '/' {
				inBlockComment = false
				i++ // skip '/'
			}
			continue
		}

		// Detect start of comments when not inside quotes
		if !inSingle && !inDouble {
			if r == '-' && i+1 < len(rs) && rs[i+1] == '-' {
				inLineComment = true
				i++ // skip second '-'
				continue
			}
			if r == '/' && i+1 < len(rs) && rs[i+1] == '*' {
				inBlockComment = true
				i++ // skip '*'
				continue
			}
		}

		// Quote state toggles
		if r == '\'' && !inDouble {
			inSingle = !inSingle
			b.WriteRune(r)
			continue
		}
		if r == '"' && !inSingle {
			inDouble = !inDouble
			b.WriteRune(r)
			continue
		}

		// Statement boundary
		if r == ';' && !inSingle && !inDouble {
			stmt := strings.TrimSpace(b.String())
			if stmt != "" {
				out = append(out, stmt)
			}
			b.Reset()
			continue
		}

		b.WriteRune(r)
	}
	if tail := strings.TrimSpace(b.String()); tail != "" {
		out = append(out, tail)
	}
	return out
}

// =========================
// ======== MODELS =========
// =========================

type Player struct {
	ID        string
	Name      string
	CreatedAt time.Time
}

type Transaction struct {
	ID          int64
	PlayerID    string
	Amount      int64
	Type        string
	Description sql.NullString
	CreatedAt   time.Time
}

type Table struct {
	ID                       string
	Name                     string
	Source                   string
	BuyIn                    int64
	MinPlayers               int
	MaxPlayers               int
	SmallBlind               int64
	BigBlind                 int64
	StartingChips            int64
	TimebankMS               int64
	AutoStartMS              int64
	AutoAdvanceMS            int64
	BlindIncreaseIntervalSec int64
	CreatedAt                time.Time
}

type Participant struct {
	TableID  string
	PlayerID string
	Seat     int
	JoinedAt time.Time
	LeftAt   sql.NullTime
	Ready    bool
}

type Hand struct {
	ID         int64
	TableID    string
	HandNo     int64
	StartedAt  time.Time
	EndedAt    sql.NullTime
	DealerSeat int
	SBSeat     int
	BBSeat     int
	ResultJSON sql.NullString
}

type HandPlayer struct {
	HandID        int64
	PlayerID      string
	Seat          int
	StartingStack int64
	HoleCardsJSON sql.NullString
}

type Action struct {
	ID        int64
	HandID    int64
	Ord       int
	Street    string
	ActorSeat int
	Action    string
	Amount    int64
	IsAllIn   bool
	CreatedAt time.Time
}

type BoardCards struct {
	HandID    int64
	Street    string
	CardsJSON string
}

type Snapshot struct {
	TableID    string
	SnapshotAt time.Time
	Payload    []byte
}

type MatchCheckpoint struct {
	TableID   string
	UpdatedAt time.Time
	Payload   []byte
}

// =========================
// ===== Players API =======
// =========================

func (db *DB) UpsertPlayer(ctx context.Context, id, name string) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO players (id, name)
		VALUES (?, ?)
		ON CONFLICT(id) DO UPDATE SET name = excluded.name
	`, id, name)
	return err
}

func (db *DB) GetPlayer(ctx context.Context, id string) (*Player, error) {
	row := db.QueryRowContext(ctx, `
		SELECT id, name, created_at
		FROM players
		WHERE id = ?
	`, id)
	var p Player
	if err := row.Scan(&p.ID, &p.Name, &p.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("player not found: %s", id)
		}
		return nil, err
	}
	return &p, nil
}

// =========================
// ===== Tables API ========
// =========================

func (db *DB) UpsertSnapshot(ctx context.Context, s Snapshot) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO table_snapshots (table_id, snapshot_at, payload)
		VALUES (?, COALESCE(?, CURRENT_TIMESTAMP), ?)
		ON CONFLICT(table_id) DO UPDATE SET
			snapshot_at = excluded.snapshot_at,
			payload     = excluded.payload
	`, s.TableID, s.SnapshotAt, s.Payload)
	return err
}

func (db *DB) UpsertTable(ctx context.Context, t *poker.TableConfig) error {
	timeBankMS := t.TimeBank.Milliseconds()
	autoStartMS := t.AutoStartDelay.Milliseconds()
	autoAdvanceMS := t.AutoAdvanceDelay.Milliseconds()
	source := strings.TrimSpace(t.Source)
	if source == "" {
		source = "user"
	}
	name := strings.TrimSpace(t.Name)
	if name == "" {
		name = t.ID
	}

	blindIncreaseSec := t.BlindIncreaseInterval.Seconds()

	query := `
		INSERT INTO tables (
			id, name, table_source, buy_in, min_players, max_players, small_blind, big_blind,
			starting_chips, timebank_ms, autostart_ms, auto_advance_ms, blind_increase_interval_sec, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, COALESCE(?, CURRENT_TIMESTAMP))
		ON CONFLICT(id) DO UPDATE SET
			name                        = excluded.name,
			table_source                = excluded.table_source,
			buy_in                      = excluded.buy_in,
			min_players                 = excluded.min_players,
			max_players                 = excluded.max_players,
			small_blind                 = excluded.small_blind,
			big_blind                   = excluded.big_blind,
			starting_chips              = excluded.starting_chips,
			timebank_ms                 = excluded.timebank_ms,
			autostart_ms                = excluded.autostart_ms,
			auto_advance_ms             = excluded.auto_advance_ms,
			blind_increase_interval_sec = excluded.blind_increase_interval_sec
	`
	args := []any{
		t.ID, name, source, t.BuyIn, t.MinPlayers, t.MaxPlayers, t.SmallBlind, t.BigBlind,
		t.StartingChips, timeBankMS, autoStartMS, autoAdvanceMS, int64(blindIncreaseSec), time.Now(),
	}

	_, err := db.ExecContext(ctx, query, args...)
	return err
}

func (db *DB) GetTable(ctx context.Context, id string) (*Table, error) {
	row := db.QueryRowContext(ctx, `
		SELECT id, name, COALESCE(table_source, 'user'),
			buy_in, min_players, max_players, small_blind, big_blind,
			starting_chips, timebank_ms, autostart_ms, auto_advance_ms,
			blind_increase_interval_sec, created_at
		FROM tables WHERE id = ?
	`, id)
	var t Table
	if err := row.Scan(&t.ID, &t.Name, &t.Source, &t.BuyIn, &t.MinPlayers, &t.MaxPlayers, &t.SmallBlind, &t.BigBlind,
		&t.StartingChips, &t.TimebankMS, &t.AutoStartMS, &t.AutoAdvanceMS,
		&t.BlindIncreaseIntervalSec, &t.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("table not found: %s", id)
		}
		return nil, err
	}
	return &t, nil
}

func (db *DB) DeleteTable(ctx context.Context, id string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM tables WHERE id = ?`, id)
	return err
}

func (db *DB) ListTableIDs(ctx context.Context) ([]string, error) {
	rows, err := db.QueryContext(ctx, `SELECT id FROM tables ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// ================================
// ===== Participants (Seats) =====
// ================================

func (db *DB) SeatPlayer(ctx context.Context, tableID, playerID string, seat int) error {
	// Ensure a players row exists so FK(table_participants.player_id → players.id) is satisfied.
	return db.withTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO players (id, name)
			VALUES (?, ?)
			ON CONFLICT(id) DO NOTHING
		`, playerID, playerID); err != nil {
			return err
		}

		// left_at MUST be NULL for an active seat; UNIQUE(table_id, seat) enforces occupancy.
		_, err := tx.ExecContext(ctx, `
			INSERT INTO table_participants (table_id, player_id, seat, joined_at, left_at, ready)
			VALUES (?, ?, ?, CURRENT_TIMESTAMP, NULL, FALSE)
			ON CONFLICT(table_id, player_id) DO UPDATE SET
				seat = excluded.seat,
				left_at = NULL
		`, tableID, playerID, seat)
		return err
	})
}

func (db *DB) UnseatPlayer(ctx context.Context, tableID, playerID string) error {
	_, err := db.ExecContext(ctx, `
		UPDATE table_participants
		SET left_at = COALESCE(left_at, CURRENT_TIMESTAMP)
		WHERE table_id = ? AND player_id = ? AND left_at IS NULL
	`, tableID, playerID)
	return err
}

func (db *DB) SetReady(ctx context.Context, tableID, playerID string, ready bool) error {
	_, err := db.ExecContext(ctx, `
		UPDATE table_participants
		SET ready = ?
		WHERE table_id = ? AND player_id = ?
	`, ready, tableID, playerID)
	return err
}

func (db *DB) ActiveParticipants(ctx context.Context, tableID string) ([]Participant, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT table_id, player_id, seat, joined_at, left_at, ready
		FROM table_participants
		WHERE table_id = ? AND left_at IS NULL
		ORDER BY seat
	`, tableID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Participant
	for rows.Next() {
		var p Participant
		if err := rows.Scan(&p.TableID, &p.PlayerID, &p.Seat, &p.JoinedAt, &p.LeftAt, &p.Ready); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (db *DB) ParticipantForPlayer(ctx context.Context, tableID, playerID string) (*Participant, error) {
	row := db.QueryRowContext(ctx, `
		SELECT table_id, player_id, seat, joined_at, left_at, ready
		FROM table_participants
		WHERE table_id = ? AND player_id = ? AND left_at IS NULL
	`, tableID, playerID)
	var p Participant
	if err := row.Scan(&p.TableID, &p.PlayerID, &p.Seat, &p.JoinedAt, &p.LeftAt, &p.Ready); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("not seated")
		}
		return nil, err
	}
	return &p, nil
}

// =======================================
// ===== Table Buy-ins / Cash-outs =======
// =======================================

func (db *DB) AddBuyIn(ctx context.Context, tableID, playerID string, amount int64) error {
	if amount <= 0 {
		return fmt.Errorf("buy-in must be positive")
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO table_buyins (table_id, player_id, amount, created_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
	`, tableID, playerID, amount)
	return err
}

func (db *DB) AddCashout(ctx context.Context, tableID, playerID string, amount int64) error {
	if amount <= 0 {
		return fmt.Errorf("cashout must be positive")
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO table_cashouts (table_id, player_id, amount, created_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
	`, tableID, playerID, amount)
	return err
}

func (db *DB) SumBuyIns(ctx context.Context, tableID, playerID string) (int64, error) {
	row := db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(amount), 0) FROM table_buyins WHERE table_id = ? AND player_id = ?
	`, tableID, playerID)
	var s int64
	if err := row.Scan(&s); err != nil {
		return 0, err
	}
	return s, nil
}

func (db *DB) SumCashouts(ctx context.Context, tableID, playerID string) (int64, error) {
	row := db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(amount), 0) FROM table_cashouts WHERE table_id = ? AND player_id = ?
	`, tableID, playerID)
	var s int64
	if err := row.Scan(&s); err != nil {
		return 0, err
	}
	return s, nil
}

// =========================
// ====== Hands API ========
// =========================

func (db *DB) GetOpenHand(ctx context.Context, tableID string) (*Hand, error) {
	row := db.QueryRowContext(ctx, `
		SELECT id, table_id, hand_no, started_at, ended_at, dealer_seat, sb_seat, bb_seat, result_json
		FROM hands
		WHERE table_id = ? AND ended_at IS NULL
		ORDER BY hand_no DESC
		LIMIT 1
	`, tableID)
	var h Hand
	if err := row.Scan(&h.ID, &h.TableID, &h.HandNo, &h.StartedAt, &h.EndedAt, &h.DealerSeat, &h.SBSeat, &h.BBSeat, &h.ResultJSON); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("no open hand")
		}
		return nil, err
	}
	return &h, nil
}

func (db *DB) NextHandNumber(ctx context.Context, tableID string) (int64, error) {
	row := db.QueryRowContext(ctx, `SELECT COALESCE(MAX(hand_no), 0) + 1 FROM hands WHERE table_id = ?`, tableID)
	var n int64
	if err := row.Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

func (db *DB) BeginHand(ctx context.Context, h *Hand) (int64, error) {
	res, err := db.ExecContext(ctx, `
		INSERT INTO hands (table_id, hand_no, started_at, dealer_seat, sb_seat, bb_seat)
		VALUES (?, ?, COALESCE(?, CURRENT_TIMESTAMP), ?, ?, ?)
	`, h.TableID, h.HandNo, h.StartedAt, h.DealerSeat, h.SBSeat, h.BBSeat)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (db *DB) EndHand(ctx context.Context, handID int64, resultJSON *string) error {
	_, err := db.ExecContext(ctx, `
		UPDATE hands
		SET ended_at = CURRENT_TIMESTAMP,
		    result_json = COALESCE(?, result_json)
		WHERE id = ? AND ended_at IS NULL
	`, resultJSON, handID)
	return err
}

func (db *DB) AddHandPlayers(ctx context.Context, players []HandPlayer) error {
	if len(players) == 0 {
		return nil
	}
	return db.withTx(ctx, func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx, `
			INSERT INTO hand_players (hand_id, player_id, seat, starting_stack, hole_cards_json)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT(hand_id, player_id) DO UPDATE SET
				seat            = excluded.seat,
				starting_stack  = excluded.starting_stack,
				hole_cards_json = excluded.hole_cards_json
		`)
		if err != nil {
			return err
		}
		defer stmt.Close()
		for _, hp := range players {
			if _, err := stmt.ExecContext(ctx, hp.HandID, hp.PlayerID, hp.Seat, hp.StartingStack, hp.HoleCardsJSON); err != nil {
				return err
			}
		}
		return nil
	})
}

// =========================
// ===== Actions API =======
// =========================

func (db *DB) AppendAction(ctx context.Context, a *Action) (int64, error) {
	res, err := db.ExecContext(ctx, `
		INSERT INTO actions (hand_id, ord, street, actor_seat, action, amount, is_allin, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, COALESCE(?, CURRENT_TIMESTAMP))
	`, a.HandID, a.Ord, a.Street, a.ActorSeat, a.Action, a.Amount, a.IsAllIn, a.CreatedAt)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (db *DB) AppendActionsBulk(ctx context.Context, actions []Action) error {
	if len(actions) == 0 {
		return nil
	}
	return db.withTx(ctx, func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx, `
			INSERT INTO actions (hand_id, ord, street, actor_seat, action, amount, is_allin, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, COALESCE(?, CURRENT_TIMESTAMP))
		`)
		if err != nil {
			return err
		}
		defer stmt.Close()
		for _, a := range actions {
			if _, err := stmt.ExecContext(ctx, a.HandID, a.Ord, a.Street, a.ActorSeat, a.Action, a.Amount, a.IsAllIn, a.CreatedAt); err != nil {
				return err
			}
		}
		return nil
	})
}

func (db *DB) ListActions(ctx context.Context, handID int64) ([]Action, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, hand_id, ord, street, actor_seat, action, amount, is_allin, created_at
		FROM actions
		WHERE hand_id = ?
		ORDER BY ord ASC
	`, handID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Action
	for rows.Next() {
		var a Action
		if err := rows.Scan(&a.ID, &a.HandID, &a.Ord, &a.Street, &a.ActorSeat, &a.Action, &a.Amount, &a.IsAllIn, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// =============================
// ===== Board Cards API =======
// =============================

func (db *DB) SetBoardCards(ctx context.Context, bc BoardCards) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO board_cards (hand_id, street, cards_json)
		VALUES (?, ?, ?)
		ON CONFLICT(hand_id, street) DO UPDATE SET
			cards_json = excluded.cards_json
	`, bc.HandID, bc.Street, bc.CardsJSON)
	return err
}

func (db *DB) GetBoardCards(ctx context.Context, handID int64) ([]BoardCards, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT hand_id, street, cards_json
		FROM board_cards
		WHERE hand_id = ?
		ORDER BY CASE street
			WHEN 'FLOP' THEN 1
			WHEN 'TURN' THEN 2
			WHEN 'RIVER' THEN 3
			ELSE 99 END
	`, handID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []BoardCards
	for rows.Next() {
		var bc BoardCards
		if err := rows.Scan(&bc.HandID, &bc.Street, &bc.CardsJSON); err != nil {
			return nil, err
		}
		out = append(out, bc)
	}
	return out, rows.Err()
}

// ==============================
// ===== Snapshot Cache API =====
// ==============================

func (db *DB) GetSnapshot(ctx context.Context, tableID string) (*Snapshot, error) {
	row := db.QueryRowContext(ctx, `
		SELECT table_id, snapshot_at, payload
		FROM table_snapshots
		WHERE table_id = ?
	`, tableID)
	var s Snapshot
	if err := row.Scan(&s.TableID, &s.SnapshotAt, &s.Payload); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("snapshot not found")
		}
		return nil, err
	}
	return &s, nil
}

func (db *DB) UpsertMatchCheckpoint(ctx context.Context, c MatchCheckpoint) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO match_checkpoints (table_id, updated_at, payload)
		VALUES (?, COALESCE(?, CURRENT_TIMESTAMP), ?)
		ON CONFLICT(table_id) DO UPDATE SET
			updated_at = excluded.updated_at,
			payload = excluded.payload
	`, c.TableID, c.UpdatedAt, c.Payload)
	return err
}

func (db *DB) GetMatchCheckpoint(ctx context.Context, tableID string) (*MatchCheckpoint, error) {
	row := db.QueryRowContext(ctx, `
		SELECT table_id, updated_at, payload
		FROM match_checkpoints
		WHERE table_id = ?
	`, tableID)
	var c MatchCheckpoint
	if err := row.Scan(&c.TableID, &c.UpdatedAt, &c.Payload); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("match checkpoint not found")
		}
		return nil, err
	}
	return &c, nil
}

func (db *DB) DeleteMatchCheckpoint(ctx context.Context, tableID string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM match_checkpoints WHERE table_id = ?`, tableID)
	return err
}

func (db *DB) DeleteSnapshot(ctx context.Context, tableID string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM table_snapshots WHERE table_id = ?`, tableID)
	return err
}

// ==============================
// ===== Auth API =====
// ==============================

// AuthUser represents a registered user
type AuthUser struct {
	Nickname      string
	UserID        string
	CreatedAt     time.Time
	LastLogin     sql.NullTime
	PayoutAddress sql.NullString
}

// UpsertAuthUser creates or updates a user registration
func (db *DB) UpsertAuthUser(ctx context.Context, nickname, userID string) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO auth_users (user_id, nickname)
		VALUES (?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			nickname = excluded.nickname
	`, userID, nickname)
	return err
}

// GetAuthUserByNickname retrieves a user by nickname
func (db *DB) GetAuthUserByNickname(ctx context.Context, nickname string) (*AuthUser, error) {
	row := db.QueryRowContext(ctx, `
		SELECT nickname, user_id, created_at, last_login, payout_address
		FROM auth_users
		WHERE nickname = ?
	`, nickname)
	var u AuthUser
	if err := row.Scan(&u.Nickname, &u.UserID, &u.CreatedAt, &u.LastLogin, &u.PayoutAddress); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}
	return &u, nil
}

// GetAuthUserByUserID retrieves a user by user ID
func (db *DB) GetAuthUserByUserID(ctx context.Context, userID string) (*AuthUser, error) {
	row := db.QueryRowContext(ctx, `
		SELECT nickname, user_id, created_at, last_login, payout_address
		FROM auth_users
		WHERE user_id = ?
	`, userID)
	var u AuthUser
	if err := row.Scan(&u.Nickname, &u.UserID, &u.CreatedAt, &u.LastLogin, &u.PayoutAddress); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}
	return &u, nil
}

// UpdateAuthUserLastLogin updates the last login timestamp
func (db *DB) UpdateAuthUserLastLogin(ctx context.Context, userID string) error {
	_, err := db.ExecContext(ctx, `
		UPDATE auth_users
		SET last_login = CURRENT_TIMESTAMP
		WHERE user_id = ?
	`, userID)
	return err
}

// UpdateAuthUserPayoutAddress updates the persisted payout address for a user.
func (db *DB) UpdateAuthUserPayoutAddress(ctx context.Context, userID, payoutAddress string) error {
	_, err := db.ExecContext(ctx, `
		UPDATE auth_users
		SET payout_address = ?
		WHERE user_id = ?
	`, payoutAddress, userID)
	return err
}

// ListAllAuthUsers returns all registered users (for loading on startup)
func (db *DB) ListAllAuthUsers(ctx context.Context) ([]AuthUser, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT nickname, user_id, created_at, last_login, payout_address
		FROM auth_users
		ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []AuthUser
	for rows.Next() {
		var u AuthUser
		if err := rows.Scan(&u.Nickname, &u.UserID, &u.CreatedAt, &u.LastLogin, &u.PayoutAddress); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}
