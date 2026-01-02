package utils

import (
	"testing"
	"time"
)

func TestLoadLocation(t *testing.T) {
	tests := []struct {
		name     string
		timezone string
		wantErr  bool
	}{
		{
			name:     "empty string returns local",
			timezone: "",
			wantErr:  false,
		},
		{
			name:     "Local returns local",
			timezone: "Local",
			wantErr:  false,
		},
		{
			name:     "valid timezone UTC",
			timezone: "UTC",
			wantErr:  false,
		},
		{
			name:     "valid timezone America/New_York",
			timezone: "America/New_York",
			wantErr:  false,
		},
		{
			name:     "valid timezone Europe/London",
			timezone: "Europe/London",
			wantErr:  false,
		},
		{
			name:     "valid timezone Asia/Tokyo",
			timezone: "Asia/Tokyo",
			wantErr:  false,
		},
		{
			name:     "invalid timezone",
			timezone: "Invalid/Timezone",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loc, err := LoadLocation(tt.timezone)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadLocation() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && loc == nil {
				t.Errorf("LoadLocation() returned nil location without error")
			}
		})
	}
}

func TestNowInTimezone(t *testing.T) {
	tests := []struct {
		name     string
		timezone string
		wantErr  bool
	}{
		{
			name:     "Local timezone",
			timezone: "Local",
			wantErr:  false,
		},
		{
			name:     "UTC timezone",
			timezone: "UTC",
			wantErr:  false,
		},
		{
			name:     "America/New_York timezone",
			timezone: "America/New_York",
			wantErr:  false,
		},
		{
			name:     "invalid timezone",
			timezone: "Invalid/Timezone",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now, err := NowInTimezone(tt.timezone)
			if (err != nil) != tt.wantErr {
				t.Errorf("NowInTimezone() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				// Verify the time is not zero
				if now.IsZero() {
					t.Errorf("NowInTimezone() returned zero time")
				}
				// Verify the location matches
				if tt.timezone == "Local" || tt.timezone == "" {
					if now.Location() != time.Local {
						t.Errorf("NowInTimezone() location = %v, want Local", now.Location())
					}
				} else {
					expectedLoc, _ := time.LoadLocation(tt.timezone)
					if now.Location().String() != expectedLoc.String() {
						t.Errorf("NowInTimezone() location = %v, want %v", now.Location(), expectedLoc)
					}
				}
			}
		})
	}
}

func TestParseTimeInLocation(t *testing.T) {
	utc, _ := time.LoadLocation("UTC")
	est, _ := time.LoadLocation("America/New_York")

	tests := []struct {
		name     string
		timeStr  string
		loc      *time.Location
		wantHour int
		wantMin  int
		wantErr  bool
	}{
		{
			name:     "valid time in UTC",
			timeStr:  "14:30",
			loc:      utc,
			wantHour: 14,
			wantMin:  30,
			wantErr:  false,
		},
		{
			name:     "valid time in EST",
			timeStr:  "09:15",
			loc:      est,
			wantHour: 9,
			wantMin:  15,
			wantErr:  false,
		},
		{
			name:     "midnight",
			timeStr:  "00:00",
			loc:      utc,
			wantHour: 0,
			wantMin:  0,
			wantErr:  false,
		},
		{
			name:     "end of day",
			timeStr:  "23:59",
			loc:      utc,
			wantHour: 23,
			wantMin:  59,
			wantErr:  false,
		},
		{
			name:     "invalid format",
			timeStr:  "25:00",
			loc:      utc,
			wantErr:  true,
		},
		{
			name:     "invalid format with text",
			timeStr:  "noon",
			loc:      utc,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTimeInLocation(tt.timeStr, tt.loc)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTimeInLocation() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Hour() != tt.wantHour {
					t.Errorf("ParseTimeInLocation() hour = %v, want %v", got.Hour(), tt.wantHour)
				}
				if got.Minute() != tt.wantMin {
					t.Errorf("ParseTimeInLocation() minute = %v, want %v", got.Minute(), tt.wantMin)
				}
				if got.Location() != tt.loc {
					t.Errorf("ParseTimeInLocation() location = %v, want %v", got.Location(), tt.loc)
				}
			}
		})
	}
}

