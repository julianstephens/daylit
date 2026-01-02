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
	case constants.RecurrenceAdHoc:
		return false // Ad-hoc tasks are not automatically scheduled
	default:
		return false
	}
}
