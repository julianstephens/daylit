package validation

import (
	"testing"

	"github.com/julianstephens/daylit/internal/models"
)

func TestValidateTasks_DuplicateNames(t *testing.T) {
	validator := New()

	tasks := []models.Task{
		{ID: "1", Name: "Task A", Active: true, Kind: models.TaskKindFlexible},
		{ID: "2", Name: "Task B", Active: true, Kind: models.TaskKindFlexible},
		{ID: "3", Name: "Task A", Active: true, Kind: models.TaskKindFlexible}, // Duplicate
	}

	result := validator.ValidateTasks(tasks)

	if !result.HasConflicts() {
		t.Error("Expected to detect duplicate task names")
	}

	found := false
	for _, conflict := range result.Conflicts {
		if conflict.Type == ConflictDuplicateTaskName {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected ConflictDuplicateTaskName conflict type")
	}
}

func TestValidateTasks_InvalidTimeFormat(t *testing.T) {
	validator := New()

	tasks := []models.Task{
		{
			ID:            "1",
			Name:          "Task A",
			Active:        true,
			Kind:          models.TaskKindFlexible,
			EarliestStart: "25:00", // Invalid hour
		},
		{
			ID:        "2",
			Name:      "Task B",
			Active:    true,
			Kind:      models.TaskKindFlexible,
			LatestEnd: "12:70", // Invalid minute
		},
		{
			ID:         "3",
			Name:       "Task C",
			Active:     true,
			Kind:       models.TaskKindAppointment,
			FixedStart: "not-a-time", // Invalid format
		},
	}

	result := validator.ValidateTasks(tasks)

	if !result.HasConflicts() {
		t.Error("Expected to detect invalid time formats")
	}

	invalidTimeCount := 0
	for _, conflict := range result.Conflicts {
		if conflict.Type == ConflictInvalidDateTime {
			invalidTimeCount++
		}
	}
	if invalidTimeCount != 3 {
		t.Errorf("Expected 3 invalid time conflicts, got %d", invalidTimeCount)
	}
}

func TestValidateTasks_OverlappingFixedAppointments(t *testing.T) {
	validator := New()

	tasks := []models.Task{
		{
			ID:         "1",
			Name:       "Meeting 1",
			Active:     true,
			Kind:       models.TaskKindAppointment,
			FixedStart: "09:00",
			FixedEnd:   "10:00",
		},
		{
			ID:         "2",
			Name:       "Meeting 2",
			Active:     true,
			Kind:       models.TaskKindAppointment,
			FixedStart: "09:30",
			FixedEnd:   "10:30",
		},
	}

	result := validator.ValidateTasks(tasks)

	if !result.HasConflicts() {
		t.Error("Expected to detect overlapping fixed appointments")
	}

	found := false
	for _, conflict := range result.Conflicts {
		if conflict.Type == ConflictOverlappingFixedTasks {
			found = true
			if len(conflict.Items) != 2 {
				t.Errorf("Expected 2 items in conflict, got %d", len(conflict.Items))
			}
		}
	}
	if !found {
		t.Error("Expected ConflictOverlappingFixedTasks conflict type")
	}
}

func TestValidateTasks_NoConflicts(t *testing.T) {
	validator := New()

	tasks := []models.Task{
		{
			ID:     "1",
			Name:   "Task A",
			Active: true,
			Kind:   models.TaskKindFlexible,
		},
		{
			ID:         "2",
			Name:       "Task B",
			Active:     true,
			Kind:       models.TaskKindAppointment,
			FixedStart: "09:00",
			FixedEnd:   "10:00",
		},
		{
			ID:         "3",
			Name:       "Task C",
			Active:     true,
			Kind:       models.TaskKindAppointment,
			FixedStart: "11:00",
			FixedEnd:   "12:00",
		},
	}

	result := validator.ValidateTasks(tasks)

	if result.HasConflicts() {
		t.Errorf("Expected no conflicts, got: %s", result.FormatReport())
	}
}

func TestValidatePlan_OverlappingSlots(t *testing.T) {
	validator := New()

	tasks := []models.Task{
		{ID: "task1", Name: "Task 1", Active: true},
		{ID: "task2", Name: "Task 2", Active: true},
	}

	plan := models.DayPlan{
		Date: "2025-01-15",
		Slots: []models.Slot{
			{Start: "09:00", End: "10:00", TaskID: "task1", Status: models.SlotStatusPlanned},
			{Start: "09:30", End: "10:30", TaskID: "task2", Status: models.SlotStatusPlanned},
		},
	}

	result := validator.ValidatePlan(plan, tasks, "08:00", "18:00")

	if !result.HasConflicts() {
		t.Error("Expected to detect overlapping slots")
	}

	found := false
	for _, conflict := range result.Conflicts {
		if conflict.Type == ConflictOverlappingSlots {
			found = true
		}
	}
	if !found {
		t.Error("Expected ConflictOverlappingSlots conflict type")
	}
}

func TestValidatePlan_MissingTaskID(t *testing.T) {
	validator := New()

	tasks := []models.Task{
		{ID: "task1", Name: "Task 1", Active: true},
	}

	plan := models.DayPlan{
		Date: "2025-01-15",
		Slots: []models.Slot{
			{Start: "09:00", End: "10:00", TaskID: "task1", Status: models.SlotStatusPlanned},
			{Start: "10:00", End: "11:00", TaskID: "nonexistent", Status: models.SlotStatusPlanned},
		},
	}

	result := validator.ValidatePlan(plan, tasks, "08:00", "18:00")

	if !result.HasConflicts() {
		t.Error("Expected to detect missing task ID")
	}

	found := false
	for _, conflict := range result.Conflicts {
		if conflict.Type == ConflictMissingTaskID {
			found = true
		}
	}
	if !found {
		t.Error("Expected ConflictMissingTaskID conflict type")
	}
}

func TestValidatePlan_ExceedsWakingWindow(t *testing.T) {
	validator := New()

	tasks := []models.Task{
		{ID: "task1", Name: "Task 1", Active: true},
		{ID: "task2", Name: "Task 2", Active: true},
		{ID: "task3", Name: "Task 3", Active: true},
	}

	// Waking window is 08:00-18:00 (10 hours)
	// Schedule 11 hours of tasks
	plan := models.DayPlan{
		Date: "2025-01-15",
		Slots: []models.Slot{
			{Start: "08:00", End: "12:00", TaskID: "task1", Status: models.SlotStatusPlanned}, // 4h
			{Start: "12:00", End: "16:00", TaskID: "task2", Status: models.SlotStatusPlanned}, // 4h
			{Start: "16:00", End: "19:00", TaskID: "task3", Status: models.SlotStatusPlanned}, // 3h (exceeds 18:00)
		},
	}

	result := validator.ValidatePlan(plan, tasks, "08:00", "18:00")

	if !result.HasConflicts() {
		t.Error("Expected to detect plan exceeding waking window")
	}

	found := false
	for _, conflict := range result.Conflicts {
		if conflict.Type == ConflictExceedsWakingWindow {
			found = true
		}
	}
	if !found {
		t.Error("Expected ConflictExceedsWakingWindow conflict type")
	}
}

func TestValidatePlan_Overcommitted(t *testing.T) {
	validator := New()

	tasks := []models.Task{
		{ID: "task1", Name: "Task 1", Active: true},
		{ID: "task2", Name: "Task 2", Active: true},
	}

	// Waking window is 08:00-18:00 (10 hours = 600 minutes)
	// Schedule 540 minutes (9 hours = 90% of capacity)
	plan := models.DayPlan{
		Date: "2025-01-15",
		Slots: []models.Slot{
			{Start: "08:00", End: "12:30", TaskID: "task1", Status: models.SlotStatusPlanned}, // 4.5h
			{Start: "12:30", End: "17:00", TaskID: "task2", Status: models.SlotStatusPlanned}, // 4.5h
		},
	}

	result := validator.ValidatePlan(plan, tasks, "08:00", "18:00")

	if !result.HasConflicts() {
		t.Error("Expected to detect overcommitted plan")
	}

	found := false
	for _, conflict := range result.Conflicts {
		if conflict.Type == ConflictOvercommitted {
			found = true
		}
	}
	if !found {
		t.Error("Expected ConflictOvercommitted conflict type")
	}
}

func TestValidatePlan_InvalidDate(t *testing.T) {
	validator := New()

	tasks := []models.Task{
		{ID: "task1", Name: "Task 1", Active: true},
	}

	plan := models.DayPlan{
		Date: "invalid-date",
		Slots: []models.Slot{
			{Start: "09:00", End: "10:00", TaskID: "task1", Status: models.SlotStatusPlanned},
		},
	}

	result := validator.ValidatePlan(plan, tasks, "08:00", "18:00")

	if !result.HasConflicts() {
		t.Error("Expected to detect invalid date")
	}

	found := false
	for _, conflict := range result.Conflicts {
		if conflict.Type == ConflictInvalidDateTime {
			found = true
		}
	}
	if !found {
		t.Error("Expected ConflictInvalidDateTime conflict type")
	}
}

func TestValidatePlan_NoConflicts(t *testing.T) {
	validator := New()

	tasks := []models.Task{
		{ID: "task1", Name: "Task 1", Active: true},
		{ID: "task2", Name: "Task 2", Active: true},
	}

	plan := models.DayPlan{
		Date: "2025-01-15",
		Slots: []models.Slot{
			{Start: "09:00", End: "10:00", TaskID: "task1", Status: models.SlotStatusPlanned},
			{Start: "10:00", End: "11:00", TaskID: "task2", Status: models.SlotStatusPlanned},
		},
	}

	result := validator.ValidatePlan(plan, tasks, "08:00", "18:00")

	if result.HasConflicts() {
		t.Errorf("Expected no conflicts, got: %s", result.FormatReport())
	}
}

func TestTimesOverlap(t *testing.T) {
	tests := []struct {
		name   string
		start1 string
		end1   string
		start2 string
		end2   string
		want   bool
	}{
		{"Completely separate", "09:00", "10:00", "11:00", "12:00", false},
		{"Adjacent (no overlap)", "09:00", "10:00", "10:00", "11:00", false},
		{"Partial overlap", "09:00", "10:00", "09:30", "10:30", true},
		{"Complete overlap", "09:00", "11:00", "09:30", "10:30", true},
		{"Same times", "09:00", "10:00", "09:00", "10:00", true},
		{"Reverse order", "11:00", "12:00", "09:00", "10:00", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := timesOverlap(tt.start1, tt.end1, tt.start2, tt.end2)
			if got != tt.want {
				t.Errorf("timesOverlap(%s, %s, %s, %s) = %v, want %v",
					tt.start1, tt.end1, tt.start2, tt.end2, got, tt.want)
			}
		})
	}
}

