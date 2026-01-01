package migration

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) (*sql.DB, string, func()) {
	// Create a temporary directory
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Open database
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	cleanup := func() {
		db.Close()
	}

	return db, dbPath, cleanup
}

func setupTestMigrations(t *testing.T, migrations map[string]string) string {
	// Create a temporary directory for migrations
	tempDir := t.TempDir()

	for filename, content := range migrations {
		path := filepath.Join(tempDir, filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test migration %s: %v", filename, err)
		}
	}

	return tempDir
}

func TestGetCurrentVersion(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	migrationsPath := setupTestMigrations(t, map[string]string{
		"001_test.sql": "CREATE TABLE test (id INTEGER);",
	})

	runner := NewRunner(db, migrationsPath)

	// Initially, version should be 0
	version, err := runner.GetCurrentVersion()
	if err != nil {
		t.Fatalf("GetCurrentVersion failed: %v", err)
	}
	if version != 0 {
		t.Errorf("expected version 0, got %d", version)
	}

	// Set a version
	if err := runner.SetVersion(5); err != nil {
		t.Fatalf("SetVersion failed: %v", err)
	}

	// Now version should be 5
	version, err = runner.GetCurrentVersion()
	if err != nil {
		t.Fatalf("GetCurrentVersion failed: %v", err)
	}
	if version != 5 {
		t.Errorf("expected version 5, got %d", version)
	}
}

func TestReadMigrationFiles(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	migrationsPath := setupTestMigrations(t, map[string]string{
		"001_init.sql":    "CREATE TABLE test1 (id INTEGER);",
		"002_update.sql":  "ALTER TABLE test1 ADD COLUMN name TEXT;",
		"003_another.sql": "CREATE TABLE test2 (id INTEGER);",
	})

	runner := NewRunner(db, migrationsPath)

	migrations, err := runner.ReadMigrationFiles()
	if err != nil {
		t.Fatalf("ReadMigrationFiles failed: %v", err)
	}

	if len(migrations) != 3 {
		t.Fatalf("expected 3 migrations, got %d", len(migrations))
	}

	// Check migrations are sorted by version
	if migrations[0].Version != 1 || migrations[0].Name != "init" {
		t.Errorf("migration 0: expected version 1 and name 'init', got version %d and name '%s'", migrations[0].Version, migrations[0].Name)
	}
	if migrations[1].Version != 2 || migrations[1].Name != "update" {
		t.Errorf("migration 1: expected version 2 and name 'update', got version %d and name '%s'", migrations[1].Version, migrations[1].Name)
	}
	if migrations[2].Version != 3 || migrations[2].Name != "another" {
		t.Errorf("migration 2: expected version 3 and name 'another', got version %d and name '%s'", migrations[2].Version, migrations[2].Name)
	}
}

func TestApplyMigrationsFromScratch(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	migrationsPath := setupTestMigrations(t, map[string]string{
		"001_init.sql": `
			CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT);
		`,
		"002_posts.sql": `
			CREATE TABLE posts (id INTEGER PRIMARY KEY, user_id INTEGER, content TEXT);
		`,
	})

	runner := NewRunner(db, migrationsPath)

	// Apply migrations
	count, err := runner.ApplyMigrations(nil)
	if err != nil {
		t.Fatalf("ApplyMigrations failed: %v", err)
	}

	if count != 2 {
		t.Errorf("expected 2 migrations applied, got %d", count)
	}

	// Verify version is now 2
	version, err := runner.GetCurrentVersion()
	if err != nil {
		t.Fatalf("GetCurrentVersion failed: %v", err)
	}
	if version != 2 {
		t.Errorf("expected version 2, got %d", version)
	}

	// Verify tables were created
	var count1, count2 int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='users'").Scan(&count1)
	if err != nil || count1 != 1 {
		t.Error("users table was not created")
	}

	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='posts'").Scan(&count2)
	if err != nil || count2 != 1 {
		t.Error("posts table was not created")
	}
}

func TestApplyMigrationsIncremental(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	migrationsPath := setupTestMigrations(t, map[string]string{
		"001_init.sql": `
			CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT);
		`,
	})

	runner := NewRunner(db, migrationsPath)

	// Apply first migration
	count, err := runner.ApplyMigrations(nil)
	if err != nil {
		t.Fatalf("ApplyMigrations (1st) failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 migration applied, got %d", count)
	}

	// Add a new migration file
	newMigration := `CREATE TABLE posts (id INTEGER PRIMARY KEY, user_id INTEGER);`
	if err := os.WriteFile(filepath.Join(migrationsPath, "002_posts.sql"), []byte(newMigration), 0644); err != nil {
		t.Fatalf("failed to write new migration: %v", err)
	}

	// Apply second migration
	count, err = runner.ApplyMigrations(nil)
	if err != nil {
		t.Fatalf("ApplyMigrations (2nd) failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 more migration applied, got %d", count)
	}

	// Verify version is now 2
	version, err := runner.GetCurrentVersion()
	if err != nil {
		t.Fatalf("GetCurrentVersion failed: %v", err)
	}
	if version != 2 {
		t.Errorf("expected version 2, got %d", version)
	}
}

