package alerts

import (
	"fmt"

	"github.com/julianstephens/daylit/daylit-cli/internal/cli"
)

type AlertDeleteCmd struct {
	ID string `arg:"" help:"Alert ID to delete."`
}

func (c *AlertDeleteCmd) Run(ctx *cli.Context) error {
	if err := ctx.Store.Load(); err != nil {
		return err
	}

	// Check if alert exists
	alert, err := ctx.Store.GetAlert(c.ID)
	if err != nil {
		return fmt.Errorf("alert not found: %w", err)
	}

	if err := ctx.Store.DeleteAlert(c.ID); err != nil {
		return fmt.Errorf("failed to delete alert: %w", err)
	}

	fmt.Printf("✓ Alert deleted: %s at %s\n", alert.Message, alert.Time)
	return nil
}
