package models

import "time"

type TaskKind string

const (
	TaskKindAppointment TaskKind = "appointment"
	TaskKindFlexible    TaskKind = "flexible"
)

type RecurrenceType string

const (
	RecurrenceDaily  RecurrenceType = "daily"
	RecurrenceWeekly RecurrenceType = "weekly"
	RecurrenceNDays  RecurrenceType = "n_days"
	RecurrenceAdHoc  RecurrenceType = "ad_hoc"
)

type EnergyBand string

const (
	EnergyLow    EnergyBand = "low"
	EnergyMedium EnergyBand = "medium"
	EnergyHigh   EnergyBand = "high"
)

type Recurrence struct {
	Type         RecurrenceType `json:"type"`
	IntervalDays int            `json:"interval_days,omitempty"`
	WeekdayMask  []time.Weekday `json:"weekday_mask,omitempty"`
}

type Task struct {
	ID                   string     `json:"id"`
	Name                 string     `json:"name"`
	Kind                 TaskKind   `json:"kind"`
	DurationMin          int        `json:"duration_min"`
	EarliestStart        string     `json:"earliest_start,omitempty"` // HH:MM format
	LatestEnd            string     `json:"latest_end,omitempty"`     // HH:MM format
	FixedStart           string     `json:"fixed_start,omitempty"`    // HH:MM format
	FixedEnd             string     `json:"fixed_end,omitempty"`      // HH:MM format
	Recurrence           Recurrence `json:"recurrence"`
	Priority             int        `json:"priority"`
	EnergyBand           EnergyBand `json:"energy_band,omitempty"`
	Active               bool       `json:"active"`
	LastDone             string     `json:"last_done,omitempty"` // YYYY-MM-DD format
	SuccessStreak        int        `json:"success_streak"`
	AvgActualDurationMin float64    `json:"avg_actual_duration_min"`
	DeletedAt            *string    `json:"deleted_at,omitempty"` // RFC3339 timestamp
}
