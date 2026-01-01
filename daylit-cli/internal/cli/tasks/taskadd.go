package tasks

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/julianstephens/daylit/daylit-cli/internal/cli"
	"github.com/julianstephens/daylit/daylit-cli/internal/constants"
	"github.com/julianstephens/daylit/daylit-cli/internal/models"
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
	// Validate priority
	if c.Priority < 1 || c.Priority > 5 {
		return fmt.Errorf("priority must be between 1 and 5")
	}

	// Validate duration is positive
	if c.Duration <= 0 {
		return fmt.Errorf("duration must be greater than zero")
	}

	// Validate interval for n_days recurrence
	if c.Recurrence == "n_days" && c.Interval < 1 {
		return fmt.Errorf("interval must be at least 1 for n_days recurrence")
	}

	// Validate weekly recurrence has weekdays
	if c.Recurrence == "weekly" && c.Weekdays == "" {
		return fmt.Errorf("weekdays must be specified for weekly recurrence")
	}

	// Validate time formats
	if c.Earliest != "" {
		if _, err := time.Parse(constants.TimeFormat, c.Earliest); err != nil {
			return fmt.Errorf("invalid Earliest time format (expected HH:MM): %w", err)
		}
	}
	if c.Latest != "" {
		if _, err := time.Parse(constants.TimeFormat, c.Latest); err != nil {
			return fmt.Errorf("invalid Latest time format (expected HH:MM): %w", err)
		}
	}
	if c.FixedStart != "" {
		if _, err := time.Parse(constants.TimeFormat, c.FixedStart); err != nil {
			return fmt.Errorf("invalid FixedStart time format (expected HH:MM): %w", err)
		}
	}
	if c.FixedEnd != "" {
		if _, err := time.Parse(constants.TimeFormat, c.FixedEnd); err != nil {
			return fmt.Errorf("invalid FixedEnd time format (expected HH:MM): %w", err)
		}
	}

	// Validate FixedStart comes before FixedEnd
	if c.FixedStart != "" && c.FixedEnd != "" {
		start, _ := time.Parse(constants.TimeFormat, c.FixedStart) // Already validated above, won't fail
		end, _ := time.Parse(constants.TimeFormat, c.FixedEnd)     // Already validated above, won't fail
		if !start.Before(end) {
			return fmt.Errorf("fixedStart must be before FixedEnd")
		}
	}

	// Validate Earliest comes before Latest
	if c.Earliest != "" && c.Latest != "" {
		earliest, _ := time.Parse(constants.TimeFormat, c.Earliest) // Already validated above, won't fail
		latest, _ := time.Parse(constants.TimeFormat, c.Latest)     // Already validated above, won't fail
		if !earliest.Before(latest) {
			return fmt.Errorf("earliest must be before Latest")
		}

		// Validate duration fits within time window
		windowMinutes := int(latest.Sub(earliest).Minutes())
		if c.Duration > windowMinutes {
			return fmt.Errorf("duration (%d minutes) must fit within time window (%d minutes)", c.Duration, windowMinutes)
		}
	}

	return nil
}

func (c *TaskAddCmd) Run(ctx *cli.Context) error {
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
		wds, err := cli.ParseWeekdays(c.Weekdays)
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