func TestApplyMigrationsNoOp(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	migrationsPath := setupTestMigrations(t, map[string]string{
		"001_init.sql": `CREATE TABLE users (id INTEGER PRIMARY KEY);`,
	})

	runner := NewRunner(db, migrationsPath)

	// Apply migrations first time
	_, err := runner.ApplyMigrations(nil)
	if err != nil {
		t.Fatalf("ApplyMigrations (1st) failed: %v", err)
	}

	// Apply migrations again (should be no-op)
	count, err := runner.ApplyMigrations(nil)
	if err != nil {
		t.Fatalf("ApplyMigrations (2nd) failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 migrations applied on second run, got %d", count)
	}
}

func TestMigrationRollbackOnError(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	migrationsPath := setupTestMigrations(t, map[string]string{
		"001_init.sql": `
			CREATE TABLE users (id INTEGER PRIMARY KEY);
			-- Invalid SQL to cause error
			THIS IS INVALID SQL;
		`,
	})

	runner := NewRunner(db, migrationsPath)

	// Apply migrations (should fail)
	_, err := runner.ApplyMigrations(nil)
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

	// Verify table was not created (rollback successful)
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='users'").Scan(&count)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if count != 0 {
		t.Error("table should not exist after failed migration")
	}
}

func TestValidateVersionNewerDatabase(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	migrationsPath := setupTestMigrations(t, map[string]string{
		"001_init.sql": `CREATE TABLE users (id INTEGER PRIMARY KEY);`,
	})

	runner := NewRunner(db, migrationsPath)

	// Ensure schema_version table exists
	if err := runner.EnsureSchemaVersionTable(); err != nil {
		t.Fatalf("EnsureSchemaVersionTable failed: %v", err)
	}

	// Set version to 10 (higher than available migrations)
	if err := runner.SetVersion(10); err != nil {
		t.Fatalf("SetVersion failed: %v", err)
	}

	// ValidateVersion should fail
	err := runner.ValidateVersion()
	if err == nil {
		t.Fatal("ValidateVersion should have failed with newer database version")
	}

	// ApplyMigrations should also fail
	_, err = runner.ApplyMigrations(nil)
	if err == nil {
		t.Fatal("ApplyMigrations should have failed with newer database version")
	}
}

func TestGetLatestVersion(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	migrationsPath := setupTestMigrations(t, map[string]string{
		"001_init.sql":   `CREATE TABLE users (id INTEGER);`,
		"003_posts.sql":  `CREATE TABLE posts (id INTEGER);`,
		"002_update.sql": `ALTER TABLE users ADD COLUMN name TEXT;`,
	})

	runner := NewRunner(db, migrationsPath)

	latestVersion, err := runner.GetLatestVersion()
	if err != nil {
		t.Fatalf("GetLatestVersion failed: %v", err)
	}

	// Should return 3 (highest version number)
	if latestVersion != 3 {
		t.Errorf("expected latest version 3, got %d", latestVersion)
	}
}

func TestMigrationFilenameValidation(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	// Invalid filename format (no underscore)
	migrationsPath := setupTestMigrations(t, map[string]string{
		"001init.sql": `CREATE TABLE users (id INTEGER);`,
	})

	runner := NewRunner(db, migrationsPath)

	_, err := runner.ReadMigrationFiles()
	if err == nil {
		t.Error("ReadMigrationFiles should have failed with invalid filename format")
	}
}

func TestMigrationVersionValidation(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	// Test zero version number
	migrationsPath := setupTestMigrations(t, map[string]string{
		"000_init.sql": `CREATE TABLE users (id INTEGER);`,
	})

	runner := NewRunner(db, migrationsPath)

	_, err := runner.ReadMigrationFiles()
	if err == nil {
		t.Error("ReadMigrationFiles should have failed with version 0")
	}
	if err != nil && !strings.Contains(err.Error(), "version must be at least 1") {
		t.Errorf("expected version validation error, got: %v", err)
	}
}

func TestDuplicateVersionDetection(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	// Two migrations with same version number
	migrationsPath := setupTestMigrations(t, map[string]string{
		"001_init.sql":  `CREATE TABLE users (id INTEGER);`,
		"001_other.sql": `CREATE TABLE posts (id INTEGER);`,
	})

	runner := NewRunner(db, migrationsPath)

	_, err := runner.ReadMigrationFiles()
	if err == nil {
		t.Error("ReadMigrationFiles should have failed with duplicate version")
	}
	if err != nil && !strings.Contains(err.Error(), "duplicate migration version") {
		t.Errorf("expected duplicate version error, got: %v", err)
	}
}
