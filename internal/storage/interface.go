package storage

import "github.com/julianstephens/daylit/internal/models"

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

	// Utils
	GetConfigPath() string
}
