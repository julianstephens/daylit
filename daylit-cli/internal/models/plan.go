package models

type SlotStatus string

const (
	SlotStatusPlanned  SlotStatus = "planned"
	SlotStatusAccepted SlotStatus = "accepted"
	SlotStatusDone     SlotStatus = "done"
	SlotStatusSkipped  SlotStatus = "skipped"
)

type FeedbackRating string

const (
	FeedbackOnTrack     FeedbackRating = "on_track"
	FeedbackTooMuch     FeedbackRating = "too_much"
	FeedbackUnnecessary FeedbackRating = "unnecessary"
)

type Feedback struct {
	Rating FeedbackRating `json:"rating"`
	Note   string         `json:"note,omitempty"`
}

type Slot struct {
	Start     string     `json:"start"` // HH:MM format
	End       string     `json:"end"`   // HH:MM format
	TaskID    string     `json:"task_id"`
	Status    SlotStatus `json:"status"`
	Feedback  *Feedback  `json:"feedback,omitempty"`
	DeletedAt *string    `json:"deleted_at,omitempty"` // RFC3339 timestamp
}

type DayPlan struct {
	Date       string  `json:"date"`                  // YYYY-MM-DD format
	Revision   int     `json:"revision"`              // Plan revision: 0 = auto-assign latest, 1+ = specific saved revisions
	AcceptedAt *string `json:"accepted_at,omitempty"` // RFC3339 timestamp when this revision was accepted; nil if never accepted
	Slots      []Slot  `json:"slots"`
	DeletedAt  *string `json:"deleted_at,omitempty"` // RFC3339 timestamp
}
