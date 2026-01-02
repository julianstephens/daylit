package models

import (
	"fmt"
	"time"

	"github.com/julianstephens/daylit/daylit-cli/internal/constants"
)

type Recurrence struct {
	Type         constants.RecurrenceType `json:"type"`
	IntervalDays int                      `json:"interval_days,omitempty"`
	WeekdayMask  []time.Weekday           `json:"weekday_mask,omitempty"`
}

type Task struct {
	ID                   string               `json:"id"`
	Name                 string               `json:"name"`
	Kind                 constants.TaskKind   `json:"kind"`
	DurationMin          int                  `json:"duration_min"`
	EarliestStart        string               `json:"earliest_start,omitempty"` // HH:MM format
	LatestEnd            string               `json:"latest_end,omitempty"`     // HH:MM format
	FixedStart           string               `json:"fixed_start,omitempty"`    // HH:MM format
	FixedEnd             string               `json:"fixed_end,omitempty"`      // HH:MM format
	Recurrence           Recurrence           `json:"recurrence"`
	Priority             int                  `json:"priority"`
	EnergyBand           constants.EnergyBand `json:"energy_band,omitempty"`
	Active               bool                 `json:"active"`
	LastDone             string               `json:"last_done,omitempty"` // YYYY-MM-DD format
	SuccessStreak        int                  `json:"success_streak"`
	AvgActualDurationMin float64              `json:"avg_actual_duration_min"`
	DeletedAt            *string              `json:"deleted_at,omitempty"` // RFC3339 timestamp
}

func (t *Task) Validate() error {
	if t.Name == "" {
		return fmt.Errorf("task name cannot be empty")
	}
	if t.DurationMin <= 0 {
		return fmt.Errorf("duration must be greater than zero")
	}
	if t.Priority < 1 || t.Priority > 5 {
		return fmt.Errorf("priority must be between 1 and 5")
	}

	// Recurrence validation
	if t.Recurrence.Type == constants.RecurrenceNDays && t.Recurrence.IntervalDays < 1 {
		return fmt.Errorf("interval must be at least 1 for n_days recurrence")
	}
	if t.Recurrence.Type == constants.RecurrenceWeekly && len(t.Recurrence.WeekdayMask) == 0 {
		return fmt.Errorf("weekdays must be specified for weekly recurrence")
	}

	return nil
}
