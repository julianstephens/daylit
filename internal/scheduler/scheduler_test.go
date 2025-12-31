package scheduler

import (
	"testing"
	"time"

	"github.com/julianstephens/daylit/internal/models"
)

func TestGeneratePlan_RespectsWeekdaysForAppointments(t *testing.T) {
	// Setup
	scheduler := New()

	// Wednesday date
	dateStr := "2025-12-31" // Dec 31 2025 is a Wednesday

	tasks := []models.Task{
		{
			ID:         "task-sat",
			Name:       "Saturday Task",
			Kind:       models.TaskKindAppointment,
			FixedStart: "09:00",
			FixedEnd:   "10:00",
			Active:     true,
			Recurrence: models.Recurrence{
				Type:        models.RecurrenceWeekly,
				WeekdayMask: []time.Weekday{time.Saturday},
			},
		},
		{
			ID:         "task-wed",
			Name:       "Wednesday Task",
			Kind:       models.TaskKindAppointment,
			FixedStart: "10:00",
			FixedEnd:   "11:00",
			Active:     true,
			Recurrence: models.Recurrence{
				Type:        models.RecurrenceWeekly,
				WeekdayMask: []time.Weekday{time.Wednesday},
			},
		},
	}

	// Execute
	plan, err := scheduler.GeneratePlan(dateStr, tasks, "08:00", "18:00")
	if err != nil {
		t.Fatalf("GeneratePlan failed: %v", err)
	}

	// Assert
	foundSat := false
	foundWed := false

	for _, slot := range plan.Slots {
		if slot.TaskID == "task-sat" {
			foundSat = true
		}
		if slot.TaskID == "task-wed" {
			foundWed = true
		}
	}

	if foundSat {
		t.Errorf("Expected Saturday task to be excluded from Wednesday plan, but it was found")
	}

	if !foundWed {
		t.Errorf("Expected Wednesday task to be included in Wednesday plan, but it was missing")
	}
}

func TestGeneratePlan_FlexibleTaskRecurrence(t *testing.T) {
	scheduler := New()
	dateStr := "2025-12-31" // Wednesday

	tasks := []models.Task{
		{
			ID:          "flex-mon",
			Name:        "Monday Flex",
			Kind:        models.TaskKindFlexible,
			DurationMin: 60,
			Active:      true,
			Recurrence: models.Recurrence{
				Type:        models.RecurrenceWeekly,
				WeekdayMask: []time.Weekday{time.Monday},
			},
		},
		{
			ID:          "flex-wed",
			Name:        "Wednesday Flex",
			Kind:        models.TaskKindFlexible,
			DurationMin: 60,
			Active:      true,
			Recurrence: models.Recurrence{
				Type:        models.RecurrenceWeekly,
				WeekdayMask: []time.Weekday{time.Wednesday},
			},
		},
	}

	plan, err := scheduler.GeneratePlan(dateStr, tasks, "09:00", "17:00")
	if err != nil {
		t.Fatalf("GeneratePlan failed: %v", err)
	}

	foundMon := false
	foundWed := false
	for _, slot := range plan.Slots {
		if slot.TaskID == "flex-mon" {
			foundMon = true
		}
		if slot.TaskID == "flex-wed" {
			foundWed = true
		}
	}

	if foundMon {
		t.Errorf("Expected Monday flexible task to be excluded, but found")
	}
	if !foundWed {
		t.Errorf("Expected Wednesday flexible task to be included, but missing")
	}
}

func TestGeneratePlan_NDaysRecurrence(t *testing.T) {
	scheduler := New()
	dateStr := "2025-12-31" // Target date

	tasks := []models.Task{
		{
			ID:          "task-due",
			Name:        "Due Task",
			Kind:        models.TaskKindFlexible,
			DurationMin: 60,
			Active:      true,
			LastDone:    "2025-12-25", // 6 days ago
			Recurrence: models.Recurrence{
				Type:         models.RecurrenceNDays,
				IntervalDays: 5, // Should be due
			},
		},
		{
			ID:          "task-not-due",
			Name:        "Not Due Task",
			Kind:        models.TaskKindFlexible,
			DurationMin: 60,
			Active:      true,
			LastDone:    "2025-12-30", // 1 day ago
			Recurrence: models.Recurrence{
				Type:         models.RecurrenceNDays,
				IntervalDays: 5, // Not due yet
			},
		},
	}

	plan, err := scheduler.GeneratePlan(dateStr, tasks, "09:00", "17:00")
	if err != nil {
		t.Fatalf("GeneratePlan failed: %v", err)
	}

	foundDue := false
	foundNotDue := false
	for _, slot := range plan.Slots {
		if slot.TaskID == "task-due" {
			foundDue = true
		}
		if slot.TaskID == "task-not-due" {
			foundNotDue = true
		}
	}

	if !foundDue {
		t.Errorf("Expected due task to be scheduled")
	}
	if foundNotDue {
		t.Errorf("Expected not-due task to be skipped")
	}
}

