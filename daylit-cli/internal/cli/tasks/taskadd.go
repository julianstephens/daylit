package tasks

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/julianstephens/daylit/daylit-cli/internal/cli"
	"github.com/julianstephens/daylit/daylit-cli/internal/constants"
	"github.com/julianstephens/daylit/daylit-cli/internal/models"
	"github.com/julianstephens/daylit/daylit-cli/internal/utils"
)

type TaskAddCmd struct {
	Name             string `arg:"" help:"Task name."`
	Duration         int    `short:"d" help:"Duration in minutes." required:""`
	Recurrence       string `short:"r" help:"Recurrence type (daily|weekly|n_days|ad_hoc|monthly_date|monthly_day|yearly|weekdays)." default:"ad_hoc"`
	Interval         int    `short:"i" help:"Interval for n_days recurrence." default:"1"`
	Weekdays         string `short:"w" help:"Comma-separated weekdays for weekly recurrence."`
	MonthDay         int    `help:"Day of month (1-31) for monthly_date or yearly recurrence."`
	Month            int    `help:"Month (1-12) for yearly recurrence."`
	WeekOccurrence   int    `help:"Week occurrence for monthly_day recurrence (-1=last, 1=first, 2=second, etc.)."`
	DayOfWeekInMonth string `help:"Day of week for monthly_day recurrence (e.g., 'monday', 'friday')."`
	Earliest         string `short:"s" help:"Earliest start time (HH:MM)."`
	Latest           string `short:"e" help:"Latest end time (HH:MM)."`
	FixedStart       string `short:"S" help:"Fixed start time for appointments (HH:MM)."`
	FixedEnd         string `short:"E" help:"Fixed end time for appointments (HH:MM)."`
	Priority         int    `short:"p" help:"Priority (1-5, lower is higher priority)." default:"3"`
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

	// Validate monthly_date recurrence
	if c.Recurrence == "monthly_date" {
		if c.MonthDay < 1 || c.MonthDay > 31 {
			return fmt.Errorf("--month-day must be between 1 and 31 for monthly_date recurrence")
		}
		// Note: We allow day 31 even though some months don't have it.
		// The scheduler will skip those months.
	}

	// Validate monthly_day recurrence
	if c.Recurrence == "monthly_day" {
		if c.WeekOccurrence < -1 || c.WeekOccurrence == 0 || c.WeekOccurrence > 5 {
			return fmt.Errorf("--week-occurrence must be -1 (last) or 1-5 for monthly_day recurrence")
		}
		if c.DayOfWeekInMonth == "" {
			return fmt.Errorf("--day-of-week-in-month must be specified for monthly_day recurrence")
		}
	}

	// Validate yearly recurrence
	if c.Recurrence == "yearly" {
		if c.Month < 1 || c.Month > 12 {
			return fmt.Errorf("--month must be between 1 and 12 for yearly recurrence")
		}
		if c.MonthDay < 1 || c.MonthDay > 31 {
			return fmt.Errorf("--month-day must be between 1 and 31 for yearly recurrence")
		}
		// Note: We allow potentially invalid dates like Feb 31.
		// The scheduler will skip years where this date doesn't exist.
	}

	// Validate time formats
	if c.Earliest != "" {
		if _, err := utils.ParseTime(c.Earliest); err != nil {
			return fmt.Errorf("invalid Earliest time format (expected HH:MM): %w", err)
		}
	}
	if c.Latest != "" {
		if _, err := utils.ParseTime(c.Latest); err != nil {
			return fmt.Errorf("invalid Latest time format (expected HH:MM): %w", err)
		}
	}
	if c.FixedStart != "" {
		if _, err := utils.ParseTime(c.FixedStart); err != nil {
			return fmt.Errorf("invalid FixedStart time format (expected HH:MM): %w", err)
		}
	}
	if c.FixedEnd != "" {
		if _, err := utils.ParseTime(c.FixedEnd); err != nil {
			return fmt.Errorf("invalid FixedEnd time format (expected HH:MM): %w", err)
		}
	}

	// Validate FixedStart comes before FixedEnd
	if c.FixedStart != "" && c.FixedEnd != "" {
		start, _ := utils.ParseTime(c.FixedStart) // Already validated above, won't fail
		end, _ := utils.ParseTime(c.FixedEnd)     // Already validated above, won't fail
		if !start.Before(end) {
			return fmt.Errorf("fixedStart must be before FixedEnd")
		}
	}

	// Validate Earliest comes before Latest
	if c.Earliest != "" && c.Latest != "" {
		earliest, _ := utils.ParseTime(c.Earliest) // Already validated above, won't fail
		latest, _ := utils.ParseTime(c.Latest)     // Already validated above, won't fail
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
	// Determine task kind
	taskKind := constants.TaskKindFlexible
	if c.FixedStart != "" && c.FixedEnd != "" {
		taskKind = constants.TaskKindAppointment
	}

	// Parse recurrence
	var recType constants.RecurrenceType
	switch c.Recurrence {
	case "daily":
		recType = constants.RecurrenceDaily
	case "weekly":
		recType = constants.RecurrenceWeekly
	case "n_days":
		recType = constants.RecurrenceNDays
	case "ad_hoc":
		recType = constants.RecurrenceAdHoc
	case "monthly_date":
		recType = constants.RecurrenceMonthlyDate
	case "monthly_day":
		recType = constants.RecurrenceMonthlyDay
	case "yearly":
		recType = constants.RecurrenceYearly
	case "weekdays":
		recType = constants.RecurrenceWeekdays
	default:
		return fmt.Errorf("invalid recurrence type: %s", c.Recurrence)
	}

	rec := models.Recurrence{
		Type:         recType,
		IntervalDays: c.Interval,
	}

	// Parse weekdays for weekly recurrence
	if recType == constants.RecurrenceWeekly && c.Weekdays != "" {
		wds, err := cli.ParseWeekdays(c.Weekdays)
		if err != nil {
			return err
		}
		rec.WeekdayMask = wds
	}

	// Set fields for monthly_date recurrence
	if recType == constants.RecurrenceMonthlyDate {
		rec.MonthDay = c.MonthDay
	}

	// Set fields for monthly_day recurrence
	if recType == constants.RecurrenceMonthlyDay {
		rec.WeekOccurrence = c.WeekOccurrence
		wd, err := cli.ParseWeekday(c.DayOfWeekInMonth)
		if err != nil {
			return fmt.Errorf("invalid --day-of-week-in-month: %w", err)
		}
		rec.DayOfWeekInMonth = wd
	}

	// Set fields for yearly recurrence
	if recType == constants.RecurrenceYearly {
		rec.Month = c.Month
		rec.MonthDay = c.MonthDay
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

	if err := task.Validate(); err != nil {
		return fmt.Errorf("invalid task: %w", err)
	}

	if err := ctx.Store.AddTask(task); err != nil {
		return err
	}

	fmt.Printf("Added task: %s (ID: %s)\n", c.Name, task.ID)
	return nil
}
