package backup

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const (
	// MaxBackups is the maximum number of backups to keep
	MaxBackups = 14
	// BackupDirName is the name of the backup directory
	BackupDirName = "backups"
	// BackupFilePrefix is the prefix for backup files
	BackupFilePrefix = "daylit-"
	// BackupFileSuffix is the suffix for backup files
	BackupFileSuffix = ".db"
)

// BackupInfo contains information about a backup file
type BackupInfo struct {
	Path      string
	Timestamp time.Time
	Size      int64
}

// Manager handles backup operations
type Manager struct {
	dbPath    string
	backupDir string
}

// NewManager creates a new backup manager
func NewManager(dbPath string) *Manager {
	configDir := filepath.Dir(dbPath)
	backupDir := filepath.Join(configDir, BackupDirName)
	return &Manager{
		dbPath:    dbPath,
		backupDir: backupDir,
	}
}

// GetBackupDir returns the backup directory path
func (m *Manager) GetBackupDir() string {
	return m.backupDir
}

// ensureBackupDir creates the backup directory if it doesn't exist
func (m *Manager) ensureBackupDir() error {
	return os.MkdirAll(m.backupDir, 0700)
}

// CreateBackup creates a new backup of the database
func (m *Manager) CreateBackup() (string, error) {
	return m.createBackup(false)
}

// createBackup creates a new backup of the database
// skipRotation parameter is used to prevent recursive backup creation during restore
func (m *Manager) createBackup(skipRotation bool) (string, error) {
	// Ensure backup directory exists
	if err := m.ensureBackupDir(); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Check if source database exists
	if _, err := os.Stat(m.dbPath); os.IsNotExist(err) {
		return "", fmt.Errorf("database does not exist: %s", m.dbPath)
	}

	// Generate backup filename with timestamp
	// Try with minute precision first
	timestamp := time.Now().Format("20060102-1504")
	backupName := fmt.Sprintf("%s%s%s", BackupFilePrefix, timestamp, BackupFileSuffix)
	backupPath := filepath.Join(m.backupDir, backupName)

	// If a backup with the same name exists, add seconds
	if _, err := os.Stat(backupPath); err == nil {
		timestamp = time.Now().Format("20060102-150405")
		backupName = fmt.Sprintf("%s%s%s", BackupFilePrefix, timestamp, BackupFileSuffix)
		backupPath = filepath.Join(m.backupDir, backupName)

		// If still exists, add a counter
		counter := 1
		for {
			if _, err := os.Stat(backupPath); os.IsNotExist(err) {
				break
			}
			backupName = fmt.Sprintf("%s%s-%d%s", BackupFilePrefix, timestamp, counter, BackupFileSuffix)
			backupPath = filepath.Join(m.backupDir, backupName)
			counter++
			if counter > 100 {
				return "", fmt.Errorf("failed to generate unique backup filename")
			}
		}
	}

	// Use SQLite backup API for safe backup
	if err := m.backupDatabase(backupPath); err != nil {
		return "", fmt.Errorf("failed to backup database: %w", err)
	}

	// Rotate old backups (unless this is part of a restore operation)
	if !skipRotation {
		if err := m.rotateBackups(); err != nil {
			// Log error but don't fail the backup operation
			fmt.Fprintf(os.Stderr, "Warning: failed to rotate old backups: %v\n", err)
		}
	}

	return backupPath, nil
}

// backupDatabase uses SQLite's backup API to safely backup the database
func (m *Manager) backupDatabase(destPath string) error {
	// For SQLite databases, the safest approach is to use a VACUUM INTO command
	// or a simple file copy when the database is properly closed

	// Open source database in read-only mode
	srcDB, err := sql.Open("sqlite", m.dbPath+"?mode=ro")
	if err != nil {
		return fmt.Errorf("failed to open source database: %w", err)
	}
	defer srcDB.Close()

	// Verify source database is valid
	var count int
	if err := srcDB.QueryRow("SELECT COUNT(*) FROM sqlite_master").Scan(&count); err != nil {
		return fmt.Errorf("source database appears to be corrupted: %w", err)
	}

	// Use VACUUM INTO to create a clean copy of the database
	// This is the recommended way to backup SQLite databases
	// Note: VACUUM INTO requires SQLite 3.27.0 or later
	// We use a parameterized query to prevent path injection
	_, err = srcDB.Exec("VACUUM INTO ?", destPath)
	if err != nil {
		// If VACUUM INTO fails (might not be supported in all SQLite versions),
		// fall back to file copy
		srcDB.Close()
		return copyFile(m.dbPath, destPath)
	}

	return nil
}

