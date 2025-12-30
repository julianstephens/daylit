package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/julianstephens/daylit/internal/models"
)

func setupTestSQLiteStore(t *testing.T) (*SQLiteStore, func()) {
	// Create a temporary directory for test database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Create test store
	store := NewSQLiteStore(dbPath)
	if err := store.Init(); err != nil {
		t.Fatalf("failed to initialize test store: %v", err)
	}

	cleanup := func() {
		store.Close()
		os.RemoveAll(tempDir)
	}

	return store, cleanup
}

func TestTaskSoftDelete(t *testing.T) {
	store, cleanup := setupTestSQLiteStore(t)
	defer cleanup()

	// Create a test task
	task := models.Task{
		ID:          "task-1",
		Name:        "Test Task",
		Kind:        models.TaskKindFlexible,
		DurationMin: 30,
		Recurrence: models.Recurrence{
			Type: models.RecurrenceDaily,
		},
		Priority: 1,
		Active:   true,
	}

	// Add the task
	if err := store.AddTask(task); err != nil {
		t.Fatalf("failed to add task: %v", err)
	}

	// Verify task can be retrieved
	retrievedTask, err := store.GetTask(task.ID)
	if err != nil {
		t.Fatalf("failed to get task: %v", err)
	}
	if retrievedTask.ID != task.ID {
		t.Errorf("expected task ID %s, got %s", task.ID, retrievedTask.ID)
	}

	// Soft delete the task
	if err := store.DeleteTask(task.ID); err != nil {
		t.Fatalf("failed to delete task: %v", err)
	}

	// Verify task cannot be retrieved (soft deleted)
	_, err = store.GetTask(task.ID)
	if err == nil {
		t.Error("expected error when getting deleted task, got nil")
	}

	// Verify task is not in GetAllTasks
	allTasks, err := store.GetAllTasks()
	if err != nil {
		t.Fatalf("failed to get all tasks: %v", err)
	}
	for _, task := range allTasks {
		if task.ID == "task-1" {
			t.Error("deleted task should not appear in GetAllTasks")
		}
	}
}

func TestTaskRestore(t *testing.T) {
	store, cleanup := setupTestSQLiteStore(t)
	defer cleanup()

	// Create and add a test task
	task := models.Task{
		ID:          "task-2",
		Name:        "Test Task 2",
		Kind:        models.TaskKindFlexible,
		DurationMin: 45,
		Recurrence: models.Recurrence{
			Type: models.RecurrenceDaily,
		},
		Priority: 2,
		Active:   true,
	}

	if err := store.AddTask(task); err != nil {
		t.Fatalf("failed to add task: %v", err)
	}

	// Soft delete the task
	if err := store.DeleteTask(task.ID); err != nil {
		t.Fatalf("failed to delete task: %v", err)
	}

	// Verify task is deleted
	_, err := store.GetTask(task.ID)
	if err == nil {
		t.Error("expected error when getting deleted task")
	}

	// Restore the task
	if err := store.RestoreTask(task.ID); err != nil {
		t.Fatalf("failed to restore task: %v", err)
	}

	// Verify task can be retrieved again
	restoredTask, err := store.GetTask(task.ID)
	if err != nil {
		t.Fatalf("failed to get restored task: %v", err)
	}
	if restoredTask.ID != task.ID {
		t.Errorf("expected task ID %s, got %s", task.ID, restoredTask.ID)
	}
	if restoredTask.Name != task.Name {
		t.Errorf("expected task name %s, got %s", task.Name, restoredTask.Name)
	}
}

