package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/kong"

	"github.com/julianstephens/daylit/daylit-cli/internal/cli"
	"github.com/julianstephens/daylit/daylit-cli/internal/cli/alerts"
	"github.com/julianstephens/daylit/daylit-cli/internal/cli/backups"
	"github.com/julianstephens/daylit/daylit-cli/internal/cli/habits"
	"github.com/julianstephens/daylit/daylit-cli/internal/cli/optimize"
	"github.com/julianstephens/daylit/daylit-cli/internal/cli/ot"
	"github.com/julianstephens/daylit/daylit-cli/internal/cli/plans"
	"github.com/julianstephens/daylit/daylit-cli/internal/cli/settings"
	"github.com/julianstephens/daylit/daylit-cli/internal/cli/system"
	"github.com/julianstephens/daylit/daylit-cli/internal/cli/tasks"
	"github.com/julianstephens/daylit/daylit-cli/internal/constants"
	"github.com/julianstephens/daylit/daylit-cli/internal/keyring"
	"github.com/julianstephens/daylit/daylit-cli/internal/logger"
	"github.com/julianstephens/daylit/daylit-cli/internal/scheduler"
	"github.com/julianstephens/daylit/daylit-cli/internal/storage"
	"github.com/julianstephens/daylit/daylit-cli/internal/storage/postgres"
)

type CLI struct {
	Version   kong.VersionFlag
	DebugMode bool   `help:"Enable debug logging." name:"debug"`
	Config    string `help:"Config file path or PostgreSQL connection string. When passing a PostgreSQL connection string via command-line flags, credentials must NOT be embedded. Use environment variables or a .pgpass file for command-line usage, or store a connection string with embedded credentials securely in the OS keyring via the 'keyring' commands." type:"string" default:"~/.config/daylit/daylit.db" env:"DAYLIT_CONFIG"`

	Init system.InitCmd `cmd:"" help:"Initialize daylit storage."`

	Migrate  system.MigrateCmd    `cmd:"" help:"Run database migrations."`
	Doctor   system.DoctorCmd     `cmd:"" help:"Run health checks and diagnostics."`
	Tui      system.TuiCmd        `cmd:"" help:"Launch the interactive TUI." default:"1"`
	Plan     plans.PlanCmd        `cmd:"" help:"Generate day plans."`
	Now      plans.NowCmd         `cmd:"" help:"Show current task."`
	Feedback plans.FeedbackCmd    `cmd:"" help:"Provide feedback on a slot."`
	Optimize optimize.OptimizeCmd `cmd:"" help:"Analyze feedback and suggest task optimizations."`
	Day      plans.DayCmd         `cmd:"" help:"Show plan for a day."`
	Debug    system.DebugCmd      `cmd:"" help:"Debug commands for troubleshooting."`
	Validate system.ValidateCmd   `cmd:"" help:"Validate tasks and plans for conflicts."`
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
	Habit habits.HabitCmd `cmd:"" help:"Manage habits and habit tracking."`
	OT    ot.OTCmd        `cmd:"" help:"Manage Once-Today (OT) intentions."`
	Alert struct {
		Add    alerts.AlertAddCmd    `cmd:"" help:"Add a new alert."`
		List   alerts.AlertListCmd   `cmd:"" help:"List all alerts."`
		Delete alerts.AlertDeleteCmd `cmd:"" help:"Delete an alert."`
	} `cmd:"" help:"Manage arbitrary scheduled notifications."`
	Keyring struct {
		Set    system.KeyringSetCmd    `cmd:"" help:"Store database connection string in OS keyring."`
		Get    system.KeyringGetCmd    `cmd:"" help:"Retrieve database connection string from OS keyring."`
		Delete system.KeyringDeleteCmd `cmd:"" help:"Remove database connection string from OS keyring."`
		Status system.KeyringStatusCmd `cmd:"" help:"Check OS keyring availability and status."`
	} `cmd:"" help:"Manage database credentials in OS keyring."`
	Settings settings.SettingsCmd `cmd:"" help:"Manage application settings."`
	Notify   system.NotifyCmd     `cmd:"" hidden:"" help:"Send a notification (used internally)."`

	store storage.Provider
}

