package optimize

import (
	"testing"

	"github.com/julianstephens/daylit/daylit-cli/internal/cli"
	"github.com/julianstephens/daylit/daylit-cli/internal/constants"
	"github.com/julianstephens/daylit/daylit-cli/internal/models"
	"github.com/julianstephens/daylit/daylit-cli/internal/optimizer"
	"github.com/julianstephens/daylit/daylit-cli/internal/scheduler"
)

// mockStore is a mock implementation of storage.Provider for testing
type mockStore struct {
	feedbackHistory map[string][]models.TaskFeedbackEntry
	tasks           []models.Task
	updateTaskCalls []models.Task
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

func (m *mockStore) GetTask(id string) (models.Task, error) {
	for _, task := range m.tasks {
		if task.ID == id {
			return task, nil
		}
	}
	return models.Task{}, nil
}

func (m *mockStore) UpdateTask(task models.Task) error {
	m.updateTaskCalls = append(m.updateTaskCalls, task)
	// Update in tasks list
	for i, t := range m.tasks {
		if t.ID == task.ID {
			m.tasks[i] = task
			return nil
		}
	}
	return nil
}

// Implement other storage.Provider methods as no-ops
func (m *mockStore) Init() error                                         { return nil }
func (m *mockStore) Load() error                                         { return nil }
func (m *mockStore) Close() error                                        { return nil }
func (m *mockStore) GetSettings() (models.Settings, error)               { return models.Settings{}, nil }
func (m *mockStore) SaveSettings(models.Settings) error                  { return nil }
func (m *mockStore) AddTask(models.Task) error                           { return nil }
func (m *mockStore) GetAllTasksIncludingDeleted() ([]models.Task, error) { return nil, nil }
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

func TestApplyOptimization_ReduceDuration(t *testing.T) {
	store := &mockStore{
		tasks: []models.Task{
			{
				ID:          "task-1",
				Name:        "Test Task",
				DurationMin: 60,
				Priority:    3,
				Active:      true,
			},
		},
	}
	ctx := &cli.Context{
		Store:     store,
		Scheduler: scheduler.New(),
	}

	opt := optimizer.Optimization{
		TaskID:   "task-1",
		TaskName: "Test Task",
		Type:     constants.OptimizationReduceDuration,
		SuggestedValue: map[string]interface{}{
			"duration_min": 45,
		},
	}

	err := applyOptimization(ctx, opt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(store.updateTaskCalls) != 1 {
		t.Fatalf("expected 1 UpdateTask call, got %d", len(store.updateTaskCalls))
	}

	updated := store.updateTaskCalls[0]
	if updated.DurationMin != 45 {
		t.Errorf("expected duration 45, got %d", updated.DurationMin)
	}
}

func TestApplyOptimization_ReduceFrequency(t *testing.T) {
	store := &mockStore{
		tasks: []models.Task{
			{
				ID:          "task-1",
				Name:        "Test Task",
				DurationMin: 30,
				Priority:    3,
				Active:      true,
				Recurrence: models.Recurrence{
					Type:         constants.RecurrenceNDays,
					IntervalDays: 1,
				},
			},
		},
	}
	ctx := &cli.Context{
		Store:     store,
		Scheduler: scheduler.New(),
	}

	opt := optimizer.Optimization{
		TaskID:   "task-1",
		TaskName: "Test Task",
		Type:     constants.OptimizationReduceFrequency,
		SuggestedValue: map[string]interface{}{
			"interval_days": 3,
		},
	}

	err := applyOptimization(ctx, opt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(store.updateTaskCalls) != 1 {
		t.Fatalf("expected 1 UpdateTask call, got %d", len(store.updateTaskCalls))
	}

	updated := store.updateTaskCalls[0]
	if updated.Recurrence.IntervalDays != 3 {
		t.Errorf("expected interval 3, got %d", updated.Recurrence.IntervalDays)
	}
}

func TestApplyOptimization_InvalidTypeAssertion(t *testing.T) {
	store := &mockStore{
		tasks: []models.Task{
			{
				ID:          "task-1",
				Name:        "Test Task",
				DurationMin: 60,
				Priority:    3,
				Active:      true,
			},
		},
	}
	ctx := &cli.Context{
		Store:     store,
		Scheduler: scheduler.New(),
	}

	// Invalid suggested value - string instead of int
	opt := optimizer.Optimization{
		TaskID:   "task-1",
		TaskName: "Test Task",
		Type:     constants.OptimizationReduceDuration,
		SuggestedValue: map[string]interface{}{
			"duration_min": "invalid",
		},
	}

	err := applyOptimization(ctx, opt)
	if err == nil {
		t.Fatal("expected error for invalid type assertion, got nil")
	}
}

func TestFormatValue_DeterministicOutput(t *testing.T) {
	value := map[string]interface{}{
		"z_key": 1,
		"a_key": 2,
		"m_key": 3,
	}

	// Call multiple times to ensure consistent output
	result1 := formatValue(value)
	result2 := formatValue(value)
	result3 := formatValue(value)

	if result1 != result2 || result2 != result3 {
		t.Errorf("formatValue output is not deterministic:\n  %s\n  %s\n  %s", result1, result2, result3)
	}

	// Check that keys are sorted
	expected := "a_key=2, m_key=3, z_key=1"
	if result1 != expected {
		t.Errorf("expected sorted keys: %s, got: %s", expected, result1)
	}
}
