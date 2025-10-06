package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/vctt94/pokerbisonrelay/pkg/poker"
)

// DB wraps sql.DB.
type DB struct {
	*sql.DB
}

// NewDB opens (or creates) a SQLite database, enables FKs, and creates schema.
func NewDB(path string) (*DB, error) {
	d, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	// Ensure FKs are enforced at the connection-level.
	if _, err := d.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
		_ = d.Close()
		return nil, fmt.Errorf("enable foreign_keys: %w", err)
	}
	if err := createSchema(d); err != nil {
		_ = d.Close()
		return nil, err
	}
	return &DB{d}, nil
}

// Close closes the underlying DB.
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
// ======== SCHEMA =========
// =========================

func createSchema(db *sql.DB) error {
	stmts := []string{
		// Players & wallet transactions
		`CREATE TABLE IF NOT EXISTS players (
			id         TEXT PRIMARY KEY,
			name       TEXT NOT NULL,
			balance    INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS transactions (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			player_id   TEXT NOT NULL,
			amount      INTEGER NOT NULL,
			type        TEXT NOT NULL,
			description TEXT,
			created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (player_id) REFERENCES players(id)
		);`,

		// Poker tables (configuration; *not* per-hand/transient state)
		`CREATE TABLE IF NOT EXISTS tables (
			id              TEXT PRIMARY KEY,
			host_id         TEXT NOT NULL,
			buy_in          INTEGER NOT NULL,
			min_players     INTEGER NOT NULL,
			max_players     INTEGER NOT NULL,
			small_blind     INTEGER NOT NULL,
			big_blind       INTEGER NOT NULL,
			min_balance     INTEGER NOT NULL,
			starting_chips  INTEGER NOT NULL,
			timebank_ms     INTEGER NOT NULL DEFAULT 0,
			autostart_ms    INTEGER NOT NULL DEFAULT 0,
			created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (host_id) REFERENCES players(id)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_tables_created_at ON tables(created_at);`,

		// Membership at a table (active row has left_at NULL).
		`CREATE TABLE IF NOT EXISTS table_participants (
			table_id   TEXT NOT NULL,
			player_id  TEXT NOT NULL,
			seat       INTEGER NOT NULL,
			joined_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			left_at    TIMESTAMP,
			ready      BOOLEAN NOT NULL DEFAULT FALSE,
			PRIMARY KEY (table_id, player_id),
			UNIQUE (table_id, seat),
			FOREIGN KEY (table_id) REFERENCES tables(id) ON DELETE CASCADE,
			FOREIGN KEY (player_id) REFERENCES players(id)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_participants_active ON table_participants(table_id, left_at);`,

		// Table-level buy-ins / cash-outs (chips ledger inside the table)
		`CREATE TABLE IF NOT EXISTS table_buyins (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			table_id   TEXT NOT NULL,
			player_id  TEXT NOT NULL,
			amount     INTEGER NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (table_id) REFERENCES tables(id) ON DELETE CASCADE,
			FOREIGN KEY (player_id) REFERENCES players(id)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_buyins_player ON table_buyins(player_id, table_id);`,

		`CREATE TABLE IF NOT EXISTS table_cashouts (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			table_id   TEXT NOT NULL,
			player_id  TEXT NOT NULL,
			amount     INTEGER NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (table_id) REFERENCES tables(id) ON DELETE CASCADE,
			FOREIGN KEY (player_id) REFERENCES players(id)
		);`,

		// Hand history (canonical, reducible state)
		`CREATE TABLE IF NOT EXISTS hands (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			table_id     TEXT NOT NULL,
			hand_no      INTEGER NOT NULL,
			started_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			ended_at     TIMESTAMP,
			dealer_seat  INTEGER NOT NULL,
			sb_seat      INTEGER NOT NULL,
			bb_seat      INTEGER NOT NULL,
			result_json  TEXT,
			UNIQUE (table_id, hand_no),
			FOREIGN KEY (table_id) REFERENCES tables(id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_hands_table ON hands(table_id, hand_no);`,

		`CREATE TABLE IF NOT EXISTS hand_players (
			hand_id         INTEGER NOT NULL,
			player_id       TEXT NOT NULL,
			seat            INTEGER NOT NULL,
			starting_stack  INTEGER NOT NULL,
			hole_cards_json TEXT,
			PRIMARY KEY (hand_id, player_id),
			FOREIGN KEY (hand_id) REFERENCES hands(id) ON DELETE CASCADE,
			FOREIGN KEY (player_id) REFERENCES players(id)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_hand_players_seat ON hand_players(hand_id, seat);`,

		// Ordered betting/actions (deterministic replay)
		`CREATE TABLE IF NOT EXISTS actions (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			hand_id     INTEGER NOT NULL,
			ord         INTEGER NOT NULL,
			street      TEXT NOT NULL,         -- "PREFLOP","FLOP","TURN","RIVER","SHOWDOWN"
			actor_seat  INTEGER NOT NULL,
			action      TEXT NOT NULL,         -- "POST_SB","POST_BB","CHECK","CALL","BET","RAISE","FOLD","ALLIN","REFUND"
			amount      INTEGER NOT NULL DEFAULT 0,
			is_allin    BOOLEAN NOT NULL DEFAULT FALSE,
			created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (hand_id) REFERENCES hands(id) ON DELETE CASCADE
		);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_actions_order ON actions(hand_id, ord);`,

		// Optional: community cards per street (audit/export)
		`CREATE TABLE IF NOT EXISTS board_cards (
			hand_id    INTEGER NOT NULL,
			street     TEXT NOT NULL,          -- "FLOP","TURN","RIVER"
			cards_json TEXT NOT NULL,
			PRIMARY KEY (hand_id, street),
			FOREIGN KEY (hand_id) REFERENCES hands(id) ON DELETE CASCADE
		);`,

		// Optional: fast-restore snapshot cache (opaque; NOT canonical)
		`CREATE TABLE IF NOT EXISTS table_snapshots (
			table_id    TEXT PRIMARY KEY,
			snapshot_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			payload     BLOB NOT NULL,
			FOREIGN KEY (table_id) REFERENCES tables(id) ON DELETE CASCADE
		);`,
	}

	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return fmt.Errorf("schema exec failed: %w\nstmt: %s", err, s)
		}
	}
	return nil
}

