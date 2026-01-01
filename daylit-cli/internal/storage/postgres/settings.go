package postgres

import (
	"fmt"

	"github.com/julianstephens/daylit/daylit-cli/internal/constants"
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
		case constants.SettingDayStart:
			settings.DayStart = value
		case constants.SettingDayEnd:
			settings.DayEnd = value
		case constants.SettingDefaultBlockMin:
			if _, err := fmt.Sscanf(value, "%d", &settings.DefaultBlockMin); err != nil {
				return storage.Settings{}, fmt.Errorf("parsing default_block_min: %w", err)
			}
		case constants.SettingNotificationsEnabled:
			settings.NotificationsEnabled = value == "true"
		case constants.SettingNotifyBlockStart:
			settings.NotifyBlockStart = value == "true"
		case constants.SettingNotifyBlockEnd:
			settings.NotifyBlockEnd = value == "true"
		case constants.SettingBlockStartOffsetMin:
			if _, err := fmt.Sscanf(value, "%d", &settings.BlockStartOffsetMin); err != nil {
				return storage.Settings{}, fmt.Errorf("parsing block_start_offset_min: %w", err)
			}
		case constants.SettingBlockEndOffsetMin:
			if _, err := fmt.Sscanf(value, "%d", &settings.BlockEndOffsetMin); err != nil {
				return storage.Settings{}, fmt.Errorf("parsing block_end_offset_min: %w", err)
			}
		case constants.SettingNotificationGracePeriodMin:
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

	if _, err := stmt.Exec(constants.SettingDayStart, settings.DayStart); err != nil {
		return err
	}
	if _, err := stmt.Exec(constants.SettingDayEnd, settings.DayEnd); err != nil {
		return err
	}
	if _, err := stmt.Exec(constants.SettingDefaultBlockMin, fmt.Sprintf("%d", settings.DefaultBlockMin)); err != nil {
		return err
	}
	if _, err := stmt.Exec(constants.SettingNotificationsEnabled, fmt.Sprintf("%v", settings.NotificationsEnabled)); err != nil {
		return err
	}
	if _, err := stmt.Exec(constants.SettingNotifyBlockStart, fmt.Sprintf("%v", settings.NotifyBlockStart)); err != nil {
		return err
	}
	if _, err := stmt.Exec(constants.SettingNotifyBlockEnd, fmt.Sprintf("%v", settings.NotifyBlockEnd)); err != nil {
		return err
	}
	if _, err := stmt.Exec(constants.SettingBlockStartOffsetMin, fmt.Sprintf("%d", settings.BlockStartOffsetMin)); err != nil {
		return err
	}
	if _, err := stmt.Exec(constants.SettingBlockEndOffsetMin, fmt.Sprintf("%d", settings.BlockEndOffsetMin)); err != nil {
		return err
	}
	if _, err := stmt.Exec(constants.SettingNotificationGracePeriodMin, fmt.Sprintf("%d", settings.NotificationGracePeriodMin)); err != nil {
		return err
	}

	return tx.Commit()
}
