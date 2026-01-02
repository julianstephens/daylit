package postgres

import (
	"fmt"

	"github.com/julianstephens/daylit/daylit-cli/internal/models"
	"github.com/julianstephens/daylit/daylit-cli/internal/storage"
)

func (s *Store) GetSettings() (storage.Settings, error) {
	rows, err := s.db.Query("SELECT key, value FROM settings")
	if err != nil {
		return storage.Settings{}, err
	}
	defer rows.Close()

	settingsMap := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return storage.Settings{}, err
		}
		settingsMap[key] = value
	}

	if len(settingsMap) == 0 {
		return storage.Settings{}, fmt.Errorf("settings not found")
	}

	settings, err := models.MapToSettings(settingsMap)
	if err != nil {
		return storage.Settings{}, err
	}

	return settings, nil
}

func (s *Store) SaveSettings(settings storage.Settings) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// PostgreSQL uses INSERT ... ON CONFLICT for upsert
	stmt, err := tx.Prepare(`
		INSERT INTO settings (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	settingsMap := models.SettingsToMap(settings)
	for key, value := range settingsMap {
		if _, err := stmt.Exec(key, value); err != nil {
			return err
		}
	}

	return tx.Commit()
}
