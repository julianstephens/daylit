package scheduler

import (
	"testing"
	"time"

	"github.com/julianstephens/daylit/daylit-cli/internal/constants"
	"github.com/julianstephens/daylit/daylit-cli/internal/models"
)

func TestGeneratePlan_MonthlyDateRecurrence(t *testing.T) {
	scheduler := New()

	// Test on the 15th - task should be scheduled
	dateStr := "2026-01-15"

	tasks := []models.Task{
		{
			ID:          "monthly-task",
			Name:        "Monthly Task",
			Kind:        constants.TaskKindFlexible,
			DurationMin: 60,
			Active:      true,
			Recurrence: models.Recurrence{
				Type:     constants.RecurrenceMonthlyDate,
				MonthDay: 15,
			},
		},
	}

	plan, err := scheduler.GeneratePlan(dateStr, tasks, "09:00", "17:00")
	if err != nil {
		t.Fatalf("GeneratePlan failed: %v", err)
	}

	found := false
	for _, slot := range plan.Slots {
		if slot.TaskID == "monthly-task" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected monthly task to be scheduled on the 15th")
	}

	// Test on the 14th - task should NOT be scheduled
	dateStr14 := "2026-01-14"
	plan14, err := scheduler.GeneratePlan(dateStr14, tasks, "09:00", "17:00")
	if err != nil {
		t.Fatalf("GeneratePlan failed: %v", err)
	}

	for _, slot := range plan14.Slots {
		if slot.TaskID == "monthly-task" {
			t.Error("Expected monthly task not to be scheduled on the 14th")
		}
	}
}

func TestGeneratePlan_MonthlyDayRecurrence_LastFriday(t *testing.T) {
	scheduler := New()

	tasks := []models.Task{
		{
			ID:          "last-friday-task",
			Name:        "Last Friday Task",
			Kind:        constants.TaskKindFlexible,
			DurationMin: 60,
			Active:      true,
			Recurrence: models.Recurrence{
				Type:             constants.RecurrenceMonthlyDay,
				WeekOccurrence:   -1,
				DayOfWeekInMonth: time.Friday,
			},
		},
	}

	// January 2026: Last Friday is the 30th
	lastFridayDate := "2026-01-30"
	plan, err := scheduler.GeneratePlan(lastFridayDate, tasks, "09:00", "17:00")
	if err != nil {
		t.Fatalf("GeneratePlan failed: %v", err)
	}

	found := false
	for _, slot := range plan.Slots {
		if slot.TaskID == "last-friday-task" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected last Friday task to be scheduled on Jan 30")
	}

	// January 23rd is a Friday but not the last Friday
	notLastFriday := "2026-01-23"
	planNotLast, err := scheduler.GeneratePlan(notLastFriday, tasks, "09:00", "17:00")
	if err != nil {
		t.Fatalf("GeneratePlan failed: %v", err)
	}

	for _, slot := range planNotLast.Slots {
		if slot.TaskID == "last-friday-task" {
			t.Error("Expected last Friday task not to be scheduled on Jan 23")
		}
	}
}

func TestGeneratePlan_MonthlyDayRecurrence_FirstMonday(t *testing.T) {
	scheduler := New()

	tasks := []models.Task{
		{
			ID:          "first-monday-task",
			Name:        "First Monday Task",
			Kind:        constants.TaskKindFlexible,
			DurationMin: 60,
			Active:      true,
			Recurrence: models.Recurrence{
				Type:             constants.RecurrenceMonthlyDay,
				WeekOccurrence:   1,
				DayOfWeekInMonth: time.Monday,
			},
		},
	}

	// January 2026: First Monday is the 5th
	firstMondayDate := "2026-01-05"
	plan, err := scheduler.GeneratePlan(firstMondayDate, tasks, "09:00", "17:00")
	if err != nil {
		t.Fatalf("GeneratePlan failed: %v", err)
	}

	found := false
	for _, slot := range plan.Slots {
		if slot.TaskID == "first-monday-task" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected first Monday task to be scheduled on Jan 5")
	}

	// Second Monday is the 12th - should not be scheduled
	secondMonday := "2026-01-12"
	planSecond, err := scheduler.GeneratePlan(secondMonday, tasks, "09:00", "17:00")
	if err != nil {
		t.Fatalf("GeneratePlan failed: %v", err)
	}

	for _, slot := range planSecond.Slots {
		if slot.TaskID == "first-monday-task" {
			t.Error("Expected first Monday task not to be scheduled on Jan 12")
		}
	}
}

func TestGeneratePlan_YearlyRecurrence(t *testing.T) {
	scheduler := New()

	tasks := []models.Task{
		{
			ID:          "new-years-task",
			Name:        "New Year's Day",
			Kind:        constants.TaskKindFlexible,
			DurationMin: 60,
			Active:      true,
			Recurrence: models.Recurrence{
				Type:     constants.RecurrenceYearly,
				Month:    1,
				MonthDay: 1,
			},
		},
	}

	// Test on January 1st - should be scheduled
	jan1 := "2026-01-01"
	plan, err := scheduler.GeneratePlan(jan1, tasks, "09:00", "17:00")
	if err != nil {
		t.Fatalf("GeneratePlan failed: %v", err)
	}

	found := false
	for _, slot := range plan.Slots {
		if slot.TaskID == "new-years-task" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected yearly task to be scheduled on January 1st")
	}

	// Test on January 2nd - should NOT be scheduled
	jan2 := "2026-01-02"
	plan2, err := scheduler.GeneratePlan(jan2, tasks, "09:00", "17:00")
	if err != nil {
		t.Fatalf("GeneratePlan failed: %v", err)
	}

	for _, slot := range plan2.Slots {
		if slot.TaskID == "new-years-task" {
			t.Error("Expected yearly task not to be scheduled on January 2nd")
		}
	}

	// Test on December 1st - should NOT be scheduled
	dec1 := "2026-12-01"
	planDec, err := scheduler.GeneratePlan(dec1, tasks, "09:00", "17:00")
	if err != nil {
		t.Fatalf("GeneratePlan failed: %v", err)
	}

	for _, slot := range planDec.Slots {
		if slot.TaskID == "new-years-task" {
			t.Error("Expected yearly task not to be scheduled on December 1st")
		}
	}
}