func TestGeneratePlan_TimeConstraints(t *testing.T) {
	scheduler := New()
	dateStr := "2025-12-31"

	tasks := []models.Task{
		{
			ID:            "early-bird",
			Name:          "Early Bird",
			Kind:          models.TaskKindFlexible,
			DurationMin:   60,
			Active:        true,
			EarliestStart: "06:00",
			LatestEnd:     "08:00", // Must be done by 8am
			Recurrence:    models.Recurrence{Type: models.RecurrenceDaily},
		},
		{
			ID:            "night-owl",
			Name:          "Night Owl",
			Kind:          models.TaskKindFlexible,
			DurationMin:   60,
			Active:        true,
			EarliestStart: "20:00", // Starts after work day
			Recurrence:    models.Recurrence{Type: models.RecurrenceDaily},
		},
	}

	// Day is 09:00 - 17:00
	plan, err := scheduler.GeneratePlan(dateStr, tasks, "09:00", "17:00")
	if err != nil {
		t.Fatalf("GeneratePlan failed: %v", err)
	}

	for _, slot := range plan.Slots {
		if slot.TaskID == "early-bird" {
			t.Errorf("Early bird task scheduled despite being outside day window (latest end 08:00 < day start 09:00)")
		}
		if slot.TaskID == "night-owl" {
			t.Errorf("Night owl task scheduled despite being outside day window (earliest start 20:00 > day end 17:00)")
		}
	}

	// Now try with a wider window that fits "early-bird"
	plan, err = scheduler.GeneratePlan(dateStr, tasks, "07:00", "17:00")
	if err != nil {
		t.Fatalf("GeneratePlan failed: %v", err)
	}

	foundEarly := false
	for _, slot := range plan.Slots {
		if slot.TaskID == "early-bird" {
			foundEarly = true
			if slot.Start < "07:00" {
				t.Errorf("Task scheduled before day start")
			}
			if slot.End > "08:00" {
				t.Errorf("Task scheduled after latest end")
			}
		}
	}
	if !foundEarly {
		t.Errorf("Expected early bird task to be scheduled in wider window")
	}
}

func TestGeneratePlan_PriorityAndLateness(t *testing.T) {
	scheduler := New()
	dateStr := "2025-12-31"

	// Create 3 tasks, each 2 hours long. Day is 4 hours long (09:00-13:00).
	// Only 2 tasks can fit.
	tasks := []models.Task{
		{
			ID:          "prio-1",
			Name:        "High Priority",
			Kind:        models.TaskKindFlexible,
			DurationMin: 120,
			Active:      true,
			Priority:    1, // Highest
			Recurrence:  models.Recurrence{Type: models.RecurrenceDaily},
		},
		{
			ID:          "prio-2-late",
			Name:        "Medium Priority Late",
			Kind:        models.TaskKindFlexible,
			DurationMin: 120,
			Active:      true,
			Priority:    2,
			LastDone:    "2025-12-01", // Very late
			Recurrence:  models.Recurrence{Type: models.RecurrenceNDays, IntervalDays: 1},
		},
		{
			ID:          "prio-2-recent",
			Name:        "Medium Priority Recent",
			Kind:        models.TaskKindFlexible,
			DurationMin: 120,
			Active:      true,
			Priority:    2,
			LastDone:    "2025-12-30", // Recently done
			Recurrence:  models.Recurrence{Type: models.RecurrenceNDays, IntervalDays: 1},
		},
	}

	plan, err := scheduler.GeneratePlan(dateStr, tasks, "09:00", "13:00")
	if err != nil {
		t.Fatalf("GeneratePlan failed: %v", err)
	}

	if len(plan.Slots) != 2 {
		t.Fatalf("Expected exactly 2 slots, got %d", len(plan.Slots))
	}

	// First slot should be High Priority
	if plan.Slots[0].TaskID != "prio-1" {
		t.Errorf("Expected first slot to be High Priority (prio-1), got %s", plan.Slots[0].TaskID)
	}

	// Second slot should be the Late Medium Priority task
	if plan.Slots[1].TaskID != "prio-2-late" {
		t.Errorf("Expected second slot to be Late Medium Priority (prio-2-late), got %s", plan.Slots[1].TaskID)
	}
}

