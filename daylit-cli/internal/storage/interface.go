package storage

import "github.com/julianstephens/daylit/daylit-cli/internal/models"

type Provider interface {
	// Lifecycle
	Init() error
	Load() error
	Close() error

	// Settings
	GetSettings() (Settings, error)
	SaveSettings(Settings) error

	// Tasks
	AddTask(models.Task) error
	GetTask(id string) (models.Task, error)
	GetAllTasks() ([]models.Task, error)
	GetAllTasksIncludingDeleted() ([]models.Task, error)
	UpdateTask(models.Task) error
	DeleteTask(id string) error
	RestoreTask(id string) error

	// Plans
	SavePlan(models.DayPlan) error
	GetPlan(date string) (models.DayPlan, error)
	// GetPlanRevision returns the specified revision of the plan for the given date.
	// The date parameter identifies the day, and revision selects the particular
	// stored revision. It returns an error if the requested revision does not exist
	// or cannot be retrieved.
	GetPlanRevision(date string, revision int) (models.DayPlan, error)
	// GetLatestPlanRevision returns the latest non-deleted revision of the plan
	// for the given date. It returns an error if no such revision exists or the
	// latest revision cannot be retrieved.
	GetLatestPlanRevision(date string) (models.DayPlan, error)
	DeletePlan(date string) error
	RestorePlan(date string) error
	// UpdateSlotNotificationTimestamp updates the notification timestamp for a specific slot
	UpdateSlotNotificationTimestamp(date string, revision int, startTime string, taskID string, notificationType string, timestamp string) error

	// Habits
	AddHabit(models.Habit) error
	GetHabit(id string) (models.Habit, error)
	GetHabitByName(name string) (models.Habit, error)
	GetAllHabits(includeArchived, includeDeleted bool) ([]models.Habit, error)
	UpdateHabit(models.Habit) error
	ArchiveHabit(id string) error
	UnarchiveHabit(id string) error
	DeleteHabit(id string) error
	RestoreHabit(id string) error

	// Habit Entries
	AddHabitEntry(models.HabitEntry) error
	GetHabitEntry(habitID, day string) (models.HabitEntry, error)
	GetHabitEntriesForDay(day string) ([]models.HabitEntry, error)
	GetHabitEntriesForHabit(habitID string, startDay, endDay string) ([]models.HabitEntry, error)
	UpdateHabitEntry(models.HabitEntry) error
	DeleteHabitEntry(id string) error
	RestoreHabitEntry(id string) error

	// OT Settings
	GetOTSettings() (models.OTSettings, error)
	SaveOTSettings(models.OTSettings) error

	// OT Entries
	AddOTEntry(models.OTEntry) error
	GetOTEntry(day string) (models.OTEntry, error)
	GetOTEntries(startDay, endDay string, includeDeleted bool) ([]models.OTEntry, error)
	UpdateOTEntry(models.OTEntry) error
	DeleteOTEntry(day string) error
	RestoreOTEntry(day string) error

	// Bulk Retrieval for Migration
	GetAllPlans() ([]models.DayPlan, error)
	GetAllHabitEntries() ([]models.HabitEntry, error)
	GetAllOTEntries() ([]models.OTEntry, error)

	// Utils
	GetConfigPath() string
}
