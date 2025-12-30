package cli

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/julianstephens/daylit/internal/models"
	"github.com/julianstephens/daylit/internal/scheduler"
	"github.com/julianstephens/daylit/internal/storage"
)

func setupTestDebugDB(t *testing.T) (*Context, func()) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	
	store := storage.NewSQLiteStore(dbPath)
	if err := store.Init(); err != nil {
		t.Fatalf("failed to initialize store: %v", err)
	}

	ctx := &Context{
		Store:     store,
		Scheduler: scheduler.New(),
	}

	cleanup := func() {
		store.Close()
	}

	return ctx, cleanup
}

func TestDebugDBPathCmd(t *testing.T) {
	ctx, cleanup := setupTestDebugDB(t)
	defer cleanup()

	// Capture stdout would be needed for full test, but we can at least
	// verify it doesn't error
	cmd := &DebugDBPathCmd{}
	err := cmd.Run(ctx)
	
	if err != nil {
		t.Errorf("debug db-path command failed: %v", err)
	}
}

func TestDebugDumpTaskCmd_Success(t *testing.T) {
	ctx, cleanup := setupTestDebugDB(t)
	defer cleanup()

	// Add a test task
	task := models.Task{
		ID:          "test-task-id",
		Name:        "Test Task",
		Kind:        models.TaskKindFlexible,
		DurationMin: 30,
		Recurrence: models.Recurrence{
			Type:         models.RecurrenceDaily,
			IntervalDays: 1,
		},
		Priority:            3,
		Active:              true,
		SuccessStreak:       0,
		AvgActualDurationMin: 30,
	}

	if err := ctx.Store.AddTask(task); err != nil {
		t.Fatalf("failed to add test task: %v", err)
	}

	cmd := &DebugDumpTaskCmd{
		ID: "test-task-id",
	}
	
	err := cmd.Run(ctx)
	if err != nil {
		t.Errorf("debug dump-task command failed: %v", err)
	}
}

func TestDebugDumpTaskCmd_NotFound(t *testing.T) {
	ctx, cleanup := setupTestDebugDB(t)
	defer cleanup()

	cmd := &DebugDumpTaskCmd{
		ID: "nonexistent-id",
	}
	
	err := cmd.Run(ctx)
	if err == nil {
		t.Error("debug dump-task should fail for non-existent task")
	}
	
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestDebugDumpPlanCmd_NotFound(t *testing.T) {
	ctx, cleanup := setupTestDebugDB(t)
	defer cleanup()

	cmd := &DebugDumpPlanCmd{
		Date: "2023-01-01",
	}
	
	err := cmd.Run(ctx)
	if err == nil {
		t.Error("debug dump-plan should fail for non-existent plan")
	}
	
	if !strings.Contains(err.Error(), "not found") && !strings.Contains(err.Error(), "no plan found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestDebugDumpPlanCmd_InvalidDate(t *testing.T) {
	ctx, cleanup := setupTestDebugDB(t)
	defer cleanup()

	cmd := &DebugDumpPlanCmd{
		Date: "invalid-date",
	}
	
	err := cmd.Run(ctx)
	if err == nil {
		t.Error("debug dump-plan should fail for invalid date")
	}
	
	if !strings.Contains(err.Error(), "invalid date") {
		t.Errorf("expected 'invalid date' error, got: %v", err)
	}
}

func TestDebugDumpPlanCmd_Success(t *testing.T) {
	ctx, cleanup := setupTestDebugDB(t)
	defer cleanup()

	// Add a test plan
	plan := models.DayPlan{
		Date:     "2023-01-01",
		Revision: 1,
		Slots:    []models.Slot{},
	}

	if err := ctx.Store.SavePlan(plan); err != nil {
		t.Fatalf("failed to save test plan: %v", err)
	}

	cmd := &DebugDumpPlanCmd{
		Date: "2023-01-01",
	}
	
	err := cmd.Run(ctx)
	if err != nil {
		t.Errorf("debug dump-plan command failed: %v", err)
	}
}

func TestGetCurrentDate(t *testing.T) {
	date := getCurrentDate()
	
	// Should be in YYYY-MM-DD format
	if len(date) != 10 {
		t.Errorf("expected date format YYYY-MM-DD, got: %s", date)
	}
	
	if !isValidDate(date) {
		t.Errorf("getCurrentDate returned invalid date: %s", date)
	}
}

func TestIsValidDate(t *testing.T) {
	tests := []struct {
		date  string
		valid bool
	}{
		{"2023-01-01", true},
		{"2023-12-31", true},
		{"2023-13-01", false},
		{"2023-01-32", false},
		{"invalid", false},
		{"2023/01/01", false},
		{"01-01-2023", false},
	}

	for _, tt := range tests {
		result := isValidDate(tt.date)
		if result != tt.valid {
			t.Errorf("isValidDate(%s) = %v, want %v", tt.date, result, tt.valid)
		}
	}
}

func TestDebugDumpPlanCmd_TodayAlias(t *testing.T) {
	ctx, cleanup := setupTestDebugDB(t)
	defer cleanup()

	// Add a plan for today
	today := getCurrentDate()
	plan := models.DayPlan{
		Date:     today,
		Revision: 1,
		Slots:    []models.Slot{},
	}

	if err := ctx.Store.SavePlan(plan); err != nil {
		t.Fatalf("failed to save test plan: %v", err)
	}

	cmd := &DebugDumpPlanCmd{
		Date: "today",
	}
	
	err := cmd.Run(ctx)
	if err != nil {
		t.Errorf("debug dump-plan with 'today' failed: %v", err)
	}
}

func TestDebugDumpTaskCmd_JSONOutput(t *testing.T) {
	ctx, cleanup := setupTestDebugDB(t)
	defer cleanup()

	// Add a test task
	task := models.Task{
		ID:          "json-test-id",
		Name:        "JSON Test",
		Kind:        models.TaskKindFlexible,
		DurationMin: 45,
		Recurrence: models.Recurrence{
			Type:         models.RecurrenceWeekly,
			IntervalDays: 7,
		},
		Priority:            2,
		Active:              true,
		SuccessStreak:       5,
		AvgActualDurationMin: 50,
	}

	if err := ctx.Store.AddTask(task); err != nil {
		t.Fatalf("failed to add test task: %v", err)
	}

	// Verify task can be retrieved and marshaled
	retrievedTask, err := ctx.Store.GetTask("json-test-id")
	if err != nil {
		t.Fatalf("failed to retrieve task: %v", err)
	}

	jsonBytes, err := json.MarshalIndent(retrievedTask, "", "  ")
	if err != nil {
		t.Errorf("failed to marshal task to JSON: %v", err)
	}

	// Verify JSON contains expected fields
	jsonStr := string(jsonBytes)
	expectedFields := []string{"id", "name", "kind", "duration_min", "priority"}
	for _, field := range expectedFields {
		if !strings.Contains(jsonStr, field) {
			t.Errorf("JSON output missing field: %s", field)
		}
	}
}
