package optimizer

import (
	"testing"

	"github.com/julianstephens/daylit/daylit-cli/internal/models"
)

// mockStore is a mock implementation of storage.Provider for testing
type mockStore struct {
	feedbackHistory map[string][]models.TaskFeedbackEntry
	tasks           []models.Task
}

func (m *mockStore) GetTaskFeedbackHistory(taskID string, limit int) ([]models.TaskFeedbackEntry, error) {
	history, ok := m.feedbackHistory[taskID]
	if !ok {
		return []models.TaskFeedbackEntry{}, nil
	}
	if len(history) > limit {
		return history[:limit], nil
	}
	return history, nil
}

func (m *mockStore) GetAllTasks() ([]models.Task, error) {
	return m.tasks, nil
}

// Implement other storage.Provider methods as no-ops
func (m *mockStore) Init() error                                         { return nil }
func (m *mockStore) Load() error                                         { return nil }
func (m *mockStore) Close() error                                        { return nil }
func (m *mockStore) GetSettings() (models.Settings, error)               { return models.Settings{}, nil }
func (m *mockStore) SaveSettings(models.Settings) error                  { return nil }
func (m *mockStore) AddTask(models.Task) error                           { return nil }
func (m *mockStore) GetTask(id string) (models.Task, error)              { return models.Task{}, nil }
func (m *mockStore) GetAllTasksIncludingDeleted() ([]models.Task, error) { return nil, nil }
func (m *mockStore) UpdateTask(models.Task) error                        { return nil }
func (m *mockStore) DeleteTask(id string) error                          { return nil }
func (m *mockStore) RestoreTask(id string) error                         { return nil }
func (m *mockStore) SavePlan(models.DayPlan) error                       { return nil }
func (m *mockStore) GetPlan(date string) (models.DayPlan, error)         { return models.DayPlan{}, nil }
func (m *mockStore) GetPlanRevision(date string, revision int) (models.DayPlan, error) {
	return models.DayPlan{}, nil
}
func (m *mockStore) GetLatestPlanRevision(date string) (models.DayPlan, error) {
	return models.DayPlan{}, nil
}
func (m *mockStore) DeletePlan(date string) error  { return nil }
func (m *mockStore) RestorePlan(date string) error { return nil }
func (m *mockStore) UpdateSlotNotificationTimestamp(date string, revision int, startTime string, taskID string, notificationType string, timestamp string) error {
	return nil
}
func (m *mockStore) AddHabit(models.Habit) error                      { return nil }
func (m *mockStore) GetHabit(id string) (models.Habit, error)         { return models.Habit{}, nil }
func (m *mockStore) GetHabitByName(name string) (models.Habit, error) { return models.Habit{}, nil }
func (m *mockStore) GetAllHabits(includeArchived, includeDeleted bool) ([]models.Habit, error) {
	return nil, nil
}
func (m *mockStore) UpdateHabit(models.Habit) error        { return nil }
func (m *mockStore) ArchiveHabit(id string) error          { return nil }
func (m *mockStore) UnarchiveHabit(id string) error        { return nil }
func (m *mockStore) DeleteHabit(id string) error           { return nil }
func (m *mockStore) RestoreHabit(id string) error          { return nil }
func (m *mockStore) AddHabitEntry(models.HabitEntry) error { return nil }
func (m *mockStore) GetHabitEntry(habitID, day string) (models.HabitEntry, error) {
	return models.HabitEntry{}, nil
}
func (m *mockStore) GetHabitEntriesForDay(day string) ([]models.HabitEntry, error) { return nil, nil }
func (m *mockStore) GetHabitEntriesForHabit(habitID string, startDay, endDay string) ([]models.HabitEntry, error) {
	return nil, nil
}
func (m *mockStore) UpdateHabitEntry(models.HabitEntry) error      { return nil }
func (m *mockStore) DeleteHabitEntry(id string) error              { return nil }
func (m *mockStore) RestoreHabitEntry(id string) error             { return nil }
func (m *mockStore) GetOTSettings() (models.OTSettings, error)     { return models.OTSettings{}, nil }
func (m *mockStore) SaveOTSettings(models.OTSettings) error        { return nil }
func (m *mockStore) AddOTEntry(models.OTEntry) error               { return nil }
func (m *mockStore) GetOTEntry(day string) (models.OTEntry, error) { return models.OTEntry{}, nil }
func (m *mockStore) GetOTEntries(startDay, endDay string, includeDeleted bool) ([]models.OTEntry, error) {
	return nil, nil
}
func (m *mockStore) UpdateOTEntry(models.OTEntry) error               { return nil }
func (m *mockStore) DeleteOTEntry(day string) error                   { return nil }
func (m *mockStore) RestoreOTEntry(day string) error                  { return nil }
func (m *mockStore) GetAllPlans() ([]models.DayPlan, error)           { return nil, nil }
func (m *mockStore) GetAllHabitEntries() ([]models.HabitEntry, error) { return nil, nil }
func (m *mockStore) GetAllOTEntries() ([]models.OTEntry, error)       { return nil, nil }
func (m *mockStore) GetConfigPath() string                            { return "" }
func (m *mockStore) AddAlert(models.Alert) error                      { return nil }
func (m *mockStore) GetAlert(id string) (models.Alert, error)         { return models.Alert{}, nil }
func (m *mockStore) GetAllAlerts() ([]models.Alert, error)            { return nil, nil }
func (m *mockStore) UpdateAlert(models.Alert) error                   { return nil }
func (m *mockStore) DeleteAlert(id string) error                      { return nil }

