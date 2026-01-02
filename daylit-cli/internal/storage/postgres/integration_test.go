package postgres

import (
	"os"
	"testing"
	"time"

	"github.com/julianstephens/daylit/daylit-cli/internal/constants"
	"github.com/julianstephens/daylit/daylit-cli/internal/models"
)

// TestStore_Integration tests PostgreSQL store with a real database
// Set POSTGRES_TEST_URL environment variable to run this test
// Example: POSTGRES_TEST_URL="postgres://daylit_user:password@localhost:5432/daylit_test?sslmode=disable"
func TestStore_Integration(t *testing.T) {
	connStr := os.Getenv("POSTGRES_TEST_URL")
	if connStr == "" {
		t.Skip("POSTGRES_TEST_URL not set, skipping PostgreSQL integration test")
	}

	// Create a new PostgreSQL store
	store := New(connStr)

	// Initialize the store
	if err := store.Init(); err != nil {
		t.Fatalf("Failed to initialize store: %v", err)
	}
	defer store.Close()

	// Test Settings
	t.Run("Settings", func(t *testing.T) {
		settings, err := store.GetSettings()
		if err != nil {
			t.Fatalf("Failed to get settings: %v", err)
		}

		// Verify default settings were created
		if settings.DayStart != constants.DefaultDayStart {
			t.Errorf("Expected day start %s, got %s", constants.DefaultDayStart, settings.DayStart)
		}

		// Update settings
		settings.DayStart = "08:00"
		if err := store.SaveSettings(settings); err != nil {
			t.Fatalf("Failed to save settings: %v", err)
		}

		// Verify update
		updated, err := store.GetSettings()
		if err != nil {
			t.Fatalf("Failed to get updated settings: %v", err)
		}
		if updated.DayStart != "08:00" {
			t.Errorf("Expected day start 08:00, got %s", updated.DayStart)
		}
	})

	// Test Tasks
	t.Run("Tasks", func(t *testing.T) {
		task := models.Task{
			ID:          "test-task-pg-1",
			Name:        "Test PostgreSQL Task",
			Kind:        "flexible",
			DurationMin: 30,
			Priority:    5,
			EnergyBand:  "high",
			Active:      true,
			Recurrence:  models.Recurrence{Type: "none"},
		}

		// Add task
		if err := store.AddTask(task); err != nil {
			t.Fatalf("Failed to add task: %v", err)
		}

		// Get task
		retrieved, err := store.GetTask(task.ID)
		if err != nil {
			t.Fatalf("Failed to get task: %v", err)
		}
		if retrieved.Name != task.Name {
			t.Errorf("Expected task name %s, got %s", task.Name, retrieved.Name)
		}

		// Update task
		task.Name = "Updated PostgreSQL Task"
		if err := store.UpdateTask(task); err != nil {
			t.Fatalf("Failed to update task: %v", err)
		}

		// Verify update
		updated, err := store.GetTask(task.ID)
		if err != nil {
			t.Fatalf("Failed to get updated task: %v", err)
		}
		if updated.Name != task.Name {
			t.Errorf("Expected task name %s, got %s", task.Name, updated.Name)
		}

		// Delete task
		if err := store.DeleteTask(task.ID); err != nil {
			t.Fatalf("Failed to delete task: %v", err)
		}

		// Verify deletion
		_, err = store.GetTask(task.ID)
		if err == nil {
			t.Error("Expected error when getting deleted task")
		}

		// Restore task
		if err := store.RestoreTask(task.ID); err != nil {
			t.Fatalf("Failed to restore task: %v", err)
		}

		// Verify restoration
		restored, err := store.GetTask(task.ID)
		if err != nil {
			t.Fatalf("Failed to get restored task: %v", err)
		}
		if restored.Name != task.Name {
			t.Errorf("Expected task name %s, got %s", task.Name, restored.Name)
		}
	})

	// Test Plans
	t.Run("Plans", func(t *testing.T) {
		// First create a task to use in the plan
		task := models.Task{
			ID:          "test-task-for-plan-pg",
			Name:        "Task for Plan",
			Kind:        "flexible",
			DurationMin: 60,
			Active:      true,
			Recurrence:  models.Recurrence{Type: "none"},
		}
		if err := store.AddTask(task); err != nil {
			t.Fatalf("Failed to add task for plan: %v", err)
		}

		// Create a plan
		plan := models.DayPlan{
			Date: "2026-01-15",
			Slots: []models.Slot{
				{
					Start:  "09:00",
					End:    "10:00",
					TaskID: task.ID,
					Status: "scheduled",
				},
			},
		}

		// Save plan
		if err := store.SavePlan(plan); err != nil {
			t.Fatalf("Failed to save plan: %v", err)
		}

		// Get plan
		retrieved, err := store.GetPlan("2026-01-15")
		if err != nil {
			t.Fatalf("Failed to get plan: %v", err)
		}
		if len(retrieved.Slots) != 1 {
			t.Errorf("Expected 1 slot, got %d", len(retrieved.Slots))
		}
		if retrieved.Slots[0].TaskID != task.ID {
			t.Errorf("Expected task ID %s, got %s", task.ID, retrieved.Slots[0].TaskID)
		}

		// Delete plan
		if err := store.DeletePlan("2026-01-15"); err != nil {
			t.Fatalf("Failed to delete plan: %v", err)
		}

		// Verify deletion
		_, err = store.GetPlan("2026-01-15")
		if err == nil {
			t.Error("Expected error when getting deleted plan")
		}

		// Restore plan
		if err := store.RestorePlan("2026-01-15"); err != nil {
			t.Fatalf("Failed to restore plan: %v", err)
		}

		// Verify restoration
		restored, err := store.GetPlan("2026-01-15")
		if err != nil {
			t.Fatalf("Failed to get restored plan: %v", err)
		}
		if len(restored.Slots) != 1 {
			t.Errorf("Expected 1 slot after restoration, got %d", len(restored.Slots))
		}
	})

	// Test Habits
	t.Run("Habits", func(t *testing.T) {
		habit := models.Habit{
			ID:        "test-habit-pg-1",
			Name:      "Test PostgreSQL Habit",
			CreatedAt: time.Now(),
		}

		// Add habit
		if err := store.AddHabit(habit); err != nil {
			t.Fatalf("Failed to add habit: %v", err)
		}

		// Get habit
		retrieved, err := store.GetHabit(habit.ID)
		if err != nil {
			t.Fatalf("Failed to get habit: %v", err)
		}
		if retrieved.Name != habit.Name {
			t.Errorf("Expected habit name %s, got %s", habit.Name, retrieved.Name)
		}

		// Archive habit
		if err := store.ArchiveHabit(habit.ID); err != nil {
			t.Fatalf("Failed to archive habit: %v", err)
		}

		// Verify archiving
		archived, err := store.GetHabit(habit.ID)
		if err != nil {
			t.Fatalf("Failed to get archived habit: %v", err)
		}
		if archived.ArchivedAt == nil {
			t.Error("Expected habit to be archived")
		}

		// Unarchive habit
		if err := store.UnarchiveHabit(habit.ID); err != nil {
			t.Fatalf("Failed to unarchive habit: %v", err)
		}

		// Verify unarchiving
		unarchived, err := store.GetHabit(habit.ID)
		if err != nil {
			t.Fatalf("Failed to get unarchived habit: %v", err)
		}
		if unarchived.ArchivedAt != nil {
			t.Error("Expected habit to not be archived")
		}
	})

	t.Log("All PostgreSQL integration tests passed!")
}
