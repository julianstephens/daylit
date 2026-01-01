package system

import (
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/julianstephens/daylit/daylit-cli/internal/backup"
	"github.com/julianstephens/daylit/daylit-cli/internal/cli"
	"github.com/julianstephens/daylit/daylit-cli/internal/migration"
	"github.com/julianstephens/daylit/daylit-cli/internal/scheduler"
	"github.com/julianstephens/daylit/daylit-cli/internal/storage"
	"github.com/julianstephens/daylit/daylit-cli/migrations"
)

func setupTestDoctorDB(t *testing.T) (*cli.Context, func()) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	store := storage.NewSQLiteStore(dbPath)
	if err := store.Init(); err != nil {
		t.Fatalf("failed to initialize store: %v", err)
	}

	ctx := &cli.Context{
		Store:     store,
		Scheduler: scheduler.New(),
	}

	cleanup := func() {
		store.Close()
	}

	return ctx, cleanup
}

func TestDoctorCmd_HealthyDB(t *testing.T) {
	ctx, cleanup := setupTestDoctorDB(t)
	defer cleanup()

	cmd := &DoctorCmd{}
	err := cmd.Run(ctx)

	// Should pass all checks (except backups which is a warning)
	if err != nil {
		t.Errorf("doctor command failed on healthy database: %v", err)
	}
}

func TestDoctorCmd_MissingBackups(t *testing.T) {
	ctx, cleanup := setupTestDoctorDB(t)
	defer cleanup()

	cmd := &DoctorCmd{}
	err := cmd.Run(ctx)

	// Missing backups is a warning, not a failure
	if err != nil {
		t.Errorf("doctor command should not fail on missing backups: %v", err)
	}
}

func TestDoctorCmd_BrokenSchema(t *testing.T) {
	ctx, cleanup := setupTestDoctorDB(t)
	defer cleanup()

	// Corrupt the schema version
	sqliteStore, ok := ctx.Store.(*storage.SQLiteStore)
	if !ok {
		t.Fatal("expected SQLiteStore")
	}

	db := sqliteStore.GetDB()
	if db == nil {
		t.Fatal("database connection is nil")
	}

	// Set an impossible future schema version
	_, err := db.Exec("DELETE FROM schema_version")
	if err != nil {
		t.Fatalf("failed to delete schema version: %v", err)
	}
	_, err = db.Exec("INSERT INTO schema_version (version) VALUES (999)")
	if err != nil {
		t.Fatalf("failed to insert corrupted schema version: %v", err)
	}

	cmd := &DoctorCmd{}
	err = cmd.Run(ctx)

	// Should fail with schema error
	if err == nil {
		t.Error("doctor command should fail with corrupted schema")
	}
}

func TestDoctorCmd_WithBackups(t *testing.T) {
	ctx, cleanup := setupTestDoctorDB(t)
	defer cleanup()

	// Create a backup
	mgr := backup.NewManager(ctx.Store.GetConfigPath())
	_, err := mgr.CreateBackup()
	if err != nil {
		t.Fatalf("failed to create backup: %v", err)
	}

	cmd := &DoctorCmd{}
	err = cmd.Run(ctx)

	// Should pass all checks including backups
	if err != nil {
		t.Errorf("doctor command failed with backups present: %v", err)
	}
}

func TestCheckMigrationsComplete_Incomplete(t *testing.T) {
	ctx, cleanup := setupTestDoctorDB(t)
	defer cleanup()

	// Downgrade schema version to simulate incomplete migrations
	sqliteStore, ok := ctx.Store.(*storage.SQLiteStore)
	if !ok {
		t.Fatal("expected SQLiteStore")
	}

	db := sqliteStore.GetDB()

	// Get the embedded SQLite migrations sub-filesystem
	subFS, err := fs.Sub(migrations.FS, "sqlite")
	if err != nil {
		t.Fatalf("failed to access sqlite migrations: %v", err)
	}

	runner := migration.NewRunner(db, subFS)

	currentVersion, err := runner.GetCurrentVersion()
	if err != nil {
		t.Fatalf("failed to get current version: %v", err)
	}

	// Set version to one less than current
	if currentVersion > 1 {
		_, err = db.Exec("DELETE FROM schema_version")
		if err != nil {
			t.Fatalf("failed to delete schema version: %v", err)
		}
		_, err = db.Exec("INSERT INTO schema_version (version) VALUES (?)", currentVersion-1)
		if err != nil {
			t.Fatalf("failed to insert downgraded schema version: %v", err)
		}

		// Check migrations should fail
		err = checkMigrationsComplete(ctx)
		if err == nil {
			t.Error("checkMigrationsComplete should fail with incomplete migrations")
		}
	}
}

func TestCheckClockTimezone(t *testing.T) {
	// Basic clock check should pass
	err := checkClockTimezone()
	if err != nil {
		t.Errorf("clock/timezone check failed: %v", err)
	}
}