func TestAnalyzeTask_NoFeedback(t *testing.T) {
	store := &mockStore{
		feedbackHistory: make(map[string][]models.TaskFeedbackEntry),
	}
	analyzer := NewFeedbackAnalyzer(store)

	task := models.Task{
		ID:          "task-1",
		Name:        "Test Task",
		DurationMin: 60,
		Active:      true,
	}

	optimizations, err := analyzer.AnalyzeTask(task, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(optimizations) != 0 {
		t.Errorf("expected no optimizations, got %d", len(optimizations))
	}
}

func TestAnalyzeTask_MostlyTooMuch(t *testing.T) {
	store := &mockStore{
		feedbackHistory: map[string][]models.TaskFeedbackEntry{
			"task-1": {
				{TaskID: "task-1", Rating: models.FeedbackTooMuch, ActualStart: "09:00", ActualEnd: "10:00"},
				{TaskID: "task-1", Rating: models.FeedbackTooMuch, ActualStart: "09:00", ActualEnd: "10:00"},
				{TaskID: "task-1", Rating: models.FeedbackTooMuch, ActualStart: "09:00", ActualEnd: "10:00"},
				{TaskID: "task-1", Rating: models.FeedbackOnTrack, ActualStart: "09:00", ActualEnd: "10:00"},
			},
		},
	}
	analyzer := NewFeedbackAnalyzer(store)

	task := models.Task{
		ID:          "task-1",
		Name:        "Test Task",
		DurationMin: 60,
		Active:      true,
	}

	optimizations, err := analyzer.AnalyzeTask(task, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(optimizations) != 1 {
		t.Fatalf("expected 1 optimization, got %d", len(optimizations))
	}

	opt := optimizations[0]
	if opt.Type != OptimizationReduceDuration {
		t.Errorf("expected OptimizationReduceDuration, got %v", opt.Type)
	}

	if opt.TaskID != "task-1" {
		t.Errorf("expected task-1, got %v", opt.TaskID)
	}
}

func TestAnalyzeTask_MostlyUnnecessary(t *testing.T) {
	store := &mockStore{
		feedbackHistory: map[string][]models.TaskFeedbackEntry{
			"task-1": {
				{TaskID: "task-1", Rating: models.FeedbackUnnecessary},
				{TaskID: "task-1", Rating: models.FeedbackUnnecessary},
				{TaskID: "task-1", Rating: models.FeedbackUnnecessary},
				{TaskID: "task-1", Rating: models.FeedbackOnTrack},
			},
		},
	}
	analyzer := NewFeedbackAnalyzer(store)

	task := models.Task{
		ID:          "task-1",
		Name:        "Test Task",
		DurationMin: 60,
		Active:      true,
		Recurrence: models.Recurrence{
			Type:         models.RecurrenceNDays,
			IntervalDays: 1,
		},
	}

	optimizations, err := analyzer.AnalyzeTask(task, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(optimizations) != 1 {
		t.Fatalf("expected 1 optimization, got %d", len(optimizations))
	}

	opt := optimizations[0]
	if opt.Type != OptimizationReduceFrequency {
		t.Errorf("expected OptimizationReduceFrequency, got %v", opt.Type)
	}
}

func TestAnalyzeTask_ShortTaskTooMuch(t *testing.T) {
	store := &mockStore{
		feedbackHistory: map[string][]models.TaskFeedbackEntry{
			"task-1": {
				{TaskID: "task-1", Rating: models.FeedbackTooMuch},
				{TaskID: "task-1", Rating: models.FeedbackTooMuch},
				{TaskID: "task-1", Rating: models.FeedbackTooMuch},
			},
		},
	}
	analyzer := NewFeedbackAnalyzer(store)

	task := models.Task{
		ID:          "task-1",
		Name:        "Short Task",
		DurationMin: 20, // Short task
		Active:      true,
	}

	optimizations, err := analyzer.AnalyzeTask(task, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(optimizations) != 1 {
		t.Fatalf("expected 1 optimization, got %d", len(optimizations))
	}

	opt := optimizations[0]
	if opt.Type != OptimizationSplitTask {
		t.Errorf("expected OptimizationSplitTask for short task, got %v", opt.Type)
	}
}

func TestAnalyzeAllTasks(t *testing.T) {
	store := &mockStore{
		feedbackHistory: map[string][]models.TaskFeedbackEntry{
			"task-1": {
				{TaskID: "task-1", Rating: models.FeedbackTooMuch},
				{TaskID: "task-1", Rating: models.FeedbackTooMuch},
			},
		},
		tasks: []models.Task{
			{
				ID:          "task-1",
				Name:        "Task 1",
				DurationMin: 60,
				Active:      true,
			},
			{
				ID:          "task-2",
				Name:        "Task 2",
				DurationMin: 30,
				Active:      true,
			},
		},
	}
	analyzer := NewFeedbackAnalyzer(store)

	optimizations, err := analyzer.AnalyzeAllTasks(10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have at least one optimization for task-1
	if len(optimizations) < 1 {
		t.Errorf("expected at least 1 optimization, got %d", len(optimizations))
	}
}
