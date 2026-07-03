package storage

import (
	"database/sql"
	"errors"
	"time"
)

// Pairing is a short-lived token a device redeems to pair.
type Pairing struct {
	Token     string
	DeviceID  string
	CreatedAt time.Time
	ExpiresAt time.Time
	Used      bool
}

// PairingRepo persists pending pairing tokens.
type PairingRepo struct{ db *sql.DB }

// Create stores a new pairing token.
func (r *PairingRepo) Create(p Pairing) error {
	_, err := r.db.Exec(`
		INSERT INTO pairings (token, device_id, created_at, expires_at, used)
		VALUES (?, ?, ?, ?, ?)`,
		p.Token, p.DeviceID, p.CreatedAt.Unix(), p.ExpiresAt.Unix(), boolToInt(p.Used))
	return err
}

// Get returns a pairing by token, or ErrNotFound.
func (r *PairingRepo) Get(token string) (Pairing, error) {
	var (
		p                   Pairing
		created, expires    int64
		used                int
	)
	err := r.db.QueryRow(
		`SELECT token, device_id, created_at, expires_at, used FROM pairings WHERE token=?`, token).
		Scan(&p.Token, &p.DeviceID, &created, &expires, &used)
	if errors.Is(err, sql.ErrNoRows) {
		return Pairing{}, ErrNotFound
	}
	if err != nil {
		return Pairing{}, err
	}
	p.CreatedAt = time.Unix(created, 0)
	p.ExpiresAt = time.Unix(expires, 0)
	p.Used = used != 0
	return p, nil
}

// MarkUsed flags a token as redeemed by deviceID.
func (r *PairingRepo) MarkUsed(token, deviceID string) error {
	_, err := r.db.Exec(`UPDATE pairings SET used=1, device_id=? WHERE token=?`, deviceID, token)
	return err
}

// DeleteExpired removes tokens that expired before now. Returns rows removed.
func (r *PairingRepo) DeleteExpired(now time.Time) (int64, error) {
	res, err := r.db.Exec(`DELETE FROM pairings WHERE expires_at < ?`, now.Unix())
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
