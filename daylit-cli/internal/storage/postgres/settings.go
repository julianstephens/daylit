package postgres

import (
	"fmt"

	"github.com/julianstephens/daylit/daylit-cli/internal/storage"
)

func (s *Store) GetSettings() (storage.Settings, error) {
	rows, err := s.db.Query("SELECT key, value FROM settings")
	if err != nil {
		return storage.Settings{}, err
	}
	defer rows.Close()

	settings := storage.Settings{}
	count := 0
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return storage.Settings{}, err
		}
		switch key {
		case "day_start":
			settings.DayStart = value
		case "day_end":
			settings.DayEnd = value
		case "default_block_min":
			if _, err := fmt.Sscanf(value, "%d", &settings.DefaultBlockMin); err != nil {
				return storage.Settings{}, fmt.Errorf("parsing default_block_min: %w", err)
			}
		case "notifications_enabled":
			settings.NotificationsEnabled = value == "true"
		case "notify_block_start":
			settings.NotifyBlockStart = value == "true"
		case "notify_block_end":
			settings.NotifyBlockEnd = value == "true"
		case "block_start_offset_min":
			if _, err := fmt.Sscanf(value, "%d", &settings.BlockStartOffsetMin); err != nil {
				return storage.Settings{}, fmt.Errorf("parsing block_start_offset_min: %w", err)
			}
		case "block_end_offset_min":
			if _, err := fmt.Sscanf(value, "%d", &settings.BlockEndOffsetMin); err != nil {
				return storage.Settings{}, fmt.Errorf("parsing block_end_offset_min: %w", err)
			}
		case "notification_grace_period_min":
			if _, err := fmt.Sscanf(value, "%d", &settings.NotificationGracePeriodMin); err != nil {
				return storage.Settings{}, fmt.Errorf("parsing notification_grace_period_min: %w", err)
			}
		}
		count++
	}

	if count == 0 {
		return storage.Settings{}, fmt.Errorf("settings not found")
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

	if _, err := stmt.Exec("day_start", settings.DayStart); err != nil {
		return err
	}
	if _, err := stmt.Exec("day_end", settings.DayEnd); err != nil {
		return err
	}
	if _, err := stmt.Exec("default_block_min", fmt.Sprintf("%d", settings.DefaultBlockMin)); err != nil {
		return err
	}
	if _, err := stmt.Exec("notifications_enabled", fmt.Sprintf("%v", settings.NotificationsEnabled)); err != nil {
		return err
	}
	if _, err := stmt.Exec("notify_block_start", fmt.Sprintf("%v", settings.NotifyBlockStart)); err != nil {
		return err
	}
	if _, err := stmt.Exec("notify_block_end", fmt.Sprintf("%v", settings.NotifyBlockEnd)); err != nil {
		return err
	}
	if _, err := stmt.Exec("block_start_offset_min", fmt.Sprintf("%d", settings.BlockStartOffsetMin)); err != nil {
		return err
	}
	if _, err := stmt.Exec("block_end_offset_min", fmt.Sprintf("%d", settings.BlockEndOffsetMin)); err != nil {
		return err
	}
	if _, err := stmt.Exec("notification_grace_period_min", fmt.Sprintf("%d", settings.NotificationGracePeriodMin)); err != nil {
		return err
	}

	return tx.Commit()
}
