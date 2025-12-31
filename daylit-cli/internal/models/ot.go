package models

import "time"

// OTSettings holds configuration for Once-Today feature
type OTSettings struct {
	ID             int  `json:"id"` // Always 1
	PromptOnEmpty  bool `json:"prompt_on_empty"`
	StrictMode     bool `json:"strict_mode"`
	DefaultLogDays int  `json:"default_log_days"`
}

// OTEntry represents a single day's Once-Today intention
type OTEntry struct {
	ID        string     `json:"id"`
	Day       string     `json:"day"` // YYYY-MM-DD format
	Title     string     `json:"title"`
	Note      string     `json:"note"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}
