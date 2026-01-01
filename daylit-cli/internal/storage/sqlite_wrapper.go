package storage

import (
	"database/sql"

	"github.com/julianstephens/daylit/daylit-cli/internal/models"
	"github.com/julianstephens/daylit/daylit-cli/internal/storage/sqlite"
)

// SQLiteStore wraps sqlite.Store for backward compatibility
type SQLiteStore struct {
	store *sqlite.Store
	// db is exported for test access
	db *sql.DB
}

// NewSQLiteStore creates a new SQLite store
func NewSQLiteStore(path string) *SQLiteStore {
	store := sqlite.NewStore(path)
	return &SQLiteStore{
		store: store,
		db:    nil, // Will be set after Init/Load
	}
}

// Lifecycle methods
func (s *SQLiteStore) Init() error {
	err := s.store.Init()
	if err == nil {
		s.db = s.store.GetDB()
	}
	return err
}

func (s *SQLiteStore) Load() error {
	err := s.store.Load()
	if err == nil {
		s.db = s.store.GetDB()
	}
	return err
}

func (s *SQLiteStore) Close() error               { return s.store.Close() }
func (s *SQLiteStore) GetConfigPath() string      { return s.store.GetConfigPath() }
func (s *SQLiteStore) GetDB() *sql.DB             {
	if s.db == nil {
		s.db = s.store.GetDB()
	}
	return s.db
}

// tableExists is exposed for test compatibility
func (s *SQLiteStore) tableExists(tableName string) (bool, error) {
	var count int
	row := s.GetDB().QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name COLLATE NOCASE = ?", tableName)
	if err := row.Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

// Settings methods
func (s *SQLiteStore) GetSettings() (Settings, error)    { return s.store.GetSettings() }
func (s *SQLiteStore) SaveSettings(settings Settings) error { return s.store.SaveSettings(settings) }

// Task methods
func (s *SQLiteStore) AddTask(task models.Task) error                       { return s.store.AddTask(task) }
func (s *SQLiteStore) GetTask(id string) (models.Task, error)               { return s.store.GetTask(id) }
func (s *SQLiteStore) GetAllTasks() ([]models.Task, error)                  { return s.store.GetAllTasks() }
func (s *SQLiteStore) GetAllTasksIncludingDeleted() ([]models.Task, error)  { return s.store.GetAllTasksIncludingDeleted() }
func (s *SQLiteStore) UpdateTask(task models.Task) error                    { return s.store.UpdateTask(task) }
func (s *SQLiteStore) DeleteTask(id string) error                           { return s.store.DeleteTask(id) }
func (s *SQLiteStore) RestoreTask(id string) error                          { return s.store.RestoreTask(id) }

// Plan methods
func (s *SQLiteStore) SavePlan(plan models.DayPlan) error                           { return s.store.SavePlan(plan) }
func (s *SQLiteStore) GetPlan(date string) (models.DayPlan, error)                  { return s.store.GetPlan(date) }
func (s *SQLiteStore) GetLatestPlanRevision(date string) (models.DayPlan, error)    { return s.store.GetLatestPlanRevision(date) }
func (s *SQLiteStore) GetPlanRevision(date string, revision int) (models.DayPlan, error) { return s.store.GetPlanRevision(date, revision) }
func (s *SQLiteStore) DeletePlan(date string) error                                 { return s.store.DeletePlan(date) }
func (s *SQLiteStore) RestorePlan(date string) error                                { return s.store.RestorePlan(date) }
func (s *SQLiteStore) UpdateSlotNotificationTimestamp(date string, revision int, startTime string, taskID string, notificationType string, timestamp string) error {
	return s.store.UpdateSlotNotificationTimestamp(date, revision, startTime, taskID, notificationType, timestamp)
}

// Habit methods
func (s *SQLiteStore) AddHabit(habit models.Habit) error                                    { return s.store.AddHabit(habit) }
func (s *SQLiteStore) GetHabit(id string) (models.Habit, error)                             { return s.store.GetHabit(id) }
func (s *SQLiteStore) GetHabitByName(name string) (models.Habit, error)                     { return s.store.GetHabitByName(name) }
func (s *SQLiteStore) GetAllHabits(includeArchived, includeDeleted bool) ([]models.Habit, error) { return s.store.GetAllHabits(includeArchived, includeDeleted) }
func (s *SQLiteStore) UpdateHabit(habit models.Habit) error                                 { return s.store.UpdateHabit(habit) }
func (s *SQLiteStore) ArchiveHabit(id string) error                                         { return s.store.ArchiveHabit(id) }
func (s *SQLiteStore) UnarchiveHabit(id string) error                                       { return s.store.UnarchiveHabit(id) }
func (s *SQLiteStore) DeleteHabit(id string) error                                          { return s.store.DeleteHabit(id) }
func (s *SQLiteStore) RestoreHabit(id string) error                                         { return s.store.RestoreHabit(id) }

// Habit Entry methods
func (s *SQLiteStore) AddHabitEntry(entry models.HabitEntry) error                               { return s.store.AddHabitEntry(entry) }
func (s *SQLiteStore) GetHabitEntry(habitID, day string) (models.HabitEntry, error)              { return s.store.GetHabitEntry(habitID, day) }
func (s *SQLiteStore) GetHabitEntriesForDay(day string) ([]models.HabitEntry, error)             { return s.store.GetHabitEntriesForDay(day) }
func (s *SQLiteStore) GetHabitEntriesForHabit(habitID string, startDay, endDay string) ([]models.HabitEntry, error) { return s.store.GetHabitEntriesForHabit(habitID, startDay, endDay) }
func (s *SQLiteStore) UpdateHabitEntry(entry models.HabitEntry) error                            { return s.store.UpdateHabitEntry(entry) }
func (s *SQLiteStore) DeleteHabitEntry(id string) error                                          { return s.store.DeleteHabitEntry(id) }
func (s *SQLiteStore) RestoreHabitEntry(id string) error                                         { return s.store.RestoreHabitEntry(id) }

// OT methods
func (s *SQLiteStore) GetOTSettings() (models.OTSettings, error)                          { return s.store.GetOTSettings() }
func (s *SQLiteStore) SaveOTSettings(settings models.OTSettings) error                    { return s.store.SaveOTSettings(settings) }
func (s *SQLiteStore) AddOTEntry(entry models.OTEntry) error                              { return s.store.AddOTEntry(entry) }
func (s *SQLiteStore) GetOTEntry(day string) (models.OTEntry, error)                      { return s.store.GetOTEntry(day) }
func (s *SQLiteStore) GetOTEntries(startDay, endDay string, includeDeleted bool) ([]models.OTEntry, error) { return s.store.GetOTEntries(startDay, endDay, includeDeleted) }
func (s *SQLiteStore) UpdateOTEntry(entry models.OTEntry) error                           { return s.store.UpdateOTEntry(entry) }
func (s *SQLiteStore) DeleteOTEntry(day string) error                                     { return s.store.DeleteOTEntry(day) }
func (s *SQLiteStore) RestoreOTEntry(day string) error                                    { return s.store.RestoreOTEntry(day) }

// Backup/Migration methods
func (s *SQLiteStore) GetAllPlans() ([]models.DayPlan, error)        { return s.store.GetAllPlans() }
func (s *SQLiteStore) GetAllHabitEntries() ([]models.HabitEntry, error) { return s.store.GetAllHabitEntries() }
func (s *SQLiteStore) GetAllOTEntries() ([]models.OTEntry, error)    { return s.store.GetAllOTEntries() }