func TestGeneratePlan_WeekdaysRecurrence(t *testing.T) {
	scheduler := New()

	tasks := []models.Task{
		{
			ID:          "weekday-task",
			Name:        "Weekday Task",
			Kind:        constants.TaskKindFlexible,
			DurationMin: 60,
			Active:      true,
			Recurrence: models.Recurrence{
				Type: constants.RecurrenceWeekdays,
			},
		},
	}

	// Test Monday - should be scheduled
	monday := "2026-01-05"
	planMon, err := scheduler.GeneratePlan(monday, tasks, "09:00", "17:00")
	if err != nil {
		t.Fatalf("GeneratePlan failed: %v", err)
	}

	foundMon := false
	for _, slot := range planMon.Slots {
		if slot.TaskID == "weekday-task" {
			foundMon = true
			break
		}
	}
	if !foundMon {
		t.Error("Expected weekday task to be scheduled on Monday")
	}

	// Test Friday - should be scheduled
	friday := "2026-01-09"
	planFri, err := scheduler.GeneratePlan(friday, tasks, "09:00", "17:00")
	if err != nil {
		t.Fatalf("GeneratePlan failed: %v", err)
	}

	foundFri := false
	for _, slot := range planFri.Slots {
		if slot.TaskID == "weekday-task" {
			foundFri = true
			break
		}
	}
	if !foundFri {
		t.Error("Expected weekday task to be scheduled on Friday")
	}

	// Test Saturday - should NOT be scheduled
	saturday := "2026-01-10"
	planSat, err := scheduler.GeneratePlan(saturday, tasks, "09:00", "17:00")
	if err != nil {
		t.Fatalf("GeneratePlan failed: %v", err)
	}

	for _, slot := range planSat.Slots {
		if slot.TaskID == "weekday-task" {
			t.Error("Expected weekday task not to be scheduled on Saturday")
		}
	}

	// Test Sunday - should NOT be scheduled
	sunday := "2026-01-11"
	planSun, err := scheduler.GeneratePlan(sunday, tasks, "09:00", "17:00")
	if err != nil {
		t.Fatalf("GeneratePlan failed: %v", err)
	}

	for _, slot := range planSun.Slots {
		if slot.TaskID == "weekday-task" {
			t.Error("Expected weekday task not to be scheduled on Sunday")
		}
	}
}

func TestGeneratePlan_MixedComplexRecurrence(t *testing.T) {
	scheduler := New()

	// Test multiple complex recurrence types together
	tasks := []models.Task{
		{
			ID:          "monthly-15",
			Name:        "Monthly 15th",
			Kind:        constants.TaskKindFlexible,
			DurationMin: 60,
			Active:      true,
			Priority:    2,
			Recurrence: models.Recurrence{
				Type:     constants.RecurrenceMonthlyDate,
				MonthDay: 15,
			},
		},
		{
			ID:          "weekdays",
			Name:        "Weekdays",
			Kind:        constants.TaskKindFlexible,
			DurationMin: 60,
			Active:      true,
			Priority:    3,
			Recurrence: models.Recurrence{
				Type: constants.RecurrenceWeekdays,
			},
		},
		{
			ID:          "yearly",
			Name:        "Yearly Jan 15",
			Kind:        constants.TaskKindFlexible,
			DurationMin: 60,
			Active:      true,
			Priority:    1, // Highest priority
			Recurrence: models.Recurrence{
				Type:     constants.RecurrenceYearly,
				Month:    1,
				MonthDay: 15,
			},
		},
	}

	// January 15, 2026 is a Thursday (weekday)
	// Should schedule: yearly (prio 1), monthly (prio 2), weekdays (prio 3)
	dateStr := "2026-01-15"
	plan, err := scheduler.GeneratePlan(dateStr, tasks, "09:00", "12:00")
	if err != nil {
		t.Fatalf("GeneratePlan failed: %v", err)
	}

	foundYearly := false
	foundMonthly := false
	foundWeekdays := false

	for _, slot := range plan.Slots {
		switch slot.TaskID {
		case "yearly":
			foundYearly = true
		case "monthly-15":
			foundMonthly = true
		case "weekdays":
			foundWeekdays = true
		}
	}

	if !foundYearly {
		t.Error("Expected yearly task to be scheduled")
	}
	if !foundMonthly {
		t.Error("Expected monthly task to be scheduled")
	}
	if !foundWeekdays {
		t.Error("Expected weekdays task to be scheduled")
	}

	// Verify priority ordering (yearly should come first)
	if len(plan.Slots) > 0 && plan.Slots[0].TaskID != "yearly" {
		t.Error("Expected highest priority (yearly) task to be scheduled first")
	}
}