func TestParseDateInLocation(t *testing.T) {
	utc, _ := time.LoadLocation("UTC")
	est, _ := time.LoadLocation("America/New_York")

	tests := []struct {
		name      string
		dateStr   string
		loc       *time.Location
		wantYear  int
		wantMonth time.Month
		wantDay   int
		wantErr   bool
	}{
		{
			name:      "valid date in UTC",
			dateStr:   "2026-01-15",
			loc:       utc,
			wantYear:  2026,
			wantMonth: time.January,
			wantDay:   15,
			wantErr:   false,
		},
		{
			name:      "valid date in EST",
			dateStr:   "2025-12-31",
			loc:       est,
			wantYear:  2025,
			wantMonth: time.December,
			wantDay:   31,
			wantErr:   false,
		},
		{
			name:     "invalid format",
			dateStr:  "2026/01/15",
			loc:      utc,
			wantErr:  true,
		},
		{
			name:     "invalid date",
			dateStr:  "2026-13-01",
			loc:      utc,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDateInLocation(tt.dateStr, tt.loc)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDateInLocation() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Year() != tt.wantYear {
					t.Errorf("ParseDateInLocation() year = %v, want %v", got.Year(), tt.wantYear)
				}
				if got.Month() != tt.wantMonth {
					t.Errorf("ParseDateInLocation() month = %v, want %v", got.Month(), tt.wantMonth)
				}
				if got.Day() != tt.wantDay {
					t.Errorf("ParseDateInLocation() day = %v, want %v", got.Day(), tt.wantDay)
				}
				if got.Location() != tt.loc {
					t.Errorf("ParseDateInLocation() location = %v, want %v", got.Location(), tt.loc)
				}
				// Should be at midnight
				if got.Hour() != 0 || got.Minute() != 0 || got.Second() != 0 {
					t.Errorf("ParseDateInLocation() time = %02d:%02d:%02d, want 00:00:00", got.Hour(), got.Minute(), got.Second())
				}
			}
		})
	}
}

func TestCombineDateAndTime(t *testing.T) {
	utc, _ := time.LoadLocation("UTC")
	est, _ := time.LoadLocation("America/New_York")

	tests := []struct {
		name     string
		dateStr  string
		timeStr  string
		loc      *time.Location
		wantYear int
		wantMon  time.Month
		wantDay  int
		wantHour int
		wantMin  int
		wantErr  bool
	}{
		{
			name:     "valid date and time in UTC",
			dateStr:  "2026-01-15",
			timeStr:  "14:30",
			loc:      utc,
			wantYear: 2026,
			wantMon:  time.January,
			wantDay:  15,
			wantHour: 14,
			wantMin:  30,
			wantErr:  false,
		},
		{
			name:     "valid date and time in EST",
			dateStr:  "2025-12-31",
			timeStr:  "23:59",
			loc:      est,
			wantYear: 2025,
			wantMon:  time.December,
			wantDay:  31,
			wantHour: 23,
			wantMin:  59,
			wantErr:  false,
		},
		{
			name:     "midnight",
			dateStr:  "2026-01-01",
			timeStr:  "00:00",
			loc:      utc,
			wantYear: 2026,
			wantMon:  time.January,
			wantDay:  1,
			wantHour: 0,
			wantMin:  0,
			wantErr:  false,
		},
		{
			name:     "invalid date format",
			dateStr:  "2026/01/15",
			timeStr:  "14:30",
			loc:      utc,
			wantErr:  true,
		},
		{
			name:     "invalid time format",
			dateStr:  "2026-01-15",
			timeStr:  "25:00",
			loc:      utc,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CombineDateAndTime(tt.dateStr, tt.timeStr, tt.loc)
			if (err != nil) != tt.wantErr {
				t.Errorf("CombineDateAndTime() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Year() != tt.wantYear {
					t.Errorf("CombineDateAndTime() year = %v, want %v", got.Year(), tt.wantYear)
				}
				if got.Month() != tt.wantMon {
					t.Errorf("CombineDateAndTime() month = %v, want %v", got.Month(), tt.wantMon)
				}
				if got.Day() != tt.wantDay {
					t.Errorf("CombineDateAndTime() day = %v, want %v", got.Day(), tt.wantDay)
				}
				if got.Hour() != tt.wantHour {
					t.Errorf("CombineDateAndTime() hour = %v, want %v", got.Hour(), tt.wantHour)
				}
				if got.Minute() != tt.wantMin {
					t.Errorf("CombineDateAndTime() minute = %v, want %v", got.Minute(), tt.wantMin)
				}
				if got.Location() != tt.loc {
					t.Errorf("CombineDateAndTime() location = %v, want %v", got.Location(), tt.loc)
				}
			}
		})
	}
}

func TestValidateTimezone(t *testing.T) {
	tests := []struct {
		name     string
		timezone string
		want     bool
	}{
		{
			name:     "empty string is valid",
			timezone: "",
			want:     true,
		},
		{
			name:     "Local is valid",
			timezone: "Local",
			want:     true,
		},
		{
			name:     "UTC is valid",
			timezone: "UTC",
			want:     true,
		},
		{
			name:     "America/New_York is valid",
			timezone: "America/New_York",
			want:     true,
		},
		{
			name:     "Europe/London is valid",
			timezone: "Europe/London",
			want:     true,
		},
		{
			name:     "Asia/Tokyo is valid",
			timezone: "Asia/Tokyo",
			want:     true,
		},
		{
			name:     "Invalid/Timezone is invalid",
			timezone: "Invalid/Timezone",
			want:     false,
		},
		{
			name:     "random string is invalid",
			timezone: "not-a-timezone",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateTimezone(tt.timezone); got != tt.want {
				t.Errorf("ValidateTimezone() = %v, want %v", got, tt.want)
			}
		})
	}
}
