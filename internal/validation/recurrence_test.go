package validation

import (
	"testing"
	"time"

	"github.com/julianstephens/daylit/internal/models"
)

func TestValidateTasks_RespectsWeekdays(t *testing.T) {
	validator := New()

	tasks := []models.Task{
		{
			ID:         "task-mon",
			Name:       "Monday Task",
			Kind:       models.TaskKindAppointment,
			FixedStart: "09:00",
			FixedEnd:   "10:00",
			Active:     true,
			Recurrence: models.Recurrence{
				Type:        models.RecurrenceWeekly,
				WeekdayMask: []time.Weekday{time.Monday},
			},
		},
		{
			ID:         "task-tue",
			Name:       "Tuesday Task",
			Kind:       models.TaskKindAppointment,
			FixedStart: "09:00",
			FixedEnd:   "10:00",
			Active:     true,
			Recurrence: models.Recurrence{
				Type:        models.RecurrenceWeekly,
				WeekdayMask: []time.Weekday{time.Tuesday},
			},
		},
	}

	result := validator.ValidateTasks(tasks)

	if result.HasConflicts() {
		t.Errorf("Expected no conflicts for disjoint weekdays, got: %s", result.FormatReport())
	}
}

func TestValidateTasks_DetectsOverlappingWeekdays(t *testing.T) {
	validator := New()

	tasks := []models.Task{
		{
			ID:         "task-mon-wed",
			Name:       "MW Task",
			Kind:       models.TaskKindAppointment,
			FixedStart: "09:00",
			FixedEnd:   "10:00",
			Active:     true,
			Recurrence: models.Recurrence{
				Type:        models.RecurrenceWeekly,
				WeekdayMask: []time.Weekday{time.Monday, time.Wednesday},
			},
		},
		{
			ID:         "task-wed-fri",
			Name:       "WF Task",
			Kind:       models.TaskKindAppointment,
			FixedStart: "09:30", // Overlaps with 09:00-10:00
			FixedEnd:   "10:30",
			Active:     true,
			Recurrence: models.Recurrence{
				Type:        models.RecurrenceWeekly,
				WeekdayMask: []time.Weekday{time.Wednesday, time.Friday},
			},
		},
	}

	result := validator.ValidateTasks(tasks)

	if !result.HasConflicts() {
		t.Error("Expected conflict for overlapping weekdays (Wednesday)")
	}
}
