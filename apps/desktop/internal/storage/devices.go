package storage

import (
	"database/sql"
	"errors"
	"time"
)

// ErrNotFound is returned when a lookup matches no row.
var ErrNotFound = errors.New("not found")

// Device is a paired Android client.
type Device struct {
	ID           string
	Name         string
	PublicKey    string
	Trusted      bool
	PairedAt     time.Time
	LastSeen     time.Time
	Capabilities []string
}

// DeviceRepo persists devices and their capabilities.
type DeviceRepo struct{ db *sql.DB }

// Upsert inserts or updates a device (capabilities are replaced wholesale).
func (r *DeviceRepo) Upsert(d Device) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO devices (id, name, public_key, trusted, paired_at, last_seen)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name=excluded.name,
			public_key=excluded.public_key,
			trusted=excluded.trusted,
			last_seen=excluded.last_seen`,
		d.ID, d.Name, d.PublicKey, boolToInt(d.Trusted),
		d.PairedAt.Unix(), d.LastSeen.Unix())
	if err != nil {
		return err
	}

	if _, err := tx.Exec(`DELETE FROM device_capabilities WHERE device_id=?`, d.ID); err != nil {
		return err
	}
	for _, c := range d.Capabilities {
		if _, err := tx.Exec(
			`INSERT INTO device_capabilities (device_id, capability) VALUES (?, ?)`,
			d.ID, c); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// Get returns one device by ID, or ErrNotFound.
func (r *DeviceRepo) Get(id string) (Device, error) {
	row := r.db.QueryRow(
		`SELECT id, name, public_key, trusted, paired_at, last_seen FROM devices WHERE id=?`, id)
	d, err := scanDevice(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Device{}, ErrNotFound
	}
	if err != nil {
		return Device{}, err
	}
	d.Capabilities, err = r.capabilities(id)
	return d, err
}

// List returns all devices ordered by name.
func (r *DeviceRepo) List() ([]Device, error) {
	rows, err := r.db.Query(
		`SELECT id, name, public_key, trusted, paired_at, last_seen FROM devices ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Device
	for rows.Next() {
		d, err := scanDevice(rows)
		if err != nil {
			return nil, err
		}
		d.Capabilities, err = r.capabilities(d.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// Delete removes a device and its capabilities (cascade).
func (r *DeviceRepo) Delete(id string) error {
	_, err := r.db.Exec(`DELETE FROM devices WHERE id=?`, id)
	return err
}

// SetTrusted updates only the trusted flag.
func (r *DeviceRepo) SetTrusted(id string, trusted bool) error {
	res, err := r.db.Exec(`UPDATE devices SET trusted=? WHERE id=?`, boolToInt(trusted), id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// TouchLastSeen records that the device was seen at t.
func (r *DeviceRepo) TouchLastSeen(id string, t time.Time) error {
	_, err := r.db.Exec(`UPDATE devices SET last_seen=? WHERE id=?`, t.Unix(), id)
	return err
}

func (r *DeviceRepo) capabilities(id string) ([]string, error) {
	rows, err := r.db.Query(
		`SELECT capability FROM device_capabilities WHERE device_id=? ORDER BY capability`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var caps []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		caps = append(caps, c)
	}
	return caps, rows.Err()
}

// scanner is satisfied by both *sql.Row and *sql.Rows.
type scanner interface{ Scan(...any) error }

func scanDevice(s scanner) (Device, error) {
	var (
		d              Device
		trusted        int
		pairedAt, seen int64
	)
	if err := s.Scan(&d.ID, &d.Name, &d.PublicKey, &trusted, &pairedAt, &seen); err != nil {
		return Device{}, err
	}
	d.Trusted = trusted != 0
	d.PairedAt = time.Unix(pairedAt, 0)
	d.LastSeen = time.Unix(seen, 0)
	return d, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
