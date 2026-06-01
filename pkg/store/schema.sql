CREATE TABLE IF NOT EXISTS users (
    id          TEXT PRIMARY KEY,
    email       TEXT NOT NULL UNIQUE,
    password    TEXT NOT NULL,
    created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS sessions (
    token       TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS neighborhoods (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    join_code   TEXT NOT NULL UNIQUE,
    created_by  TEXT NOT NULL REFERENCES users(id),
    created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS neighborhood_members (
    neighborhood_id TEXT NOT NULL REFERENCES neighborhoods(id) ON DELETE CASCADE,
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    joined_at       TEXT NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (neighborhood_id, user_id)
);

CREATE TABLE IF NOT EXISTS listings (
    id              TEXT PRIMARY KEY,
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    neighborhood_id TEXT NOT NULL REFERENCES neighborhoods(id) ON DELETE CASCADE,
    title           TEXT NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    category        TEXT NOT NULL DEFAULT 'other',
    min_tier        INTEGER NOT NULL DEFAULT 0,
    created_at      TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS trust_edges (
    owner_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    neighbor_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tier        INTEGER NOT NULL DEFAULT 1,
    PRIMARY KEY (owner_id, neighbor_id)
);
