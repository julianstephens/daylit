package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/julianstephens/daylit/internal/backup"
)

type BackupCreateCmd struct{}

func (c *BackupCreateCmd) Run(ctx *Context) error {
	// Perform a manual backup
	if err := ctx.Store.Load(); err != nil {
		return err
	}

	mgr := backup.NewManager(ctx.Store.GetConfigPath())
	backupPath, err := mgr.CreateBackup()
	if err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}

	fmt.Printf("✓ Backup created: %s\n", filepath.Base(backupPath))
	return nil
}

type BackupListCmd struct{}

func (c *BackupListCmd) Run(ctx *Context) error {
	if err := ctx.Store.Load(); err != nil {
		return err
	}

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

func (c *BackupRestoreCmd) Run(ctx *Context) error {
	if err := ctx.Store.Load(); err != nil {
		return err
	}

	mgr := backup.NewManager(ctx.Store.GetConfigPath())

	// Determine the full path to the backup file
	backupPath := c.BackupFile
	if !filepath.IsAbs(backupPath) {
		// If it's not an absolute path, check if it exists relative to backup directory
		possiblePath := filepath.Join(mgr.GetBackupDir(), c.BackupFile)
		if _, err := os.Stat(possiblePath); err == nil {
			backupPath = possiblePath
		}
	}

	// Verify backup file exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file not found: %s", backupPath)
	}

	// Show warning and ask for confirmation
	fmt.Println("⚠️  WARNING: This will replace your current database with the backup.")
	fmt.Println("A backup of your current database will be created before restoring.")
	fmt.Printf("\nRestore from: %s\n", filepath.Base(backupPath))
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
	fmt.Println("Restart any running daylit processes to use the restored database.")

	return nil
}
