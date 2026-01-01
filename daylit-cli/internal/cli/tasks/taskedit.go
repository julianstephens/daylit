package tasks

import (
	"fmt"
	"time"

	"github.com/julianstephens/daylit/daylit-cli/internal/cli"
	"github.com/julianstephens/daylit/daylit-cli/internal/constants"
	"github.com/julianstephens/daylit/daylit-cli/internal/models"
)

type TaskEditCmd struct {
	ID         string  `arg:"" help:"Task ID."`
	Name       *string `help:"New task name."`
	Duration   *int    `short:"d" help:"New duration in minutes."`
	Recurrence *string `short:"r" help:"New recurrence type (daily|weekly|n_days|ad_hoc)."`
	Interval   *int    `short:"i" help:"New interval for n_days recurrence."`
	Weekdays   *string `short:"w" help:"New comma-separated weekdays for weekly recurrence."`
	Earliest   *string `short:"s" help:"New earliest start time (HH:MM)."`
	Latest     *string `short:"e" help:"New latest end time (HH:MM)."`
	FixedStart *string `short:"S" help:"New fixed start time for appointments (HH:MM)."`
	FixedEnd   *string `short:"E" help:"New fixed end time for appointments (HH:MM)."`
	Priority   *int    `short:"p" help:"New priority (1-5)."`
	Active     *bool   `help:"Set active status."`
}

func (c *TaskEditCmd) Run(ctx *cli.Context) error {
	if err := ctx.Store.Load(); err != nil {
		return err
	}

	task, err := ctx.Store.GetTask(c.ID)
	if err != nil {
		return fmt.Errorf("failed to find task: %w", err)
	}

	if c.Name != nil {
		task.Name = *c.Name
	}
	if c.Duration != nil {
		if *c.Duration <= 0 {
			return fmt.Errorf("duration must be positive")
		}
		task.DurationMin = *c.Duration
	}
	if c.Priority != nil {
		if *c.Priority < 1 || *c.Priority > 5 {
			return fmt.Errorf("priority must be between 1 and 5")
		}
		task.Priority = *c.Priority
	}
	if c.Active != nil {
		task.Active = *c.Active
	}

	// Update recurrence
	if c.Recurrence != nil {
		switch *c.Recurrence {
		case "daily":
			task.Recurrence.Type = models.RecurrenceDaily
		case "weekly":
			task.Recurrence.Type = models.RecurrenceWeekly
		case "n_days":
			task.Recurrence.Type = models.RecurrenceNDays
		case "ad_hoc":
			task.Recurrence.Type = models.RecurrenceAdHoc
		default:
			return fmt.Errorf("invalid recurrence type: %s", *c.Recurrence)
		}
	}

	if c.Interval != nil {
		if *c.Interval <= 0 {
			return fmt.Errorf("interval must be positive")
		}
		task.Recurrence.IntervalDays = *c.Interval
	}

	if c.Weekdays != nil {
		weekdays, err := cli.ParseWeekdays(*c.Weekdays)
		if err != nil {
			return err
		}
		task.Recurrence.WeekdayMask = weekdays
	}

	// Update time constraints
	if c.Earliest != nil {
		if _, err := time.Parse(constants.TimeFormat, *c.Earliest); err != nil {
			return fmt.Errorf("invalid earliest time: %w", err)
		}
		task.EarliestStart = *c.Earliest
	}
	if c.Latest != nil {
		if _, err := time.Parse(constants.TimeFormat, *c.Latest); err != nil {
			return fmt.Errorf("invalid latest time: %w", err)
		}
		task.LatestEnd = *c.Latest
	}
	if c.FixedStart != nil {
		if _, err := time.Parse(constants.TimeFormat, *c.FixedStart); err != nil {
			return fmt.Errorf("invalid fixed start time: %w", err)
		}
		task.FixedStart = *c.FixedStart
	}
	if c.FixedEnd != nil {
		if _, err := time.Parse(constants.TimeFormat, *c.FixedEnd); err != nil {
			return fmt.Errorf("invalid fixed end time: %w", err)
		}
		task.FixedEnd = *c.FixedEnd
	}

	// Update kind based on fixed times
	if task.FixedStart != "" && task.FixedEnd != "" {
		task.Kind = models.TaskKindAppointment
	} else {
		task.Kind = models.TaskKindFlexible
	}

	if err := ctx.Store.UpdateTask(task); err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	fmt.Printf("Task updated: %s\n", task.Name)
	return nil
}
