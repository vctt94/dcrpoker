CREATE TABLE IF NOT EXISTS settlement_escrows (
    match_id   TEXT NOT NULL,
    seat       INTEGER NOT NULL,
    escrow_id  TEXT NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (match_id, seat)
);

CREATE INDEX IF NOT EXISTS idx_settlement_escrows_match_id
    ON settlement_escrows(match_id);

CREATE TABLE IF NOT EXISTS referee_escrows (
    escrow_id   TEXT PRIMARY KEY,
    updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    payload     BLOB NOT NULL
);

CREATE TABLE IF NOT EXISTS referee_branch_gammas (
    match_id    TEXT NOT NULL,
    branch      INTEGER NOT NULL,
    gamma_hex   TEXT NOT NULL,
    updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (match_id, branch)
);

CREATE INDEX IF NOT EXISTS idx_referee_branch_gammas_match_id
    ON referee_branch_gammas(match_id);

CREATE TABLE IF NOT EXISTS referee_presigns (
    match_id    TEXT NOT NULL,
    branch      INTEGER NOT NULL,
    input_id    TEXT NOT NULL,
    updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    payload     BLOB NOT NULL,
    PRIMARY KEY (match_id, branch, input_id)
);

CREATE INDEX IF NOT EXISTS idx_referee_presigns_match_id
    ON referee_presigns(match_id);

CREATE TABLE IF NOT EXISTS pending_settlements (
    match_id     TEXT PRIMARY KEY,
    table_id     TEXT NOT NULL,
    winner_id    TEXT NOT NULL,
    winner_seat  INTEGER NOT NULL,
    updated_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