// ListBackups returns a list of all available backups, sorted by timestamp (newest first)
func (m *Manager) ListBackups() ([]BackupInfo, error) {
	// Check if backup directory exists
	if _, err := os.Stat(m.backupDir); os.IsNotExist(err) {
		return []BackupInfo{}, nil
	}

	entries, err := os.ReadDir(m.backupDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	var backups []BackupInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasPrefix(name, BackupFilePrefix) || !strings.HasSuffix(name, BackupFileSuffix) {
			continue
		}

		// Parse timestamp from filename
		timestampStr := strings.TrimPrefix(name, BackupFilePrefix)
		timestampStr = strings.TrimSuffix(timestampStr, BackupFileSuffix)

		// Remove counter suffix if present (format: YYYYMMDD-HHMM-N or YYYYMMDD-HHMMSS-N)
		// Counter is always after the last hyphen and is all digits
		parts := strings.Split(timestampStr, "-")
		if len(parts) > 2 {
			// Check if last part is a counter (all digits, not 4 or 6 chars which would be time)
			lastPart := parts[len(parts)-1]
			if len(lastPart) != 4 && len(lastPart) != 6 {
				// Could be a counter, check if all digits
				isCounter := true
				for _, c := range lastPart {
					if c < '0' || c > '9' {
						isCounter = false
						break
					}
				}
				if isCounter {
					// Remove the counter part
					timestampStr = strings.Join(parts[:len(parts)-1], "-")
				}
			}
		}

		var timestamp time.Time
		// Try different timestamp formats
		timestamp, err := time.Parse("20060102-1504", timestampStr)
		if err != nil {
			timestamp, err = time.Parse("20060102-150405", timestampStr)
			if err != nil {
				// Skip files with invalid timestamp format
				continue
			}
		}

		path := filepath.Join(m.backupDir, name)
		info, err := os.Stat(path)
		if err != nil {
			continue
		}

		backups = append(backups, BackupInfo{
			Path:      path,
			Timestamp: timestamp,
			Size:      info.Size(),
		})
	}

	// Sort by timestamp, newest first
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Timestamp.After(backups[j].Timestamp)
	})

	return backups, nil
}

// rotateBackups removes old backups beyond the retention limit
func (m *Manager) rotateBackups() error {
	backups, err := m.ListBackups()
	if err != nil {
		return err
	}

	if len(backups) <= MaxBackups {
		return nil
	}

	// Delete oldest backups
	for i := MaxBackups; i < len(backups); i++ {
		if err := os.Remove(backups[i].Path); err != nil {
			return fmt.Errorf("failed to remove old backup %s: %w", backups[i].Path, err)
		}
	}

	return nil
}

// RestoreBackup restores the database from a backup file
func (m *Manager) RestoreBackup(backupPath string) error {
	// Check if backup file exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file does not exist: %s", backupPath)
	}

	// Verify backup file is valid SQLite database
	if err := m.verifyBackup(backupPath); err != nil {
		return fmt.Errorf("backup file is corrupted or invalid: %w", err)
	}

	// Create a backup of the current database before restoring
	if _, err := os.Stat(m.dbPath); err == nil {
		// Current database exists, backup it first
		// Use skipRotation=true to prevent infinite recursion
		currentBackup, err := m.createBackup(true)
		if err != nil {
			return fmt.Errorf("failed to backup current database before restore: %w", err)
		}
		fmt.Printf("Created backup of current database: %s\n", filepath.Base(currentBackup))
	}

	// Copy backup file to database location
	// We use a temporary file and atomic rename to ensure safety
	tempPath := m.dbPath + ".restore.tmp"

	if err := copyFile(backupPath, tempPath); err != nil {
		return fmt.Errorf("failed to copy backup file: %w", err)
	}

	// Rename temporary file to actual database (atomic operation)
	if err := os.Rename(tempPath, m.dbPath); err != nil {
		// Clean up temp file on error
		if removeErr := os.Remove(tempPath); removeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to remove temporary file %s: %v\n", tempPath, removeErr)
		}
		return fmt.Errorf("failed to restore database: %w", err)
	}

	return nil
}

// verifyBackup checks if a backup file is a valid SQLite database
func (m *Manager) verifyBackup(path string) error {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return err
	}
	defer db.Close()

	// Try to query sqlite_master to verify it's a valid database
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master").Scan(&count)
	if err != nil {
		return err
	}

	return nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	if _, err := destFile.ReadFrom(sourceFile); err != nil {
		return err
	}

	// Sync to ensure data is written to disk
	return destFile.Sync()
}
