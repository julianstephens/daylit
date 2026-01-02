package models

import (
	"fmt"

	"github.com/julianstephens/daylit/daylit-cli/internal/constants"
)

// MapToSettings converts a map of key-value pairs to a Settings struct.
func MapToSettings(data map[string]string) (Settings, error) {
	settings := Settings{}

	for key, value := range data {
		switch key {
		case constants.SettingDayStart:
			settings.DayStart = value
		case constants.SettingDayEnd:
			settings.DayEnd = value
		case constants.SettingDefaultBlockMin:
			if _, err := fmt.Sscanf(value, "%d", &settings.DefaultBlockMin); err != nil {
				return Settings{}, fmt.Errorf("parsing default_block_min: %w", err)
			}
		case constants.SettingNotificationsEnabled:
			settings.NotificationsEnabled = value == "true"
		case constants.SettingNotifyBlockStart:
			settings.NotifyBlockStart = value == "true"
		case constants.SettingNotifyBlockEnd:
			settings.NotifyBlockEnd = value == "true"
		case constants.SettingBlockStartOffsetMin:
			if _, err := fmt.Sscanf(value, "%d", &settings.BlockStartOffsetMin); err != nil {
				return Settings{}, fmt.Errorf("parsing block_start_offset_min: %w", err)
			}
		case constants.SettingBlockEndOffsetMin:
			if _, err := fmt.Sscanf(value, "%d", &settings.BlockEndOffsetMin); err != nil {
				return Settings{}, fmt.Errorf("parsing block_end_offset_min: %w", err)
			}
		case constants.SettingNotificationGracePeriodMin:
			if _, err := fmt.Sscanf(value, "%d", &settings.NotificationGracePeriodMin); err != nil {
				return Settings{}, fmt.Errorf("parsing notification_grace_period_min: %w", err)
			}
		}
	}
	return settings, nil
}

// SettingsToMap converts a Settings struct to a map of key-value pairs.
func SettingsToMap(settings Settings) map[string]string {
	return map[string]string{
		constants.SettingDayStart:                   settings.DayStart,
		constants.SettingDayEnd:                     settings.DayEnd,
		constants.SettingDefaultBlockMin:            fmt.Sprintf("%d", settings.DefaultBlockMin),
		constants.SettingNotificationsEnabled:       fmt.Sprintf("%v", settings.NotificationsEnabled),
		constants.SettingNotifyBlockStart:           fmt.Sprintf("%v", settings.NotifyBlockStart),
		constants.SettingNotifyBlockEnd:             fmt.Sprintf("%v", settings.NotifyBlockEnd),
		constants.SettingBlockStartOffsetMin:        fmt.Sprintf("%d", settings.BlockStartOffsetMin),
		constants.SettingBlockEndOffsetMin:          fmt.Sprintf("%d", settings.BlockEndOffsetMin),
		constants.SettingNotificationGracePeriodMin: fmt.Sprintf("%d", settings.NotificationGracePeriodMin),
	}
}

// ApplyDefaultSettings applies default values to missing settings.
func ApplyDefaultSettings(settings *Settings) {
	if settings.DayStart == "" {
		settings.DayStart = constants.DefaultDayStart
	}
	if settings.DayEnd == "" {
		settings.DayEnd = constants.DefaultDayEnd
	}
	if settings.DefaultBlockMin == 0 {
		settings.DefaultBlockMin = constants.DefaultBlockMin
	}

	if settings.BlockStartOffsetMin == 0 {
		settings.BlockStartOffsetMin = constants.DefaultBlockStartOffsetMin
	}
	if settings.BlockEndOffsetMin == 0 {
		settings.BlockEndOffsetMin = constants.DefaultBlockEndOffsetMin
	}
	if settings.NotificationGracePeriodMin == 0 {
		settings.NotificationGracePeriodMin = constants.DefaultNotificationGracePeriodMin
	}
}
