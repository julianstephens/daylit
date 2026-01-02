package system

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/julianstephens/daylit/daylit-cli/internal/cli"
	"github.com/julianstephens/daylit/daylit-cli/internal/constants"
	"github.com/julianstephens/daylit/daylit-cli/internal/models"
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

func createTestTask(id, name string) models.Task {
	return models.Task{
		ID:          id,
		Name:        name,
		Kind:        constants.TaskKindFlexible,
		DurationMin: 60,
		Recurrence: models.Recurrence{
			Type: constants.RecurrenceAdHoc,
		},
		Priority:   1,
		EnergyBand: constants.EnergyMedium,
		Active:     true,
	}
}

func createTestPlan(date string, revision int, taskIDs []string) models.DayPlan {
	plan := models.DayPlan{
		Date:     date,
		Revision: revision,
		Slots:    []models.Slot{},
	}

	startHour := 9
	for _, taskID := range taskIDs {
		slot := models.Slot{
			Start:  fmt.Sprintf("%02d:00", startHour),
			End:    fmt.Sprintf("%02d:30", startHour),
			TaskID: taskID,
			Status: constants.SlotStatusPlanned,
		}
		plan.Slots = append(plan.Slots, slot)
		startHour++
	}

	return plan
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

func TestInitCmd_MigrationFromSQLiteToSQLite(t *testing.T) {
	tempDir := t.TempDir()

	// Create and populate source database
	sourceDBPath := filepath.Join(tempDir, "source.db")
	sourceStore := storage.NewSQLiteStore(sourceDBPath)
	if err := sourceStore.Init(); err != nil {
		t.Fatalf("failed to init source store: %v", err)
	}

	// Add test data to source
	sourceSettings := storage.Settings{
		DayStart:                   "08:30",
		DayEnd:                     "21:00",
		DefaultBlockMin:            45,
		NotificationsEnabled:       false,
		NotifyBlockStart:           true,
		NotifyBlockEnd:             false,
		BlockStartOffsetMin:        10,
		BlockEndOffsetMin:          15,
		NotificationGracePeriodMin: 5,
	}
	if err := sourceStore.SaveSettings(sourceSettings); err != nil {
		t.Fatalf("failed to save settings: %v", err)
	}

	sourceStore.Close()

	// Create destination database
	destDBPath := filepath.Join(tempDir, "dest.db")
	destStore := storage.NewSQLiteStore(destDBPath)

	ctx := &cli.Context{
		Store:     destStore,
		Scheduler: scheduler.New(),
	}

	// Run init with migration
	cmd := &InitCmd{Source: sourceDBPath}
	err := cmd.Run(ctx)
	if err != nil {
		t.Fatalf("init with migration failed: %v", err)
	}

	// Verify destination was created
	if _, err := os.Stat(destDBPath); os.IsNotExist(err) {
		t.Fatalf("destination database was not created")
	}

	// Verify settings were migrated
	destSettings, err := destStore.GetSettings()
	if err != nil {
		t.Fatalf("failed to get settings from destination: %v", err)
	}

	if destSettings.DayStart != sourceSettings.DayStart {
		t.Errorf("DayStart not migrated correctly: got %s, want %s", destSettings.DayStart, sourceSettings.DayStart)
	}
	if destSettings.DayEnd != sourceSettings.DayEnd {
		t.Errorf("DayEnd not migrated correctly: got %s, want %s", destSettings.DayEnd, sourceSettings.DayEnd)
	}
	if destSettings.DefaultBlockMin != sourceSettings.DefaultBlockMin {
		t.Errorf("DefaultBlockMin not migrated correctly: got %d, want %d", destSettings.DefaultBlockMin, sourceSettings.DefaultBlockMin)
	}

	destStore.Close()
}

func TestInitCmd_MigrationPreventsSourceDestinationConflict(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Create a database
	store := storage.NewSQLiteStore(dbPath)
	if err := store.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}
	store.Close()

	// Try to migrate to the same location with force - should fail
	ctx := &cli.Context{
		Store:     storage.NewSQLiteStore(dbPath),
		Scheduler: scheduler.New(),
	}

	cmd := &InitCmd{Force: true, Source: dbPath}
	err := cmd.Run(ctx)

	if err == nil {
		t.Fatal("expected error when source and destination are the same with --force, got nil")
	}

	if !filepath.IsAbs(dbPath) {
		t.Error("dbPath should be absolute")
	}
}

func TestInitCmd_MigrationWithNonExistentSource(t *testing.T) {
	tempDir := t.TempDir()
	destDBPath := filepath.Join(tempDir, "dest.db")
	nonExistentSource := filepath.Join(tempDir, "nonexistent.db")

	destStore := storage.NewSQLiteStore(destDBPath)
	ctx := &cli.Context{
		Store:     destStore,
		Scheduler: scheduler.New(),
	}

	cmd := &InitCmd{Source: nonExistentSource}
	err := cmd.Run(ctx)

	if err == nil {
		t.Fatal("expected error when migrating from non-existent source, got nil")
	}

	destStore.Close()
}

func TestInitCmd_MigrationWithTasksAndPlans(t *testing.T) {
	tempDir := t.TempDir()

	// Create and populate source database with actual data
	sourceDBPath := filepath.Join(tempDir, "source.db")
	sourceStore := storage.NewSQLiteStore(sourceDBPath)
	if err := sourceStore.Init(); err != nil {
		t.Fatalf("failed to init source store: %v", err)
	}

	// Add a task to source
	task := createTestTask("task-1", "Test Task")
	if err := sourceStore.AddTask(task); err != nil {
		t.Fatalf("failed to add task to source: %v", err)
	}

	// Add a plan to source
	plan := createTestPlan("2024-01-01", 1, []string{"task-1"})
	if err := sourceStore.SavePlan(plan); err != nil {
		t.Fatalf("failed to save plan to source: %v", err)
	}

	sourceStore.Close()

	// Create destination database
	destDBPath := filepath.Join(tempDir, "dest.db")
	destStore := storage.NewSQLiteStore(destDBPath)

	ctx := &cli.Context{
		Store:     destStore,
		Scheduler: scheduler.New(),
	}

	// Run init with migration
	cmd := &InitCmd{Source: sourceDBPath}
	err := cmd.Run(ctx)
	if err != nil {
		t.Fatalf("init with migration failed: %v", err)
	}

	// Verify task was migrated
	tasks, err := destStore.GetAllTasks()
	if err != nil {
		t.Fatalf("failed to get tasks from destination: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].ID != "task-1" {
		t.Errorf("expected task ID 'task-1', got '%s'", tasks[0].ID)
	}
	if tasks[0].Name != "Test Task" {
		t.Errorf("expected task name 'Test Task', got '%s'", tasks[0].Name)
	}

	// Verify plan was migrated
	migratedPlan, err := destStore.GetPlan("2024-01-01")
	if err != nil {
		t.Fatalf("failed to get plan from destination: %v", err)
	}
	if migratedPlan.Date != "2024-01-01" {
		t.Errorf("expected plan date '2024-01-01', got '%s'", migratedPlan.Date)
	}
	if len(migratedPlan.Slots) != 1 {
		t.Fatalf("expected 1 slot, got %d", len(migratedPlan.Slots))
	}
	if migratedPlan.Slots[0].TaskID != "task-1" {
		t.Errorf("expected slot task ID 'task-1', got '%s'", migratedPlan.Slots[0].TaskID)
	}

	destStore.Close()
}
