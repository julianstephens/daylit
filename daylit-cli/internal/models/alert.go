package models

import (
	"fmt"
	"time"
)

type Alert struct {
	ID         string     `json:"id"`
	Message    string     `json:"message"`
	Time       string     `json:"time"`                // HH:MM format
	Date       string     `json:"date,omitempty"`      // YYYY-MM-DD (for one-time alerts)
	Recurrence Recurrence `json:"recurrence"`          // Re-use existing Recurrence struct
	Active     bool       `json:"active"`
	LastSent   *time.Time `json:"last_sent,omitempty"` // RFC3339 timestamp
	CreatedAt  time.Time  `json:"created_at"`
}

func (a *Alert) Validate() error {
	if a.Message == "" {
		return fmt.Errorf("alert message cannot be empty")
	}

	if a.Time == "" {
		return fmt.Errorf("alert time cannot be empty")
	}

	// Validate time format (HH:MM)
	if _, err := time.Parse("15:04", a.Time); err != nil {
		return fmt.Errorf("invalid time format (expected HH:MM): %w", err)
	}

	// Validate date format if provided (one-time alert)
	if a.Date != "" {
		if _, err := time.Parse("2006-01-02", a.Date); err != nil {
			return fmt.Errorf("invalid date format (expected YYYY-MM-DD): %w", err)
		}
	}

	// If not a one-time alert, validate recurrence
	if a.Date == "" {
		if a.Recurrence.Type == RecurrenceWeekly && len(a.Recurrence.WeekdayMask) == 0 {
			return fmt.Errorf("weekdays must be specified for weekly recurrence")
		}
		if a.Recurrence.Type == RecurrenceNDays && a.Recurrence.IntervalDays < 1 {
			return fmt.Errorf("interval must be at least 1 for n_days recurrence")
		}
	}

	return nil
}

// IsOneTime returns true if this is a one-time alert (has a date)
func (a *Alert) IsOneTime() bool {
	return a.Date != ""
}

// IsDueToday checks if the alert should fire today based on its recurrence pattern
func (a *Alert) IsDueToday(today time.Time) bool {
	// One-time alerts: check if date matches
	if a.IsOneTime() {
		dateStr := today.Format("2006-01-02")
		return a.Date == dateStr
	}

	// Recurring alerts: check recurrence pattern
	switch a.Recurrence.Type {
	case RecurrenceDaily:
		return true
	case RecurrenceWeekly:
		todayWeekday := today.Weekday()
		for _, wd := range a.Recurrence.WeekdayMask {
			if wd == todayWeekday {
				return true
			}
		}
		return false
	case RecurrenceNDays:
		// For n_days recurrence, we would need to track when it was last completed
		// For now, we'll rely on LastSent to determine if it should fire
		return true
	case RecurrenceAdHoc:
		// Ad-hoc alerts don't recur
		return false
	default:
		return false
	}
}
