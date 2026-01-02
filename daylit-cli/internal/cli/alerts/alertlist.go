package alerts

import (
	"fmt"
	"strings"

	"github.com/julianstephens/daylit/daylit-cli/internal/cli"
	"github.com/julianstephens/daylit/daylit-cli/internal/models"
)

type AlertListCmd struct{}

func (c *AlertListCmd) Run(ctx *cli.Context) error {
	if err := ctx.Store.Load(); err != nil {
		return err
	}

	alerts, err := ctx.Store.GetAllAlerts()
	if err != nil {
		return fmt.Errorf("failed to get alerts: %w", err)
	}

	if len(alerts) == 0 {
		fmt.Println("No alerts configured.")
		return nil
	}

	fmt.Printf("%-36s %-30s %-8s %-20s %-8s\n", "ID", "Message", "Time", "Recurrence", "Active")
	fmt.Println(strings.Repeat("-", 110))

	for _, alert := range alerts {
		message := alert.Message
		if len(message) > 28 {
			message = message[:25] + "..."
		}

		recurrence := formatRecurrenceForList(alert)
		if len(recurrence) > 18 {
			recurrence = recurrence[:15] + "..."
		}

		activeStr := "Yes"
		if !alert.Active {
			activeStr = "No"
		}

		fmt.Printf("%-36s %-30s %-8s %-20s %-8s\n",
			alert.ID, message, alert.Time, recurrence, activeStr)
	}

	return nil
}

func formatRecurrenceForList(alert models.Alert) string {
	if alert.Date != "" {
		return fmt.Sprintf("Once on %s", alert.Date)
	}

	switch alert.Recurrence.Type {
	case models.RecurrenceDaily:
		return "Daily"
	case models.RecurrenceWeekly:
		days := make([]string, len(alert.Recurrence.WeekdayMask))
		for i, wd := range alert.Recurrence.WeekdayMask {
			days[i] = wd.String()[:3]
		}
		return fmt.Sprintf("Weekly: %s", strings.Join(days, ","))
	case models.RecurrenceNDays:
		if alert.Recurrence.IntervalDays == 1 {
			return "Daily"
		}
		return fmt.Sprintf("Every %dd", alert.Recurrence.IntervalDays)
	default:
		return "One-time"
	}
}
