package sqlite

import (
	"fmt"

	"github.com/julianstephens/daylit/daylit-cli/internal/models"
)

func (s *Store) GetSettings() (models.Settings, error) {
	rows, err := s.db.Query("SELECT key, value FROM settings")
	if err != nil {
		return models.Settings{}, err
	}
	defer rows.Close()

	settingsMap := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return models.Settings{}, err
		}
		settingsMap[key] = value
	}

	if len(settingsMap) == 0 {
		return models.Settings{}, fmt.Errorf("settings not found")
	}

	settings, err := models.MapToSettings(settingsMap)
	if err != nil {
		return models.Settings{}, err
	}

	return settings, nil
}

func (s *Store) SaveSettings(settings models.Settings) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)")
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
