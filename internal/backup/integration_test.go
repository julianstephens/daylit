package backup

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// TestIntegrationBackupRestoreWorkflow tests the complete backup and restore workflow
func TestIntegrationBackupRestoreWorkflow(t *testing.T) {
	// Create a temporary directory for test
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Step 1: Create a database with sample data
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}

	// Create tables similar to actual daylit database
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS tasks (
		id TEXT PRIMARY KEY,
		name TEXT,
		duration INTEGER
	)`)
	if err != nil {
		t.Fatalf("failed to create tasks table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS plans (
		date TEXT PRIMARY KEY
	)`)
	if err != nil {
		t.Fatalf("failed to create plans table: %v", err)
	}

	// Insert initial data
	_, err = db.Exec("INSERT INTO tasks (id, name, duration) VALUES (?, ?, ?)", "task1", "Test Task 1", 30)
	if err != nil {
		t.Fatalf("failed to insert task: %v", err)
	}
	_, err = db.Exec("INSERT INTO plans (date) VALUES (?)", "2025-01-01")
	if err != nil {
		t.Fatalf("failed to insert plan: %v", err)
	}
	db.Close()

	// Step 2: Create a backup
	mgr := NewManager(dbPath)
	backup1Path, err := mgr.CreateBackup()
	if err != nil {
		t.Fatalf("failed to create backup: %v", err)
	}

	// Step 3: Modify the database
	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	_, err = db.Exec("INSERT INTO tasks (id, name, duration) VALUES (?, ?, ?)", "task2", "Test Task 2", 60)
	if err != nil {
		t.Fatalf("failed to insert second task: %v", err)
	}
	db.Close()

	// Verify database now has 2 tasks
	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count)
	if err != nil {
		t.Fatalf("failed to count tasks: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 tasks after modification, got %d", count)
	}
	db.Close()

	// Step 4: Restore from backup
	err = mgr.RestoreBackup(backup1Path)
	if err != nil {
		t.Fatalf("failed to restore backup: %v", err)
	}

	// Step 5: Verify database is restored to original state (1 task)
	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open database after restore: %v", err)
	}
	defer db.Close()

	err = db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count)
	if err != nil {
		t.Fatalf("failed to count tasks after restore: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 task after restore, got %d", count)
	}

	// Verify the data is correct
	var id, name string
	var duration int
	err = db.QueryRow("SELECT id, name, duration FROM tasks WHERE id = ?", "task1").Scan(&id, &name, &duration)
	if err != nil {
		t.Fatalf("failed to query task after restore: %v", err)
	}
	if name != "Test Task 1" || duration != 30 {
		t.Errorf("task data mismatch after restore: got name=%s, duration=%d", name, duration)
	}

	// Verify a backup was created before restore
	backups, err := mgr.ListBackups()
	if err != nil {
		t.Fatalf("failed to list backups: %v", err)
	}
	// Should have at least 2 backups: original + pre-restore
	if len(backups) < 2 {
		t.Errorf("expected at least 2 backups after restore, got %d", len(backups))
	}
}

// TestMultipleDayBackups tests that backups work correctly when created on different days
func TestMultipleDayBackups(t *testing.T) {
	dbPath, cleanup := setupTestDB(t)
	defer cleanup()

	mgr := NewManager(dbPath)

	// Create multiple backups
	for i := 0; i < 3; i++ {
		_, err := mgr.CreateBackup()
		if err != nil {
			t.Fatalf("CreateBackup #%d failed: %v", i, err)
		}
	}

	backups, err := mgr.ListBackups()
	if err != nil {
		t.Fatalf("ListBackups failed: %v", err)
	}

	if len(backups) != 3 {
		t.Errorf("expected 3 backups, got %d", len(backups))
	}

	// Verify all backups are valid SQLite databases
	for _, backup := range backups {
		db, err := sql.Open("sqlite", backup.Path)
		if err != nil {
			t.Errorf("failed to open backup %s: %v", backup.Path, err)
			continue
		}
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM test_data").Scan(&count)
		if err != nil {
			t.Errorf("failed to query backup %s: %v", backup.Path, err)
		}
		db.Close()
	}
}

// TestBackupWithNoDatabase tests that backup fails gracefully when database doesn't exist
func TestBackupWithNoDatabase(t *testing.T) {
	tempDir := t.TempDir()
	nonExistentDB := filepath.Join(tempDir, "nonexistent.db")

	mgr := NewManager(nonExistentDB)
	_, err := mgr.CreateBackup()
	if err == nil {
		t.Error("expected error when backing up non-existent database")
	}
}

// TestRestoreWithCorruptedBackup tests restore fails for corrupted backup
func TestRestoreWithCorruptedBackup(t *testing.T) {
	dbPath, cleanup := setupTestDB(t)
	defer cleanup()

	mgr := NewManager(dbPath)

	// Create a corrupted backup file
	corruptedPath := filepath.Join(mgr.GetBackupDir(), "corrupted.db")
	err := os.MkdirAll(mgr.GetBackupDir(), 0700)
	if err != nil {
		t.Fatalf("failed to create backup dir: %v", err)
	}
	err = os.WriteFile(corruptedPath, []byte("not a valid sqlite database"), 0600)
	if err != nil {
		t.Fatalf("failed to create corrupted file: %v", err)
	}

	// Attempt to restore from corrupted backup
	err = mgr.RestoreBackup(corruptedPath)
	if err == nil {
		t.Error("expected error when restoring from corrupted backup")
	}
}

// TestBackupDirectoryCreation tests that backup directory is created if it doesn't exist
func TestBackupDirectoryCreation(t *testing.T) {
	dbPath, cleanup := setupTestDB(t)
	defer cleanup()

	mgr := NewManager(dbPath)

	// Remove backup directory if it exists
	os.RemoveAll(mgr.GetBackupDir())

	// Create a backup - should create the directory
	backupPath, err := mgr.CreateBackup()
	if err != nil {
		t.Fatalf("CreateBackup failed: %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(mgr.GetBackupDir()); os.IsNotExist(err) {
		t.Error("backup directory was not created")
	}

	// Verify backup file exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Error("backup file was not created")
	}
}
