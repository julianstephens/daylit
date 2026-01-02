package backups

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/julianstephens/daylit/daylit-cli/internal/backup"
	"github.com/julianstephens/daylit/daylit-cli/internal/cli"
)

type BackupCreateCmd struct{}

func (c *BackupCreateCmd) Run(ctx *cli.Context) error {
	// Perform a manual backup
	mgr := backup.NewManager(ctx.Store.GetConfigPath())
	backupPath, err := mgr.CreateBackup()
	if err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}

	fmt.Printf("✓ Backup created: %s\n", filepath.Base(backupPath))
	return nil
}

type BackupListCmd struct{}

func (c *BackupListCmd) Run(ctx *cli.Context) error {
	mgr := backup.NewManager(ctx.Store.GetConfigPath())
	backups, err := mgr.ListBackups()
	if err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}

	if len(backups) == 0 {
		fmt.Println("No backups found.")
		fmt.Printf("Backups are stored in: %s\n", mgr.GetBackupDir())
		return nil
	}

	fmt.Printf("Available backups (%d total, keeping most recent %d):\n\n", len(backups), backup.MaxBackups)
	for _, b := range backups {
		sizeKB := float64(b.Size) / 1024.0
		timestamp := b.Timestamp.Format("2006-01-02 15:04:05")
		filename := filepath.Base(b.Path)
		fmt.Printf("  %s  %s  (%.1f KB)\n", timestamp, filename, sizeKB)
	}
	fmt.Printf("\nBackup directory: %s\n", mgr.GetBackupDir())

	return nil
}

type BackupRestoreCmd struct {
	BackupFile string `arg:"" help:"Path or filename of the backup to restore."`
}

func (c *BackupRestoreCmd) Run(ctx *cli.Context) error {
	mgr := backup.NewManager(ctx.Store.GetConfigPath())

	// Determine the full path to the backup file
	backupPath := c.BackupFile

	// If it's an absolute path, use it directly
	if filepath.IsAbs(backupPath) {
		// Verify absolute path exists
		if _, err := os.Stat(backupPath); os.IsNotExist(err) {
			return fmt.Errorf("backup file not found: %s", backupPath)
		}
	} else {
		// For relative paths, first check current directory
		if _, err := os.Stat(backupPath); err == nil {
			// File exists in current directory - convert to absolute path
			absPath, err := filepath.Abs(backupPath)
			if err != nil {
				return fmt.Errorf("failed to resolve backup path: %w", err)
			}
			backupPath = absPath
		} else {
			// Check backup directory
			possiblePath := filepath.Join(mgr.GetBackupDir(), c.BackupFile)
			if _, err := os.Stat(possiblePath); err == nil {
				backupPath = possiblePath
			} else {
				return fmt.Errorf("backup file not found: tried current directory and %s", mgr.GetBackupDir())
			}
		}
	}

	// Show warning and ask for confirmation
	fmt.Println("⚠️  WARNING: This will replace your current database with the backup.")
	fmt.Println("⚠️  IMPORTANT: All daylit processes (including TUI) must be stopped before restore.")
	fmt.Println("             Concurrent access during restore can cause data corruption.")
	fmt.Println("A backup of your current database will be created before restoring.")
	fmt.Printf("\nRestore from: %s\n", backupPath)
	fmt.Print("Continue? [y/N]: ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	response = strings.TrimSpace(strings.ToLower(response))

	if response != "y" && response != "yes" {
		fmt.Println("Restore cancelled.")
		return nil
	}

	// Close the current store connection before restoring
	if err := ctx.Store.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to close database connection: %v\n", err)
	}

	// Perform restore
	if err := mgr.RestoreBackup(backupPath); err != nil {
		return fmt.Errorf("restore failed: %w", err)
	}

	fmt.Println("✓ Database restored successfully!")
	fmt.Println("⚠️  Remember to restart any daylit processes that were stopped for the restore.")
	fmt.Println("    The restored database is now active.")

	return nil
}
