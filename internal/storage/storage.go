package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/julianstephens/daylit/internal/models"
)

type Settings struct {
	DayStart        string `json:"day_start"`
	DayEnd          string `json:"day_end"`
	DefaultBlockMin int    `json:"default_block_min"`
}

type Store struct {
	Version  int                       `json:"version"`
	Settings Settings                  `json:"settings"`
	Tasks    map[string]models.Task    `json:"tasks"`
	Plans    map[string]models.DayPlan `json:"plans"`
}

type Storage struct {
	path  string
	store *Store
}

func New(configPath string) (*Storage, error) {
	return &Storage{
		path: configPath,
	}, nil
}

func (s *Storage) Init() error {
	// Create config directory if it doesn't exist
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Check if file already exists
	if _, err := os.Stat(s.path); err == nil {
		return fmt.Errorf("storage already initialized at %s", s.path)
	}

	// Initialize with default settings
	s.store = &Store{
		Version: 1,
		Settings: Settings{
			DayStart:        "07:00",
			DayEnd:          "22:00",
			DefaultBlockMin: 30,
		},
		Tasks: make(map[string]models.Task),
		Plans: make(map[string]models.DayPlan),
	}

	return s.save()
}

func (s *Storage) Load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("storage not initialized, run 'daylit init' first")
		}
		return fmt.Errorf("failed to read storage: %w", err)
	}

	s.store = &Store{}
	if err := json.Unmarshal(data, s.store); err != nil {
		return fmt.Errorf("failed to parse storage: %w", err)
	}

	// Ensure maps are initialized
	if s.store.Tasks == nil {
		s.store.Tasks = make(map[string]models.Task)
	}
	if s.store.Plans == nil {
		s.store.Plans = make(map[string]models.DayPlan)
	}

	return nil
}

func (s *Storage) save() error {
	data, err := json.MarshalIndent(s.store, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize storage: %w", err)
	}

	if err := os.WriteFile(s.path, data, 0600); err != nil {
		return fmt.Errorf("failed to write storage: %w", err)
	}

	return nil
}

func (s *Storage) GetSettings() Settings {
	if s.store == nil {
		return Settings{}
	}
	return s.store.Settings
}

func (s *Storage) AddTask(task models.Task) error {
	if s.store == nil {
		return fmt.Errorf("storage not loaded")
	}

	s.store.Tasks[task.ID] = task
	return s.save()
}

func (s *Storage) GetTask(id string) (models.Task, error) {
	if s.store == nil {
		return models.Task{}, fmt.Errorf("storage not loaded")
	}

	task, ok := s.store.Tasks[id]
	if !ok {
		return models.Task{}, fmt.Errorf("task not found: %s", id)
	}

	return task, nil
}

func (s *Storage) GetAllTasks() []models.Task {
	if s.store == nil {
		return nil
	}

	tasks := make([]models.Task, 0, len(s.store.Tasks))
	for _, task := range s.store.Tasks {
		tasks = append(tasks, task)
	}

	return tasks
}

func (s *Storage) UpdateTask(task models.Task) error {
	if s.store == nil {
		return fmt.Errorf("storage not loaded")
	}

	if _, ok := s.store.Tasks[task.ID]; !ok {
		return fmt.Errorf("task not found: %s", task.ID)
	}

	s.store.Tasks[task.ID] = task
	return s.save()
}

func (s *Storage) SavePlan(plan models.DayPlan) error {
	if s.store == nil {
		return fmt.Errorf("storage not loaded")
	}

	s.store.Plans[plan.Date] = plan
	return s.save()
}

func (s *Storage) GetPlan(date string) (models.DayPlan, error) {
	if s.store == nil {
		return models.DayPlan{}, fmt.Errorf("storage not loaded")
	}

	plan, ok := s.store.Plans[date]
	if !ok {
		return models.DayPlan{}, fmt.Errorf("no plan found for date: %s", date)
	}

	return plan, nil
}

// GetConfigPath returns the path to the underlying configuration/storage file.
//
// Concurrency note:
//   - Storage is not safe for concurrent use by multiple goroutines without external
//     synchronization.
//   - Running multiple daylit processes that share the same storage/config path at the
//     same time is not supported and may lead to data loss or corruption.
func (s *Storage) GetConfigPath() string {
	return s.path
}
