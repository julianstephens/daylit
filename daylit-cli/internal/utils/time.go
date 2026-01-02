package utils

import (
	"time"

	"github.com/julianstephens/daylit/daylit-cli/internal/constants"
)

// ParseTime parses a time string in the standard format (HH:MM).
func ParseTime(timeStr string) (time.Time, error) {
	return time.Parse(constants.TimeFormat, timeStr)
}

// ParseTimeToMinutes parses a time string (HH:MM) and returns the number of minutes from midnight.
func ParseTimeToMinutes(timeStr string) (int, error) {
	t, err := ParseTime(timeStr)
	if err != nil {
		return 0, err
	}
	return t.Hour()*60 + t.Minute(), nil
}

// ValidateTimeFormat checks if the string matches the standard time format.
func ValidateTimeFormat(timeStr string) bool {
	_, err := ParseTime(timeStr)
	return err == nil
}