func TestValidationResult_FormatReport(t *testing.T) {
	result := ValidationResult{
		Conflicts: []Conflict{
			{
				Type:        ConflictOverlappingSlots,
				Description: "Mon: 09:00-10:00 \"Task A\" overlaps \"Task B\"",
			},
			{
				Type:        ConflictExceedsWakingWindow,
				Description: "Mon: 11.0h scheduled exceeds 10.0h waking window",
			},
		},
	}

	report := result.FormatReport()
	if report == "" {
		t.Error("Expected non-empty report")
	}
	if report == "No conflicts detected." {
		t.Error("Expected conflicts in report")
	}
}

func TestValidationResult_FormatReport_NoConflicts(t *testing.T) {
	result := ValidationResult{Conflicts: []Conflict{}}

	report := result.FormatReport()
	if report != "No conflicts detected." {
		t.Errorf("Expected 'No conflicts detected.', got: %s", report)
	}
}

func TestValidateTasks_SkipsDeletedTasks(t *testing.T) {
	validator := New()

	deleted := "2025-01-15T10:00:00Z"
	tasks := []models.Task{
		{ID: "1", Name: "Task A", Active: true, Kind: models.TaskKindFlexible},
		{ID: "2", Name: "Task A", Active: true, Kind: models.TaskKindFlexible, DeletedAt: &deleted}, // Deleted duplicate
	}

	result := validator.ValidateTasks(tasks)

	// Should not report duplicate since one is deleted
	if result.HasConflicts() {
		t.Errorf("Expected no conflicts (deleted tasks should be skipped), got: %s", result.FormatReport())
	}
}

func TestValidatePlan_SkipsDeletedSlots(t *testing.T) {
	validator := New()

	tasks := []models.Task{
		{ID: "task1", Name: "Task 1", Active: true},
		{ID: "task2", Name: "Task 2", Active: true},
	}

	deleted := "2025-01-15T10:00:00Z"
	plan := models.DayPlan{
		Date: "2025-01-15",
		Slots: []models.Slot{
			{Start: "09:00", End: "10:00", TaskID: "task1", Status: models.SlotStatusPlanned},
			{Start: "09:30", End: "10:30", TaskID: "task2", Status: models.SlotStatusPlanned, DeletedAt: &deleted}, // Deleted overlap
		},
	}

	result := validator.ValidatePlan(plan, tasks, "08:00", "18:00")

	// Should not report overlap since one slot is deleted
	if result.HasConflicts() {
		t.Errorf("Expected no conflicts (deleted slots should be skipped), got: %s", result.FormatReport())
	}
}
