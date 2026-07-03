package storage

import "database/sql"

// Automation is a stored, user-defined action (config is opaque JSON).
type Automation struct {
	ID      string
	Name    string
	Config  string // JSON blob interpreted by the automation service
	Enabled bool
}

// AutomationRepo persists automation configurations.
type AutomationRepo struct{ db *sql.DB }

// Upsert inserts or updates an automation.
func (r *AutomationRepo) Upsert(a Automation) error {
	_, err := r.db.Exec(`
		INSERT INTO automations (id, name, config, enabled) VALUES (?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name=excluded.name, config=excluded.config, enabled=excluded.enabled`,
		a.ID, a.Name, a.Config, boolToInt(a.Enabled))
	return err
}

// List returns all automations.
func (r *AutomationRepo) List() ([]Automation, error) {
	rows, err := r.db.Query(`SELECT id, name, config, enabled FROM automations ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Automation
	for rows.Next() {
		var (
			a       Automation
			enabled int
		)
		if err := rows.Scan(&a.ID, &a.Name, &a.Config, &enabled); err != nil {
			return nil, err
		}
		a.Enabled = enabled != 0
		out = append(out, a)
	}
	return out, rows.Err()
}

// Delete removes an automation by ID.
func (r *AutomationRepo) Delete(id string) error {
	_, err := r.db.Exec(`DELETE FROM automations WHERE id=?`, id)
	return err
}
