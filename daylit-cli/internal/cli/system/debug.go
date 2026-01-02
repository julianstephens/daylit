package system

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/julianstephens/daylit/daylit-cli/internal/cli"
)

type DebugCmd struct {
	DBPath      *DebugDBPathCmd      `cmd:"" help:"Show database path."`
	DumpPlan    *DebugDumpPlanCmd    `cmd:"" help:"Dump plan data as JSON."`
	DumpTask    *DebugDumpTaskCmd    `cmd:"" help:"Dump task data as JSON."`
	DumpHabit   *DebugDumpHabitCmd   `cmd:"" help:"Dump habit data as JSON."`
	DumpOT      *DebugDumpOTCmd      `cmd:"" help:"Dump OT intention data as JSON."`
	DumpAlert   *DebugDumpAlertCmd   `cmd:"" help:"Dump alert data as JSON."`
	DumpSettings *DebugDumpSettingsCmd `cmd:"" help:"Dump settings data as JSON."`
}

type DebugDBPathCmd struct{}

func (cmd *DebugDBPathCmd) Run(ctx *cli.Context) error {
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

func (cmd *DebugDumpPlanCmd) Run(ctx *cli.Context) error {
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
		if errors.Is(err, sql.ErrNoRows) || err.Error() == "plan not found" {
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

func (cmd *DebugDumpTaskCmd) Run(ctx *cli.Context) error {
	// Load the database
	if err := ctx.Store.Load(); err != nil {
		return fmt.Errorf("failed to load database: %w", err)
	}

	// Get the task
	task, err := ctx.Store.GetTask(cmd.ID)
	if err != nil {
		// Check if it's a not found error
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("task not found: %s", cmd.ID)
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

type DebugDumpHabitCmd struct {
	ID string `arg:"" help:"ID of the habit to dump."`
}

func (cmd *DebugDumpHabitCmd) Run(ctx *cli.Context) error {
	// Load the database
	if err := ctx.Store.Load(); err != nil {
		return fmt.Errorf("failed to load database: %w", err)
	}

	// Get the habit
	habit, err := ctx.Store.GetHabit(cmd.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("habit not found: %s", cmd.ID)
		}
		return fmt.Errorf("failed to get habit: %w", err)
	}

	// Marshal to JSON
	jsonBytes, err := json.MarshalIndent(habit, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal habit: %w", err)
	}

	fmt.Println(string(jsonBytes))
	return nil
}

type DebugDumpOTCmd struct {
	Day string `arg:"" help:"Day of the OT entry to dump (YYYY-MM-DD or 'today')."`
}

func (cmd *DebugDumpOTCmd) Run(ctx *cli.Context) error {
	// Load the database
	if err := ctx.Store.Load(); err != nil {
		return fmt.Errorf("failed to load database: %w", err)
	}

	// Handle 'today' as a special case
	day := cmd.Day
	if day == "today" {
		day = getCurrentDate()
	}

	// Validate date format
	if !isValidDate(day) {
		return fmt.Errorf("invalid date format: %s (expected YYYY-MM-DD or 'today')", day)
	}

	// Get the OT entry
	ot, err := ctx.Store.GetOTEntry(day)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("OT entry not found for day: %s", day)
		}
		return fmt.Errorf("failed to get OT entry: %w", err)
	}

	// Marshal to JSON
	jsonBytes, err := json.MarshalIndent(ot, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal OT entry: %w", err)
	}

	fmt.Println(string(jsonBytes))
	return nil
}

type DebugDumpAlertCmd struct {
	ID string `arg:"" help:"ID of the alert to dump."`
}

func (cmd *DebugDumpAlertCmd) Run(ctx *cli.Context) error {
	// Load the database
	if err := ctx.Store.Load(); err != nil {
		return fmt.Errorf("failed to load database: %w", err)
	}

	// Get the alert
	alert, err := ctx.Store.GetAlert(cmd.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("alert not found: %s", cmd.ID)
		}
		return fmt.Errorf("failed to get alert: %w", err)
	}

	// Marshal to JSON
	jsonBytes, err := json.MarshalIndent(alert, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal alert: %w", err)
	}

	fmt.Println(string(jsonBytes))
	return nil
}

type DebugDumpSettingsCmd struct{}

func (cmd *DebugDumpSettingsCmd) Run(ctx *cli.Context) error {
	// Load the database
	if err := ctx.Store.Load(); err != nil {
		return fmt.Errorf("failed to load database: %w", err)
	}

	// Get the settings
	settings, err := ctx.Store.GetSettings()
	if err != nil {
		return fmt.Errorf("failed to get settings: %w", err)
	}

	// Marshal to JSON
	jsonBytes, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	fmt.Println(string(jsonBytes))
	return nil
}
