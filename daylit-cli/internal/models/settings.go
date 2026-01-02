package models

// Settings represents application-wide settings
type Settings struct {
	DayStart                   string `json:"day_start"`                     // the time the day starts, e.g. "08:00"
	DayEnd                     string `json:"day_end"`                       // the time the day ends, e.g. "18:00"
	DefaultBlockMin            int    `json:"default_block_min"`             // the default block duration in minutes
	NotificationsEnabled       bool   `json:"notifications_enabled"`         // whether notifications are enabled
	NotifyBlockStart           bool   `json:"notify_block_start"`            // whether to notify at the start of a block
	NotifyBlockEnd             bool   `json:"notify_block_end"`              // whether to notify at the end of a block
	BlockStartOffsetMin        int    `json:"block_start_offset_min"`        // the offset in minutes for block start notifications
	BlockEndOffsetMin          int    `json:"block_end_offset_min"`          // the offset in minutes for block end notifications
	NotificationGracePeriodMin int    `json:"notification_grace_period_min"` // grace period for late notifications in minutes
	Timezone                   string `json:"timezone"`                      // IANA timezone name (e.g. "America/New_York", "Europe/London", or "Local" for system timezone)
}
