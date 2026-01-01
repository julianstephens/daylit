package postgres

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/julianstephens/daylit/daylit-cli/internal/models"
)

// OT Settings

func (s *Store) GetOTSettings() (models.OTSettings, error) {
	rows, err := s.db.Query("SELECT key, value FROM settings WHERE key LIKE 'ot_%'")
	if err != nil {
		return models.OTSettings{}, err
	}
	defer rows.Close()

	settings := models.OTSettings{
		PromptOnEmpty:  true,
		StrictMode:     true,
		DefaultLogDays: 14,
	}

	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return models.OTSettings{}, err
		}
		switch key {
		case "ot_prompt_on_empty":
			settings.PromptOnEmpty = value == "true"
		case "ot_strict_mode":
			settings.StrictMode = value == "true"
		case "ot_default_log_days":
			if _, err := fmt.Sscanf(value, "%d", &settings.DefaultLogDays); err != nil {
				return models.OTSettings{}, fmt.Errorf("parsing ot_default_log_days: %w", err)
			}
		}
	}

	return settings, rows.Err()
}

func (s *Store) SaveOTSettings(settings models.OTSettings) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT INTO settings (key, value) VALUES ($1, $2) ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value")
	if err != nil {
		return err
	}
	defer stmt.Close()

	if _, err := stmt.Exec("ot_prompt_on_empty", fmt.Sprintf("%v", settings.PromptOnEmpty)); err != nil {
		return err
	}
	if _, err := stmt.Exec("ot_strict_mode", fmt.Sprintf("%v", settings.StrictMode)); err != nil {
		return err
	}
	if _, err := stmt.Exec("ot_default_log_days", fmt.Sprintf("%d", settings.DefaultLogDays)); err != nil {
		return err
	}

	return tx.Commit()
}

// OT Entries

func (s *Store) AddOTEntry(entry models.OTEntry) error {
	return s.UpdateOTEntry(entry)
}

func (s *Store) GetOTEntry(day string) (models.OTEntry, error) {
	row := s.db.QueryRow(`
		SELECT id, day, title, note, created_at, updated_at, deleted_at
		FROM ot_entries WHERE day = $1 AND deleted_at IS NULL`, day)

	var e models.OTEntry
	var createdAt, updatedAt string
	var deletedAt sql.NullString

	err := row.Scan(&e.ID, &e.Day, &e.Title, &e.Note, &createdAt, &updatedAt, &deletedAt)
	if err != nil {
		return models.OTEntry{}, err
	}

	e.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return models.OTEntry{}, fmt.Errorf("failed to parse created_at: %w", err)
	}
	e.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		return models.OTEntry{}, fmt.Errorf("failed to parse updated_at: %w", err)
	}
	if deletedAt.Valid {
		t, err := time.Parse(time.RFC3339, deletedAt.String)
		if err != nil {
			return models.OTEntry{}, fmt.Errorf("failed to parse deleted_at: %w", err)
		}
		e.DeletedAt = &t
	}

	return e, nil
}

func (s *Store) GetOTEntries(startDay, endDay string, includeDeleted bool) ([]models.OTEntry, error) {
	query := `
		SELECT id, day, title, note, created_at, updated_at, deleted_at
		FROM ot_entries WHERE day >= $1 AND day <= $2`
	if !includeDeleted {
		query += " AND deleted_at IS NULL"
	}
	query += " ORDER BY day DESC"

	rows, err := s.db.Query(query, startDay, endDay)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []models.OTEntry
	for rows.Next() {
		var e models.OTEntry
		var createdAt, updatedAt string
		var deletedAt sql.NullString

		err := rows.Scan(&e.ID, &e.Day, &e.Title, &e.Note, &createdAt, &updatedAt, &deletedAt)
		if err != nil {
			return nil, err
		}

		e.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse created_at for entry %s: %w", e.ID, err)
		}
		e.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse updated_at for entry %s: %w", e.ID, err)
		}
		if deletedAt.Valid {
			t, err := time.Parse(time.RFC3339, deletedAt.String)
			if err != nil {
				return nil, fmt.Errorf("failed to parse deleted_at for entry %s: %w", e.ID, err)
			}
			e.DeletedAt = &t
		}

		entries = append(entries, e)
	}

	return entries, nil
}

func (s *Store) GetAllOTEntries() ([]models.OTEntry, error) {
	rows, err := s.db.Query(`
		SELECT id, day, title, note, created_at, updated_at, deleted_at
		FROM ot_entries
		ORDER BY day DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []models.OTEntry
	for rows.Next() {
		var e models.OTEntry
		var createdAt, updatedAt string
		var deletedAt sql.NullString

		err := rows.Scan(&e.ID, &e.Day, &e.Title, &e.Note, &createdAt, &updatedAt, &deletedAt)
		if err != nil {
			return nil, err
		}

		e.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse created_at for entry %s: %w", e.ID, err)
		}
		e.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse updated_at for entry %s: %w", e.ID, err)
		}
		if deletedAt.Valid {
			t, err := time.Parse(time.RFC3339, deletedAt.String)
			if err != nil {
				return nil, fmt.Errorf("failed to parse deleted_at for entry %s: %w", e.ID, err)
			}
			e.DeletedAt = &t
		}

		entries = append(entries, e)
	}

	return entries, nil
}

func (s *Store) UpdateOTEntry(entry models.OTEntry) error {
	var deletedAt sql.NullString
	if entry.DeletedAt != nil {
		deletedAt = sql.NullString{String: entry.DeletedAt.Format(time.RFC3339), Valid: true}
	}

	_, err := s.db.Exec(`
		INSERT INTO ot_entries (id, day, title, note, created_at, updated_at, deleted_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT(day) DO UPDATE SET
			title = EXCLUDED.title,
			note = EXCLUDED.note,
			updated_at = EXCLUDED.updated_at,
			deleted_at = EXCLUDED.deleted_at`,
		entry.ID, entry.Day, entry.Title, entry.Note,
		entry.CreatedAt.Format(time.RFC3339), entry.UpdatedAt.Format(time.RFC3339), deletedAt)

	return err
}

func (s *Store) DeleteOTEntry(day string) error {
	result, err := s.db.Exec(`
		UPDATE ot_entries SET deleted_at = $1 WHERE day = $2 AND deleted_at IS NULL`,
		time.Now().Format(time.RFC3339), day)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("OT entry not found or already deleted")
	}

	return nil
}

func (s *Store) RestoreOTEntry(day string) error {
	result, err := s.db.Exec(`
		UPDATE ot_entries SET deleted_at = NULL WHERE day = $1 AND deleted_at IS NOT NULL`,
		day)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("OT entry not found or not deleted")
	}

	return nil
}
