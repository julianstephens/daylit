package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

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
	if !ok || task.DeletedAt != nil {
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
		if task.DeletedAt == nil {
			tasks = append(tasks, task)
		}
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

	task, ok := s.store.Tasks[id]
	if !ok {
		return fmt.Errorf("task not found: %s", id)
	}

	// Soft delete: set deleted_at timestamp
	now := time.Now().UTC().Format(time.RFC3339)
	task.DeletedAt = &now
	s.store.Tasks[id] = task
	return s.save()
}

func (s *JSONStore) RestoreTask(id string) error {
	if s.store == nil {
		return fmt.Errorf("storage not loaded")
	}

	task, ok := s.store.Tasks[id]
	if !ok {
		return fmt.Errorf("task not found: %s", id)
	}

	// Only allow restoring tasks that are currently soft-deleted
	if task.DeletedAt == nil {
		return fmt.Errorf("cannot restore a task that is not deleted: %s", id)
	}

	// Restore by clearing deleted_at
	task.DeletedAt = nil
	s.store.Tasks[id] = task
	return s.save()
}

func (s *JSONStore) SavePlan(plan models.DayPlan) error {
	if s.store == nil {
		return fmt.Errorf("storage not loaded")
	}

	// Check if plan is deleted - forbid adding slots to deleted plans
	if existingPlan, ok := s.store.Plans[plan.Date]; ok && existingPlan.DeletedAt != nil {
		return fmt.Errorf("cannot save slots to a deleted plan: %s", plan.Date)
	}

	// Filter out soft-deleted slots to keep behavior consistent with SQLite
	// which hard-deletes existing slots before inserting
	if len(plan.Slots) > 0 {
		filteredSlots := make([]models.Slot, 0, len(plan.Slots))
		for _, slot := range plan.Slots {
			if slot.DeletedAt == nil {
				filteredSlots = append(filteredSlots, slot)
			}
		}
		plan.Slots = filteredSlots
	}

	s.store.Plans[plan.Date] = plan
	return s.save()
}

func (s *JSONStore) GetPlan(date string) (models.DayPlan, error) {
	if s.store == nil {
		return models.DayPlan{}, fmt.Errorf("storage not loaded")
	}

	plan, ok := s.store.Plans[date]
	if !ok || plan.DeletedAt != nil {
		return models.DayPlan{}, fmt.Errorf("no plan found for date: %s", date)
	}

	// Filter out soft-deleted slots before returning the plan
	if len(plan.Slots) > 0 {
		filteredSlots := make([]models.Slot, 0, len(plan.Slots))
		for _, slot := range plan.Slots {
			if slot.DeletedAt == nil {
				filteredSlots = append(filteredSlots, slot)
			}
		}
		plan.Slots = filteredSlots
	}

	return plan, nil
}

func (s *JSONStore) DeletePlan(date string) error {
	if s.store == nil {
		return fmt.Errorf("storage not loaded")
	}

	plan, ok := s.store.Plans[date]
	if !ok {
		return fmt.Errorf("plan not found for date: %s", date)
	}

	// Soft delete: set deleted_at timestamp for plan and all its slots
	now := time.Now().UTC().Format(time.RFC3339)
	plan.DeletedAt = &now
	
	// Soft delete all slots in the plan
	for i := range plan.Slots {
		if plan.Slots[i].DeletedAt == nil {
			plan.Slots[i].DeletedAt = &now
		}
	}
	
	s.store.Plans[date] = plan
	return s.save()
}

func (s *JSONStore) RestorePlan(date string) error {
	if s.store == nil {
		return fmt.Errorf("storage not loaded")
	}

	plan, ok := s.store.Plans[date]
	if !ok {
		return fmt.Errorf("plan not found for date: %s", date)
	}

	// Only allow restoring plans that are currently soft-deleted
	if plan.DeletedAt == nil {
		return fmt.Errorf("plan is not deleted for date: %s", date)
	}

	// Restore by clearing deleted_at on the plan and on slots that were
	// deleted as part of the same DeletePlan operation. This avoids
	// resurrecting slots that were individually soft-deleted earlier.
	planDeletedAt := plan.DeletedAt
	plan.DeletedAt = nil

	if planDeletedAt != nil {
		for i := range plan.Slots {
			if plan.Slots[i].DeletedAt != nil && *plan.Slots[i].DeletedAt == *planDeletedAt {
				plan.Slots[i].DeletedAt = nil
			}
		}
	}
	
	s.store.Plans[date] = plan
	return s.save()
}

func (s *JSONStore) GetConfigPath() string {
	return s.path
}
