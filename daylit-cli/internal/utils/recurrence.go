package utils

import (
	"math"
	"time"

	"github.com/julianstephens/daylit/daylit-cli/internal/constants"
	"github.com/julianstephens/daylit/daylit-cli/internal/models"
)

// ShouldScheduleTask determines if a task should be scheduled on the given date
// based on its recurrence pattern. This logic is shared between validation and
// scheduling to ensure consistency.
func ShouldScheduleTask(task models.Task, date time.Time) bool {
	switch task.Recurrence.Type {
	case constants.RecurrenceDaily:
		return true
	case constants.RecurrenceWeekly:
		if len(task.Recurrence.WeekdayMask) == 0 {
			return false
		}
		for _, wd := range task.Recurrence.WeekdayMask {
			if date.Weekday() == wd {
				return true
			}
		}
		return false
	case constants.RecurrenceNDays:
		if task.LastDone == "" {
			return true
		}
		lastDone, err := time.Parse(constants.DateFormat, task.LastDone)
		if err != nil {
			return false
		}
		// Use date-based arithmetic to avoid DST issues with explicit rounding
		daysSince := int(math.Round(date.Sub(lastDone).Hours() / 24))
		return daysSince >= task.Recurrence.IntervalDays
	case constants.RecurrenceMonthlyDate:
		// Schedule on the specified day of each month
		// If the day doesn't exist in the current month (e.g., Feb 31), skip it
		if date.Day() != task.Recurrence.MonthDay {
			return false
		}
		// Double-check the month day hasn't wrapped (e.g., setting day 31 in February)
		return date.Day() == task.Recurrence.MonthDay
	case constants.RecurrenceMonthlyDay:
		// Schedule on a specific weekday occurrence in the month
		// e.g., "last Friday" or "first Monday"
		return isNthWeekdayOfMonth(date, task.Recurrence.DayOfWeekInMonth, task.Recurrence.WeekOccurrence)
	case constants.RecurrenceYearly:
		// Schedule on a specific date each year
		// If the date doesn't exist (e.g., Feb 29 in non-leap years), skip it
		if date.Month() != time.Month(task.Recurrence.Month) {
			return false
		}
		return date.Day() == task.Recurrence.MonthDay
	case constants.RecurrenceWeekdays:
		// Schedule every weekday (Monday through Friday)
		wd := date.Weekday()
		return wd >= time.Monday && wd <= time.Friday
	case constants.RecurrenceAdHoc:
		return false // Ad-hoc tasks are not automatically scheduled
	default:
		return false
	}
}

// isNthWeekdayOfMonth checks if the given date is the nth occurrence of a weekday in its month
// occurrence: -1 for last, 1 for first, 2 for second, etc.
func isNthWeekdayOfMonth(date time.Time, weekday time.Weekday, occurrence int) bool {
	if date.Weekday() != weekday {
		return false
	}

	if occurrence == -1 {
		// Check if this is the last occurrence of the weekday in the month
		// Add 7 days and see if we're still in the same month
		nextWeek := date.AddDate(0, 0, 7)
		return nextWeek.Month() != date.Month()
	}

	// Count which occurrence this is
	day := date.Day()
	occurrenceNum := (day-1)/7 + 1

	// Validate that the occurrence number is reasonable for this month
	// A month can have at most 5 occurrences of any weekday
	if occurrence < 1 || occurrence > 5 {
		return false
	}

	return occurrenceNum == occurrence
}
