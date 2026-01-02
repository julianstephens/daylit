package models

import (
	"fmt"
	"strings"
	"time"
)

type Alert struct {
	ID         string     `json:"id"`
	Message    string     `json:"message"`
	Time       string     `json:"time"`           // HH:MM format
	Date       string     `json:"date,omitempty"` // YYYY-MM-DD (for one-time alerts)
	Recurrence Recurrence `json:"recurrence"`     // Re-use existing Recurrence struct
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
		// For n_days recurrence, an alert is due when today falls on an IntervalDays boundary
		// relative to the base date (LastSent if available, otherwise CreatedAt).
		interval := a.Recurrence.IntervalDays
		if interval < 1 {
			return false
		}

		// Normalize today to a date-only value
		todayDate := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())

		// Determine the base date: last sent date if available, otherwise created-at date
		var baseDate time.Time
		if a.LastSent != nil {
			baseDate = time.Date(a.LastSent.Year(), a.LastSent.Month(), a.LastSent.Day(), 0, 0, 0, 0, a.LastSent.Location())
		} else {
			baseDate = time.Date(a.CreatedAt.Year(), a.CreatedAt.Month(), a.CreatedAt.Day(), 0, 0, 0, 0, a.CreatedAt.Location())
		}

		// If today is before the base date, it cannot be due yet
		if todayDate.Before(baseDate) {
			return false
		}

		daysSince := int(todayDate.Sub(baseDate).Hours() / 24)

		// Fire on exact interval boundaries (0, interval, 2*interval, etc.)
		return daysSince%interval == 0
	case RecurrenceAdHoc:
		// Ad-hoc alerts don't recur
		return false
	default:
		return false
	}
}

// FormatRecurrence returns a human-readable string describing the alert's recurrence pattern
func (a *Alert) FormatRecurrence() string {
	if a.Date != "" {
		return fmt.Sprintf("Once on %s", a.Date)
	}

	switch a.Recurrence.Type {
	case RecurrenceDaily:
		return "Daily"
	case RecurrenceWeekly:
		days := make([]string, len(a.Recurrence.WeekdayMask))
		for i, wd := range a.Recurrence.WeekdayMask {
			days[i] = wd.String()[:3]
		}
		return fmt.Sprintf("Weekly: %s", strings.Join(days, ", "))
	case RecurrenceNDays:
		if a.Recurrence.IntervalDays == 1 {
			return "Daily"
		}
		return fmt.Sprintf("Every %d days", a.Recurrence.IntervalDays)
	default:
		return "One-time"
	}
}
