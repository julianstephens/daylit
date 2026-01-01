package main

import (
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
	Config  string `help:"Config file path or PostgreSQL connection string. For PostgreSQL, credentials must NOT be embedded in the connection string. Use environment variables, .pgpass, or OS keyring instead." type:"string" default:"~/.config/daylit/daylit.db"`

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
	if strings.HasPrefix(CLI.Config, "postgres://") || strings.HasPrefix(CLI.Config, "postgresql://") {
		// PostgreSQL connection string detected - validate for embedded credentials
		if storage.HasEmbeddedCredentials(CLI.Config) {
			fmt.Fprintf(os.Stderr, "❌ Error: PostgreSQL connection strings with embedded credentials are NOT allowed.\n")
			fmt.Fprintf(os.Stderr, "       Use one of these secure alternatives:\n")
			fmt.Fprintf(os.Stderr, "       1. OS keyring:    daylit config set connection-string \"postgresql://user:password@host:5432/daylit\"\n")
			fmt.Fprintf(os.Stderr, "       2. Environment:   export DAYLIT_DB_CONNECTION=\"postgresql://user:password@host:5432/daylit\"\n")
			fmt.Fprintf(os.Stderr, "       3. .pgpass file:  Use connection string without password: \"postgresql://user@host:5432/daylit\"\n")
			fmt.Fprintf(os.Stderr, "\n       For more information, see docs/POSTGRES_SETUP.md\n")
			os.Exit(1)
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
