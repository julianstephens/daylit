package utils

import (
	"fmt"
	"time"

	"github.com/julianstephens/daylit/daylit-cli/internal/constants"
	"github.com/julianstephens/daylit/daylit-cli/internal/models"
)

// GetTodayInTimezone returns today's date string (YYYY-MM-DD) in the specified timezone.
// This ensures that "today" is determined by the user's configured timezone, not the system timezone.
func GetTodayInTimezone(timezone string) (string, error) {
	now, err := NowInTimezone(timezone)
	if err != nil {
		return "", err
	}
	return now.Format(constants.DateFormat), nil
}

// GetTodayFromSettings returns today's date string (YYYY-MM-DD) using the timezone from settings.
func GetTodayFromSettings(settings models.Settings) (string, error) {
	return GetTodayInTimezone(settings.Timezone)
}

// LoadLocation loads a timezone location from an IANA timezone name.
// If the timezone is "Local" or empty, it returns the system's local timezone.
func LoadLocation(timezone string) (*time.Location, error) {
	if timezone == "" || timezone == "Local" {
		return time.Local, nil
	}
	return time.LoadLocation(timezone)
}

// NowInTimezone returns the current time in the specified timezone.
func NowInTimezone(timezone string) (time.Time, error) {
	loc, err := LoadLocation(timezone)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid timezone %q: %w", timezone, err)
	}
	return time.Now().In(loc), nil
}

// ParseTime parses a time string in the standard format (HH:MM).
func ParseTime(timeStr string) (time.Time, error) {
	return time.Parse(constants.TimeFormat, timeStr)
}

// ParseTimeInLocation parses a time string (HH:MM) in the specified timezone.
// It returns a time.Time with the date set to the zero value and the timezone set.
func ParseTimeInLocation(timeStr string, loc *time.Location) (time.Time, error) {
	t, err := time.Parse(constants.TimeFormat, timeStr)
	if err != nil {
		return time.Time{}, err
	}
	// The parsed time has no date, so we need to set it to a zero date in the specified location
	return time.Date(0, 1, 1, t.Hour(), t.Minute(), 0, 0, loc), nil
}

// ParseTimeToMinutes parses a time string (HH:MM) and returns the number of minutes from midnight.
func ParseTimeToMinutes(timeStr string) (int, error) {
	t, err := ParseTime(timeStr)
	if err != nil {
		return 0, err
	}
	return t.Hour()*60 + t.Minute(), nil
}

// ParseDateInLocation parses a date string (YYYY-MM-DD) in the specified timezone.
func ParseDateInLocation(dateStr string, loc *time.Location) (time.Time, error) {
	t, err := time.Parse(constants.DateFormat, dateStr)
	if err != nil {
		return time.Time{}, err
	}
	// Return the date at midnight in the specified timezone
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc), nil
}

// CombineDateAndTime combines a date string (YYYY-MM-DD) and time string (HH:MM)
// into a single time.Time in the specified timezone.
func CombineDateAndTime(dateStr, timeStr string, loc *time.Location) (time.Time, error) {
	date, err := time.Parse(constants.DateFormat, dateStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date format: %w", err)
	}

	timeOfDay, err := time.Parse(constants.TimeFormat, timeStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid time format: %w", err)
	}

	return time.Date(
		date.Year(), date.Month(), date.Day(),
		timeOfDay.Hour(), timeOfDay.Minute(), 0, 0,
		loc,
	), nil
}

// ValidateTimeFormat checks if the string matches the standard time format.
func ValidateTimeFormat(timeStr string) bool {
	_, err := ParseTime(timeStr)
	return err == nil
}

// ValidateTimezone checks if the timezone name is valid.
func ValidateTimezone(timezone string) bool {
	if timezone == "" || timezone == "Local" {
		return true
	}
	_, err := time.LoadLocation(timezone)
	return err == nil
}
