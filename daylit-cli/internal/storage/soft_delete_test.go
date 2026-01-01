package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/julianstephens/daylit/daylit-cli/internal/models"
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
		ID:            "task-8",
		Name:          "Full Task",
		Kind:          models.TaskKindAppointment,
		DurationMin:   60,
		EarliestStart: "09:00",
		LatestEnd:     "17:00",
		FixedStart:    "10:00",
		FixedEnd:      "11:00",
		Recurrence: models.Recurrence{
			Type:        models.RecurrenceWeekly,
			WeekdayMask: []time.Weekday{time.Monday, time.Wednesday, time.Friday},
		},
		Priority:             3,
		EnergyBand:           models.EnergyHigh,
		Active:               true,
		LastDone:             "2024-01-10",
		SuccessStreak:        5,
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

// Edge case tests

func TestDeleteAlreadyDeletedTask(t *testing.T) {
	store, cleanup := setupTestSQLiteStore(t)
	defer cleanup()

	// Create and add a test task
	task := models.Task{
		ID:          "task-double-delete",
		Name:        "Double Delete Task",
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

	// Delete the task once
	if err := store.DeleteTask(task.ID); err != nil {
		t.Fatalf("failed to delete task: %v", err)
	}

	// Try to delete again - should return error
	err := store.DeleteTask(task.ID)
	if err == nil {
		t.Error("expected error when deleting already deleted task, got nil")
	}
}

func TestRestoreNonDeletedTask(t *testing.T) {
	store, cleanup := setupTestSQLiteStore(t)
	defer cleanup()

	// Create and add a test task (not deleted)
	task := models.Task{
		ID:          "task-restore-active",
		Name:        "Restore Active Task",
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

	// Try to restore a non-deleted task - should return error
	err := store.RestoreTask(task.ID)
	if err == nil {
		t.Error("expected error when restoring non-deleted task, got nil")
	}
}

func TestDeleteAlreadyDeletedPlan(t *testing.T) {
	store, cleanup := setupTestSQLiteStore(t)
	defer cleanup()

	// Create a task
	task := models.Task{
		ID:          "task-plan-double-delete",
		Name:        "Plan Double Delete Task",
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
		Date: "2024-02-01",
		Slots: []models.Slot{
			{
				Start:  "09:00",
				End:    "09:30",
				TaskID: task.ID,
				Status: models.SlotStatusPlanned,
			},
		},
	}

	if err := store.SavePlan(plan); err != nil {
		t.Fatalf("failed to save plan: %v", err)
	}

	// Delete the plan once
	if err := store.DeletePlan(plan.Date); err != nil {
		t.Fatalf("failed to delete plan: %v", err)
	}

	// Try to delete again - should return error
	err := store.DeletePlan(plan.Date)
	if err == nil {
		t.Error("expected error when deleting already deleted plan, got nil")
	}
}

func TestRestoreNonDeletedPlan(t *testing.T) {
	store, cleanup := setupTestSQLiteStore(t)
	defer cleanup()

	// Create a task
	task := models.Task{
		ID:          "task-restore-active-plan",
		Name:        "Restore Active Plan Task",
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

	// Create and save a plan (not deleted)
	plan := models.DayPlan{
		Date: "2024-02-02",
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

	// Try to restore a non-deleted plan - should return error
	err := store.RestorePlan(plan.Date)
	if err == nil {
		t.Error("expected error when restoring non-deleted plan, got nil")
	}
}

func TestRestorePlanTimestampMatching(t *testing.T) {
	store, cleanup := setupTestSQLiteStore(t)
	defer cleanup()

	// Create a task
	task := models.Task{
		ID:          "task-timestamp-match",
		Name:        "Timestamp Match Task",
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

	// Create a plan with two slots
	plan := models.DayPlan{
		Date: "2024-02-03",
		Slots: []models.Slot{
			{
				Start:  "09:00",
				End:    "09:30",
				TaskID: task.ID,
				Status: models.SlotStatusPlanned,
			},
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

	// Manually soft-delete one slot individually (simulating a slot being deleted before the plan)
	// This requires direct database access since there's no API for individual slot deletion yet
	db := store.GetDB()
	if db == nil {
		t.Fatal("database connection is nil")
	}

	earlyDeleteTime := "2024-02-03T08:00:00Z"
	_, err := db.Exec("UPDATE slots SET deleted_at = ? WHERE plan_date = ? AND start_time = ?",
		earlyDeleteTime, plan.Date, "09:00")
	if err != nil {
		t.Fatalf("failed to manually delete slot: %v", err)
	}

	// Now delete the entire plan (this should soft-delete the remaining slot with a different timestamp)
	if err := store.DeletePlan(plan.Date); err != nil {
		t.Fatalf("failed to delete plan: %v", err)
	}

	// Restore the plan
	if err := store.RestorePlan(plan.Date); err != nil {
		t.Fatalf("failed to restore plan: %v", err)
	}

	// Verify the plan is restored
	restoredPlan, err := store.GetPlan(plan.Date)
	if err != nil {
		t.Fatalf("failed to get restored plan: %v", err)
	}

	// The restored plan should only have 1 slot (the one that was deleted with the plan)
	// The slot that was individually deleted earlier should NOT be restored
	if len(restoredPlan.Slots) != 1 {
		t.Errorf("expected 1 slot after restore (only plan-level deletion), got %d", len(restoredPlan.Slots))
	}

	// Verify the restored slot is the correct one (10:00-10:30)
	if len(restoredPlan.Slots) == 1 {
		if restoredPlan.Slots[0].Start != "10:00" {
			t.Errorf("expected restored slot to start at 10:00, got %s", restoredPlan.Slots[0].Start)
		}
	}
}

// JSONStore Tests

func setupTestJSONStore(t *testing.T) (*JSONStore, func()) {
	// Create a temporary directory for test JSON file
	tempDir := t.TempDir()
	jsonPath := filepath.Join(tempDir, "test.json")

	// Create test store
	store := NewJSONStore(jsonPath)
	if err := store.Init(); err != nil {
		t.Fatalf("failed to initialize test JSON store: %v", err)
	}

	cleanup := func() {
		store.Close()
		os.RemoveAll(tempDir)
	}

	return store, cleanup
}

func TestJSONTaskSoftDelete(t *testing.T) {
	store, cleanup := setupTestJSONStore(t)
	defer cleanup()

	// Create a test task
	task := models.Task{
		ID:          "json-task-1",
		Name:        "JSON Test Task",
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
		if task.ID == "json-task-1" {
			t.Error("deleted task should not appear in GetAllTasks")
		}
	}
}

func TestJSONTaskRestore(t *testing.T) {
	store, cleanup := setupTestJSONStore(t)
	defer cleanup()

	// Create and add a test task
	task := models.Task{
		ID:          "json-task-2",
		Name:        "JSON Test Task 2",
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

func TestJSONPlanSoftDelete(t *testing.T) {
	store, cleanup := setupTestJSONStore(t)
	defer cleanup()

	// Create a task for the plan
	task := models.Task{
		ID:          "json-task-3",
		Name:        "JSON Test Task 3",
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

func TestJSONPlanRestore(t *testing.T) {
	store, cleanup := setupTestJSONStore(t)
	defer cleanup()

	// Create a task for the plan
	task := models.Task{
		ID:          "json-task-4",
		Name:        "JSON Test Task 4",
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

	// Create a test plan with slots
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

func TestJSONPlanDeleteCascadesToSlots(t *testing.T) {
	store, cleanup := setupTestJSONStore(t)
	defer cleanup()

	// Create a task
	task := models.Task{
		ID:          "json-task-5",
		Name:        "JSON Test Task 5",
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

	// Create a plan with multiple slots
	plan := models.DayPlan{
		Date: "2024-01-17",
		Slots: []models.Slot{
			{
				Start:  "09:00",
				End:    "09:30",
				TaskID: task.ID,
				Status: models.SlotStatusPlanned,
			},
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

	// Delete the plan
	if err := store.DeletePlan(plan.Date); err != nil {
		t.Fatalf("failed to delete plan: %v", err)
	}

	// Restore the plan
	if err := store.RestorePlan(plan.Date); err != nil {
		t.Fatalf("failed to restore plan: %v", err)
	}

	// Verify all slots are restored
	restoredPlan, err := store.GetPlan(plan.Date)
	if err != nil {
		t.Fatalf("failed to get restored plan: %v", err)
	}

	if len(restoredPlan.Slots) != 2 {
		t.Errorf("expected 2 slots after restore, got %d", len(restoredPlan.Slots))
	}

	// Verify all slots have nil DeletedAt
	for i, slot := range restoredPlan.Slots {
		if slot.DeletedAt != nil {
			t.Errorf("slot %d still has DeletedAt set after restore", i)
		}
	}
}
