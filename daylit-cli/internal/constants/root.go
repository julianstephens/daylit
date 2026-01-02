package constants

import "time"

// ConflictType represents the type of validation conflict
type ConflictType string

// SessionState represents the current state of the TUI application
type SessionState int

const (
	AppName            = "daylit"
	DefaultKeyringUser = "database-connection"
	DefaultConfigPath  = "~/.config/daylit/daylit.db"

	// DateFormat is the standard date format used throughout the application (YYYY-MM-DD)
	DateFormat = "2006-01-02"

	// TimeFormat is the standard time format used throughout the application (HH:MM)
	TimeFormat = "15:04"

	// Backup constants
	MaxBackups       = 14
	BackupDirName    = "backups"
	BackupFilePrefix = "daylit-"
	BackupFileSuffix = ".db"

	// Notify constants
	NotifyMaxRetries = 3
	NotifyRetryDelay = 100 * time.Millisecond

	// Slot Status constants
	SlotStatusPlanned  = "planned"
	SlotStatusAccepted = "accepted"
	SlotStatusDone     = "done"
	SlotStatusSkipped  = "skipped"

	// Task Kind constants
	TaskKindAppointment = "appointment"
	TaskKindFlexible    = "flexible"

	// Recurrence Type constants
	RecurrenceDaily  = "daily"
	RecurrenceWeekly = "weekly"
	RecurrenceNDays  = "n_days"
	RecurrenceAdHoc  = "ad_hoc"

	// Energy Band constants
	EnergyLow    = "low"
	EnergyMedium = "medium"
	EnergyHigh   = "high"

	// Notification constants
	NotifierLockfileName   = "daylit-tray.lock"
	NotificationDurationMs = 5000
	TrayAppIdentifier      = "com.daylit.daylit-tray"

	// NumMainTabs is the number of main navigation tabs in the TUI
	NumMainTabs = 7 // Now, Plan, Tasks, Habits, OT, Alerts, Settings

	// Conflict Types
	ConflictOverlappingFixedTasks ConflictType = "overlapping_fixed_tasks"
	ConflictOverlappingSlots      ConflictType = "overlapping_slots"
	ConflictExceedsWakingWindow   ConflictType = "exceeds_waking_window"
	ConflictOvercommitted         ConflictType = "overcommitted"
	ConflictMissingTaskID         ConflictType = "missing_task_id"
	ConflictDuplicateTaskName     ConflictType = "duplicate_task_name"
	ConflictInvalidDateTime       ConflictType = "invalid_datetime"

	// TUI Session States
	StateNow SessionState = iota
	StatePlan
	StateTasks
	StateHabits
	StateOT
	StateAlerts
	StateSettings
	StateFeedback
	StateEditing
	StateConfirmDelete
	StateConfirmRestore
	StateConfirmOverwrite
	StateConfirmArchive
	StateAddHabit
	StateAddAlert
	StateEditOT
	StateEditSettings
)
