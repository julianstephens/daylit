package tasks

import (
	"fmt"

	"github.com/julianstephens/daylit/daylit-cli/internal/cli"
	"github.com/julianstephens/daylit/daylit-cli/internal/constants"
	"github.com/julianstephens/daylit/daylit-cli/internal/utils"
)

type TaskEditCmd struct {
	ID               string  `arg:"" help:"Task ID."`
	Name             *string `help:"New task name."`
	Duration         *int    `short:"d" help:"New duration in minutes."`
	Recurrence       *string `short:"r" help:"New recurrence type (daily|weekly|n_days|ad_hoc|monthly_date|monthly_day|yearly|weekdays)."`
	Interval         *int    `short:"i" help:"New interval for n_days recurrence."`
	Weekdays         *string `short:"w" help:"New comma-separated weekdays for weekly recurrence."`
	MonthDay         *int    `help:"New day of month (1-31) for monthly_date or yearly recurrence."`
	Month            *int    `help:"New month (1-12) for yearly recurrence."`
	WeekOccurrence   *int    `help:"New week occurrence for monthly_day recurrence (-1=last, 1=first, 2=second, etc.)."`
	DayOfWeekInMonth *string `help:"New day of week for monthly_day recurrence (e.g., 'monday', 'friday')."`
	Earliest         *string `short:"s" help:"New earliest start time (HH:MM)."`
	Latest           *string `short:"e" help:"New latest end time (HH:MM)."`
	FixedStart       *string `short:"S" help:"New fixed start time for appointments (HH:MM)."`
	FixedEnd         *string `short:"E" help:"New fixed end time for appointments (HH:MM)."`
	Priority         *int    `short:"p" help:"New priority (1-5)."`
	Active           *bool   `help:"Set active status."`
}

func (c *TaskEditCmd) Run(ctx *cli.Context) error {
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
			task.Recurrence.Type = constants.RecurrenceDaily
		case "weekly":
			task.Recurrence.Type = constants.RecurrenceWeekly
		case "n_days":
			task.Recurrence.Type = constants.RecurrenceNDays
		case "ad_hoc":
			task.Recurrence.Type = constants.RecurrenceAdHoc
		case "monthly_date":
			task.Recurrence.Type = constants.RecurrenceMonthlyDate
		case "monthly_day":
			task.Recurrence.Type = constants.RecurrenceMonthlyDay
		case "yearly":
			task.Recurrence.Type = constants.RecurrenceYearly
		case "weekdays":
			task.Recurrence.Type = constants.RecurrenceWeekdays
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

	if c.MonthDay != nil {
		if *c.MonthDay < 1 || *c.MonthDay > 31 {
			return fmt.Errorf("month_day must be between 1 and 31")
		}
		task.Recurrence.MonthDay = *c.MonthDay
	}

	if c.Month != nil {
		if *c.Month < 1 || *c.Month > 12 {
			return fmt.Errorf("month must be between 1 and 12")
		}
		task.Recurrence.Month = *c.Month
	}

	if c.WeekOccurrence != nil {
		if *c.WeekOccurrence < -1 || *c.WeekOccurrence == 0 || *c.WeekOccurrence > 5 {
			return fmt.Errorf("week_occurrence must be -1 (last) or 1-5")
		}
		task.Recurrence.WeekOccurrence = *c.WeekOccurrence
	}

	if c.DayOfWeekInMonth != nil {
		wd, err := cli.ParseWeekday(*c.DayOfWeekInMonth)
		if err != nil {
			return fmt.Errorf("invalid day_of_week_in_month: %w", err)
		}
		task.Recurrence.DayOfWeekInMonth = wd
	}

	// Update time constraints
	if c.Earliest != nil {
		if _, err := utils.ParseTime(*c.Earliest); err != nil {
			return fmt.Errorf("invalid earliest time: %w", err)
		}
		task.EarliestStart = *c.Earliest
	}
	if c.Latest != nil {
		if _, err := utils.ParseTime(*c.Latest); err != nil {
			return fmt.Errorf("invalid latest time: %w", err)
		}
		task.LatestEnd = *c.Latest
	}
	if c.FixedStart != nil {
		if _, err := utils.ParseTime(*c.FixedStart); err != nil {
			return fmt.Errorf("invalid fixed start time: %w", err)
		}
		task.FixedStart = *c.FixedStart
	}
	if c.FixedEnd != nil {
		if _, err := utils.ParseTime(*c.FixedEnd); err != nil {
			return fmt.Errorf("invalid fixed end time: %w", err)
		}
		task.FixedEnd = *c.FixedEnd
	}

	// Update kind based on fixed times
	if task.FixedStart != "" && task.FixedEnd != "" {
		task.Kind = constants.TaskKindAppointment
	} else {
		task.Kind = constants.TaskKindFlexible
	}

	if err := task.Validate(); err != nil {
		return fmt.Errorf("invalid task: %w", err)
	}

	if err := ctx.Store.UpdateTask(task); err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	fmt.Printf("Task updated: %s\n", task.Name)
	return nil
}
