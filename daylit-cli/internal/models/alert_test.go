package models

import (
	"testing"
	"time"

	"github.com/julianstephens/daylit/daylit-cli/internal/constants"
)

func TestAlert_Validate(t *testing.T) {
	tests := []struct {
		name    string
		alert   Alert
		wantErr bool
	}{
		{
			name: "valid daily alert",
			alert: Alert{
				ID:      "test-id",
				Message: "Test alert",
				Time:    "10:00",
				Recurrence: Recurrence{
					Type: constants.RecurrenceDaily,
				},
				Active:    true,
				CreatedAt: time.Now(),
			},
			wantErr: false,
		},
		{
			name: "valid one-time alert",
			alert: Alert{
				ID:        "test-id",
				Message:   "Test alert",
				Time:      "14:30",
				Date:      "2026-01-15",
				Active:    true,
				CreatedAt: time.Now(),
			},
			wantErr: false,
		},
		{
			name: "empty message",
			alert: Alert{
				ID:      "test-id",
				Message: "",
				Time:    "10:00",
				Active:  true,
			},
			wantErr: true,
		},
		{
			name: "empty time",
			alert: Alert{
				ID:      "test-id",
				Message: "Test",
				Time:    "",
				Active:  true,
			},
			wantErr: true,
		},
		{
			name: "invalid time format",
			alert: Alert{
				ID:      "test-id",
				Message: "Test",
				Time:    "25:00",
				Active:  true,
			},
			wantErr: true,
		},
		{
			name: "invalid date format",
			alert: Alert{
				ID:      "test-id",
				Message: "Test",
				Time:    "10:00",
				Date:    "2026/01/15",
				Active:  true,
			},
			wantErr: true,
		},
		{
			name: "weekly without weekdays",
			alert: Alert{
				ID:      "test-id",
				Message: "Test",
				Time:    "10:00",
				Recurrence: Recurrence{
					Type:        constants.RecurrenceWeekly,
					WeekdayMask: []time.Weekday{},
				},
				Active: true,
			},
			wantErr: true,
		},
		{
			name: "n_days with invalid interval",
			alert: Alert{
				ID:      "test-id",
				Message: "Test",
				Time:    "10:00",
				Recurrence: Recurrence{
					Type:         constants.RecurrenceNDays,
					IntervalDays: 0,
				},
				Active: true,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.alert.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Alert.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAlert_IsOneTime(t *testing.T) {
	tests := []struct {
		name  string
		alert Alert
		want  bool
	}{
		{
			name: "one-time with date",
			alert: Alert{
				Date: "2026-01-15",
			},
			want: true,
		},
		{
			name: "recurring without date",
			alert: Alert{
				Date: "",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.alert.IsOneTime(); got != tt.want {
				t.Errorf("Alert.IsOneTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAlert_IsDueToday(t *testing.T) {
	tests := []struct {
		name  string
		alert Alert
		today time.Time
		want  bool
	}{
		{
			name: "daily alert is always due",
			alert: Alert{
				Recurrence: Recurrence{
					Type: constants.RecurrenceDaily,
				},
			},
			today: time.Now(),
			want:  true,
		},
		{
			name: "weekly alert on matching weekday",
			alert: Alert{
				Recurrence: Recurrence{
					Type:        constants.RecurrenceWeekly,
					WeekdayMask: []time.Weekday{time.Monday, time.Wednesday, time.Friday},
				},
			},
			today: time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC), // Monday
			want:  true,
		},
		{
			name: "weekly alert on non-matching weekday",
			alert: Alert{
				Recurrence: Recurrence{
					Type:        constants.RecurrenceWeekly,
					WeekdayMask: []time.Weekday{time.Monday, time.Wednesday, time.Friday},
				},
			},
			today: time.Date(2026, 1, 6, 0, 0, 0, 0, time.UTC), // Tuesday
			want:  false,
		},
		{
			name: "one-time alert on matching date",
			alert: Alert{
				Date: "2026-01-15",
			},
			today: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
			want:  true,
		},
		{
			name: "one-time alert on non-matching date",
			alert: Alert{
				Date: "2026-01-15",
			},
			today: time.Date(2026, 1, 16, 0, 0, 0, 0, time.UTC),
			want:  false,
		},
		{
			name: "n_days alert on first occurrence",
			alert: Alert{
				Recurrence: Recurrence{
					Type:         constants.RecurrenceNDays,
					IntervalDays: 3,
				},
				CreatedAt: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC),
			},
			today: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			want:  true,
		},
		{
			name: "n_days alert on exact interval boundary",
			alert: Alert{
				Recurrence: Recurrence{
					Type:         constants.RecurrenceNDays,
					IntervalDays: 3,
				},
				CreatedAt: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC),
			},
			today: time.Date(2026, 1, 4, 0, 0, 0, 0, time.UTC), // 3 days later
			want:  true,
		},
		{
			name: "n_days alert not on interval boundary",
			alert: Alert{
				Recurrence: Recurrence{
					Type:         constants.RecurrenceNDays,
					IntervalDays: 3,
				},
				CreatedAt: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC),
			},
			today: time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC), // 2 days later
			want:  false,
		},
		{
			name: "n_days alert with LastSent as base",
			alert: Alert{
				Recurrence: Recurrence{
					Type:         constants.RecurrenceNDays,
					IntervalDays: 3,
				},
				CreatedAt: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC),
				LastSent:  ptrTime(time.Date(2026, 1, 5, 10, 30, 0, 0, time.UTC)),
			},
			today: time.Date(2026, 1, 8, 0, 0, 0, 0, time.UTC), // 3 days after LastSent
			want:  true,
		},
		{
			name: "n_days alert with invalid interval",
			alert: Alert{
				Recurrence: Recurrence{
					Type:         constants.RecurrenceNDays,
					IntervalDays: 0,
				},
				CreatedAt: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC),
			},
			today: time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC),
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.alert.IsDueToday(tt.today); got != tt.want {
				t.Errorf("Alert.IsDueToday() = %v, want %v", got, tt.want)
			}
		})
	}
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
