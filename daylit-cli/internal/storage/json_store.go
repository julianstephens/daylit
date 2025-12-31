package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/julianstephens/daylit/daylit-cli/internal/models"
)

type Store struct {
	Version  int                               `json:"version"`
	Settings Settings                          `json:"settings"`
	Tasks    map[string]models.Task            `json:"tasks"`
	Plans    map[string]map[int]models.DayPlan `json:"plans"` // date -> revision -> plan
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
		Plans: make(map[string]map[int]models.DayPlan),
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
		s.store.Plans = make(map[string]map[int]models.DayPlan)
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

func (s *JSONStore) GetAllTasksIncludingDeleted() ([]models.Task, error) {
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

	// Prevent bypassing the delete/restore workflow by ensuring plans cannot be saved
	// with DeletedAt manually set. Use DeletePlan/RestorePlan for managing deletion state.
	if plan.DeletedAt != nil {
		return fmt.Errorf("cannot save a plan with deleted_at set; use DeletePlan to soft-delete or RestorePlan to restore")
	}

	// Determine the revision number if not set
	if plan.Revision == 0 {
		revisions, ok := s.store.Plans[plan.Date]
		if !ok || len(revisions) == 0 {
			// No existing plan, start with revision 1
			plan.Revision = 1
		} else {
			// Find the latest non-deleted revision
			latestRev := 0
			var latestPlan models.DayPlan
			for rev, p := range revisions {
				if p.DeletedAt == nil && rev > latestRev {
					latestRev = rev
					latestPlan = p
				}
			}

			if latestRev == 0 {
				// All plans are deleted, create new revision 1
				plan.Revision = 1
			} else if latestPlan.AcceptedAt != nil {
				// Latest plan is accepted - must create new revision
				plan.Revision = latestRev + 1
			} else {
				// Latest plan is not accepted - can overwrite
				plan.Revision = latestRev
			}
		}
	} else {
		// If revision is manually set, validate that it doesn't overwrite an accepted plan
		// unless it's the same plan being updated (same accepted_at timestamp)
		if revisions, ok := s.store.Plans[plan.Date]; ok {
			if existingPlan, ok := revisions[plan.Revision]; ok && existingPlan.DeletedAt == nil && existingPlan.AcceptedAt != nil {
				// Check if we're updating the same plan (same accepted_at timestamp)
				planAcceptedAtStr := ""
				if plan.AcceptedAt != nil {
					planAcceptedAtStr = *plan.AcceptedAt
				}
				existingAcceptedAtStr := ""
				if existingPlan.AcceptedAt != nil {
					existingAcceptedAtStr = *existingPlan.AcceptedAt
				}
				if planAcceptedAtStr != existingAcceptedAtStr {
					return fmt.Errorf("cannot overwrite accepted plan: %s revision %d", plan.Date, plan.Revision)
				}
			}
		}
	}

	// Check if the specific revision is deleted
	if revisions, ok := s.store.Plans[plan.Date]; ok {
		if existingPlan, ok := revisions[plan.Revision]; ok && existingPlan.DeletedAt != nil {
			return fmt.Errorf("cannot save slots to a deleted plan: %s revision %d", plan.Date, plan.Revision)
		}
	}

	// Filter out soft-deleted slots so that SavePlan only persists non-deleted slots,
	// matching the SQLite store's behavior where existing non-soft-deleted slots are
	// hard-deleted before new ones are inserted during the save operation.
	if len(plan.Slots) > 0 {
		filteredSlots := make([]models.Slot, 0, len(plan.Slots))
		for _, slot := range plan.Slots {
			if slot.DeletedAt == nil {
				filteredSlots = append(filteredSlots, slot)
			}
		}
		plan.Slots = filteredSlots
	}

	// Initialize the date map if it doesn't exist
	if _, ok := s.store.Plans[plan.Date]; !ok {
		s.store.Plans[plan.Date] = make(map[int]models.DayPlan)
	}

	s.store.Plans[plan.Date][plan.Revision] = plan
	return s.save()
}

func (s *JSONStore) GetPlan(date string) (models.DayPlan, error) {
	return s.GetLatestPlanRevision(date)
}

func (s *JSONStore) GetLatestPlanRevision(date string) (models.DayPlan, error) {
	if s.store == nil {
		return models.DayPlan{}, fmt.Errorf("storage not loaded")
	}

	revisions, ok := s.store.Plans[date]
	if !ok || len(revisions) == 0 {
		return models.DayPlan{}, fmt.Errorf("no plan found for date: %s", date)
	}

	// Find the latest non-deleted revision
	latestRev := 0
	var latestPlan models.DayPlan
	for rev, p := range revisions {
		if p.DeletedAt == nil && rev > latestRev {
			latestRev = rev
			latestPlan = p
		}
	}

	if latestRev == 0 {
		return models.DayPlan{}, fmt.Errorf("no active plan found for date: %s", date)
	}

	// Filter out soft-deleted slots before returning the plan
	if len(latestPlan.Slots) > 0 {
		filteredSlots := make([]models.Slot, 0, len(latestPlan.Slots))
		for _, slot := range latestPlan.Slots {
			if slot.DeletedAt == nil {
				filteredSlots = append(filteredSlots, slot)
			}
		}
		latestPlan.Slots = filteredSlots
	}

	return latestPlan, nil
}

func (s *JSONStore) GetPlanRevision(date string, revision int) (models.DayPlan, error) {
	if s.store == nil {
		return models.DayPlan{}, fmt.Errorf("storage not loaded")
	}

	revisions, ok := s.store.Plans[date]
	if !ok {
		return models.DayPlan{}, fmt.Errorf("no plan found for date: %s", date)
	}

	plan, ok := revisions[revision]
	if !ok {
		return models.DayPlan{}, fmt.Errorf("no plan found for date: %s revision: %d", date, revision)
	}

	if plan.DeletedAt != nil {
		return models.DayPlan{}, fmt.Errorf("plan for date %s revision %d has been deleted; use 'daylit restore plan %s' to restore it", date, revision, date)
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

	revisions, ok := s.store.Plans[date]
	if !ok || len(revisions) == 0 {
		return fmt.Errorf("no plan found for date: %s", date)
	}

	// Check if there are any non-deleted revisions
	hasActiveRevision := false
	for _, p := range revisions {
		if p.DeletedAt == nil {
			hasActiveRevision = true
			break
		}
	}

	if !hasActiveRevision {
		return fmt.Errorf("no active plans found for date: %s", date)
	}

	// Soft delete all non-deleted revisions
	now := time.Now().UTC().Format(time.RFC3339)
	for rev, plan := range revisions {
		if plan.DeletedAt == nil {
			plan.DeletedAt = &now
			// Soft delete all slots in the plan
			for i := range plan.Slots {
				if plan.Slots[i].DeletedAt == nil {
					plan.Slots[i].DeletedAt = &now
				}
			}
			revisions[rev] = plan
		}
	}

	return s.save()
}

func (s *JSONStore) RestorePlan(date string) error {
	if s.store == nil {
		return fmt.Errorf("storage not loaded")
	}

	revisions, ok := s.store.Plans[date]
	if !ok || len(revisions) == 0 {
		return fmt.Errorf("no plan found for date: %s", date)
	}

	// Find the most recent deletion timestamp
	var mostRecentDeletedAt *string
	for _, p := range revisions {
		if p.DeletedAt != nil {
			if mostRecentDeletedAt == nil || *p.DeletedAt > *mostRecentDeletedAt {
				mostRecentDeletedAt = p.DeletedAt
			}
		}
	}

	if mostRecentDeletedAt == nil {
		return fmt.Errorf("no deleted plans found for date: %s", date)
	}

	// Restore all revisions with matching deleted_at timestamp
	for rev, plan := range revisions {
		if plan.DeletedAt != nil && *plan.DeletedAt == *mostRecentDeletedAt {
			plan.DeletedAt = nil
			// Restore slots with matching timestamp
			for i := range plan.Slots {
				if plan.Slots[i].DeletedAt != nil && *plan.Slots[i].DeletedAt == *mostRecentDeletedAt {
					plan.Slots[i].DeletedAt = nil
				}
			}
			revisions[rev] = plan
		}
	}

	return s.save()
}

// Habit stubs - JSON store doesn't support habits yet
func (s *JSONStore) AddHabit(habit models.Habit) error {
	return fmt.Errorf("habits are not supported in JSON store, please use SQLite")
}

func (s *JSONStore) GetHabit(id string) (models.Habit, error) {
	return models.Habit{}, fmt.Errorf("habits are not supported in JSON store, please use SQLite")
}

func (s *JSONStore) GetHabitByName(name string) (models.Habit, error) {
	return models.Habit{}, fmt.Errorf("habits are not supported in JSON store, please use SQLite")
}

func (s *JSONStore) GetAllHabits(includeArchived, includeDeleted bool) ([]models.Habit, error) {
	return nil, fmt.Errorf("habits are not supported in JSON store, please use SQLite")
}

func (s *JSONStore) UpdateHabit(habit models.Habit) error {
	return fmt.Errorf("habits are not supported in JSON store, please use SQLite")
}

func (s *JSONStore) ArchiveHabit(id string) error {
	return fmt.Errorf("habits are not supported in JSON store, please use SQLite")
}

func (s *JSONStore) UnarchiveHabit(id string) error {
	return fmt.Errorf("habits are not supported in JSON store, please use SQLite")
}

func (s *JSONStore) DeleteHabit(id string) error {
	return fmt.Errorf("habits are not supported in JSON store, please use SQLite")
}

func (s *JSONStore) RestoreHabit(id string) error {
	return fmt.Errorf("habits are not supported in JSON store, please use SQLite")
}

func (s *JSONStore) AddHabitEntry(entry models.HabitEntry) error {
	return fmt.Errorf("habit entries are not supported in JSON store, please use SQLite")
}

func (s *JSONStore) GetHabitEntry(habitID, day string) (models.HabitEntry, error) {
	return models.HabitEntry{}, fmt.Errorf("habit entries are not supported in JSON store, please use SQLite")
}

func (s *JSONStore) GetHabitEntriesForDay(day string) ([]models.HabitEntry, error) {
	return nil, fmt.Errorf("habit entries are not supported in JSON store, please use SQLite")
}

func (s *JSONStore) GetHabitEntriesForHabit(habitID string, startDay, endDay string) ([]models.HabitEntry, error) {
	return nil, fmt.Errorf("habit entries are not supported in JSON store, please use SQLite")
}

func (s *JSONStore) UpdateHabitEntry(entry models.HabitEntry) error {
	return fmt.Errorf("habit entries are not supported in JSON store, please use SQLite")
}

func (s *JSONStore) DeleteHabitEntry(id string) error {
	return fmt.Errorf("habit entries are not supported in JSON store, please use SQLite")
}

func (s *JSONStore) RestoreHabitEntry(id string) error {
	return fmt.Errorf("habit entries are not supported in JSON store, please use SQLite")
}

func (s *JSONStore) GetOTSettings() (models.OTSettings, error) {
	return models.OTSettings{}, fmt.Errorf("OT is not supported in JSON store, please use SQLite")
}

func (s *JSONStore) SaveOTSettings(settings models.OTSettings) error {
	return fmt.Errorf("OT is not supported in JSON store, please use SQLite")
}

func (s *JSONStore) AddOTEntry(entry models.OTEntry) error {
	return fmt.Errorf("OT entries are not supported in JSON store, please use SQLite")
}

func (s *JSONStore) GetOTEntry(day string) (models.OTEntry, error) {
	return models.OTEntry{}, fmt.Errorf("OT entries are not supported in JSON store, please use SQLite")
}

func (s *JSONStore) GetOTEntries(startDay, endDay string, includeDeleted bool) ([]models.OTEntry, error) {
	return nil, fmt.Errorf("OT entries are not supported in JSON store, please use SQLite")
}

func (s *JSONStore) UpdateOTEntry(entry models.OTEntry) error {
	return fmt.Errorf("OT entries are not supported in JSON store, please use SQLite")
}

func (s *JSONStore) DeleteOTEntry(day string) error {
	return fmt.Errorf("OT entries are not supported in JSON store, please use SQLite")
}

func (s *JSONStore) RestoreOTEntry(day string) error {
	return fmt.Errorf("OT entries are not supported in JSON store, please use SQLite")
}

func (s *JSONStore) GetConfigPath() string {
	return s.path
}
