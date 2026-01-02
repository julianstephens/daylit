package tasks

import (
	"fmt"

	"github.com/julianstephens/daylit/daylit-cli/internal/cli"
)

type TaskRestoreCmd struct {
	ID string `arg:"" help:"Task ID to restore."`
}

func (c *TaskRestoreCmd) Run(ctx *cli.Context) error {
	if err := ctx.Store.RestoreTask(c.ID); err != nil {
		return fmt.Errorf("failed to restore task: %w", err)
	}

	fmt.Printf("Restored task with ID: %s\n", c.ID)
	return nil
}
