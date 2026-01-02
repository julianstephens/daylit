package system

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/julianstephens/daylit/daylit-cli/internal/cli"
	"github.com/julianstephens/daylit/daylit-cli/internal/constants"
	"github.com/julianstephens/daylit/daylit-cli/internal/models"
	"github.com/julianstephens/daylit/daylit-cli/internal/scheduler"
	"github.com/julianstephens/daylit/daylit-cli/internal/storage"
)

func setupTestDebugDB(t *testing.T) (*cli.Context, func()) {
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
		Kind:        constants.TaskKindFlexible,
		DurationMin: 30,
		Recurrence: models.Recurrence{
			Type:         constants.RecurrenceDaily,
			IntervalDays: 1,
		},
		Priority:             3,
		Active:               true,
		SuccessStreak:        0,
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
		Kind:        constants.TaskKindFlexible,
		DurationMin: 45,
		Recurrence: models.Recurrence{
			Type:         constants.RecurrenceWeekly,
			IntervalDays: 7,
		},
		Priority:             2,
		Active:               true,
		SuccessStreak:        5,
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

func TestDebugDumpHabitCmd_Success(t *testing.T) {
	ctx, cleanup := setupTestDebugDB(t)
	defer cleanup()

	habit := models.Habit{
		ID:   "test-habit-id",
		Name: "Test Habit",
	}

	if err := ctx.Store.AddHabit(habit); err != nil {
		t.Fatalf("failed to add test habit: %v", err)
	}

	cmd := &DebugDumpHabitCmd{
		ID: "test-habit-id",
	}

	err := cmd.Run(ctx)
	if err != nil {
		t.Errorf("debug dump-habit command failed: %v", err)
	}
}

func TestDebugDumpHabitCmd_NotFound(t *testing.T) {
	ctx, cleanup := setupTestDebugDB(t)
	defer cleanup()

	cmd := &DebugDumpHabitCmd{
		ID: "nonexistent-habit",
	}

	err := cmd.Run(ctx)
	if err == nil {
		t.Error("debug dump-habit should fail for non-existent habit")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestDebugDumpHabitCmd_JSONOutput(t *testing.T) {
	ctx, cleanup := setupTestDebugDB(t)
	defer cleanup()

	habit := models.Habit{
		ID:   "json-habit-id",
		Name: "JSON Habit",
	}

	if err := ctx.Store.AddHabit(habit); err != nil {
		t.Fatalf("failed to add test habit: %v", err)
	}

	// Verify habit can be retrieved and marshaled
	retrievedHabit, err := ctx.Store.GetHabit("json-habit-id")
	if err != nil {
		t.Fatalf("failed to retrieve habit: %v", err)
	}

	jsonBytes, err := json.MarshalIndent(retrievedHabit, "", "  ")
	if err != nil {
		t.Errorf("failed to marshal habit to JSON: %v", err)
	}

	jsonStr := string(jsonBytes)
	expectedFields := []string{"id", "name"}
	for _, field := range expectedFields {
		if !strings.Contains(jsonStr, field) {
			t.Errorf("JSON output missing field: %s", field)
		}
	}
}

func TestDebugDumpOTCmd_Success(t *testing.T) {
	ctx, cleanup := setupTestDebugDB(t)
	defer cleanup()

	ot := models.OTEntry{
		Day:   "2023-01-01",
		Title: "Test OT",
		Note:  "Test Note",
	}

	if err := ctx.Store.AddOTEntry(ot); err != nil {
		t.Fatalf("failed to add test OT entry: %v", err)
	}

	cmd := &DebugDumpOTCmd{
		Day: "2023-01-01",
	}

	err := cmd.Run(ctx)
	if err != nil {
		t.Errorf("debug dump-ot command failed: %v", err)
	}
}

func TestDebugDumpOTCmd_NotFound(t *testing.T) {
	ctx, cleanup := setupTestDebugDB(t)
	defer cleanup()

	cmd := &DebugDumpOTCmd{
		Day: "2023-01-01",
	}

	err := cmd.Run(ctx)
	if err == nil {
		t.Error("debug dump-ot should fail for non-existent OT entry")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestDebugDumpOTCmd_InvalidDate(t *testing.T) {
	ctx, cleanup := setupTestDebugDB(t)
	defer cleanup()

	cmd := &DebugDumpOTCmd{
		Day: "invalid-date",
	}

	err := cmd.Run(ctx)
	if err == nil {
		t.Error("debug dump-ot should fail for invalid date")
	}

	if !strings.Contains(err.Error(), "invalid date") {
		t.Errorf("expected 'invalid date' error, got: %v", err)
	}
}

func TestDebugDumpOTCmd_TodayAlias(t *testing.T) {
	ctx, cleanup := setupTestDebugDB(t)
	defer cleanup()

	today := getCurrentDate()
	ot := models.OTEntry{
		Day:   today,
		Title: "Today OT",
	}

	if err := ctx.Store.AddOTEntry(ot); err != nil {
		t.Fatalf("failed to add test OT entry: %v", err)
	}

	cmd := &DebugDumpOTCmd{
		Day: "today",
	}

	err := cmd.Run(ctx)
	if err != nil {
		t.Errorf("debug dump-ot with 'today' failed: %v", err)
	}
}

func TestDebugDumpOTCmd_JSONOutput(t *testing.T) {
	ctx, cleanup := setupTestDebugDB(t)
	defer cleanup()

	ot := models.OTEntry{
		Day:   "2023-01-02",
		Title: "JSON OT",
		Note:  "JSON Note",
	}

	if err := ctx.Store.AddOTEntry(ot); err != nil {
		t.Fatalf("failed to add test OT entry: %v", err)
	}

	retrievedOT, err := ctx.Store.GetOTEntry("2023-01-02")
	if err != nil {
		t.Fatalf("failed to retrieve OT entry: %v", err)
	}

	jsonBytes, err := json.MarshalIndent(retrievedOT, "", "  ")
	if err != nil {
		t.Errorf("failed to marshal OT entry to JSON: %v", err)
	}

	jsonStr := string(jsonBytes)
	expectedFields := []string{"day", "title", "note"}
	for _, field := range expectedFields {
		if !strings.Contains(jsonStr, field) {
			t.Errorf("JSON output missing field: %s", field)
		}
	}
}

func TestDebugDumpAlertCmd_Success(t *testing.T) {
	ctx, cleanup := setupTestDebugDB(t)
	defer cleanup()

	alert := models.Alert{
		ID:      "test-alert-id",
		Message: "Test Alert",
		Time:    "10:00",
		Recurrence: models.Recurrence{
			Type: constants.RecurrenceDaily,
		},
		Active: true,
	}

	if err := ctx.Store.AddAlert(alert); err != nil {
		t.Fatalf("failed to add test alert: %v", err)
	}

	cmd := &DebugDumpAlertCmd{
		ID: "test-alert-id",
	}

	err := cmd.Run(ctx)
	if err != nil {
		t.Errorf("debug dump-alert command failed: %v", err)
	}
}

func TestDebugDumpAlertCmd_NotFound(t *testing.T) {
	ctx, cleanup := setupTestDebugDB(t)
	defer cleanup()

	cmd := &DebugDumpAlertCmd{
		ID: "nonexistent-alert",
	}

	err := cmd.Run(ctx)
	if err == nil {
		t.Error("debug dump-alert should fail for non-existent alert")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestDebugDumpAlertCmd_JSONOutput(t *testing.T) {
	ctx, cleanup := setupTestDebugDB(t)
	defer cleanup()

	alert := models.Alert{
		ID:      "json-alert-id",
		Message: "JSON Alert",
		Time:    "12:00",
		Recurrence: models.Recurrence{
			Type: constants.RecurrenceDaily,
		},
		Active: true,
	}

	if err := ctx.Store.AddAlert(alert); err != nil {
		t.Fatalf("failed to add test alert: %v", err)
	}

	retrievedAlert, err := ctx.Store.GetAlert("json-alert-id")
	if err != nil {
		t.Fatalf("failed to retrieve alert: %v", err)
	}

	jsonBytes, err := json.MarshalIndent(retrievedAlert, "", "  ")
	if err != nil {
		t.Errorf("failed to marshal alert to JSON: %v", err)
	}

	jsonStr := string(jsonBytes)
	expectedFields := []string{"id", "message", "time", "recurrence", "active"}
	for _, field := range expectedFields {
		if !strings.Contains(jsonStr, field) {
			t.Errorf("JSON output missing field: %s", field)
		}
	}
}

func TestDebugDumpSettingsCmd_Success(t *testing.T) {
	ctx, cleanup := setupTestDebugDB(t)
	defer cleanup()

	settings := models.Settings{
		DayStart: "09:00",
		DayEnd:   "17:00",
	}

	if err := ctx.Store.SaveSettings(settings); err != nil {
		t.Fatalf("failed to save settings: %v", err)
	}

	cmd := &DebugDumpSettingsCmd{}

	err := cmd.Run(ctx)
	if err != nil {
		t.Errorf("debug dump-settings command failed: %v", err)
	}
}

func TestDebugDumpSettingsCmd_JSONOutput(t *testing.T) {
	ctx, cleanup := setupTestDebugDB(t)
	defer cleanup()

	settings := models.Settings{
		DayStart: "08:30",
		DayEnd:   "18:30",
	}

	if err := ctx.Store.SaveSettings(settings); err != nil {
		t.Fatalf("failed to save settings: %v", err)
	}

	retrievedSettings, err := ctx.Store.GetSettings()
	if err != nil {
		t.Fatalf("failed to retrieve settings: %v", err)
	}

	jsonBytes, err := json.MarshalIndent(retrievedSettings, "", "  ")
	if err != nil {
		t.Errorf("failed to marshal settings to JSON: %v", err)
	}

	jsonStr := string(jsonBytes)
	expectedFields := []string{"day_start", "day_end"}
	for _, field := range expectedFields {
		if !strings.Contains(jsonStr, field) {
			t.Errorf("JSON output missing field: %s", field)
		}
	}
}
