package storage

import (
	"testing"
	"time"

	"github.com/julianstephens/daylit/daylit-cli/internal/models"
)

// Test that accepting a plan creates revision 1
func TestPlanAcceptCreatesRevision1(t *testing.T) {
	store, cleanup := setupTestSQLiteStore(t)
	defer cleanup()

	// Create a task
	task := models.Task{
		ID:          "task-rev-1",
		Name:        "Test Task",
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
	now := time.Now().UTC().Format(time.RFC3339)
	plan := models.DayPlan{
		Date:       "2024-03-01",
		Revision:   0, // Let it auto-assign
		AcceptedAt: &now,
		Slots: []models.Slot{
			{
				Start:  "09:00",
				End:    "09:30",
				TaskID: task.ID,
				Status: models.SlotStatusAccepted,
			},
		},
	}

	if err := store.SavePlan(plan); err != nil {
		t.Fatalf("failed to save plan: %v", err)
	}

	// Retrieve the plan and verify revision is 1
	retrieved, err := store.GetPlan(plan.Date)
	if err != nil {
		t.Fatalf("failed to get plan: %v", err)
	}

	if retrieved.Revision != 1 {
		t.Errorf("expected revision 1, got %d", retrieved.Revision)
	}

	if retrieved.AcceptedAt == nil {
		t.Error("expected plan to be accepted")
	}
}

// Test that regenerating an accepted plan creates revision 2
func TestRegenerateAcceptedPlanCreatesRevision2(t *testing.T) {
	store, cleanup := setupTestSQLiteStore(t)
	defer cleanup()

	// Create a task
	task := models.Task{
		ID:          "task-rev-2",
		Name:        "Test Task",
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

	// Create and save accepted plan (revision 1)
	now := time.Now().UTC().Format(time.RFC3339)
	plan1 := models.DayPlan{
		Date:       "2024-03-02",
		Revision:   0,
		AcceptedAt: &now,
		Slots: []models.Slot{
			{
				Start:  "09:00",
				End:    "09:30",
				TaskID: task.ID,
				Status: models.SlotStatusAccepted,
			},
		},
	}

	if err := store.SavePlan(plan1); err != nil {
		t.Fatalf("failed to save plan1: %v", err)
	}

	// Create and save a new plan (should be revision 2)
	plan2 := models.DayPlan{
		Date:       "2024-03-02",
		Revision:   0, // Let it auto-assign
		AcceptedAt: nil,
		Slots: []models.Slot{
			{
				Start:  "10:00",
				End:    "10:30",
				TaskID: task.ID,
				Status: models.SlotStatusPlanned,
			},
		},
	}

	if err := store.SavePlan(plan2); err != nil {
		t.Fatalf("failed to save plan2: %v", err)
	}

	// Retrieve latest plan and verify it's revision 2
	latest, err := store.GetPlan(plan2.Date)
	if err != nil {
		t.Fatalf("failed to get latest plan: %v", err)
	}

	if latest.Revision != 2 {
		t.Errorf("expected revision 2, got %d", latest.Revision)
	}

	if latest.AcceptedAt != nil {
		t.Error("expected plan2 to not be accepted")
	}

	// Verify revision 1 is still intact
	rev1, err := store.GetPlanRevision(plan1.Date, 1)
	if err != nil {
		t.Fatalf("failed to get revision 1: %v", err)
	}

	if rev1.Revision != 1 {
		t.Errorf("expected revision 1, got %d", rev1.Revision)
	}

	if rev1.AcceptedAt == nil {
		t.Error("expected revision 1 to still be accepted")
	}

	if len(rev1.Slots) != 1 || rev1.Slots[0].Start != "09:00" {
		t.Error("revision 1 data was modified")
	}
}

// Test that regenerating an unaccepted plan overwrites it
func TestRegenerateUnacceptedPlanOverwrites(t *testing.T) {
	store, cleanup := setupTestSQLiteStore(t)
	defer cleanup()

	// Create a task
	task := models.Task{
		ID:          "task-rev-3",
		Name:        "Test Task",
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

	// Create and save unaccepted plan
	plan1 := models.DayPlan{
		Date:       "2024-03-03",
		Revision:   0,
		AcceptedAt: nil,
		Slots: []models.Slot{
			{
				Start:  "09:00",
				End:    "09:30",
				TaskID: task.ID,
				Status: models.SlotStatusPlanned,
			},
		},
	}

	if err := store.SavePlan(plan1); err != nil {
		t.Fatalf("failed to save plan1: %v", err)
	}

	// Regenerate plan (should overwrite revision 1)
	plan2 := models.DayPlan{
		Date:       "2024-03-03",
		Revision:   0,
		AcceptedAt: nil,
		Slots: []models.Slot{
			{
				Start:  "10:00",
				End:    "10:30",
				TaskID: task.ID,
				Status: models.SlotStatusPlanned,
			},
		},
	}

	if err := store.SavePlan(plan2); err != nil {
		t.Fatalf("failed to save plan2: %v", err)
	}

	// Retrieve plan and verify it's still revision 1 but with new data
	latest, err := store.GetPlan(plan2.Date)
	if err != nil {
		t.Fatalf("failed to get latest plan: %v", err)
	}

	if latest.Revision != 1 {
		t.Errorf("expected revision 1 (overwritten), got %d", latest.Revision)
	}

	if len(latest.Slots) != 1 || latest.Slots[0].Start != "10:00" {
		t.Error("plan was not overwritten correctly")
	}
}

// Test GetLatestPlanRevision returns the latest non-deleted revision
func TestGetLatestPlanRevision(t *testing.T) {
	store, cleanup := setupTestSQLiteStore(t)
	defer cleanup()

	// Create a task
	task := models.Task{
		ID:          "task-rev-4",
		Name:        "Test Task",
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

	// Create multiple accepted revisions using Revision=0 to test auto-assignment
	for i := 1; i <= 3; i++ {
		now := time.Now().UTC().Format(time.RFC3339)
		plan := models.DayPlan{
			Date:       "2024-03-04",
			Revision:   0, // Let SavePlan auto-assign
			AcceptedAt: &now,
			Slots: []models.Slot{
				{
					Start:  "09:00",
					End:    "09:30",
					TaskID: task.ID,
					Status: models.SlotStatusAccepted,
				},
			},
		}
		if err := store.SavePlan(plan); err != nil {
			t.Fatalf("failed to save plan revision %d: %v", i, err)
		}
	}

	// Get latest revision
	latest, err := store.GetLatestPlanRevision("2024-03-04")
	if err != nil {
		t.Fatalf("failed to get latest revision: %v", err)
	}

	if latest.Revision != 3 {
		t.Errorf("expected latest revision 3, got %d", latest.Revision)
	}
}

// Test feedback attaches to the correct (latest) revision
func TestFeedbackAttachesToLatestRevision(t *testing.T) {
	store, cleanup := setupTestSQLiteStore(t)
	defer cleanup()

	// Create a task
	task := models.Task{
		ID:          "task-rev-5",
		Name:        "Test Task",
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

	// Create revision 1
	now1 := time.Now().UTC().Format(time.RFC3339)
	plan1 := models.DayPlan{
		Date:       "2024-03-05",
		Revision:   0, // Use auto-assignment
		AcceptedAt: &now1,
		Slots: []models.Slot{
			{
				Start:  "09:00",
				End:    "09:30",
				TaskID: task.ID,
				Status: models.SlotStatusAccepted,
			},
		},
	}
	if err := store.SavePlan(plan1); err != nil {
		t.Fatalf("failed to save plan1: %v", err)
	}

	// Create revision 2 with a later timestamp to simulate real-world usage
	time.Sleep(10 * time.Millisecond) // Ensure different timestamp
	now2 := time.Now().UTC().Format(time.RFC3339)
	plan2 := models.DayPlan{
		Date:       "2024-03-05",
		Revision:   0, // Use auto-assignment to create new revision
		AcceptedAt: &now2,
		Slots: []models.Slot{
			{
				Start:  "10:00",
				End:    "10:30",
				TaskID: task.ID,
				Status: models.SlotStatusAccepted,
			},
		},
	}
	if err := store.SavePlan(plan2); err != nil {
		t.Fatalf("failed to save plan2: %v", err)
	}

	// Get latest plan and add feedback
	latest, err := store.GetPlan("2024-03-05")
	if err != nil {
		t.Fatalf("failed to get latest plan: %v", err)
	}

	if latest.Revision != 2 {
		t.Fatalf("expected latest revision 2, got %d", latest.Revision)
	}

	// Add feedback to the latest revision
	latest.Slots[0].Feedback = &models.Feedback{
		Rating: models.FeedbackOnTrack,
		Note:   "Great!",
	}
	latest.Slots[0].Status = models.SlotStatusDone

	if err := store.SavePlan(latest); err != nil {
		t.Fatalf("failed to save plan with feedback: %v", err)
	}

	// Verify feedback is on revision 2
	rev2, err := store.GetPlanRevision("2024-03-05", 2)
	if err != nil {
		t.Fatalf("failed to get revision 2: %v", err)
	}

	if rev2.Slots[0].Feedback == nil {
		t.Fatal("feedback not saved on revision 2")
	}

	if rev2.Slots[0].Feedback.Rating != models.FeedbackOnTrack {
		t.Error("feedback rating incorrect")
	}

	// Verify revision 1 is unchanged (no feedback)
	rev1, err := store.GetPlanRevision("2024-03-05", 1)
	if err != nil {
		t.Fatalf("failed to get revision 1: %v", err)
	}

	if rev1.Slots[0].Feedback != nil {
		t.Error("feedback should not be on revision 1")
	}
}

// Test JSONStore revision functionality
func TestJSONStorePlanRevisions(t *testing.T) {
	store, cleanup := setupTestJSONStore(t)
	defer cleanup()

	// Create a task
	task := models.Task{
		ID:          "json-task-rev",
		Name:        "JSON Test Task",
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

	// Create and save accepted plan (revision 1)
	now := time.Now().UTC().Format(time.RFC3339)
	plan1 := models.DayPlan{
		Date:       "2024-03-10",
		Revision:   0,
		AcceptedAt: &now,
		Slots: []models.Slot{
			{
				Start:  "09:00",
				End:    "09:30",
				TaskID: task.ID,
				Status: models.SlotStatusAccepted,
			},
		},
	}

	if err := store.SavePlan(plan1); err != nil {
		t.Fatalf("failed to save plan1: %v", err)
	}

	// Create revision 2
	plan2 := models.DayPlan{
		Date:       "2024-03-10",
		Revision:   0,
		AcceptedAt: nil,
		Slots: []models.Slot{
			{
				Start:  "10:00",
				End:    "10:30",
				TaskID: task.ID,
				Status: models.SlotStatusPlanned,
			},
		},
	}

	if err := store.SavePlan(plan2); err != nil {
		t.Fatalf("failed to save plan2: %v", err)
	}

	// Get latest (should be revision 2)
	latest, err := store.GetLatestPlanRevision("2024-03-10")
	if err != nil {
		t.Fatalf("failed to get latest: %v", err)
	}

	if latest.Revision != 2 {
		t.Errorf("expected revision 2, got %d", latest.Revision)
	}

	// Get specific revision 1
	rev1, err := store.GetPlanRevision("2024-03-10", 1)
	if err != nil {
		t.Fatalf("failed to get revision 1: %v", err)
	}

	if rev1.Revision != 1 {
		t.Errorf("expected revision 1, got %d", rev1.Revision)
	}

	if len(rev1.Slots) != 1 || rev1.Slots[0].Start != "09:00" {
		t.Error("revision 1 data incorrect")
	}
}