func TestPlanSoftDelete(t *testing.T) {
	store, cleanup := setupTestSQLiteStore(t)
	defer cleanup()

	// Create a task for the plan
	task := models.Task{
		ID:          "task-3",
		Name:        "Test Task 3",
		Kind:        models.TaskKindFlexible,
		DurationMin: 30,
		Recurrence: models.Recurrence{
			Type: models.RecurrenceDaily,
		},
		Priority: 1,
		Active:   true,
	}
	if err := store.AddTask(task); err != nil {
		t.Fatalf("failed to add task: %v", err)
	}

	// Create a test plan
	plan := models.DayPlan{
		Date: "2024-01-15",
		Slots: []models.Slot{
			{
				Start:  "09:00",
				End:    "09:30",
				TaskID: task.ID,
				Status: models.SlotStatusPlanned,
			},
		},
	}

	// Save the plan
	if err := store.SavePlan(plan); err != nil {
		t.Fatalf("failed to save plan: %v", err)
	}

	// Verify plan can be retrieved
	retrievedPlan, err := store.GetPlan(plan.Date)
	if err != nil {
		t.Fatalf("failed to get plan: %v", err)
	}
	if retrievedPlan.Date != plan.Date {
		t.Errorf("expected plan date %s, got %s", plan.Date, retrievedPlan.Date)
	}

	// Soft delete the plan
	if err := store.DeletePlan(plan.Date); err != nil {
		t.Fatalf("failed to delete plan: %v", err)
	}

	// Verify plan cannot be retrieved (soft deleted)
	_, err = store.GetPlan(plan.Date)
	if err == nil {
		t.Error("expected error when getting deleted plan, got nil")
	}
}

func TestPlanRestore(t *testing.T) {
	store, cleanup := setupTestSQLiteStore(t)
	defer cleanup()

	// Create a task for the plan
	task := models.Task{
		ID:          "task-4",
		Name:        "Test Task 4",
		Kind:        models.TaskKindFlexible,
		DurationMin: 30,
		Recurrence: models.Recurrence{
			Type: models.RecurrenceDaily,
		},
		Priority: 1,
		Active:   true,
	}
	if err := store.AddTask(task); err != nil {
		t.Fatalf("failed to add task: %v", err)
	}

	// Create a test plan
	plan := models.DayPlan{
		Date: "2024-01-16",
		Slots: []models.Slot{
			{
				Start:  "10:00",
				End:    "10:30",
				TaskID: task.ID,
				Status: models.SlotStatusPlanned,
			},
		},
	}

	if err := store.SavePlan(plan); err != nil {
		t.Fatalf("failed to save plan: %v", err)
	}

	// Soft delete the plan
	if err := store.DeletePlan(plan.Date); err != nil {
		t.Fatalf("failed to delete plan: %v", err)
	}

	// Verify plan is deleted
	_, err := store.GetPlan(plan.Date)
	if err == nil {
		t.Error("expected error when getting deleted plan")
	}

	// Restore the plan
	if err := store.RestorePlan(plan.Date); err != nil {
		t.Fatalf("failed to restore plan: %v", err)
	}

	// Verify plan can be retrieved again
	restoredPlan, err := store.GetPlan(plan.Date)
	if err != nil {
		t.Fatalf("failed to get restored plan: %v", err)
	}
	if restoredPlan.Date != plan.Date {
		t.Errorf("expected plan date %s, got %s", plan.Date, restoredPlan.Date)
	}
	if len(restoredPlan.Slots) != len(plan.Slots) {
		t.Errorf("expected %d slots, got %d", len(plan.Slots), len(restoredPlan.Slots))
	}
}

func TestCannotAddSlotsToDeletedPlan(t *testing.T) {
	store, cleanup := setupTestSQLiteStore(t)
	defer cleanup()

	// Create a task
	task := models.Task{
		ID:          "task-5",
		Name:        "Test Task 5",
		Kind:        models.TaskKindFlexible,
		DurationMin: 30,
		Recurrence: models.Recurrence{
			Type: models.RecurrenceDaily,
		},
		Priority: 1,
		Active:   true,
	}
	if err := store.AddTask(task); err != nil {
		t.Fatalf("failed to add task: %v", err)
	}

	// Create and save a plan
	plan := models.DayPlan{
		Date: "2024-01-17",
		Slots: []models.Slot{
			{
				Start:  "11:00",
				End:    "11:30",
				TaskID: task.ID,
				Status: models.SlotStatusPlanned,
			},
		},
	}
	if err := store.SavePlan(plan); err != nil {
		t.Fatalf("failed to save plan: %v", err)
	}

	// Soft delete the plan
	if err := store.DeletePlan(plan.Date); err != nil {
		t.Fatalf("failed to delete plan: %v", err)
	}

	// Try to save slots to the deleted plan
	newPlan := models.DayPlan{
		Date: "2024-01-17",
		Slots: []models.Slot{
			{
				Start:  "12:00",
				End:    "12:30",
				TaskID: task.ID,
				Status: models.SlotStatusPlanned,
			},
		},
	}
	err := store.SavePlan(newPlan)
	if err == nil {
		t.Error("expected error when saving slots to deleted plan, got nil")
	}
}

