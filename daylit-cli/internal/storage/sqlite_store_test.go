package storage

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

// setupMinimalTestStore creates a SQLite store without running migrations
func setupMinimalTestStore(t *testing.T) (*SQLiteStore, func()) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	store := NewSQLiteStore(dbPath)
	
	// Open the database manually without running Init (which runs migrations)
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	store.db = db

	cleanup := func() {
		store.Close()
		os.RemoveAll(tempDir)
	}

	return store, cleanup
}

func TestTableExists(t *testing.T) {
	t.Run("table exists", func(t *testing.T) {
		store, cleanup := setupMinimalTestStore(t)
		defer cleanup()

		// Create a test table
		_, err := store.db.Exec("CREATE TABLE test_table (id INTEGER PRIMARY KEY, name TEXT)")
		if err != nil {
			t.Fatalf("failed to create test table: %v", err)
		}

		exists, err := store.tableExists("test_table")
		if err != nil {
			t.Errorf("tableExists() returned unexpected error: %v", err)
		}
		if !exists {
			t.Error("tableExists() = false, want true for existing table")
		}
	})

	t.Run("table does not exist", func(t *testing.T) {
		store, cleanup := setupMinimalTestStore(t)
		defer cleanup()

		exists, err := store.tableExists("nonexistent_table")
		if err != nil {
			t.Errorf("tableExists() returned unexpected error: %v", err)
		}
		if exists {
			t.Error("tableExists() = true, want false for nonexistent table")
		}
	})

	t.Run("multiple tables - check specific one", func(t *testing.T) {
		store, cleanup := setupMinimalTestStore(t)
		defer cleanup()

		// Create multiple tables
		tables := []string{"table1", "table2", "table3"}
		for _, tableName := range tables {
			_, err := store.db.Exec("CREATE TABLE " + tableName + " (id INTEGER PRIMARY KEY)")
			if err != nil {
				t.Fatalf("failed to create table %s: %v", tableName, err)
			}
		}

		// Check each table exists
		for _, tableName := range tables {
			exists, err := store.tableExists(tableName)
			if err != nil {
				t.Errorf("tableExists(%s) returned unexpected error: %v", tableName, err)
			}
			if !exists {
				t.Errorf("tableExists(%s) = false, want true", tableName)
			}
		}

		// Check nonexistent table
		exists, err := store.tableExists("table4")
		if err != nil {
			t.Errorf("tableExists() returned unexpected error: %v", err)
		}
		if exists {
			t.Error("tableExists(table4) = true, want false for nonexistent table")
		}
	})

	t.Run("case sensitivity", func(t *testing.T) {
		store, cleanup := setupMinimalTestStore(t)
		defer cleanup()

		// Create a table with lowercase name
		_, err := store.db.Exec("CREATE TABLE lowercase_table (id INTEGER PRIMARY KEY)")
		if err != nil {
			t.Fatalf("failed to create test table: %v", err)
		}

		// SQLite table names are case-sensitive in queries
		exists, err := store.tableExists("lowercase_table")
		if err != nil {
			t.Errorf("tableExists() returned unexpected error: %v", err)
		}
		if !exists {
			t.Error("tableExists('lowercase_table') = false, want true")
		}

		// Check with different case - should not exist
		exists, err = store.tableExists("LOWERCASE_TABLE")
		if err != nil {
			t.Errorf("tableExists() returned unexpected error: %v", err)
		}
		if exists {
			t.Error("tableExists('LOWERCASE_TABLE') = true, want false (case-sensitive)")
		}
	})

	t.Run("empty table name", func(t *testing.T) {
		store, cleanup := setupMinimalTestStore(t)
		defer cleanup()

		exists, err := store.tableExists("")
		if err != nil {
			t.Errorf("tableExists('') returned unexpected error: %v", err)
		}
		if exists {
			t.Error("tableExists('') = true, want false for empty table name")
		}
	})

	t.Run("table name with special characters", func(t *testing.T) {
		store, cleanup := setupMinimalTestStore(t)
		defer cleanup()

		// Try to check for a table with SQL injection attempt
		// This should safely return false, not cause an error
		exists, err := store.tableExists("'; DROP TABLE test; --")
		if err != nil {
			t.Errorf("tableExists() with special characters returned unexpected error: %v", err)
		}
		if exists {
			t.Error("tableExists() with SQL injection attempt should return false")
		}
	})

	t.Run("closed database", func(t *testing.T) {
		store, cleanup := setupMinimalTestStore(t)
		
		// Close the database
		store.Close()
		cleanup()

		// Attempt to check table existence on closed database
		_, err := store.tableExists("some_table")
		if err == nil {
			t.Error("tableExists() on closed database should return an error")
		}
	})

	t.Run("habits table in initialized store", func(t *testing.T) {
		// Use the full setup that runs migrations
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "test.db")
		store := NewSQLiteStore(dbPath)
		
		if err := store.Init(); err != nil {
			t.Fatalf("failed to initialize store: %v", err)
		}
		defer func() {
			store.Close()
			os.RemoveAll(tempDir)
		}()

		// After initialization with migrations, habits table should exist
		exists, err := store.tableExists("habits")
		if err != nil {
			t.Errorf("tableExists('habits') returned unexpected error: %v", err)
		}
		if !exists {
			t.Error("tableExists('habits') = false, want true after running migrations")
		}
	})
}
