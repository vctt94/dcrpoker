CREATE TABLE IF NOT EXISTS players (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Authentication: user_id is primary key, nickname is just for UI display (not unique)
CREATE TABLE IF NOT EXISTS auth_users (
    user_id     TEXT PRIMARY KEY,
    nickname    TEXT NOT NULL,
    payout_address TEXT,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_login  TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_auth_users_user_id ON auth_users(user_id);

-- Active sessions (optional, for session management)
CREATE TABLE IF NOT EXISTS auth_sessions (
    token       TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL,
    nickname    TEXT NOT NULL,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at  TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES auth_users(user_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_auth_sessions_user_id ON auth_sessions(user_id);

-- Poker tables (configuration; not per-hand/transient state)
CREATE TABLE IF NOT EXISTS tables (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    table_source    TEXT NOT NULL DEFAULT 'user',
    buy_in          INTEGER NOT NULL,
    min_players     INTEGER NOT NULL,
    max_players     INTEGER NOT NULL,
    small_blind     INTEGER NOT NULL,
    big_blind       INTEGER NOT NULL,
    starting_chips  INTEGER NOT NULL,
    timebank_ms     INTEGER NOT NULL DEFAULT 0,
    autostart_ms    INTEGER NOT NULL DEFAULT 0,
    auto_advance_ms INTEGER NOT NULL DEFAULT 1000,
    blind_increase_interval_sec INTEGER NOT NULL DEFAULT 0,
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_tables_created_at ON tables(created_at);
CREATE INDEX IF NOT EXISTS idx_tables_source ON tables(table_source);

-- Membership at a table (active row has left_at NULL)
CREATE TABLE IF NOT EXISTS table_participants (
    table_id   TEXT NOT NULL,
    player_id  TEXT NOT NULL,
    seat       INTEGER NOT NULL,
    joined_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    left_at    TIMESTAMP,
    ready      BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (table_id, player_id),
    FOREIGN KEY (table_id) REFERENCES tables(id) ON DELETE CASCADE,
    FOREIGN KEY (player_id) REFERENCES players(id)
);

CREATE INDEX IF NOT EXISTS idx_participants_active ON table_participants(table_id, left_at);

-- Table-level buy-ins / cash-outs (chips ledger inside the table)
CREATE TABLE IF NOT EXISTS table_buyins (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    table_id   TEXT NOT NULL,
    player_id  TEXT NOT NULL,
    amount     INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (table_id) REFERENCES tables(id) ON DELETE CASCADE,
    FOREIGN KEY (player_id) REFERENCES players(id)
);

CREATE INDEX IF NOT EXISTS idx_buyins_player ON table_buyins(player_id, table_id);

CREATE TABLE IF NOT EXISTS table_cashouts (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    table_id   TEXT NOT NULL,
    player_id  TEXT NOT NULL,
    amount     INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (table_id) REFERENCES tables(id) ON DELETE CASCADE,
    FOREIGN KEY (player_id) REFERENCES players(id)
);

-- Hand history (canonical, reducible state)
CREATE TABLE IF NOT EXISTS hands (
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
);

CREATE INDEX IF NOT EXISTS idx_hands_table ON hands(table_id, hand_no);

CREATE TABLE IF NOT EXISTS hand_players (
    hand_id         INTEGER NOT NULL,
    player_id       TEXT NOT NULL,
    seat            INTEGER NOT NULL,
    starting_stack  INTEGER NOT NULL,
    hole_cards_json TEXT,
    PRIMARY KEY (hand_id, player_id),
    FOREIGN KEY (hand_id) REFERENCES hands(id) ON DELETE CASCADE,
    FOREIGN KEY (player_id) REFERENCES players(id)
);

CREATE INDEX IF NOT EXISTS idx_hand_players_seat ON hand_players(hand_id, seat);

-- Ordered betting/actions (deterministic replay)
CREATE TABLE IF NOT EXISTS actions (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    hand_id     INTEGER NOT NULL,
    ord         INTEGER NOT NULL,
    street      TEXT NOT NULL,
    actor_seat  INTEGER NOT NULL,
    action      TEXT NOT NULL,
    amount      INTEGER NOT NULL DEFAULT 0,
    is_allin    BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (hand_id) REFERENCES hands(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_actions_order ON actions(hand_id, ord);

-- Optional: community cards per street (audit/export)
CREATE TABLE IF NOT EXISTS board_cards (
    hand_id    INTEGER NOT NULL,
    street     TEXT NOT NULL,
    cards_json TEXT NOT NULL,
    PRIMARY KEY (hand_id, street),
    FOREIGN KEY (hand_id) REFERENCES hands(id) ON DELETE CASCADE
);

-- Optional: fast-restore snapshot cache (opaque; NOT canonical)
CREATE TABLE IF NOT EXISTS table_snapshots (
    table_id    TEXT PRIMARY KEY,
    snapshot_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    payload     BLOB NOT NULL,
    FOREIGN KEY (table_id) REFERENCES tables(id) ON DELETE CASCADE
);

-- Stable between-hand checkpoints used for graceful shutdown/restore.
CREATE TABLE IF NOT EXISTS match_checkpoints (
    table_id    TEXT PRIMARY KEY,
    updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    payload     BLOB NOT NULL,
    FOREIGN KEY (table_id) REFERENCES tables(id) ON DELETE CASCADE
);