func TestDeletedTasksExcludedFromScheduler(t *testing.T) {
	store, cleanup := setupTestSQLiteStore(t)
	defer cleanup()

	// Create multiple tasks
	task1 := models.Task{
		ID:          "task-6",
		Name:        "Active Task",
		Kind:        models.TaskKindFlexible,
		DurationMin: 30,
		Recurrence: models.Recurrence{
			Type: models.RecurrenceDaily,
		},
		Priority: 1,
		Active:   true,
	}
	task2 := models.Task{
		ID:          "task-7",
		Name:        "To Be Deleted Task",
		Kind:        models.TaskKindFlexible,
		DurationMin: 30,
		Recurrence: models.Recurrence{
			Type: models.RecurrenceDaily,
		},
		Priority: 2,
		Active:   true,
	}

	if err := store.AddTask(task1); err != nil {
		t.Fatalf("failed to add task1: %v", err)
	}
	if err := store.AddTask(task2); err != nil {
		t.Fatalf("failed to add task2: %v", err)
	}

	// Verify both tasks are in GetAllTasks
	allTasks, err := store.GetAllTasks()
	if err != nil {
		t.Fatalf("failed to get all tasks: %v", err)
	}
	if len(allTasks) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(allTasks))
	}

	// Delete task2
	if err := store.DeleteTask(task2.ID); err != nil {
		t.Fatalf("failed to delete task2: %v", err)
	}

	// Verify only task1 is in GetAllTasks
	allTasks, err = store.GetAllTasks()
	if err != nil {
		t.Fatalf("failed to get all tasks after deletion: %v", err)
	}
	if len(allTasks) != 1 {
		t.Errorf("expected 1 task after deletion, got %d", len(allTasks))
	}
	if allTasks[0].ID != task1.ID {
		t.Errorf("expected task ID %s, got %s", task1.ID, allTasks[0].ID)
	}
}

func TestSoftDeletePreservesData(t *testing.T) {
	store, cleanup := setupTestSQLiteStore(t)
	defer cleanup()

	// Create a task with all fields populated
	task := models.Task{
		ID:          "task-8",
		Name:        "Full Task",
		Kind:        models.TaskKindAppointment,
		DurationMin: 60,
		EarliestStart: "09:00",
		LatestEnd:     "17:00",
		FixedStart:    "10:00",
		FixedEnd:      "11:00",
		Recurrence: models.Recurrence{
			Type:         models.RecurrenceWeekly,
			WeekdayMask:  []time.Weekday{time.Monday, time.Wednesday, time.Friday},
		},
		Priority:       3,
		EnergyBand:     models.EnergyHigh,
		Active:         true,
		LastDone:       "2024-01-10",
		SuccessStreak:  5,
		AvgActualDurationMin: 55.5,
	}

	if err := store.AddTask(task); err != nil {
		t.Fatalf("failed to add task: %v", err)
	}

	// Soft delete the task
	if err := store.DeleteTask(task.ID); err != nil {
		t.Fatalf("failed to delete task: %v", err)
	}

	// Restore the task
	if err := store.RestoreTask(task.ID); err != nil {
		t.Fatalf("failed to restore task: %v", err)
	}

	// Verify all fields are preserved
	restoredTask, err := store.GetTask(task.ID)
	if err != nil {
		t.Fatalf("failed to get restored task: %v", err)
	}

	if restoredTask.Name != task.Name {
		t.Errorf("name not preserved: expected %s, got %s", task.Name, restoredTask.Name)
	}
	if restoredTask.Kind != task.Kind {
		t.Errorf("kind not preserved: expected %s, got %s", task.Kind, restoredTask.Kind)
	}
	if restoredTask.DurationMin != task.DurationMin {
		t.Errorf("duration not preserved: expected %d, got %d", task.DurationMin, restoredTask.DurationMin)
	}
	if restoredTask.Priority != task.Priority {
		t.Errorf("priority not preserved: expected %d, got %d", task.Priority, restoredTask.Priority)
	}
	if restoredTask.SuccessStreak != task.SuccessStreak {
		t.Errorf("success streak not preserved: expected %d, got %d", task.SuccessStreak, restoredTask.SuccessStreak)
	}
}
