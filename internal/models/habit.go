package models

import "time"

// Habit represents a recurring practice to track
type Habit struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	CreatedAt  time.Time  `json:"created_at"`
	ArchivedAt *time.Time `json:"archived_at,omitempty"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty"`
}

// HabitEntry represents a single day's record of a habit
type HabitEntry struct {
	ID        string     `json:"id"`
	HabitID   string     `json:"habit_id"`
	Day       string     `json:"day"` // YYYY-MM-DD format
	Note      string     `json:"note"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}
