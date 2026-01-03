package system

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/julianstephens/daylit/daylit-cli/internal/cli"
	"github.com/julianstephens/daylit/daylit-cli/internal/storage"
	"github.com/julianstephens/daylit/daylit-cli/internal/storage/postgres"
	"github.com/julianstephens/daylit/daylit-cli/internal/storage/sqlite"
)

type InitCmd struct {
	Force  bool   `help:"Force reset by deleting existing database before initialization."`
	Source string `help:"Source database path or connection string to migrate data from."`
}

func (c *InitCmd) Run(ctx *cli.Context) error {
	// If force flag is provided, delete existing database
	if c.Force {
		dbPath := ctx.Store.GetConfigPath()
		// Don't delete if it's the source (user error protection)
		if c.Source != "" {
			// Normalize paths to absolute for accurate comparison
			absDbPath, err := filepath.Abs(dbPath)
			if err == nil {
				dbPath = absDbPath
			}
			absSource, err := filepath.Abs(c.Source)
			if err == nil && absSource == dbPath {
				return fmt.Errorf("cannot use --force when source and destination are the same: %s", dbPath)
			}
		}
		if _, err := os.Stat(dbPath); err == nil {
			// Database exists, close it first to prevent file locking issues
			if err := ctx.Store.Close(); err != nil {
				return fmt.Errorf("failed to close existing database: %w", err)
			}
			// Then delete it
			if err := os.Remove(dbPath); err != nil {
				return fmt.Errorf("failed to delete existing database: %w", err)
			}
			fmt.Printf("Deleted existing database at: %s\n", dbPath)
		} else if !os.IsNotExist(err) {
			// Some other error occurred while checking the database; surface it to the user
			return fmt.Errorf("failed to access existing database: %w", err)
		}
	}

	// Initialize destination store
	if err := ctx.Store.Init(); err != nil {
		return err
	}
	fmt.Printf("Initialized daylit storage at: %s\n", ctx.Store.GetConfigPath())

	// If source is provided, migrate data
	if c.Source != "" {
		fmt.Printf("Migrating data from: %s\n", c.Source)
		if err := c.migrateData(ctx, c.Source); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
		fmt.Println("Migration completed successfully!")
	}

	return nil
}

func (c *InitCmd) migrateData(ctx *cli.Context, sourcePath string) error {
	// Determine source store type and instantiate it
	var sourceStore storage.Provider
	if strings.HasPrefix(sourcePath, "postgres://") || strings.HasPrefix(sourcePath, "postgresql://") {
		// Validate source connection string for embedded credentials
		if valid, err := postgres.ValidateConnString(sourcePath); !valid {
			if errors.Is(err, postgres.ErrEmbeddedCredentials) {
				return fmt.Errorf("PostgreSQL source connection string contains embedded credentials. Use environment variables or .pgpass instead")
			}
			// For other validation errors, we can return them or proceed (and likely fail later).
			return err
		}
		sourceStore = postgres.New(sourcePath)
	} else {
		// Default to SQLite for file paths
		sourceStore = sqlite.NewStore(sourcePath)
	}

	// Load the source store
	if err := sourceStore.Load(); err != nil {
		return fmt.Errorf("failed to load source database: %w", err)
	}
	defer sourceStore.Close()

	// Migrate Settings
	fmt.Println("  Migrating settings...")
	settings, err := sourceStore.GetSettings()
	if err != nil {
		return fmt.Errorf("failed to get settings from source: %w", err)
	}
	if err := ctx.Store.SaveSettings(settings); err != nil {
		return fmt.Errorf("failed to save settings to destination: %w", err)
	}

	// Migrate Tasks
	fmt.Println("  Migrating tasks...")
	tasks, err := sourceStore.GetAllTasksIncludingDeleted()
	if err != nil {
		return fmt.Errorf("failed to get tasks from source: %w", err)
	}
	for _, task := range tasks {
		if err := ctx.Store.AddTask(task); err != nil {
			return fmt.Errorf("failed to add task %s: %w", task.ID, err)
		}
	}
	fmt.Printf("    Migrated %d tasks\n", len(tasks))

	// Migrate Plans
	fmt.Println("  Migrating plans...")
	plans, err := sourceStore.GetAllPlans()
	if err != nil {
		return fmt.Errorf("failed to get plans from source: %w", err)
	}
	for _, plan := range plans {
		if err := ctx.Store.SavePlan(plan); err != nil {
			return fmt.Errorf("failed to save plan for date %s revision %d: %w", plan.Date, plan.Revision, err)
		}
	}
	fmt.Printf("    Migrated %d plans\n", len(plans))

	// Migrate Habits
	fmt.Println("  Migrating habits...")
	habits, err := sourceStore.GetAllHabits(true, true)
	if err != nil {
		return fmt.Errorf("failed to get habits from source: %w", err)
	}
	for _, habit := range habits {
		if err := ctx.Store.AddHabit(habit); err != nil {
			return fmt.Errorf("failed to add habit %s: %w", habit.ID, err)
		}
	}
	fmt.Printf("    Migrated %d habits\n", len(habits))

	// Migrate Habit Entries
	fmt.Println("  Migrating habit entries...")
	habitEntries, err := sourceStore.GetAllHabitEntries()
	if err != nil {
		return fmt.Errorf("failed to get habit entries from source: %w", err)
	}
	for _, entry := range habitEntries {
		if err := ctx.Store.AddHabitEntry(entry); err != nil {
			return fmt.Errorf("failed to add habit entry %s: %w", entry.ID, err)
		}
	}
	fmt.Printf("    Migrated %d habit entries\n", len(habitEntries))

	// Migrate OT Settings
	fmt.Println("  Migrating OT settings...")
	otSettings, err := sourceStore.GetOTSettings()
	if err != nil {
		return fmt.Errorf("failed to get OT settings from source: %w", err)
	}
	if err := ctx.Store.SaveOTSettings(otSettings); err != nil {
		return fmt.Errorf("failed to save OT settings to destination: %w", err)
	}

	// Migrate OT Entries
	fmt.Println("  Migrating OT entries...")
	otEntries, err := sourceStore.GetAllOTEntries()
	if err != nil {
		return fmt.Errorf("failed to get OT entries from source: %w", err)
	}
	for _, entry := range otEntries {
		if err := ctx.Store.AddOTEntry(entry); err != nil {
			return fmt.Errorf("failed to add OT entry %s: %w", entry.ID, err)
		}
	}
	fmt.Printf("    Migrated %d OT entries\n", len(otEntries))

	return nil
}
