package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/julianstephens/daylit/internal/models"
)

type Store struct {
	Version  int                       `json:"version"`
	Settings Settings                  `json:"settings"`
	Tasks    map[string]models.Task    `json:"tasks"`
	Plans    map[string]models.DayPlan `json:"plans"`
}

type JSONStore struct {
	path  string
	store *Store
}

func NewJSONStore(configPath string) *JSONStore {
	return &JSONStore{
		path: configPath,
	}
}

func (s *JSONStore) Init() error {
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

func (s *JSONStore) Load() error {
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

func (s *JSONStore) Close() error {
	return nil
}

func (s *JSONStore) save() error {
	data, err := json.MarshalIndent(s.store, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize storage: %w", err)
	}

	if err := os.WriteFile(s.path, data, 0600); err != nil {
		return fmt.Errorf("failed to write storage: %w", err)
	}

	return nil
}

func (s *JSONStore) GetSettings() (Settings, error) {
	if s.store == nil {
		return Settings{}, fmt.Errorf("storage not loaded")
	}
	return s.store.Settings, nil
}

func (s *JSONStore) SaveSettings(settings Settings) error {
	if s.store == nil {
		return fmt.Errorf("storage not loaded")
	}
	s.store.Settings = settings
	return s.save()
}

func (s *JSONStore) AddTask(task models.Task) error {
	if s.store == nil {
		return fmt.Errorf("storage not loaded")
	}

	s.store.Tasks[task.ID] = task
	return s.save()
}

func (s *JSONStore) GetTask(id string) (models.Task, error) {
	if s.store == nil {
		return models.Task{}, fmt.Errorf("storage not loaded")
	}

	task, ok := s.store.Tasks[id]
	if !ok {
		return models.Task{}, fmt.Errorf("task not found: %s", id)
	}

	return task, nil
}

func (s *JSONStore) GetAllTasks() ([]models.Task, error) {
	if s.store == nil {
		return nil, fmt.Errorf("storage not loaded")
	}

	tasks := make([]models.Task, 0, len(s.store.Tasks))
	for _, task := range s.store.Tasks {
		tasks = append(tasks, task)
	}

	return tasks, nil
}

func (s *JSONStore) UpdateTask(task models.Task) error {
	if s.store == nil {
		return fmt.Errorf("storage not loaded")
	}

	if _, ok := s.store.Tasks[task.ID]; !ok {
		return fmt.Errorf("task not found: %s", task.ID)
	}

	s.store.Tasks[task.ID] = task
	return s.save()
}

func (s *JSONStore) DeleteTask(id string) error {
	if s.store == nil {
		return fmt.Errorf("storage not loaded")
	}

	if _, ok := s.store.Tasks[id]; !ok {
		return fmt.Errorf("task not found: %s", id)
	}

	delete(s.store.Tasks, id)
	return s.save()
}

func (s *JSONStore) SavePlan(plan models.DayPlan) error {
	if s.store == nil {
		return fmt.Errorf("storage not loaded")
	}

	s.store.Plans[plan.Date] = plan
	return s.save()
}

func (s *JSONStore) GetPlan(date string) (models.DayPlan, error) {
	if s.store == nil {
		return models.DayPlan{}, fmt.Errorf("storage not loaded")
	}

	plan, ok := s.store.Plans[date]
	if !ok {
		return models.DayPlan{}, fmt.Errorf("no plan found for date: %s", date)
	}

	return plan, nil
}

func (s *JSONStore) GetConfigPath() string {
	return s.path
}
