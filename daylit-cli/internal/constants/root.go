package constants

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

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

// ConfirmationMsg is a message to trigger a confirmation dialog
type ConfirmationMsg struct {
	Message string
	Action  func() tea.Cmd
}

// FeedbackMsg is a message to trigger feedback
type FeedbackMsg struct{}

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
	NotifyMaxRetries       = 3
	NotifyRetryDelay       = 100 * time.Millisecond
	NotifierLockfileName   = "daylit-notifier.lock"
	NotificationDurationMs = 5000
	TrayAppIdentifier      = "com.julianstephens.daylit"

	// Slot Status constants
	SlotStatusPlanned  = "planned"
	SlotStatusAccepted = "accepted"
	SlotStatusDone     = "done"
	SlotStatusSkipped  = "skipped"

	// Task Kind constants
	TaskKindAppointment TaskKind = "appointment"
	TaskKindFlexible    TaskKind = "flexible"

	// Recurrence constants
	RecurrenceAdHoc       RecurrenceType = "ad-hoc"
	RecurrenceDaily       RecurrenceType = "daily"
	RecurrenceWeekly      RecurrenceType = "weekly"
	RecurrenceNDays       RecurrenceType = "n-days"
	RecurrenceMonthlyDate RecurrenceType = "monthly-date"
	RecurrenceMonthlyDay  RecurrenceType = "monthly-day"
	RecurrenceYearly      RecurrenceType = "yearly"
	RecurrenceWeekdays    RecurrenceType = "weekdays"

	// Energy Band constants
	EnergyHigh   EnergyBand = "high"
	EnergyMedium EnergyBand = "medium"
	EnergyLow    EnergyBand = "low"

	// Conflict Types
	ConflictDuplicateTaskName     ConflictType = "duplicate_task_name"
	ConflictInvalidDateTime       ConflictType = "invalid_date_time"
	ConflictOverlappingFixedTasks ConflictType = "overlapping_fixed_tasks"
	ConflictMissingTaskID         ConflictType = "missing_task_id"
	ConflictOverlappingSlots      ConflictType = "overlapping_slots"
	ConflictExceedsWakingWindow   ConflictType = "exceeds_waking_window"
	ConflictOvercommitted         ConflictType = "overcommitted"

	// Session States
	StateTasks SessionState = iota
	StatePlan
	StateNow
	StateHabits
	StateOT
	StateAlerts
	StateSettings
	StateEditing
	StateAddHabit
	StateAddAlert
	StateEditOT
	StateEditSettings
	StateFeedback
	StateConfirmation
	StateConfirmDelete
	StateConfirmRestore
	StateConfirmOverwrite
	StateConfirmArchive
)