func TestGeneratePlan_MixedScheduling(t *testing.T) {
	scheduler := New()
	dateStr := "2025-12-31"

	tasks := []models.Task{
		{
			ID:         "fixed-lunch",
			Name:       "Lunch",
			Kind:       models.TaskKindAppointment,
			FixedStart: "12:00",
			FixedEnd:   "13:00",
			Active:     true,
			Recurrence: models.Recurrence{Type: models.RecurrenceDaily},
		},
		{
			ID:          "flex-morning",
			Name:        "Morning Work",
			Kind:        models.TaskKindFlexible,
			DurationMin: 120, // 2 hours
			Active:      true,
			Recurrence:  models.Recurrence{Type: models.RecurrenceDaily},
		},
		{
			ID:          "flex-afternoon",
			Name:        "Afternoon Work",
			Kind:        models.TaskKindFlexible,
			DurationMin: 120, // 2 hours
			Active:      true,
			Recurrence:  models.Recurrence{Type: models.RecurrenceDaily},
		},
	}

	// Day: 09:00 - 17:00
	// Lunch: 12:00 - 13:00
	// Morning gap: 09:00 - 12:00 (3 hours) -> Fits Morning Work (2h)
	// Afternoon gap: 13:00 - 17:00 (4 hours) -> Fits Afternoon Work (2h)

	plan, err := scheduler.GeneratePlan(dateStr, tasks, "09:00", "17:00")
	if err != nil {
		t.Fatalf("GeneratePlan failed: %v", err)
	}

	if len(plan.Slots) != 3 {
		t.Fatalf("Expected 3 slots, got %d", len(plan.Slots))
	}

	// Verify order
	if plan.Slots[0].TaskID != "flex-morning" && plan.Slots[0].TaskID != "flex-afternoon" {
		t.Errorf("Expected flexible task in first slot")
	}

	// Check if lunch is in the middle (or at its fixed time)
	lunchFound := false
	for _, slot := range plan.Slots {
		if slot.TaskID == "fixed-lunch" {
			lunchFound = true
			if slot.Start != "12:00" || slot.End != "13:00" {
				t.Errorf("Lunch scheduled at wrong time: %s-%s", slot.Start, slot.End)
			}
		}
	}
	if !lunchFound {
		t.Errorf("Lunch not found in plan")
	}
}

func TestGeneratePlan_EdgeCases(t *testing.T) {
	scheduler := New()
	dateStr := "2025-12-31"

	tasks := []models.Task{
		{
			ID:          "zero-duration",
			Name:        "Zero Duration",
			Kind:        models.TaskKindFlexible,
			DurationMin: 0,
			Active:      true,
			Recurrence:  models.Recurrence{Type: models.RecurrenceDaily},
		},
		{
			ID:          "too-long",
			Name:        "Too Long",
			Kind:        models.TaskKindFlexible,
			DurationMin: 600, // 10 hours
			Active:      true,
			Recurrence:  models.Recurrence{Type: models.RecurrenceDaily},
		},
	}

	// Day: 09:00 - 17:00 (8 hours = 480 mins)
	plan, err := scheduler.GeneratePlan(dateStr, tasks, "09:00", "17:00")
	if err != nil {
		t.Fatalf("GeneratePlan failed: %v", err)
	}

	for _, slot := range plan.Slots {
		if slot.TaskID == "too-long" {
			t.Errorf("Task 'Too Long' should not have been scheduled")
		}
		// Zero duration tasks might be scheduled depending on implementation,
		// but usually they are not useful. Let's see if they are allowed.
		// Based on code: `if task.DurationMin > block.end-block.start` -> 0 > X is false, so it fits.
		// `endTime := startTime + task.DurationMin` -> same as start.
		// `if endTime > block.end` -> start > end (false).
		// So zero duration tasks are technically allowed.
	}
}

func TestGeneratePlan_ErrorHandling(t *testing.T) {
	scheduler := New()
	tasks := []models.Task{}

	// Invalid date
	_, err := scheduler.GeneratePlan("invalid-date", tasks, "09:00", "17:00")
	if err == nil {
		t.Error("Expected error for invalid date, got nil")
	}

	// Invalid day start
	_, err = scheduler.GeneratePlan("2025-12-31", tasks, "invalid-time", "17:00")
	if err == nil {
		t.Error("Expected error for invalid day start, got nil")
	}

	// Invalid day end
	_, err = scheduler.GeneratePlan("2025-12-31", tasks, "09:00", "invalid-time")
	if err == nil {
		t.Error("Expected error for invalid day end, got nil")
	}
}