// =========================
// ======== MODELS =========
// =========================

type Player struct {
	ID        string
	Name      string
	Balance   int64
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
	ID            string
	HostID        string
	BuyIn         int64
	MinPlayers    int
	MaxPlayers    int
	SmallBlind    int64
	BigBlind      int64
	MinBalance    int64
	StartingChips int64
	TimebankMS    int64
	AutoStartMS   int64
	CreatedAt     time.Time
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
		SELECT id, name, balance, created_at
		FROM players
		WHERE id = ?
	`, id)
	var p Player
	if err := row.Scan(&p.ID, &p.Name, &p.Balance, &p.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("player not found: %s", id)
		}
		return nil, err
	}
	return &p, nil
}

func (db *DB) GetPlayerBalance(ctx context.Context, id string) (int64, error) {
	row := db.QueryRowContext(ctx, `SELECT balance FROM players WHERE id = ?`, id)
	var bal int64
	if err := row.Scan(&bal); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, fmt.Errorf("player not found: %s", id)
		}
		return 0, err
	}
	return bal, nil
}

// Update wallet balance and record a ledger transaction atomically.
func (db *DB) UpdatePlayerBalance(ctx context.Context, playerID string, amount int64, typ, description string) error {
	return db.withTx(ctx, func(tx *sql.Tx) error {
		// Ensure player exists (default name = id if new).
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO players (id, name, balance)
			VALUES (?, ?, 0)
			ON CONFLICT(id) DO NOTHING
		`, playerID, playerID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE players SET balance = balance + ? WHERE id = ?
		`, amount, playerID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO transactions (player_id, amount, type, description)
			VALUES (?, ?, ?, ?)
		`, playerID, amount, typ, description); err != nil {
			return err
		}
		return nil
	})
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
	_, err := db.ExecContext(ctx, `
		INSERT INTO tables (
			id, host_id, buy_in, min_players, max_players, small_blind, big_blind,
			min_balance, starting_chips, timebank_ms, autostart_ms, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, COALESCE(?, CURRENT_TIMESTAMP))
		ON CONFLICT(id) DO UPDATE SET
			host_id        = excluded.host_id,
			buy_in         = excluded.buy_in,
			min_players    = excluded.min_players,
			max_players    = excluded.max_players,
			small_blind    = excluded.small_blind,
			big_blind      = excluded.big_blind,
			min_balance    = excluded.min_balance,
			starting_chips = excluded.starting_chips,
			timebank_ms    = excluded.timebank_ms,
			autostart_ms   = excluded.autostart_ms
	`, t.ID, t.HostID, t.BuyIn, t.MinPlayers, t.MaxPlayers, t.SmallBlind, t.BigBlind,
		t.MinBalance, t.StartingChips, t.TimeBank, t.AutoStartDelay, time.Now())
	return err
}

func (db *DB) GetTable(ctx context.Context, id string) (*Table, error) {
	row := db.QueryRowContext(ctx, `
		SELECT id, host_id, buy_in, min_players, max_players, small_blind, big_blind,
		       min_balance, starting_chips, timebank_ms, autostart_ms, created_at
		FROM tables WHERE id = ?
	`, id)
	var t Table
	if err := row.Scan(&t.ID, &t.HostID, &t.BuyIn, &t.MinPlayers, &t.MaxPlayers, &t.SmallBlind, &t.BigBlind,
		&t.MinBalance, &t.StartingChips, &t.TimebankMS, &t.AutoStartMS, &t.CreatedAt); err != nil {
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
	// left_at MUST be NULL for an active seat; UNIQUE(table_id, seat) enforces occupancy.
	_, err := db.ExecContext(ctx, `
		INSERT INTO table_participants (table_id, player_id, seat, joined_at, left_at, ready)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP, NULL, FALSE)
		ON CONFLICT(table_id, player_id) DO UPDATE SET
			seat = excluded.seat,
			left_at = NULL
	`, tableID, playerID, seat)
	return err
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

func (db *DB) DeleteSnapshot(ctx context.Context, tableID string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM table_snapshots WHERE table_id = ?`, tableID)
	return err
}