func (c *CLI) AfterApply(ctx *kong.Context) error {
	// Determine config directory for logger initialization
	configPath := c.Config
	if configPath == constants.DefaultConfigPath {
		configPath = os.ExpandEnv(configPath)
	}
	configDir := filepath.Dir(configPath)

	// Initialize logger
	// For debug command, always enable debug logging
	cmdPath := ctx.Command()
	isDebugCmd := cmdPath == "debug" || strings.HasPrefix(cmdPath, "debug ")
	debugEnabled := c.DebugMode || isDebugCmd

	if err := logger.Init(logger.Config{
		Debug:     debugEnabled,
		ConfigDir: configDir,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to initialize logger: %v\n", err)
	}

	// Skip keyring lookup for keyring management commands
	if cmdPath == "keyring" || strings.HasPrefix(cmdPath, "keyring ") {
		return nil
	}

	// Initialize storage based on config format
	var store storage.Provider

	configToUse := c.Config

	// If config is still the default SQLite path and no DAYLIT_CONFIG env var is set,
	// try to retrieve from keyring
	if configToUse == constants.DefaultConfigPath && os.Getenv("DAYLIT_CONFIG") == "" {
		keyringConnStr, err := keyring.GetConnectionString()
		if err == nil {
			// Successfully retrieved from keyring
			configToUse = keyringConnStr
			logger.Debug("Using connection string from OS keyring")
		} else if !errors.Is(err, keyring.ErrNotFound) {
			// Keyring error (not just "not found")
			logger.Warn("Failed to access OS keyring, falling back to default SQLite configuration", "error", err)
		}
		// If ErrNotFound, silently fall back to SQLite
	}

	// Check for Postgres URL or DSN format
	isPostgres := strings.HasPrefix(configToUse, "postgres://") ||
		strings.HasPrefix(configToUse, "postgresql://") ||
		// Simple DSN heuristic: contains space and common keys
		(strings.Contains(configToUse, " ") &&
			(strings.Contains(configToUse, "host=") ||
				strings.Contains(configToUse, "dbname=") ||
				strings.Contains(configToUse, "user=") ||
				strings.Contains(configToUse, "sslmode=")))

	if isPostgres {
		// PostgreSQL connection string detected - validate for embedded credentials
		// We only enforce this check if the config was NOT sourced from the environment
		// or keyring (e.g. came from command line flags, which are visible in the process list).
		envConfig := os.Getenv("DAYLIT_CONFIG")
		configFromEnv := envConfig != "" && envConfig == configToUse
		configFromKeyring := configToUse != c.Config

		_, err := postgres.ValidateConnString(configToUse)
		hasPasswordError := err != nil && errors.Is(err, postgres.ErrEmbeddedCredentials)

		if !configFromEnv && !configFromKeyring && hasPasswordError {
			fmt.Fprintf(os.Stderr, "❌ Error: PostgreSQL connection strings with embedded credentials are NOT allowed via command line flags.\n")
			fmt.Fprintf(os.Stderr, "       Use one of these secure alternatives:\n")
			fmt.Fprintf(os.Stderr, "       1. Environment:   export DAYLIT_CONFIG=\"postgresql://user:your_password@host:5432/daylit\"\n")
			fmt.Fprintf(os.Stderr, "       2. .pgpass file:  Create ~/.pgpass with credentials\n")
			fmt.Fprintf(os.Stderr, "       3. OS keyring:    daylit keyring set \"postgresql://user:your_password@host:5432/daylit\"\n")
			fmt.Fprintf(os.Stderr, "\n       For more information, see docs/user-guides/POSTGRES_SETUP.md\n")
			os.Exit(1)
		} else if configFromEnv && hasPasswordError {
			// Warn user about embedded credentials in environment variable
			logger.Warn("Using embedded credentials in DAYLIT_CONFIG environment variable. Consider using a .pgpass file or OS keyring for better security.")
		}
		logger.Debug("Using PostgreSQL storage backend")
		store = postgres.New(configToUse)
	} else {
		// Default to SQLite
		logger.Debug("Using SQLite storage backend", "path", configToUse)
		store = storage.NewSQLiteStore(configToUse)
	}

	c.store = store

	// Load the store before running the command (Init command will handle its own loading)
	if !c.Init.Force && ctx.Command() != "init" {
		if err := store.Load(); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	kongCLI := CLI{}
	ctx := kong.Parse(&kongCLI,
		kong.Name(constants.AppName),
		kong.Description("Daily structure scheduler / time-blocking companion"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact:             true,
			NoExpandSubcommands: true,
		}),
		kong.Vars{"version": constants.Version},
	)

	appCtx := &cli.Context{
		Store:     kongCLI.store,
		Scheduler: scheduler.New(),
	}

	err := ctx.Run(appCtx)
	if err != nil {
		logger.Error("Command execution failed", "error", err)
		os.Exit(1)
	}
}
