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
	UpdateTask(models.Task) error
	DeleteTask(id string) error
	RestoreTask(id string) error

	// Plans
	SavePlan(models.DayPlan) error
	GetPlan(date string) (models.DayPlan, error)
	DeletePlan(date string) error
	RestorePlan(date string) error

	// Utils
	GetConfigPath() string
}
