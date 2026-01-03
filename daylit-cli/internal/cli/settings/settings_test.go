package settings

import (
	"path/filepath"
	"testing"

	"github.com/julianstephens/daylit/daylit-cli/internal/cli"
	"github.com/julianstephens/daylit/daylit-cli/internal/scheduler"
	"github.com/julianstephens/daylit/daylit-cli/internal/storage/sqlite"
)

func setupTestDB(t *testing.T) (*cli.Context, func()) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	store := sqlite.NewStore(dbPath)
	if err := store.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}

	ctx := &cli.Context{
		Store:     store,
		Scheduler: scheduler.New(),
	}

	cleanup := func() {
		if err := store.Close(); err != nil {
			t.Errorf("failed to close store: %v", err)
		}
	}

	return ctx, cleanup
}

func TestSettingsCmd_List(t *testing.T) {
	ctx, cleanup := setupTestDB(t)
	defer cleanup()

	cmd := &SettingsCmd{
		List: true,
	}

	err := cmd.Run(ctx)
	if err != nil {
		t.Errorf("settings list failed: %v", err)
	}
}

func TestSettingsCmd_UpdateOTPromptOnEmpty(t *testing.T) {
	ctx, cleanup := setupTestDB(t)
	defer cleanup()

	// Get initial settings
	otSettings, err := ctx.Store.GetOTSettings()
	if err != nil {
		t.Fatalf("failed to get OT settings: %v", err)
	}
	initialValue := otSettings.PromptOnEmpty

	// Toggle the value
	newValue := !initialValue
	cmd := &SettingsCmd{
		OTPromptOnEmpty: &newValue,
	}

	err = cmd.Run(ctx)
	if err != nil {
		t.Errorf("settings update failed: %v", err)
	}

	// Verify the change
	updatedSettings, err := ctx.Store.GetOTSettings()
	if err != nil {
		t.Fatalf("failed to get updated OT settings: %v", err)
	}

	if updatedSettings.PromptOnEmpty != newValue {
		t.Errorf("expected PromptOnEmpty to be %v, got %v", newValue, updatedSettings.PromptOnEmpty)
	}
}

func TestSettingsCmd_UpdateOTStrictMode(t *testing.T) {
	ctx, cleanup := setupTestDB(t)
	defer cleanup()

	// Get initial settings
	otSettings, err := ctx.Store.GetOTSettings()
	if err != nil {
		t.Fatalf("failed to get OT settings: %v", err)
	}
	initialValue := otSettings.StrictMode

	// Toggle the value
	newValue := !initialValue
	cmd := &SettingsCmd{
		OTStrictMode: &newValue,
	}

	err = cmd.Run(ctx)
	if err != nil {
		t.Errorf("settings update failed: %v", err)
	}

	// Verify the change
	updatedSettings, err := ctx.Store.GetOTSettings()
	if err != nil {
		t.Fatalf("failed to get updated OT settings: %v", err)
	}

	if updatedSettings.StrictMode != newValue {
		t.Errorf("expected StrictMode to be %v, got %v", newValue, updatedSettings.StrictMode)
	}
}

func TestSettingsCmd_UpdateOTDefaultLogDays(t *testing.T) {
	ctx, cleanup := setupTestDB(t)
	defer cleanup()

	newValue := 30
	cmd := &SettingsCmd{
		OTDefaultLogDays: &newValue,
	}

	err := cmd.Run(ctx)
	if err != nil {
		t.Errorf("settings update failed: %v", err)
	}

	// Verify the change
	updatedSettings, err := ctx.Store.GetOTSettings()
	if err != nil {
		t.Fatalf("failed to get updated OT settings: %v", err)
	}

	if updatedSettings.DefaultLogDays != newValue {
		t.Errorf("expected DefaultLogDays to be %d, got %d", newValue, updatedSettings.DefaultLogDays)
	}
}

func TestSettingsCmd_UpdateOTDefaultLogDays_InvalidValue(t *testing.T) {
	ctx, cleanup := setupTestDB(t)
	defer cleanup()

	// Test with 0 (invalid)
	zeroValue := 0
	cmd := &SettingsCmd{
		OTDefaultLogDays: &zeroValue,
	}

	err := cmd.Run(ctx)
	if err == nil {
		t.Error("expected error for OTDefaultLogDays = 0, got nil")
	}

	// Test with negative value (invalid)
	negativeValue := -5
	cmd = &SettingsCmd{
		OTDefaultLogDays: &negativeValue,
	}

	err = cmd.Run(ctx)
	if err == nil {
		t.Error("expected error for OTDefaultLogDays = -5, got nil")
	}
}

func TestSettingsCmd_UpdateMultipleOTSettings(t *testing.T) {
	ctx, cleanup := setupTestDB(t)
	defer cleanup()

	promptOnEmpty := false
	strictMode := true
	defaultLogDays := 21

	cmd := &SettingsCmd{
		OTPromptOnEmpty:  &promptOnEmpty,
		OTStrictMode:     &strictMode,
		OTDefaultLogDays: &defaultLogDays,
	}

	err := cmd.Run(ctx)
	if err != nil {
		t.Errorf("settings update failed: %v", err)
	}

	// Verify all changes
	updatedSettings, err := ctx.Store.GetOTSettings()
	if err != nil {
		t.Fatalf("failed to get updated OT settings: %v", err)
	}

	if updatedSettings.PromptOnEmpty != promptOnEmpty {
		t.Errorf("expected PromptOnEmpty to be %v, got %v", promptOnEmpty, updatedSettings.PromptOnEmpty)
	}
	if updatedSettings.StrictMode != strictMode {
		t.Errorf("expected StrictMode to be %v, got %v", strictMode, updatedSettings.StrictMode)
	}
	if updatedSettings.DefaultLogDays != defaultLogDays {
		t.Errorf("expected DefaultLogDays to be %d, got %d", defaultLogDays, updatedSettings.DefaultLogDays)
	}
}
