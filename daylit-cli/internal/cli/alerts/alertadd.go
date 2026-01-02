package alerts

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/julianstephens/daylit/daylit-cli/internal/cli"
	"github.com/julianstephens/daylit/daylit-cli/internal/models"
	"github.com/julianstephens/daylit/daylit-cli/internal/utils"
)

type AlertAddCmd struct {
	Message    string `arg:"" help:"Alert message."`
	Time       string `help:"Time for alert (HH:MM)." required:""`
	Date       string `help:"Date for one-time alert (YYYY-MM-DD)."`
	Recurrence string `help:"Recurrence type (daily|weekly|n_days). Required if --date not set."`
	Interval   int    `help:"Interval for n_days recurrence." default:"1"`
	Weekdays   string `help:"Comma-separated weekdays for weekly recurrence (e.g., mon,wed,fri)."`
}

func (c *AlertAddCmd) Validate() error {
	// Validate time format
	if _, err := utils.ParseTime(c.Time); err != nil {
		return fmt.Errorf("invalid time format (expected HH:MM): %w", err)
	}

	// One-time alert (date specified)
	if c.Date != "" {
		if _, err := time.Parse("2006-01-02", c.Date); err != nil {
			return fmt.Errorf("invalid date format (expected YYYY-MM-DD): %w", err)
		}
		// If date is specified, recurrence should not be
		if c.Recurrence != "" {
			return fmt.Errorf("cannot specify both date and recurrence")
		}
		return nil
	}

	// Recurring alert (no date)
	if c.Recurrence == "" {
		return fmt.Errorf("must specify either --date for one-time alert or --recurrence for recurring alert")
	}

	// Validate recurrence type
	validRecurrence := map[string]bool{
		"daily":  true,
		"weekly": true,
		"n_days": true,
	}
	if !validRecurrence[c.Recurrence] {
		return fmt.Errorf("invalid recurrence type: %s (must be daily, weekly, or n_days)", c.Recurrence)
	}

	// Validate weekly recurrence has weekdays
	if c.Recurrence == "weekly" && c.Weekdays == "" {
		return fmt.Errorf("weekdays must be specified for weekly recurrence")
	}

	// Validate interval for n_days recurrence
	if c.Recurrence == "n_days" && c.Interval < 1 {
		return fmt.Errorf("interval must be at least 1 for n_days recurrence")
	}

	return nil
}

func (c *AlertAddCmd) Run(ctx *cli.Context) error {
	if err := c.Validate(); err != nil {
		return err
	}

	if err := ctx.Store.Load(); err != nil {
		return err
	}

	alert := models.Alert{
		ID:        uuid.New().String(),
		Message:   c.Message,
		Time:      c.Time,
		Date:      c.Date,
		Active:    true,
		CreatedAt: time.Now(),
	}

	// Set recurrence if not one-time
	if c.Date == "" {
		alert.Recurrence.Type = models.RecurrenceType(c.Recurrence)
		alert.Recurrence.IntervalDays = c.Interval

		// Parse weekdays for weekly recurrence
		if c.Recurrence == "weekly" {
			weekdays, err := cli.ParseWeekdays(c.Weekdays)
			if err != nil {
				return fmt.Errorf("failed to parse weekdays: %w", err)
			}
			alert.Recurrence.WeekdayMask = weekdays
		}
	}

	if err := ctx.Store.AddAlert(alert); err != nil {
		return fmt.Errorf("failed to add alert: %w", err)
	}

	fmt.Printf("✓ Alert added: %s at %s", alert.Message, alert.Time)
	if alert.Date != "" {
		fmt.Printf(" on %s", alert.Date)
	} else {
		fmt.Printf(" (%s)", alert.FormatRecurrence())
	}
	fmt.Println()

	return nil
}
