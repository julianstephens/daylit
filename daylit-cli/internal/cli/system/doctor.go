package system

import (
	"fmt"
	"time"

	"github.com/julianstephens/daylit/daylit-cli/internal/backup"
	"github.com/julianstephens/daylit/daylit-cli/internal/cli"
	"github.com/julianstephens/daylit/daylit-cli/internal/migration"
	"github.com/julianstephens/daylit/daylit-cli/internal/storage"
)

type DoctorCmd struct{}

func (cmd *DoctorCmd) Run(ctx *cli.Context) error {
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

	// Check 2: Schema version valid (only if DB is reachable)
	if dbReachable {
		if err := checkSchemaVersion(ctx); err != nil {
			fmt.Printf("❌ Schema version: FAIL\n")
			fmt.Printf("   Error: %v\n", err)
			hasError = true
		} else {
			fmt.Printf("✓ Schema version: OK\n")
		}
	} else {
		fmt.Printf("⊘ Schema version: SKIPPED (database not reachable)\n")
	}

	// Check 3: Migrations complete (only if DB is reachable)
	if dbReachable {
		if err := checkMigrationsComplete(ctx); err != nil {
			fmt.Printf("❌ Migrations complete: FAIL\n")
			fmt.Printf("   Error: %v\n", err)
			hasError = true
		} else {
			fmt.Printf("✓ Migrations complete: OK\n")
		}
	} else {
		fmt.Printf("⊘ Migrations complete: SKIPPED (database not reachable)\n")
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

	// Check 7: Habit integrity (only if DB is reachable)
	if dbReachable {
		if err := checkHabitsIntegrity(ctx); err != nil {
			fmt.Printf("❌ Habit integrity: FAIL\n")
			fmt.Printf("   Error: %v\n", err)
			hasError = true
		} else {
			fmt.Printf("✓ Habit integrity: OK\n")
		}
	} else {
		fmt.Printf("⊘ Habit integrity: SKIPPED (database not reachable)\n")
	}

	// Check 8: Habit entries duplicates (only if DB is reachable)
	if dbReachable {
		if err := checkHabitEntriesDuplicates(ctx); err != nil {
			fmt.Printf("❌ Habit entries duplicates: FAIL\n")
			fmt.Printf("   Error: %v\n", err)
			hasError = true
		} else {
			fmt.Printf("✓ Habit entries duplicates: OK\n")
		}
	} else {
		fmt.Printf("⊘ Habit entries duplicates: SKIPPED (database not reachable)\n")
	}

	// Check 9: OT settings (only if DB is reachable)
	if dbReachable {
		if err := checkOTSettings(ctx); err != nil {
			fmt.Printf("❌ OT settings: FAIL\n")
			fmt.Printf("   Error: %v\n", err)
			hasError = true
		} else {
			fmt.Printf("✓ OT settings: OK\n")
		}
	} else {
		fmt.Printf("⊘ OT settings: SKIPPED (database not reachable)\n")
	}

	// Check 10: Date formats (only if DB is reachable)
	if dbReachable {
		if err := checkOTEntriesDates(ctx); err != nil {
			fmt.Printf("❌ Date formats: FAIL\n")
			fmt.Printf("   Error: %v\n", err)
			hasError = true
		} else {
			fmt.Printf("✓ Date formats: OK\n")
		}
	} else {
		fmt.Printf("⊘ Date formats: SKIPPED (database not reachable)\n")
	}

	// Check 11: Timestamp integrity (only if DB is reachable)
	if dbReachable {
		if err := checkTimestampIntegrity(ctx); err != nil {
			fmt.Printf("❌ Timestamp integrity: FAIL\n")
			fmt.Printf("   Error: %v\n", err)
			hasError = true
		} else {
			fmt.Printf("✓ Timestamp integrity: OK\n")
		}
	} else {
		fmt.Printf("⊘ Timestamp integrity: SKIPPED (database not reachable)\n")
	}

	fmt.Println()
	if hasError {
		fmt.Println("Diagnostics completed with errors.")
		return fmt.Errorf("one or more health checks failed")
	}

	fmt.Println("All diagnostics passed!")
	return nil
}

func checkDBReachable(ctx *cli.Context) error {
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

func checkSchemaVersion(ctx *cli.Context) error {
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
	runner := migration.NewRunner(db, migrationsPath, "sqlite")

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

func checkMigrationsComplete(ctx *cli.Context) error {
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
	runner := migration.NewRunner(db, migrationsPath, "sqlite")

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

func checkBackupsPresent(ctx *cli.Context) error {
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

func checkValidation(ctx *cli.Context) error {
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

	return nil
}

func checkHabitsIntegrity(ctx *cli.Context) error {
	sqliteStore, ok := ctx.Store.(*storage.SQLiteStore)
	if !ok {
		return nil // Not SQLite, skip
	}

	db := sqliteStore.GetDB()
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}

	// Check for orphaned habit entries (entries referencing non-existent habits)
	var orphanedCount int
	err := db.QueryRow(`
		SELECT COUNT(*)
		FROM habit_entries he
		LEFT JOIN habits h ON he.habit_id = h.id
		WHERE h.id IS NULL AND he.deleted_at IS NULL
	`).Scan(&orphanedCount)
	if err != nil {
		return fmt.Errorf("failed to check orphaned habit entries: %w", err)
	}
	if orphanedCount > 0 {
		return fmt.Errorf("found %d orphaned habit entries (referencing non-existent habits)", orphanedCount)
	}

	return nil
}

func checkHabitEntriesDuplicates(ctx *cli.Context) error {
	sqliteStore, ok := ctx.Store.(*storage.SQLiteStore)
	if !ok {
		return nil // Not SQLite, skip
	}

	db := sqliteStore.GetDB()
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}

	// Check for duplicate habit entries (multiple entries for same habit + day)
	var duplicateCount int
	err := db.QueryRow(`
		SELECT COUNT(*)
		FROM (
			SELECT habit_id, day, COUNT(*) as cnt
			FROM habit_entries
			WHERE deleted_at IS NULL
			GROUP BY habit_id, day
			HAVING cnt > 1
		)
	`).Scan(&duplicateCount)
	if err != nil {
		return fmt.Errorf("failed to check duplicate habit entries: %w", err)
	}
	if duplicateCount > 0 {
		return fmt.Errorf("found %d habit+day combinations with duplicate entries", duplicateCount)
	}

	return nil
}

func checkOTSettings(ctx *cli.Context) error {
	sqliteStore, ok := ctx.Store.(*storage.SQLiteStore)
	if !ok {
		return nil // Not SQLite, skip
	}

	db := sqliteStore.GetDB()
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}

	// Check if ot_settings row exists
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM ot_settings WHERE id = 1`).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check ot_settings: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("ot_settings row missing (run 'daylit ot init')")
	}

	return nil
}

func checkOTEntriesDates(ctx *cli.Context) error {
	sqliteStore, ok := ctx.Store.(*storage.SQLiteStore)
	if !ok {
		return nil // Not SQLite, skip
	}

	db := sqliteStore.GetDB()
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}

	// Check for invalid date formats
	var invalidCount int
	err := db.QueryRow(`
		SELECT COUNT(*)
		FROM ot_entries
		WHERE day NOT GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'
	`).Scan(&invalidCount)
	if err != nil {
		return fmt.Errorf("failed to check OT entry dates: %w", err)
	}
	if invalidCount > 0 {
		return fmt.Errorf("found %d OT entries with invalid date format", invalidCount)
	}

	// Check for invalid date formats in habit_entries
	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM habit_entries
		WHERE day NOT GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'
	`).Scan(&invalidCount)
	if err != nil {
		return fmt.Errorf("failed to check habit entry dates: %w", err)
	}
	if invalidCount > 0 {
		return fmt.Errorf("found %d habit entries with invalid date format", invalidCount)
	}

	return nil
}

func checkTimestampIntegrity(ctx *cli.Context) error {
	sqliteStore, ok := ctx.Store.(*storage.SQLiteStore)
	if !ok {
		return nil // Not SQLite, skip
	}

	db := sqliteStore.GetDB()
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}

	// Check habit entries
	var corruptedCount int
	err := db.QueryRow(`
		SELECT COUNT(*)
		FROM habit_entries
		WHERE created_at = '' OR updated_at = ''
	`).Scan(&corruptedCount)
	if err != nil {
		return fmt.Errorf("failed to check habit entry timestamps: %w", err)
	}
	if corruptedCount > 0 {
		return fmt.Errorf("found %d habit entries with corrupted timestamps", corruptedCount)
	}

	// Check OT entries
	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM ot_entries
		WHERE created_at = '' OR updated_at = ''
	`).Scan(&corruptedCount)
	if err != nil {
		return fmt.Errorf("failed to check OT entry timestamps: %w", err)
	}
	if corruptedCount > 0 {
		return fmt.Errorf("found %d OT entries with corrupted timestamps", corruptedCount)
	}

	// Check habits
	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM habits
		WHERE created_at = ''
	`).Scan(&corruptedCount)
	if err != nil {
		return fmt.Errorf("failed to check habit timestamps: %w", err)
	}
	if corruptedCount > 0 {
		return fmt.Errorf("found %d habits with corrupted timestamps", corruptedCount)
	}

	return nil
}
