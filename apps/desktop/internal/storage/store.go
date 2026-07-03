// Package storage is the SQLite data layer.
//
// It uses the pure-Go modernc.org/sqlite driver so the backend builds without
// CGO (important on the toolchains this project targets). Each entity has its
// own repository type; Store owns the *sql.DB and the schema.
package storage

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite" // registers the "sqlite" driver
)

// schema is applied on every Open. Statements are idempotent (IF NOT EXISTS),
// which is enough while the schema is additive. A migration table can replace
// this if columns ever need to change.
const schema = `
CREATE TABLE IF NOT EXISTS devices (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    public_key  TEXT NOT NULL DEFAULT '',
    trusted     INTEGER NOT NULL DEFAULT 0,
    paired_at   INTEGER NOT NULL,
    last_seen   INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS device_capabilities (
    device_id   TEXT NOT NULL,
    capability  TEXT NOT NULL,
    PRIMARY KEY (device_id, capability),
    FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS pairings (
    token      TEXT PRIMARY KEY,
    device_id  TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    expires_at INTEGER NOT NULL,
    used       INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS automations (
    id       TEXT PRIMARY KEY,
    name     TEXT NOT NULL,
    config   TEXT NOT NULL DEFAULT '{}',
    enabled  INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS logs (
    id      INTEGER PRIMARY KEY AUTOINCREMENT,
    ts      INTEGER NOT NULL,
    level   TEXT NOT NULL,
    message TEXT NOT NULL
);
`

// Store holds the database handle and repositories.
type Store struct {
	db *sql.DB

	Devices     *DeviceRepo
	Settings    *SettingsRepo
	Pairings    *PairingRepo
	Automations *AutomationRepo
	Logs        *LogRepo
}

// Open opens (creating if needed) the SQLite database at path and applies the
// schema. Use ":memory:" for tests.
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	// modernc/sqlite is safe for concurrent use but a single writer avoids
	// "database is locked"; the backend is low-write, so keep one connection.
	db.SetMaxOpenConns(1)
	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		db.Close()
		return nil, err
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	s := &Store{db: db}
	s.Devices = &DeviceRepo{db: db}
	s.Settings = &SettingsRepo{db: db}
	s.Pairings = &PairingRepo{db: db}
	s.Automations = &AutomationRepo{db: db}
	s.Logs = &LogRepo{db: db}
	return s, nil
}

// Close closes the underlying database.
func (s *Store) Close() error { return s.db.Close() }
