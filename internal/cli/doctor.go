package cli

import (
	"fmt"
	"time"

	"github.com/julianstephens/daylit/internal/backup"
	"github.com/julianstephens/daylit/internal/migration"
	"github.com/julianstephens/daylit/internal/storage"
)

type DoctorCmd struct{}

func (cmd *DoctorCmd) Run(ctx *Context) error {
	fmt.Println("Running diagnostics...")
	fmt.Println()

	hasError := false
	dbReachable := false

	// Check 1: DB reachable
	if err := checkDBReachable(ctx); err != nil {
		fmt.Printf("❌ Database reachable: FAIL\n")
		fmt.Printf("   Error: %v\n", err)
		hasError = true
	} else {
		fmt.Printf("✓ Database reachable: OK\n")
		dbReachable = true
	}

	// Check 2: Schema version valid
	if err := checkSchemaVersion(ctx); err != nil {
		fmt.Printf("❌ Schema version: FAIL\n")
		fmt.Printf("   Error: %v\n", err)
		hasError = true
	} else {
		fmt.Printf("✓ Schema version: OK\n")
	}

	// Check 3: Migrations complete
	if err := checkMigrationsComplete(ctx); err != nil {
		fmt.Printf("❌ Migrations complete: FAIL\n")
		fmt.Printf("   Error: %v\n", err)
		hasError = true
	} else {
		fmt.Printf("✓ Migrations complete: OK\n")
	}

	// Check 4: Backups present (warning only)
	if err := checkBackupsPresent(ctx); err != nil {
		fmt.Printf("⚠ Backups present: WARNING\n")
		fmt.Printf("   %v\n", err)
	} else {
		fmt.Printf("✓ Backups present: OK\n")
	}

	// Check 5: Validation passes (only if DB is reachable)
	if dbReachable {
		if err := checkValidation(ctx); err != nil {
			fmt.Printf("❌ Data validation: FAIL\n")
			fmt.Printf("   Error: %v\n", err)
			hasError = true
		} else {
			fmt.Printf("✓ Data validation: OK\n")
		}
	} else {
		fmt.Printf("⊘ Data validation: SKIPPED (database not reachable)\n")
	}

	// Check 6: Clock/timezone sanity
	if err := checkClockTimezone(); err != nil {
		fmt.Printf("❌ Clock/timezone: FAIL\n")
		fmt.Printf("   Error: %v\n", err)
		hasError = true
	} else {
		fmt.Printf("✓ Clock/timezone: OK\n")
	}

	fmt.Println()
	if hasError {
		fmt.Println("Diagnostics completed with errors.")
		return fmt.Errorf("one or more health checks failed")
	}

	fmt.Println("All diagnostics passed!")
	return nil
}

func checkDBReachable(ctx *Context) error {
	// Try to load the database
	if err := ctx.Store.Load(); err != nil {
		return fmt.Errorf("failed to load database: %w", err)
	}

	// For SQLite, also try a simple query
	if sqliteStore, ok := ctx.Store.(*storage.SQLiteStore); ok {
		db := sqliteStore.GetDB()
		if db == nil {
			return fmt.Errorf("database connection is nil")
		}
		var result int
		if err := db.QueryRow("SELECT 1").Scan(&result); err != nil {
			return fmt.Errorf("failed to query database: %w", err)
		}
	}

	return nil
}

func checkSchemaVersion(ctx *Context) error {
	sqliteStore, ok := ctx.Store.(*storage.SQLiteStore)
	if !ok {
		// JSON store doesn't have schema version
		return nil
	}

	db := sqliteStore.GetDB()
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}

	migrationsPath := sqliteStore.GetMigrationsPath()
	runner := migration.NewRunner(db, migrationsPath)

	currentVersion, err := runner.GetCurrentVersion()
	if err != nil {
		return fmt.Errorf("failed to get current schema version: %w", err)
	}

	latestVersion, err := runner.GetLatestVersion()
	if err != nil {
		return fmt.Errorf("failed to get latest schema version: %w", err)
	}

	if currentVersion > latestVersion {
		return fmt.Errorf("database schema version (%d) is newer than supported version (%d)", currentVersion, latestVersion)
	}

	return nil
}

func checkMigrationsComplete(ctx *Context) error {
	sqliteStore, ok := ctx.Store.(*storage.SQLiteStore)
	if !ok {
		// JSON store doesn't have migrations
		return nil
	}

	db := sqliteStore.GetDB()
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}

	migrationsPath := sqliteStore.GetMigrationsPath()
	runner := migration.NewRunner(db, migrationsPath)

	currentVersion, err := runner.GetCurrentVersion()
	if err != nil {
		return fmt.Errorf("failed to get current schema version: %w", err)
	}

	latestVersion, err := runner.GetLatestVersion()
	if err != nil {
		return fmt.Errorf("failed to get latest schema version: %w", err)
	}

	if currentVersion < latestVersion {
		return fmt.Errorf("migrations incomplete: current version %d, latest version %d", currentVersion, latestVersion)
	}

	return nil
}

func checkBackupsPresent(ctx *Context) error {
	mgr := backup.NewManager(ctx.Store.GetConfigPath())
	backups, err := mgr.ListBackups()
	if err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}

	if len(backups) == 0 {
		return fmt.Errorf("no backups found - consider creating one with 'daylit backup create'")
	}

	return nil
}

func checkValidation(ctx *Context) error {
	// Try to get settings
	if _, err := ctx.Store.GetSettings(); err != nil {
		return fmt.Errorf("failed to get settings: %w", err)
	}

	// Try to get all tasks
	tasks, err := ctx.Store.GetAllTasks()
	if err != nil {
		return fmt.Errorf("failed to get tasks: %w", err)
	}

	// Basic validation: check for duplicate IDs
	taskIDs := make(map[string]bool)
	for _, task := range tasks {
		if taskIDs[task.ID] {
			return fmt.Errorf("duplicate task ID found: %s", task.ID)
		}
		taskIDs[task.ID] = true
	}

	return nil
}

func checkClockTimezone() error {
	// Check if system time is reasonable
	now := time.Now()

	// Check if time is in a reasonable range (after 2020 and before 2100)
	if now.Year() < 2020 || now.Year() > 2100 {
		return fmt.Errorf("system time appears incorrect: %s", now.Format(time.RFC3339))
	}

	// Check if timezone is set
	_, offset := now.Zone()
	if offset == 0 && now.Location() == time.UTC {
		// This might be intentional, so just note it
		fmt.Printf("   Note: timezone is UTC\n")
	}

	return nil
}
