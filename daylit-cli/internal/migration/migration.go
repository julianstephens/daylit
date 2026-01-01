package migration

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Migration represents a single database migration
type Migration struct {
	Version int
	Name    string
	SQL     string
}

// Runner manages database schema migrations
type Runner struct {
	db *sql.DB
	fs fs.FS
}

// NewRunner creates a new migration runner
func NewRunner(db *sql.DB, migrationFS fs.FS) *Runner {
	return &Runner{
		db: db,
		fs: migrationFS,
	}
}

// EnsureSchemaVersionTable creates the schema_version table if it doesn't exist
func (r *Runner) EnsureSchemaVersionTable() error {
	_, err := r.db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_version (
			version INTEGER PRIMARY KEY
		)
	`)
	return err
}

// GetCurrentVersion returns the current schema version from the database
// Returns 0 if no version is set (fresh database)
func (r *Runner) GetCurrentVersion() (int, error) {
	if err := r.EnsureSchemaVersionTable(); err != nil {
		return 0, fmt.Errorf("failed to ensure schema_version table: %w", err)
	}

	var version int
	err := r.db.QueryRow("SELECT version FROM schema_version").Scan(&version)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// No version set yet, this is a fresh database
			return 0, nil
		}
		return 0, fmt.Errorf("failed to get current version: %w", err)
	}
	return version, nil
}

// SetVersion sets the current schema version in the database
func (r *Runner) SetVersion(version int) error {
	if err := r.EnsureSchemaVersionTable(); err != nil {
		return fmt.Errorf("failed to ensure schema_version table: %w", err)
	}

	// Delete any existing version and insert the new one
	_, err := r.db.Exec("DELETE FROM schema_version")
	if err != nil {
		return fmt.Errorf("failed to clear version: %w", err)
	}

	_, err = r.db.Exec("INSERT INTO schema_version (version) VALUES (?)", version)
	if err != nil {
		return fmt.Errorf("failed to set version: %w", err)
	}
	return nil
}

// ReadMigrationFiles reads and parses migration files from the migrations directory
// Returns migrations sorted by version number
func (r *Runner) ReadMigrationFiles() ([]Migration, error) {
	files, err := fs.ReadDir(r.fs, ".")
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var migrations []Migration
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".sql") {
			continue
		}

		// Parse version from filename (e.g., "001_init.sql" -> 1)
		parts := strings.SplitN(file.Name(), "_", 2)
		if len(parts) < 2 {
			return nil, fmt.Errorf("invalid migration filename format: %s (expected NNN_name.sql)", file.Name())
		}

		version, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil, fmt.Errorf("invalid version number in filename %s: %w", file.Name(), err)
		}
		if version < 1 {
			return nil, fmt.Errorf("invalid version number in filename %s: version must be at least 1", file.Name())
		}

		// Read migration SQL
		content, err := fs.ReadFile(r.fs, file.Name())
		if err != nil {
			return nil, fmt.Errorf("failed to read migration file %s: %w", file.Name(), err)
		}

		migrations = append(migrations, Migration{
			Version: version,
			Name:    strings.TrimSuffix(parts[1], ".sql"),
			SQL:     string(content),
		})
	}

	// Sort by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	// Validate that there are no duplicate versions
	for i := 1; i < len(migrations); i++ {
		if migrations[i].Version == migrations[i-1].Version {
			return nil, fmt.Errorf("duplicate migration version %d", migrations[i].Version)
		}
	}

	return migrations, nil
}

// GetLatestVersion returns the highest migration version available
func (r *Runner) GetLatestVersion() (int, error) {
	migrations, err := r.ReadMigrationFiles()
	if err != nil {
		return 0, err
	}

	if len(migrations) == 0 {
		return 0, nil
	}

	return migrations[len(migrations)-1].Version, nil
}

// ApplyMigrations applies all pending migrations up to the latest version
// Returns the number of migrations applied
func (r *Runner) ApplyMigrations(logFn func(string)) (int, error) {
	if logFn == nil {
		logFn = func(s string) {} // no-op logger
	}

	currentVersion, err := r.GetCurrentVersion()
	if err != nil {
		return 0, fmt.Errorf("failed to get current version: %w", err)
	}

	migrations, err := r.ReadMigrationFiles()
	if err != nil {
		return 0, fmt.Errorf("failed to read migrations: %w", err)
	}

	if len(migrations) == 0 {
		logFn("No migration files found")
		return 0, nil
	}

	latestVersion := migrations[len(migrations)-1].Version

	// Check if database is newer than supported version
	if currentVersion > latestVersion {
		return 0, fmt.Errorf("database schema version (%d) is newer than supported version (%d) - please upgrade the application", currentVersion, latestVersion)
	}

	// Filter migrations that need to be applied
	var pendingMigrations []Migration
	for _, m := range migrations {
		if m.Version > currentVersion {
			pendingMigrations = append(pendingMigrations, m)
		}
	}

	if len(pendingMigrations) == 0 {
		logFn(fmt.Sprintf("Database schema is up to date (version %d)", currentVersion))
		return 0, nil
	}

	logFn(fmt.Sprintf("Current schema version: %d", currentVersion))
	logFn(fmt.Sprintf("Target schema version: %d", latestVersion))
	logFn(fmt.Sprintf("Applying %d migration(s)...", len(pendingMigrations)))

	startTime := time.Now()
	appliedCount := 0

	for _, migration := range pendingMigrations {
		logFn(fmt.Sprintf("  Applying migration %d: %s", migration.Version, migration.Name))

		// Execute migration in a transaction
		tx, err := r.db.Begin()
		if err != nil {
			return appliedCount, fmt.Errorf("failed to begin transaction for migration %d: %w", migration.Version, err)
		}

		// Execute the migration SQL
		if _, err := tx.Exec(migration.SQL); err != nil {
			_ = tx.Rollback()
			return appliedCount, fmt.Errorf("failed to apply migration %d (%s): %w", migration.Version, migration.Name, err)
		}

		// Update the schema version within the same transaction
		if _, err := tx.Exec("DELETE FROM schema_version"); err != nil {
			_ = tx.Rollback()
			return appliedCount, fmt.Errorf("failed to clear version in migration %d: %w", migration.Version, err)
		}

		if _, err := tx.Exec("INSERT INTO schema_version (version) VALUES (?)", migration.Version); err != nil {
			_ = tx.Rollback()
			return appliedCount, fmt.Errorf("failed to set version in migration %d: %w", migration.Version, err)
		}

		// Commit the transaction
		if err := tx.Commit(); err != nil {
			return appliedCount, fmt.Errorf("failed to commit migration %d: %w", migration.Version, err)
		}

		appliedCount++
		logFn(fmt.Sprintf("  ✓ Migration %d applied successfully", migration.Version))
	}

	duration := time.Since(startTime)
	logFn(fmt.Sprintf("Applied %d migration(s) in %v", appliedCount, duration))

	return appliedCount, nil
}

// ValidateVersion checks if the database version is compatible with the application
func (r *Runner) ValidateVersion() error {
	currentVersion, err := r.GetCurrentVersion()
	if err != nil {
		return err
	}

	latestVersion, err := r.GetLatestVersion()
	if err != nil {
		return err
	}

	if currentVersion > latestVersion {
		return fmt.Errorf("database schema version (%d) is newer than supported version (%d) - please upgrade the application", currentVersion, latestVersion)
	}

	return nil
}
