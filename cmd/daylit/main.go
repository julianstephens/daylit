package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/alecthomas/kong"

	"github.com/julianstephens/daylit/internal/cli"
	_ "github.com/julianstephens/daylit/internal/constants" // import for init validation
	"github.com/julianstephens/daylit/internal/scheduler"
	"github.com/julianstephens/daylit/internal/storage"
)

var CLI struct {
	Version kong.VersionFlag
	Config  string `help:"Config file path." type:"path" default:"~/.config/daylit/daylit.db"`

	Init     cli.InitCmd     `cmd:"" help:"Initialize daylit storage."`
	Tui      cli.TuiCmd      `cmd:"" help:"Launch the interactive TUI." default:"1"`
	Plan     cli.PlanCmd     `cmd:"" help:"Generate day plans."`
	Now      cli.NowCmd      `cmd:"" help:"Show current task."`
	Feedback cli.FeedbackCmd `cmd:"" help:"Provide feedback on a slot."`
	Day      cli.DayCmd      `cmd:"" help:"Show plan for a day."`
	Backup   struct {
		Create  cli.BackupCreateCmd  `cmd:"" help:"Create a manual backup." default:"1"`
		List    cli.BackupListCmd    `cmd:"" help:"List available backups."`
		Restore cli.BackupRestoreCmd `cmd:"" help:"Restore from a backup."`
	} `cmd:"" help:"Manage database backups."`
	Task struct {
		Add    cli.TaskAddCmd    `cmd:"" help:"Add a new task."`
		Edit   cli.TaskEditCmd   `cmd:"" help:"Edit an existing task."`
		Delete cli.TaskDeleteCmd `cmd:"" help:"Delete a task."`
		List   cli.TaskListCmd   `cmd:"" help:"List all tasks."`
	} `cmd:"" help:"Manage tasks."`
}

func main() {
	ctx := kong.Parse(&CLI,
		kong.Name("daylit"),
		kong.Description("Daily structure scheduler / time-blocking companion"),
		kong.UsageOnError(),
		kong.Vars{"version": "v0.2.0"},
	)

	// Determine storage type based on extension
	var store storage.Provider
	if filepath.Ext(CLI.Config) == ".json" {
		store = storage.NewJSONStore(CLI.Config)
	} else {
		store = storage.NewSQLiteStore(CLI.Config)
	}

	appCtx := &cli.Context{
		Store:     store,
		Scheduler: scheduler.New(),
	}

	err := ctx.Run(appCtx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
