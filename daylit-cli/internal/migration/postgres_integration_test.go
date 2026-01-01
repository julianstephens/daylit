package migration

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/lib/pq"
)

// setupPostgresTestDB creates a test PostgreSQL database connection
// Set POSTGRES_TEST_URL environment variable to run this test
// Example: POSTGRES_TEST_URL="postgres://user:password@localhost:5432/testdb?sslmode=disable"
func setupPostgresTestDB(t *testing.T) (*sql.DB, func()) {
	connStr := os.Getenv("POSTGRES_TEST_URL")
	if connStr == "" {
		t.Skip("POSTGRES_TEST_URL not set, skipping PostgreSQL integration test")
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatalf("failed to open postgres database: %v", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		t.Fatalf("failed to ping postgres database: %v", err)
	}

	cleanup := func() {
		// Clean up test tables
		db.Exec("DROP TABLE IF EXISTS schema_version")
		db.Exec("DROP TABLE IF EXISTS test_users")
		db.Exec("DROP TABLE IF EXISTS test_posts")
		db.Close()
	}

	return db, cleanup
}

// TestPostgresSetVersion verifies SetVersion works with PostgreSQL $1 placeholders
func TestPostgresSetVersion(t *testing.T) {
	db, cleanup := setupPostgresTestDB(t)
	defer cleanup()

	migrationsPath := setupTestMigrations(t, map[string]string{
		"001_init.sql": "CREATE TABLE test_users (id SERIAL PRIMARY KEY);",
	})

	runner, err := NewRunner(db, migrationsPath, DriverPostgres)
	if err != nil {
		t.Fatalf("failed to create migration runner: %v", err)
	}

	// Ensure schema_version table exists
	if err := runner.EnsureSchemaVersionTable(); err != nil {
		t.Fatalf("failed to ensure schema_version table: %v", err)
	}

	// Set version
	if err := runner.SetVersion(1); err != nil {
		t.Fatalf("SetVersion failed: %v", err)
	}

	// Get version
	version, err := runner.GetCurrentVersion()
	if err != nil {
		t.Fatalf("GetCurrentVersion failed: %v", err)
	}
	if version != 1 {
		t.Errorf("expected version 1, got %d", version)
	}

	// Update version
	if err := runner.SetVersion(2); err != nil {
		t.Fatalf("SetVersion(2) failed: %v", err)
	}

	version, err = runner.GetCurrentVersion()
	if err != nil {
		t.Fatalf("GetCurrentVersion failed: %v", err)
	}
	if version != 2 {
		t.Errorf("expected version 2, got %d", version)
	}
}

// TestPostgresApplyMigrations verifies ApplyMigrations works with PostgreSQL $1 placeholders
func TestPostgresApplyMigrations(t *testing.T) {
	db, cleanup := setupPostgresTestDB(t)
	defer cleanup()

	migrationsPath := setupTestMigrations(t, map[string]string{
		"001_init.sql": `
			CREATE TABLE test_users (
				id SERIAL PRIMARY KEY,
				name TEXT NOT NULL
			);
		`,
		"002_posts.sql": `
			CREATE TABLE test_posts (
				id SERIAL PRIMARY KEY,
				user_id INTEGER NOT NULL REFERENCES test_users(id),
				title TEXT NOT NULL
			);
		`,
	})

	runner, err := NewRunner(db, migrationsPath, DriverPostgres)
	if err != nil {
		t.Fatalf("failed to create migration runner: %v", err)
	}

	// Initially, version should be 0
	version, err := runner.GetCurrentVersion()
	if err != nil {
		t.Fatalf("GetCurrentVersion failed: %v", err)
	}
	if version != 0 {
		t.Errorf("expected initial version 0, got %d", version)
	}

	// Apply migrations
	count, err := runner.ApplyMigrations(nil)
	if err != nil {
		t.Fatalf("ApplyMigrations failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 migrations applied, got %d", count)
	}

	// Verify final version
	version, err = runner.GetCurrentVersion()
	if err != nil {
		t.Fatalf("GetCurrentVersion failed: %v", err)
	}
	if version != 2 {
		t.Errorf("expected version 2, got %d", version)
	}

	// Verify tables were created
	var exists bool
	err = db.QueryRow("SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'test_users')").Scan(&exists)
	if err != nil {
		t.Fatalf("failed to check test_users table: %v", err)
	}
	if !exists {
		t.Error("test_users table was not created")
	}

	err = db.QueryRow("SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'test_posts')").Scan(&exists)
	if err != nil {
		t.Fatalf("failed to check test_posts table: %v", err)
	}
	if !exists {
		t.Error("test_posts table was not created")
	}

	// Apply migrations again (should be no-op)
	count, err = runner.ApplyMigrations(nil)
	if err != nil {
		t.Fatalf("ApplyMigrations (2nd) failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 migrations on second run, got %d", count)
	}
}

// TestPostgresApplyMigrationsIncremental verifies incremental migrations work with PostgreSQL
func TestPostgresApplyMigrationsIncremental(t *testing.T) {
	db, cleanup := setupPostgresTestDB(t)
	defer cleanup()

	// First migration
	migrationsPath := setupTestMigrations(t, map[string]string{
		"001_init.sql": `
			CREATE TABLE test_users (
				id SERIAL PRIMARY KEY,
				name TEXT NOT NULL
			);
		`,
	})

	runner, err := NewRunner(db, migrationsPath, DriverPostgres)
	if err != nil {
		t.Fatalf("failed to create migration runner: %v", err)
	}

	// Apply first migration
	count, err := runner.ApplyMigrations(nil)
	if err != nil {
		t.Fatalf("ApplyMigrations (1st) failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 migration applied, got %d", count)
	}

	// Verify version
	version, err := runner.GetCurrentVersion()
	if err != nil {
		t.Fatalf("GetCurrentVersion failed: %v", err)
	}
	if version != 1 {
		t.Errorf("expected version 1, got %d", version)
	}

	// Add a second migration file
	migrationsPath = setupTestMigrations(t, map[string]string{
		"001_init.sql": `
			CREATE TABLE test_users (
				id SERIAL PRIMARY KEY,
				name TEXT NOT NULL
			);
		`,
		"002_posts.sql": `
			CREATE TABLE test_posts (
				id SERIAL PRIMARY KEY,
				title TEXT NOT NULL
			);
		`,
	})

	runner, err = NewRunner(db, migrationsPath, DriverPostgres)
	if err != nil {
		t.Fatalf("failed to create migration runner (2nd): %v", err)
	}

	// Apply second migration
	count, err = runner.ApplyMigrations(nil)
	if err != nil {
		t.Fatalf("ApplyMigrations (2nd) failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 migration applied in second run, got %d", count)
	}

	// Verify final version
	version, err = runner.GetCurrentVersion()
	if err != nil {
		t.Fatalf("GetCurrentVersion (2nd) failed: %v", err)
	}
	if version != 2 {
		t.Errorf("expected version 2, got %d", version)
	}
}

// TestPostgresMigrationRollbackOnError verifies transaction rollback works with PostgreSQL
func TestPostgresMigrationRollbackOnError(t *testing.T) {
	db, cleanup := setupPostgresTestDB(t)
	defer cleanup()

	migrationsPath := setupTestMigrations(t, map[string]string{
		"001_bad.sql": `
			CREATE TABLE test_users (id SERIAL PRIMARY KEY);
			THIS IS INVALID SQL;
		`,
	})

	runner, err := NewRunner(db, migrationsPath, DriverPostgres)
	if err != nil {
		t.Fatalf("failed to create migration runner: %v", err)
	}

	// Apply migrations (should fail)
	_, err = runner.ApplyMigrations(nil)
	if err == nil {
		t.Fatal("ApplyMigrations should have failed with invalid SQL")
	}

	// Verify version is still 0 (transaction rolled back)
	version, err := runner.GetCurrentVersion()
	if err != nil {
		t.Fatalf("GetCurrentVersion failed: %v", err)
	}
	if version != 0 {
		t.Errorf("expected version 0 after failed migration, got %d", version)
	}

	// Verify table was not created (transaction rolled back)
	var exists bool
	err = db.QueryRow("SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'test_users')").Scan(&exists)
	if err != nil {
		t.Fatalf("failed to check test_users table: %v", err)
	}
	if exists {
		t.Error("test_users table should not exist after rollback")
	}
}
