package tasks

import (
	"fmt"

	"github.com/julianstephens/daylit/daylit-cli/internal/cli"
)

type TaskDeleteCmd struct {
	ID string `arg:"" help:"Task ID to delete."`
}

func (c *TaskDeleteCmd) Run(ctx *cli.Context) error {
	// Check if task exists first
	task, err := ctx.Store.GetTask(c.ID)
	if err != nil {
		return fmt.Errorf("failed to find task with ID %s: %w", c.ID, err)
	}

	if err := ctx.Store.DeleteTask(c.ID); err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	fmt.Printf("Deleted task: %s (ID: %s)\n", task.Name, c.ID)
	return nil
}
