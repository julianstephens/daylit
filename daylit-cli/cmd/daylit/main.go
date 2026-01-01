package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/alecthomas/kong"

	"github.com/julianstephens/daylit/daylit-cli/internal/cli"
	"github.com/julianstephens/daylit/daylit-cli/internal/cli/backups"
	"github.com/julianstephens/daylit/daylit-cli/internal/cli/habits"
	"github.com/julianstephens/daylit/daylit-cli/internal/cli/ot"
	"github.com/julianstephens/daylit/daylit-cli/internal/cli/plans"
	"github.com/julianstephens/daylit/daylit-cli/internal/cli/settings"
	"github.com/julianstephens/daylit/daylit-cli/internal/cli/system"
	"github.com/julianstephens/daylit/daylit-cli/internal/cli/tasks"
	_ "github.com/julianstephens/daylit/daylit-cli/internal/constants"
	"github.com/julianstephens/daylit/daylit-cli/internal/scheduler"
	"github.com/julianstephens/daylit/daylit-cli/internal/storage"
)

var CLI struct {
	Version kong.VersionFlag
	Config  string `help:"Config file path or PostgreSQL connection string. For PostgreSQL, credentials must NOT be embedded in the connection string. Use environment variables or a .pgpass file instead." type:"string" default:"~/.config/daylit/daylit.db" env:"DAYLIT_CONFIG"`

	Init     system.InitCmd     `cmd:"" help:"Initialize daylit storage."`
	Migrate  system.MigrateCmd  `cmd:"" help:"Run database migrations."`
	Doctor   system.DoctorCmd   `cmd:"" help:"Run health checks and diagnostics."`
	Tui      system.TuiCmd      `cmd:"" help:"Launch the interactive TUI." default:"1"`
	Plan     plans.PlanCmd      `cmd:"" help:"Generate day plans."`
	Now      plans.NowCmd       `cmd:"" help:"Show current task."`
	Feedback plans.FeedbackCmd  `cmd:"" help:"Provide feedback on a slot."`
	Day      plans.DayCmd       `cmd:"" help:"Show plan for a day."`
	Debug    system.DebugCmd    `cmd:"" help:"Debug commands for troubleshooting."`
	Validate system.ValidateCmd `cmd:"" help:"Validate tasks and plans for conflicts."`
	Backup   struct {
		Create  backups.BackupCreateCmd  `cmd:"" help:"Create a manual backup." default:"1"`
		List    backups.BackupListCmd    `cmd:"" help:"List available backups."`
		Restore backups.BackupRestoreCmd `cmd:"" help:"Restore from a backup."`
	} `cmd:"" help:"Manage database backups."`
	Task struct {
		Add    tasks.TaskAddCmd    `cmd:"" help:"Add a new task."`
		Edit   tasks.TaskEditCmd   `cmd:"" help:"Edit an existing task."`
		Delete tasks.TaskDeleteCmd `cmd:"" help:"Delete a task."`
		List   tasks.TaskListCmd   `cmd:"" help:"List all tasks."`
	} `cmd:"" help:"Manage tasks."`
	Plans struct {
		Delete plans.PlanDeleteCmd `cmd:"" help:"Delete a plan."`
	} `cmd:"" help:"Manage plans."`
	Restore struct {
		Task tasks.TaskRestoreCmd `cmd:"" help:"Restore a deleted task."`
		Plan plans.PlanRestoreCmd `cmd:"" help:"Restore a deleted plan."`
	} `cmd:"" help:"Restore deleted items."`
	Habit    habits.HabitCmd      `cmd:"" help:"Manage habits and habit tracking."`
	OT       ot.OTCmd             `cmd:"" help:"Manage Once-Today (OT) intentions."`
	Settings settings.SettingsCmd `cmd:"" help:"Manage application settings."`
	Notify   system.NotifyCmd     `cmd:"" hidden:"" help:"Send a notification (used internally)."`
}

func main() {
	ctx := kong.Parse(&CLI,
		kong.Name("daylit"),
		kong.Description("Daily structure scheduler / time-blocking companion"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact:             true,
			NoExpandSubcommands: true,
		}),
		kong.Vars{"version": "v0.4.0"},
	)

	// Initialize storage based on config format
	var store storage.Provider

	// Check for Postgres URL or DSN format
	isPostgres := strings.HasPrefix(CLI.Config, "postgres://") ||
		strings.HasPrefix(CLI.Config, "postgresql://") ||
		// Simple DSN heuristic: contains space and common keys
		(strings.Contains(CLI.Config, " ") &&
			(strings.Contains(CLI.Config, "host=") ||
				strings.Contains(CLI.Config, "dbname=") ||
				strings.Contains(CLI.Config, "user=") ||
				strings.Contains(CLI.Config, "sslmode=")))

	if isPostgres {
		// PostgreSQL connection string detected - validate for embedded credentials
		// We only enforce this check if the config was NOT sourced from the environment
		// (e.g. came from command line flags, which are visible in the process list).
		envConfig := os.Getenv("DAYLIT_CONFIG")
		configFromEnv := envConfig != "" && envConfig == CLI.Config

		_, err := storage.ValidatePostgresConnString(CLI.Config)
		hasPasswordError := err != nil && errors.Is(err, storage.ErrEmbeddedCredentials)

		if !configFromEnv && hasPasswordError {
			fmt.Fprintf(os.Stderr, "❌ Error: PostgreSQL connection strings with embedded credentials are NOT allowed via command line flags.\n")
			fmt.Fprintf(os.Stderr, "       Use one of these secure alternatives:\n")
			fmt.Fprintf(os.Stderr, "       1. Environment:   export DAYLIT_CONFIG=\"postgresql://user:your_password@host:5432/daylit\"\n")
			fmt.Fprintf(os.Stderr, "       2. .pgpass file:  Create ~/.pgpass with credentials\n")
			fmt.Fprintf(os.Stderr, "\n       For more information, see docs/user-guides/POSTGRES_SETUP.md\n")
			os.Exit(1)
		} else if configFromEnv && hasPasswordError {
			// Warn user about embedded credentials in environment variable
			fmt.Fprintf(os.Stderr, "⚠️  Warning: Using embedded credentials in DAYLIT_CONFIG environment variable.\n")
			fmt.Fprintf(os.Stderr, "            Consider using a .pgpass file for better security.\n")
		}
		store = storage.NewPostgresStore(CLI.Config)
	} else {
		// Default to SQLite
		store = storage.NewSQLiteStore(CLI.Config)
	}

	appCtx := &cli.Context{
		Store:     store,
		Scheduler: scheduler.New(),
	}

	// Load the store before running the command (Init command will handle its own loading)
	if !CLI.Init.Force && ctx.Selected() != nil && ctx.Selected().Name != "init" {
		if err := store.Load(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}

	err := ctx.Run(appCtx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
