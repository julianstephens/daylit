package plans

import (
	"fmt"

	"github.com/julianstephens/daylit/daylit-cli/internal/cli"
)

type PlanRestoreCmd struct {
	Date string `arg:"" help:"Date of the plan to restore (YYYY-MM-DD)."`
}

func (c *PlanRestoreCmd) Run(ctx *cli.Context) error {
	if err := ctx.Store.Load(); err != nil {
		return err
	}

	if err := ctx.Store.RestorePlan(c.Date); err != nil {
		return fmt.Errorf("failed to restore plan: %w", err)
	}

	fmt.Printf("Restored plan for date: %s\n", c.Date)
	return nil
}
