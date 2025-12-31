package cli

import (
	"fmt"
	"os"
)

type InitCmd struct {
	Force bool `help:"Force reset by deleting existing database before initialization."`
}

func (c *InitCmd) Run(ctx *Context) error {
	// If force flag is provided, delete existing database
	if c.Force {
		dbPath := ctx.Store.GetConfigPath()
		if _, err := os.Stat(dbPath); err == nil {
			// Database exists, delete it
			if err := os.Remove(dbPath); err != nil {
				return fmt.Errorf("failed to delete existing database: %w", err)
			}
			fmt.Printf("Deleted existing database at: %s\n", dbPath)
		}
	}

	if err := ctx.Store.Init(); err != nil {
		return err
	}
	fmt.Printf("Initialized daylit storage at: %s\n", ctx.Store.GetConfigPath())
	return nil
}
