package backup

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) (string, func()) {
	// Create a temporary directory for test database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Create a test database with sample data
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	// Create test table
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS test_data (
		id INTEGER PRIMARY KEY,
		name TEXT,
		value INTEGER
	)`)
	if err != nil {
		t.Fatalf("failed to create test table: %v", err)
	}

	// Insert sample data
	_, err = db.Exec("INSERT INTO test_data (id, name, value) VALUES (1, 'test1', 100)")
	if err != nil {
		t.Fatalf("failed to insert test data: %v", err)
	}
	_, err = db.Exec("INSERT INTO test_data (id, name, value) VALUES (2, 'test2', 200)")
	if err != nil {
		t.Fatalf("failed to insert test data: %v", err)
	}

	db.Close()

	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return dbPath, cleanup
}

func TestCreateBackup(t *testing.T) {
	dbPath, cleanup := setupTestDB(t)
	defer cleanup()

	mgr := NewManager(dbPath)
	backupPath, err := mgr.CreateBackup()
	if err != nil {
		t.Fatalf("CreateBackup failed: %v", err)
	}

	// Verify backup file exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Errorf("backup file was not created: %s", backupPath)
	}

	// Verify backup file is a valid SQLite database
	db, err := sql.Open("sqlite", backupPath)
	if err != nil {
		t.Fatalf("failed to open backup database: %v", err)
	}
	defer db.Close()

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM test_data").Scan(&count)
	if err != nil {
		t.Fatalf("failed to query backup database: %v", err)
	}

	if count != 2 {
		t.Errorf("expected 2 rows in backup, got %d", count)
	}
}

func TestBackupRotation(t *testing.T) {
	dbPath, cleanup := setupTestDB(t)
	defer cleanup()

	mgr := NewManager(dbPath)

	// Create more than MaxBackups backups
	numBackups := MaxBackups + 5
	for i := 0; i < numBackups; i++ {
		_, err := mgr.CreateBackup()
		if err != nil {
			t.Fatalf("CreateBackup #%d failed: %v", i, err)
		}
		// Sleep briefly to ensure unique timestamps
		time.Sleep(10 * time.Millisecond)
	}

	// Verify only MaxBackups remain
	backups, err := mgr.ListBackups()
	if err != nil {
		t.Fatalf("ListBackups failed: %v", err)
	}

	if len(backups) != MaxBackups {
		t.Errorf("expected %d backups after rotation, got %d", MaxBackups, len(backups))
	}

	// Verify backups are sorted newest first
	for i := 1; i < len(backups); i++ {
		if backups[i].Timestamp.After(backups[i-1].Timestamp) {
			t.Errorf("backups are not sorted correctly: backup %d is newer than backup %d", i, i-1)
		}
	}
}

func TestListBackups(t *testing.T) {
	dbPath, cleanup := setupTestDB(t)
	defer cleanup()

	mgr := NewManager(dbPath)

	// Initially no backups
	backups, err := mgr.ListBackups()
	if err != nil {
		t.Fatalf("ListBackups failed: %v", err)
	}
	if len(backups) != 0 {
		t.Errorf("expected 0 backups initially, got %d", len(backups))
	}

	// Create some backups
	numBackups := 3
	for i := 0; i < numBackups; i++ {
		_, err := mgr.CreateBackup()
		if err != nil {
			t.Fatalf("CreateBackup #%d failed: %v", i, err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	// List backups
	backups, err = mgr.ListBackups()
	if err != nil {
		t.Fatalf("ListBackups failed: %v", err)
	}

	if len(backups) != numBackups {
		t.Errorf("expected %d backups, got %d", numBackups, len(backups))
	}

	// Verify all backups have valid info
	for _, backup := range backups {
		if backup.Path == "" {
			t.Error("backup path is empty")
		}
		if backup.Size == 0 {
			t.Error("backup size is 0")
		}
		if backup.Timestamp.IsZero() {
			t.Error("backup timestamp is zero")
		}
	}
}

func TestRestoreBackup(t *testing.T) {
	dbPath, cleanup := setupTestDB(t)
	defer cleanup()

	mgr := NewManager(dbPath)

	// Create a backup
	backupPath, err := mgr.CreateBackup()
	if err != nil {
		t.Fatalf("CreateBackup failed: %v", err)
	}

	// Modify the original database
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	_, err = db.Exec("INSERT INTO test_data (id, name, value) VALUES (3, 'test3', 300)")
	if err != nil {
		t.Fatalf("failed to insert data: %v", err)
	}
	db.Close()

	// Verify database has 3 rows
	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM test_data").Scan(&count)
	if err != nil {
		t.Fatalf("failed to query database: %v", err)
	}
	db.Close()
	if count != 3 {
		t.Errorf("expected 3 rows before restore, got %d", count)
	}

	// Restore from backup
	err = mgr.RestoreBackup(backupPath)
	if err != nil {
		t.Fatalf("RestoreBackup failed: %v", err)
	}

	// Verify database is restored to original state (2 rows)
	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open database after restore: %v", err)
	}
	defer db.Close()

	err = db.QueryRow("SELECT COUNT(*) FROM test_data").Scan(&count)
	if err != nil {
		t.Fatalf("failed to query database after restore: %v", err)
	}

	if count != 2 {
		t.Errorf("expected 2 rows after restore, got %d", count)
	}
}

func TestRestoreBackupCreatesPreRestoreBackup(t *testing.T) {
	dbPath, cleanup := setupTestDB(t)
	defer cleanup()

	mgr := NewManager(dbPath)

	// Create initial backup
	backupPath, err := mgr.CreateBackup()
	if err != nil {
		t.Fatalf("CreateBackup failed: %v", err)
	}

	// Count initial backups
	backups, err := mgr.ListBackups()
	if err != nil {
		t.Fatalf("ListBackups failed: %v", err)
	}
	initialCount := len(backups)

	// Restore from backup (should create another backup first)
	err = mgr.RestoreBackup(backupPath)
	if err != nil {
		t.Fatalf("RestoreBackup failed: %v", err)
	}

	// Verify an additional backup was created
	backups, err = mgr.ListBackups()
	if err != nil {
		t.Fatalf("ListBackups failed: %v", err)
	}

	if len(backups) != initialCount+1 {
		t.Errorf("expected %d backups after restore, got %d", initialCount+1, len(backups))
	}
}

func TestVerifyBackup(t *testing.T) {
	dbPath, cleanup := setupTestDB(t)
	defer cleanup()

	mgr := NewManager(dbPath)

	// Create a valid backup
	backupPath, err := mgr.CreateBackup()
	if err != nil {
		t.Fatalf("CreateBackup failed: %v", err)
	}

	// Verify valid backup
	err = mgr.verifyBackup(backupPath)
	if err != nil {
		t.Errorf("verifyBackup failed for valid backup: %v", err)
	}

	// Create an invalid backup file
	invalidPath := filepath.Join(mgr.GetBackupDir(), "invalid.db")
	err = os.WriteFile(invalidPath, []byte("not a database"), 0600)
	if err != nil {
		t.Fatalf("failed to create invalid file: %v", err)
	}

	// Verify invalid backup fails
	err = mgr.verifyBackup(invalidPath)
	if err == nil {
		t.Error("verifyBackup should fail for invalid backup")
	}
}

func TestUniqueBackupFilenames(t *testing.T) {
	dbPath, cleanup := setupTestDB(t)
	defer cleanup()

	mgr := NewManager(dbPath)

	// Create multiple backups in quick succession
	paths := make(map[string]bool)
	for i := 0; i < 5; i++ {
		backupPath, err := mgr.CreateBackup()
		if err != nil {
			t.Fatalf("CreateBackup #%d failed: %v", i, err)
		}
		
		filename := filepath.Base(backupPath)
		if paths[filename] {
			t.Errorf("duplicate backup filename: %s", filename)
		}
		paths[filename] = true
	}
}
