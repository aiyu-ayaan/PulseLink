package storage

import (
	"database/sql"
	"time"
)

// LogEntry is a persisted application log line surfaced in the Logs page.
type LogEntry struct {
	ID      int64
	TS      time.Time
	Level   string
	Message string
}

// LogRepo persists a bounded history of log lines.
type LogRepo struct{ db *sql.DB }

// Add appends a log line.
func (r *LogRepo) Add(level, message string) error {
	_, err := r.db.Exec(`INSERT INTO logs (ts, level, message) VALUES (?, ?, ?)`,
		time.Now().Unix(), level, message)
	return err
}

// Recent returns up to limit newest entries, newest first.
func (r *LogRepo) Recent(limit int) ([]LogEntry, error) {
	rows, err := r.db.Query(
		`SELECT id, ts, level, message FROM logs ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []LogEntry
	for rows.Next() {
		var (
			e  LogEntry
			ts int64
		)
		if err := rows.Scan(&e.ID, &ts, &e.Level, &e.Message); err != nil {
			return nil, err
		}
		e.TS = time.Unix(ts, 0)
		out = append(out, e)
	}
	return out, rows.Err()
}

// Prune keeps only the newest keep rows.
func (r *LogRepo) Prune(keep int) error {
	_, err := r.db.Exec(
		`DELETE FROM logs WHERE id NOT IN (SELECT id FROM logs ORDER BY id DESC LIMIT ?)`, keep)
	return err
}
