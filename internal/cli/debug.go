package cli

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/julianstephens/daylit/internal/storage"
)

type DebugCmd struct {
	DBPath   *DebugDBPathCmd   `cmd:"" help:"Show database path."`
	DumpPlan *DebugDumpPlanCmd `cmd:"" help:"Dump plan data as JSON."`
	DumpTask *DebugDumpTaskCmd `cmd:"" help:"Dump task data as JSON."`
}

type DebugDBPathCmd struct{}

func (cmd *DebugDBPathCmd) Run(ctx *Context) error {
	path := ctx.Store.GetConfigPath()

	// Output in machine-readable format
	output := map[string]string{
		"path": path,
	}

	jsonBytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal output: %w", err)
	}

	fmt.Println(string(jsonBytes))
	return nil
}

type DebugDumpPlanCmd struct {
	Date string `arg:"" help:"Date of the plan to dump (YYYY-MM-DD or 'today')."`
}

func (cmd *DebugDumpPlanCmd) Run(ctx *Context) error {
	// Load the database
	if err := ctx.Store.Load(); err != nil {
		return fmt.Errorf("failed to load database: %w", err)
	}

	// Handle 'today' as a special case
	date := cmd.Date
	if date == "today" {
		date = getCurrentDate()
	}

	// Validate date format
	if !isValidDate(date) {
		return fmt.Errorf("invalid date format: %s (expected YYYY-MM-DD or 'today')", date)
	}

	// Get the plan
	plan, err := ctx.Store.GetPlan(date)
	if err != nil {
		// Try to provide a helpful error message
		if err.Error() == "sql: no rows in result set" || err.Error() == "plan not found" {
			return fmt.Errorf("no plan found for date: %s", date)
		}
		return fmt.Errorf("failed to get plan: %w", err)
	}

	// Marshal to JSON
	jsonBytes, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal plan: %w", err)
	}

	fmt.Println(string(jsonBytes))
	return nil
}

type DebugDumpTaskCmd struct {
	ID string `arg:"" help:"ID of the task to dump."`
}

func (cmd *DebugDumpTaskCmd) Run(ctx *Context) error {
	// Load the database
	if err := ctx.Store.Load(); err != nil {
		return fmt.Errorf("failed to load database: %w", err)
	}

	// Get the task
	task, err := ctx.Store.GetTask(cmd.ID)
	if err != nil {
		// Check if it's a not found error
		sqliteStore, ok := ctx.Store.(*storage.SQLiteStore)
		if ok && sqliteStore != nil {
			// For SQLite, sql.ErrNoRows means not found
			if err.Error() == "sql: no rows in result set" {
				return fmt.Errorf("task not found: %s", cmd.ID)
			}
		}
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Marshal to JSON
	jsonBytes, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	fmt.Println(string(jsonBytes))
	return nil
}

func getCurrentDate() string {
	return time.Now().Format("2006-01-02")
}

func isValidDate(dateStr string) bool {
	_, err := time.Parse("2006-01-02", dateStr)
	return err == nil
}
