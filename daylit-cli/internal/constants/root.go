package constants

import "time"

// ConflictType represents the type of validation conflict
type ConflictType string

// SessionState represents the current state of the TUI application
type SessionState int

// TaskKind represents the kind of task
type TaskKind string

// RecurrenceType represents the type of recurrence for tasks or alerts
type RecurrenceType string

// EnergyBand represents the energy band of a task
type EnergyBand string

const (
	AppName            = "daylit"
	DefaultKeyringUser = "database-connection"
	DefaultConfigPath  = "~/.config/daylit/daylit.db"
	Version            = "v0.5.0"

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
	TaskKindAppointment TaskKind = "appointment"
	TaskKindFlexible    TaskKind = "flexible"

	// Recurrence Type constants
	RecurrenceDaily       RecurrenceType = "daily"
	RecurrenceWeekly      RecurrenceType = "weekly"
	RecurrenceNDays       RecurrenceType = "n_days"
	RecurrenceAdHoc       RecurrenceType = "ad_hoc"
	RecurrenceMonthlyDate RecurrenceType = "monthly_date" // e.g., 15th of every month
	RecurrenceMonthlyDay  RecurrenceType = "monthly_day"  // e.g., last Friday of the month
	RecurrenceYearly      RecurrenceType = "yearly"       // e.g., every year on January 1st
	RecurrenceWeekdays    RecurrenceType = "weekdays"     // every weekday (Mon-Fri)

	// Energy Band constants
	EnergyLow    EnergyBand = "low"
	EnergyMedium EnergyBand = "medium"
	EnergyHigh   EnergyBand = "high"

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
