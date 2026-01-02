package constants

const (
	// General Settings
	SettingDayStart                   = "day_start"
	SettingDayEnd                     = "day_end"
	SettingDefaultBlockMin            = "default_block_min"
	SettingNotificationsEnabled       = "notifications_enabled"
	SettingNotifyBlockStart           = "notify_block_start"
	SettingNotifyBlockEnd             = "notify_block_end"
	SettingBlockStartOffsetMin        = "block_start_offset_min"
	SettingBlockEndOffsetMin          = "block_end_offset_min"
	SettingNotificationGracePeriodMin = "notification_grace_period_min"
	SettingTimezone                   = "timezone"

	// OT Settings
	SettingOTPromptOnEmpty  = "ot_prompt_on_empty"
	SettingOTStrictMode     = "ot_strict_mode"
	SettingOTDefaultLogDays = "ot_default_log_days"

	// Default Settings Values
	DefaultDayStart                   = "07:00"
	DefaultDayEnd                     = "22:00"
	DefaultBlockMin                   = 30
	DefaultNotificationsEnabled       = true
	DefaultNotifyBlockStart           = true
	DefaultNotifyBlockEnd             = true
	DefaultBlockStartOffsetMin        = 5
	DefaultBlockEndOffsetMin          = 5
	DefaultNotificationGracePeriodMin = 10
	DefaultTimezone                   = "Local" // Use system local timezone by default
)
