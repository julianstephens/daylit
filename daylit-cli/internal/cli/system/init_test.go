package system

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/julianstephens/daylit/daylit-cli/internal/cli"
	"github.com/julianstephens/daylit/daylit-cli/internal/scheduler"
	"github.com/julianstephens/daylit/daylit-cli/internal/storage"
)

func setupTestInitDB(t *testing.T) (*cli.Context, string, func()) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	store := storage.NewSQLiteStore(dbPath)

	ctx := &cli.Context{
		Store:     store,
		Scheduler: scheduler.New(),
	}

	cleanup := func() {
		if err := store.Close(); err != nil {
			t.Errorf("failed to close store: %v", err)
		}
	}

	return ctx, dbPath, cleanup
}

func TestInitCmd_Success(t *testing.T) {
	ctx, dbPath, cleanup := setupTestInitDB(t)
	defer cleanup()

	cmd := &InitCmd{}
	err := cmd.Run(ctx)

	if err != nil {
		t.Errorf("init command failed: %v", err)
	}

	// Verify database file was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Errorf("database file was not created at %s", dbPath)
	}
}

func TestInitCmd_Idempotent(t *testing.T) {
	ctx, _, cleanup := setupTestInitDB(t)
	defer cleanup()

	cmd := &InitCmd{}

	// Run init first time
	err := cmd.Run(ctx)
	if err != nil {
		t.Fatalf("first init failed: %v", err)
	}

	// Run init second time - should be idempotent
	err = cmd.Run(ctx)
	if err != nil {
		t.Errorf("second init failed (should be idempotent): %v", err)
	}
}

func TestInitCmd_ForceDeletesExisting(t *testing.T) {
	ctx, dbPath, cleanup := setupTestInitDB(t)
	defer cleanup()

	// First, create and initialize database
	normalCmd := &InitCmd{}
	err := normalCmd.Run(ctx)
	if err != nil {
		t.Fatalf("initial init failed: %v", err)
	}

	// Verify database exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Fatalf("database file was not created")
	}

	// Add some data to verify it gets wiped
	// Get initial settings (created by Init)
	initialSettings, err := ctx.Store.GetSettings()
	if err != nil {
		t.Fatalf("failed to get initial settings: %v", err)
	}

	// Modify settings to mark this as "used"
	initialSettings.DayStart = "08:00"
	err = ctx.Store.SaveSettings(initialSettings)
	if err != nil {
		t.Fatalf("failed to save modified settings: %v", err)
	}

	// Close the store before forcing reset
	if err := ctx.Store.Close(); err != nil {
		t.Fatalf("failed to close store before force reset: %v", err)
	}

	// Now run init with force flag
	forceCmd := &InitCmd{Force: true}
	err = forceCmd.Run(ctx)
	if err != nil {
		t.Fatalf("init with force failed: %v", err)
	}

	// Verify database exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Fatalf("database file was not recreated after force")
	}

	// Load the fresh database and verify it has default settings
	err = ctx.Store.Load()
	if err != nil {
		t.Fatalf("failed to load store after force: %v", err)
	}

	newSettings, err := ctx.Store.GetSettings()
	if err != nil {
		t.Fatalf("failed to get settings after force: %v", err)
	}

	// Check that settings are back to defaults
	if newSettings.DayStart != "07:00" {
		t.Errorf("expected default DayStart '07:00', got '%s'", newSettings.DayStart)
	}
}

func TestInitCmd_ForceWithNonExistentDatabase(t *testing.T) {
	ctx, dbPath, cleanup := setupTestInitDB(t)
	defer cleanup()

	// Verify database doesn't exist initially
	if _, err := os.Stat(dbPath); !os.IsNotExist(err) {
		t.Fatalf("database file should not exist initially")
	}

	// Run init with force flag on non-existent database
	forceCmd := &InitCmd{Force: true}
	err := forceCmd.Run(ctx)
	if err != nil {
		t.Fatalf("init with force on non-existent database failed: %v", err)
	}

	// Verify database was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Errorf("database file was not created")
	}
}
