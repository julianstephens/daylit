package cli

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/julianstephens/daylit/daylit-cli/internal/backup"
	"github.com/julianstephens/daylit/daylit-cli/internal/constants"
	"github.com/julianstephens/daylit/daylit-cli/internal/logger"
	"github.com/julianstephens/daylit/daylit-cli/internal/models"
	"github.com/julianstephens/daylit/daylit-cli/internal/scheduler"
	"github.com/julianstephens/daylit/daylit-cli/internal/storage"
)

type Context struct {
	Store     storage.Provider
	Scheduler *scheduler.Scheduler
}

// PerformAutomaticBackup creates an automatic backup and silently handles errors
func (c *Context) PerformAutomaticBackup() {
	mgr := backup.NewManager(c.Store.GetConfigPath())
	_, err := mgr.CreateBackup()
	if err != nil {
		// Log warning but don't interrupt user workflow
		logger.Warn("Automatic backup failed", "error", err)
	}
}

// ParseWeekdays parses a comma-separated list of weekdays
func ParseWeekdays(s string) ([]time.Weekday, error) {
	parts := strings.Split(s, ",")
	var weekdays []time.Weekday

	dayMap := map[string]time.Weekday{
		"sun":       time.Sunday,
		"sunday":    time.Sunday,
		"mon":       time.Monday,
		"monday":    time.Monday,
		"tue":       time.Tuesday,
		"tuesday":   time.Tuesday,
		"wed":       time.Wednesday,
		"wednesday": time.Wednesday,
		"thu":       time.Thursday,
		"thursday":  time.Thursday,
		"fri":       time.Friday,
		"friday":    time.Friday,
		"sat":       time.Saturday,
		"saturday":  time.Saturday,
	}

	for _, part := range parts {
		part = strings.TrimSpace(strings.ToLower(part))
		if wd, ok := dayMap[part]; ok {
			weekdays = append(weekdays, wd)
		} else {
			// Try parsing as number (0=Sunday, 6=Saturday)
			num, err := strconv.Atoi(part)
			if err == nil && num >= 0 && num <= 6 {
				weekdays = append(weekdays, time.Weekday(num))
			} else {
				return nil, fmt.Errorf("invalid weekday: %s", part)
			}
		}
	}

	return weekdays, nil
}

// FormatRecurrence formats a recurrence rule into a human-readable string
func FormatRecurrence(rec models.Recurrence) string {
	switch rec.Type {
	case models.RecurrenceDaily:
		return "daily"
	case models.RecurrenceWeekly:
		if len(rec.WeekdayMask) > 0 {
			var days []string
			for _, wd := range rec.WeekdayMask {
				days = append(days, wd.String()[:3])
			}
			return fmt.Sprintf("weekly on %s", strings.Join(days, ","))
		}
		return "weekly"
	case models.RecurrenceNDays:
		return fmt.Sprintf("every %d days", rec.IntervalDays)
	case models.RecurrenceAdHoc:
		return "ad-hoc"
	default:
		return "unknown"
	}
}

// CalculateSlotDuration returns the duration of a slot in minutes.
// Returns 0 if the time format is invalid (which the caller should check).
func CalculateSlotDuration(slot models.Slot) int {
	start, err := time.Parse(constants.TimeFormat, slot.Start)
	if err != nil {
		return 0
	}
	end, err := time.Parse(constants.TimeFormat, slot.End)
	if err != nil {
		return 0
	}
	return int(end.Sub(start).Minutes())
}
