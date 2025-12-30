package cli

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/julianstephens/daylit/internal/models"
)

type TaskAddCmd struct {
	Name       string `arg:"" help:"Task name."`
	Duration   int    `short:"d" help:"Duration in minutes." required:""`
	Recurrence string `short:"r" help:"Recurrence type (daily|weekly|n_days|ad_hoc)." default:"ad_hoc"`
	Interval   int    `short:"i" help:"Interval for n_days recurrence." default:"1"`
	Weekdays   string `short:"w" help:"Comma-separated weekdays for weekly recurrence."`
	Earliest   string `short:"s" help:"Earliest start time (HH:MM)."`
	Latest     string `short:"e" help:"Latest end time (HH:MM)."`
	FixedStart string `short:"S" help:"Fixed start time for appointments (HH:MM)."`
	FixedEnd   string `short:"E" help:"Fixed end time for appointments (HH:MM)."`
	Priority   int    `short:"p" help:"Priority (1-5, lower is higher priority)." default:"3"`
}

func (c *TaskAddCmd) Validate() error {
	if c.Priority < 1 || c.Priority > 5 {
		return fmt.Errorf("priority must be between 1 and 5")
	}
	return nil
}

func (c *TaskAddCmd) Run(ctx *Context) error {
	if err := ctx.Store.Load(); err != nil {
		return err
	}

	// Determine task kind
	taskKind := models.TaskKindFlexible
	if c.FixedStart != "" && c.FixedEnd != "" {
		taskKind = models.TaskKindAppointment
	}

	// Parse recurrence
	var recType models.RecurrenceType
	switch c.Recurrence {
	case "daily":
		recType = models.RecurrenceDaily
	case "weekly":
		recType = models.RecurrenceWeekly
	case "n_days":
		recType = models.RecurrenceNDays
	case "ad_hoc":
		recType = models.RecurrenceAdHoc
	default:
		return fmt.Errorf("invalid recurrence type: %s", c.Recurrence)
	}

	rec := models.Recurrence{
		Type:         recType,
		IntervalDays: c.Interval,
	}

	// Parse weekdays for weekly recurrence
	if recType == models.RecurrenceWeekly && c.Weekdays != "" {
		wds, err := parseWeekdays(c.Weekdays)
		if err != nil {
			return err
		}
		rec.WeekdayMask = wds
	}

	// Create task
	task := models.Task{
		ID:                   uuid.New().String(),
		Name:                 c.Name,
		Kind:                 taskKind,
		DurationMin:          c.Duration,
		EarliestStart:        c.Earliest,
		LatestEnd:            c.Latest,
		FixedStart:           c.FixedStart,
		FixedEnd:             c.FixedEnd,
		Recurrence:           rec,
		Priority:             c.Priority,
		Active:               true,
		SuccessStreak:        0,
		AvgActualDurationMin: float64(c.Duration),
	}

	if err := ctx.Store.AddTask(task); err != nil {
		return err
	}

	fmt.Printf("Added task: %s (ID: %s)\n", c.Name, task.ID)
	return nil
}
