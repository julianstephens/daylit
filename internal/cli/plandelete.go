package cli

import (
	"fmt"
)

type PlanDeleteCmd struct {
	Date string `arg:"" help:"Date of the plan to delete (YYYY-MM-DD)."`
}

func (c *PlanDeleteCmd) Run(ctx *Context) error {
	if err := ctx.Store.Load(); err != nil {
		return err
	}

	// Check if plan exists first
	_, err := ctx.Store.GetPlan(c.Date)
	if err != nil {
		return fmt.Errorf("failed to find plan for date %s: %w", c.Date, err)
	}

	if err := ctx.Store.DeletePlan(c.Date); err != nil {
		return fmt.Errorf("failed to delete plan: %w", err)
	}

	fmt.Printf("Deleted plan for date: %s\n", c.Date)
	return nil
}
