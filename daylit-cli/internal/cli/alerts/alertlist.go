package alerts

import (
	"fmt"
	"strings"

	"github.com/julianstephens/daylit/daylit-cli/internal/cli"
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

		recurrence := alert.FormatRecurrence()
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
