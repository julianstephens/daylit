package models

import (
	"fmt"
	"time"

	"github.com/julianstephens/daylit/daylit-cli/internal/constants"
)

type Recurrence struct {
	Type             constants.RecurrenceType `json:"type"`
	IntervalDays     int                      `json:"interval_days,omitempty"`
	WeekdayMask      []time.Weekday           `json:"weekday_mask,omitempty"`
	MonthDay         int                      `json:"month_day,omitempty"`            // Day of month (1-31) for monthly_date
	WeekOccurrence   int                      `json:"week_occurrence,omitempty"`      // Week occurrence (-1=last, 1=first, 2=second, etc.) for monthly_day
	Month            int                      `json:"month,omitempty"`                // Month (1-12) for yearly
	DayOfWeekInMonth time.Weekday             `json:"day_of_week_in_month,omitempty"` // Weekday for monthly_day (e.g., Friday for "last Friday")
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
	if t.Recurrence.Type == constants.RecurrenceMonthlyDate {
		if t.Recurrence.MonthDay < 1 || t.Recurrence.MonthDay > 31 {
			return fmt.Errorf("month day must be between 1 and 31 for monthly_date recurrence")
		}
	}
	if t.Recurrence.Type == constants.RecurrenceMonthlyDay {
		if t.Recurrence.WeekOccurrence < -1 || t.Recurrence.WeekOccurrence == 0 || t.Recurrence.WeekOccurrence > 5 {
			return fmt.Errorf("week occurrence must be -1 (last) or 1-5 for monthly_day recurrence")
		}
	}
	if t.Recurrence.Type == constants.RecurrenceYearly {
		if t.Recurrence.Month < 1 || t.Recurrence.Month > 12 {
			return fmt.Errorf("month must be between 1 and 12 for yearly recurrence")
		}
		if t.Recurrence.MonthDay < 1 || t.Recurrence.MonthDay > 31 {
			return fmt.Errorf("month day must be between 1 and 31 for yearly recurrence")
		}
	}

	return nil
}
