package system

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/julianstephens/daylit/daylit-cli/internal/cli"
	"github.com/julianstephens/daylit/daylit-cli/internal/constants"
	"github.com/julianstephens/daylit/daylit-cli/internal/models"
	"github.com/julianstephens/daylit/daylit-cli/internal/storage"
)

func setupTestStore(t *testing.T) (*storage.SQLiteStore, func()) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	store := storage.NewSQLiteStore(dbPath)
	if err := store.Init(); err != nil {
		t.Fatalf("failed to initialize test store: %v", err)
	}

	cleanup := func() {
		store.Close()
		os.RemoveAll(tempDir)
	}

	return store, cleanup
}

// Helper function to calculate end time correctly handling hour overflow
func calculateEndTime(startMinutes, durationMin int) string {
	endMinutes := startMinutes + durationMin
	endHour := endMinutes / 60
	endMin := endMinutes % 60
	return fmt.Sprintf("%02d:%02d", endHour, endMin)
}

func TestNotifyCmd_Idempotency(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// Create a task
	task := models.Task{
		ID:          "task-notify-1",
		Name:        "Test Task",
		Kind:        constants.TaskKindFlexible,
		DurationMin: 30,
		Recurrence: models.Recurrence{
			Type: constants.RecurrenceDaily,
		},
		Priority: 1,
		Active:   true,
	}
	if err := store.AddTask(task); err != nil {
		t.Fatalf("failed to add task: %v", err)
	}

	// Create a plan with a slot that should trigger notification
	now := time.Now()
	currentMinutes := now.Hour()*60 + now.Minute()

	// We want a slot that triggers a notification.
	// Default offset is 5 min, grace period is 10 min.
	// If we set startTime = currentMinutes + 3:
	// triggerTime = (currentMinutes + 3) - 5 = currentMinutes - 2
	// minutesLate = currentMinutes - (currentMinutes - 2) = 2
	// 2 <= 10 (grace period), so it should trigger.
	startMinutes := currentMinutes + 3

	// Skip if near end of day to avoid crossing midnight (which would make startTime invalid for today)
	// We also need endTime (start + 30) to be valid.
	if startMinutes+30 >= 24*60 {
		t.Skip("Skipping test near end of day")
	}

	startHour := startMinutes / 60
	startMin := startMinutes % 60
	startTime := fmt.Sprintf("%02d:%02d", startHour, startMin)
	endTime := calculateEndTime(startMinutes, 30)

	nowStr := time.Now().UTC().Format(time.RFC3339)
	plan := models.DayPlan{
		Date:       now.Format("2006-01-02"),
		Revision:   0,
		AcceptedAt: &nowStr,
		Slots: []models.Slot{
			{
				Start:  startTime,
				End:    endTime,
				TaskID: task.ID,
				Status: constants.SlotStatusAccepted,
			},
		},
	}

	if err := store.SavePlan(plan); err != nil {
		t.Fatalf("failed to save plan: %v", err)
	}

	// Create context
	ctx := &cli.Context{
		Store: store,
	}

	// Run notify command first time
	cmd := &NotifyCmd{DryRun: true}
	if err := cmd.Run(ctx); err != nil {
		t.Fatalf("first notify run failed: %v", err)
	}

	// Get the plan to check notification timestamp
	retrievedPlan, err := store.GetPlan(plan.Date)
	if err != nil {
		t.Fatalf("failed to retrieve plan: %v", err)
	}

	if len(retrievedPlan.Slots) == 0 {
		t.Fatal("no slots in retrieved plan")
	}

	firstSlot := retrievedPlan.Slots[0]
	if firstSlot.LastNotifiedStart == nil {
		t.Error("expected LastNotifiedStart to be set after first run")
	}

	// Store the first notification time
	firstNotificationTime := firstSlot.LastNotifiedStart

	// Run notify command second time (should be idempotent)
	if err := cmd.Run(ctx); err != nil {
		t.Fatalf("second notify run failed: %v", err)
	}

	// Get the plan again
	retrievedPlan2, err := store.GetPlan(plan.Date)
	if err != nil {
		t.Fatalf("failed to retrieve plan second time: %v", err)
	}

	secondSlot := retrievedPlan2.Slots[0]
	if secondSlot.LastNotifiedStart == nil {
		t.Error("expected LastNotifiedStart to still be set after second run")
	}

	// The notification timestamp should be the same (idempotent)
	if firstNotificationTime != nil && secondSlot.LastNotifiedStart != nil {
		if *firstNotificationTime != *secondSlot.LastNotifiedStart {
			t.Error("notification timestamp changed on second run - not idempotent")
		}
	}

	// Run a third time to be sure
	if err := cmd.Run(ctx); err != nil {
		t.Fatalf("third notify run failed: %v", err)
	}

	retrievedPlan3, err := store.GetPlan(plan.Date)
	if err != nil {
		t.Fatalf("failed to retrieve plan third time: %v", err)
	}

	thirdSlot := retrievedPlan3.Slots[0]
	if firstNotificationTime != nil && thirdSlot.LastNotifiedStart != nil {
		if *firstNotificationTime != *thirdSlot.LastNotifiedStart {
			t.Error("notification timestamp changed on third run - not idempotent")
		}
	}
}

func TestNotifyCmd_GracePeriod(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// Create a task
	task := models.Task{
		ID:          "task-grace-1",
		Name:        "Test Grace Period",
		Kind:        constants.TaskKindFlexible,
		DurationMin: 30,
		Recurrence: models.Recurrence{
			Type: constants.RecurrenceDaily,
		},
		Priority: 1,
		Active:   true,
	}
	if err := store.AddTask(task); err != nil {
		t.Fatalf("failed to add task: %v", err)
	}

	now := time.Now()
	currentMinutes := now.Hour()*60 + now.Minute()

	// Test 1: Notification within grace period (5 minutes late)
	t.Run("WithinGracePeriod", func(t *testing.T) {
		// Set start time to now. With 5 min offset, notification should have happened 5 mins ago.
		// This is within the 10 min grace period.
		startMinutes := currentMinutes
		startHour := startMinutes / 60
		startMin := startMinutes % 60
		startTime := fmt.Sprintf("%02d:%02d", startHour, startMin)
		endTime := calculateEndTime(startMinutes, 30)

		nowStr := time.Now().UTC().Format(time.RFC3339)
		plan := models.DayPlan{
			Date:       now.Format("2006-01-02"),
			Revision:   0,
			AcceptedAt: &nowStr,
			Slots: []models.Slot{
				{
					Start:  startTime,
					End:    endTime,
					TaskID: task.ID,
					Status: constants.SlotStatusAccepted,
				},
			},
		}

		if err := store.SavePlan(plan); err != nil {
			t.Fatalf("failed to save plan: %v", err)
		}

		ctx := &cli.Context{Store: store}
		cmd := &NotifyCmd{DryRun: true}

		if err := cmd.Run(ctx); err != nil {
			t.Fatalf("notify run failed: %v", err)
		}

		retrievedPlan, err := store.GetPlan(plan.Date)
		if err != nil {
			t.Fatalf("failed to retrieve plan: %v", err)
		}

		if len(retrievedPlan.Slots) == 0 {
			t.Fatal("no slots in retrieved plan")
		}

		if retrievedPlan.Slots[0].LastNotifiedStart == nil {
			t.Error("expected notification to be sent within grace period")
		}
	})

	// Test 2: Notification outside grace period (15 minutes late)
	t.Run("OutsideGracePeriod", func(t *testing.T) {
		// Use a different date to avoid conflict
		tomorrow := now.AddDate(0, 0, 1)
		triggerMinutes := currentMinutes - 15
		startHour := triggerMinutes / 60
		startMin := triggerMinutes % 60
		startTime := fmt.Sprintf("%02d:%02d", startHour, startMin)
		endTime := calculateEndTime(triggerMinutes, 30)

		nowStr := time.Now().UTC().Format(time.RFC3339)
		plan := models.DayPlan{
			Date:       tomorrow.Format("2006-01-02"),
			Revision:   0,
			AcceptedAt: &nowStr,
			Slots: []models.Slot{
				{
					Start:  startTime,
					End:    endTime,
					TaskID: task.ID,
					Status: constants.SlotStatusAccepted,
				},
			},
		}

		if err := store.SavePlan(plan); err != nil {
			t.Fatalf("failed to save plan: %v", err)
		}

		ctx := &cli.Context{Store: store}
		cmd := &NotifyCmd{DryRun: true}

		if err := cmd.Run(ctx); err != nil {
			t.Fatalf("notify run failed: %v", err)
		}

		retrievedPlan, err := store.GetPlan(plan.Date)
		if err != nil {
			t.Fatalf("failed to retrieve plan: %v", err)
		}

		if len(retrievedPlan.Slots) == 0 {
			t.Fatal("no slots in retrieved plan")
		}

		if retrievedPlan.Slots[0].LastNotifiedStart != nil {
			t.Error("expected notification to be skipped outside grace period")
		}
	})
}

func TestNotifyCmd_NoNotificationBeforeTime(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// Create a task
	task := models.Task{
		ID:          "task-future-1",
		Name:        "Future Task",
		Kind:        constants.TaskKindFlexible,
		DurationMin: 30,
		Recurrence: models.Recurrence{
			Type: constants.RecurrenceDaily,
		},
		Priority: 1,
		Active:   true,
	}
	if err := store.AddTask(task); err != nil {
		t.Fatalf("failed to add task: %v", err)
	}

	now := time.Now()
	currentMinutes := now.Hour()*60 + now.Minute()

	// Create a slot that should trigger 10 minutes from now
	triggerMinutes := currentMinutes + 10
	startHour := triggerMinutes / 60
	startMin := triggerMinutes % 60
	startTime := fmt.Sprintf("%02d:%02d", startHour, startMin)
	endTime := calculateEndTime(triggerMinutes, 30)

	nowStr := time.Now().UTC().Format(time.RFC3339)
	plan := models.DayPlan{
		Date:       now.Format("2006-01-02"),
		Revision:   0,
		AcceptedAt: &nowStr,
		Slots: []models.Slot{
			{
				Start:  startTime,
				End:    endTime,
				TaskID: task.ID,
				Status: constants.SlotStatusAccepted,
			},
		},
	}

	if err := store.SavePlan(plan); err != nil {
		t.Fatalf("failed to save plan: %v", err)
	}

	ctx := &cli.Context{Store: store}
	cmd := &NotifyCmd{DryRun: true}

	if err := cmd.Run(ctx); err != nil {
		t.Fatalf("notify run failed: %v", err)
	}

	retrievedPlan, err := store.GetPlan(plan.Date)
	if err != nil {
		t.Fatalf("failed to retrieve plan: %v", err)
	}

	if len(retrievedPlan.Slots) == 0 {
		t.Fatal("no slots in retrieved plan")
	}

	if retrievedPlan.Slots[0].LastNotifiedStart != nil {
		t.Error("expected no notification for future slot")
	}
}

func TestNotifyCmd_DisabledNotifications(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// Disable notifications in settings
	settings, err := store.GetSettings()
	if err != nil {
		t.Fatalf("failed to get settings: %v", err)
	}
	settings.NotificationsEnabled = false
	if err := store.SaveSettings(settings); err != nil {
		t.Fatalf("failed to save settings: %v", err)
	}

	// Create a task
	task := models.Task{
		ID:          "task-disabled-1",
		Name:        "Test Task",
		Kind:        constants.TaskKindFlexible,
		DurationMin: 30,
		Recurrence: models.Recurrence{
			Type: constants.RecurrenceDaily,
		},
		Priority: 1,
		Active:   true,
	}
	if err := store.AddTask(task); err != nil {
		t.Fatalf("failed to add task: %v", err)
	}

	now := time.Now()
	currentMinutes := now.Hour()*60 + now.Minute()
	triggerMinutes := currentMinutes - 2
	startHour := triggerMinutes / 60
	startMin := triggerMinutes % 60
	startTime := fmt.Sprintf("%02d:%02d", startHour, startMin)
	endTime := calculateEndTime(triggerMinutes, 30)

	nowStr := time.Now().UTC().Format(time.RFC3339)
	plan := models.DayPlan{
		Date:       now.Format("2006-01-02"),
		Revision:   0,
		AcceptedAt: &nowStr,
		Slots: []models.Slot{
			{
				Start:  startTime,
				End:    endTime,
				TaskID: task.ID,
				Status: constants.SlotStatusAccepted,
			},
		},
	}

	if err := store.SavePlan(plan); err != nil {
		t.Fatalf("failed to save plan: %v", err)
	}

	ctx := &cli.Context{Store: store}
	cmd := &NotifyCmd{DryRun: true}

	if err := cmd.Run(ctx); err != nil {
		t.Fatalf("notify run failed: %v", err)
	}

	retrievedPlan, err := store.GetPlan(plan.Date)
	if err != nil {
		t.Fatalf("failed to retrieve plan: %v", err)
	}

	if len(retrievedPlan.Slots) == 0 {
		t.Fatal("no slots in retrieved plan")
	}

	if retrievedPlan.Slots[0].LastNotifiedStart != nil {
		t.Error("expected no notification when notifications are disabled")
	}
}

func TestUpdateSlotNotificationTimestamp(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// Create a task
	task := models.Task{
		ID:          "task-timestamp-1",
		Name:        "Test Task",
		Kind:        constants.TaskKindFlexible,
		DurationMin: 30,
		Recurrence: models.Recurrence{
			Type: constants.RecurrenceDaily,
		},
		Priority: 1,
		Active:   true,
	}
	if err := store.AddTask(task); err != nil {
		t.Fatalf("failed to add task: %v", err)
	}

	// Create a plan
	nowStr := time.Now().UTC().Format(time.RFC3339)
	plan := models.DayPlan{
		Date:       "2024-03-01",
		Revision:   0,
		AcceptedAt: &nowStr,
		Slots: []models.Slot{
			{
				Start:  "09:00",
				End:    "09:30",
				TaskID: task.ID,
				Status: constants.SlotStatusAccepted,
			},
		},
	}

	if err := store.SavePlan(plan); err != nil {
		t.Fatalf("failed to save plan: %v", err)
	}

	// Retrieve plan to get the actual revision assigned by SavePlan
	savedPlan, err := store.GetPlan("2024-03-01")
	if err != nil {
		t.Fatalf("failed to retrieve plan after save: %v", err)
	}

	// Update start notification timestamp
	timestamp := time.Now().Format(time.RFC3339)
	err = store.UpdateSlotNotificationTimestamp("2024-03-01", savedPlan.Revision, "09:00", task.ID, "start", timestamp)
	if err != nil {
		t.Fatalf("failed to update start notification timestamp: %v", err)
	}

	// Retrieve and verify
	retrievedPlan, err := store.GetPlan("2024-03-01")
	if err != nil {
		t.Fatalf("failed to retrieve plan: %v", err)
	}

	if len(retrievedPlan.Slots) == 0 {
		t.Fatal("no slots in retrieved plan")
	}

	slot := retrievedPlan.Slots[0]
	if slot.LastNotifiedStart == nil {
		t.Error("expected LastNotifiedStart to be set")
	}

	// Update end notification timestamp
	timestamp2 := time.Now().Format(time.RFC3339)
	err = store.UpdateSlotNotificationTimestamp("2024-03-01", retrievedPlan.Revision, "09:00", task.ID, "end", timestamp2)
	if err != nil {
		t.Fatalf("failed to update end notification timestamp: %v", err)
	}

	// Retrieve and verify
	retrievedPlan2, err := store.GetPlan("2024-03-01")
	if err != nil {
		t.Fatalf("failed to retrieve plan: %v", err)
	}

	slot2 := retrievedPlan2.Slots[0]
	if slot2.LastNotifiedEnd == nil {
		t.Error("expected LastNotifiedEnd to be set")
	}

	// Verify both are set
	if slot2.LastNotifiedStart == nil {
		t.Error("expected LastNotifiedStart to still be set")
	}
}

func TestIsDatabaseBusyError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "database is locked",
			err:      fmt.Errorf("database is locked"),
			expected: true,
		},
		{
			name:     "database busy",
			err:      fmt.Errorf("database busy"),
			expected: true,
		},
		{
			name:     "other error",
			err:      fmt.Errorf("some other error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isDatabaseBusyError(tt.err)
			if result != tt.expected {
				t.Errorf("isDatabaseBusyError(%v) = %v, expected %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestNotifyCmd_BothStartAndEndNotifications(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// Create a task
	task := models.Task{
		ID:          "task-both-1",
		Name:        "Test Both Notifications",
		Kind:        constants.TaskKindFlexible,
		DurationMin: 30,
		Recurrence: models.Recurrence{
			Type: constants.RecurrenceDaily,
		},
		Priority: 1,
		Active:   true,
	}
	if err := store.AddTask(task); err != nil {
		t.Fatalf("failed to add task: %v", err)
	}

	now := time.Now()
	currentMinutes := now.Hour()*60 + now.Minute()

	// Skip test if running near midnight to avoid negative time calculations
	if currentMinutes < 40 {
		t.Skip("Skipping test when running too close to midnight (currentMinutes < 40)")
	}

	// Create a slot where both start and end should have triggered
	triggerMinutes := currentMinutes - 35 // Started 35 minutes ago
	startHour := triggerMinutes / 60
	startMin := triggerMinutes % 60
	startTime := fmt.Sprintf("%02d:%02d", startHour, startMin)

	endTriggerMinutes := currentMinutes - 2 // Ends 2 minutes ago (within grace period)
	endHour := endTriggerMinutes / 60
	endMin := endTriggerMinutes % 60
	endTime := fmt.Sprintf("%02d:%02d", endHour, endMin)

	nowStr := time.Now().UTC().Format(time.RFC3339)
	plan := models.DayPlan{
		Date:       now.Format("2006-01-02"),
		Revision:   0,
		AcceptedAt: &nowStr,
		Slots: []models.Slot{
			{
				Start:  startTime,
				End:    endTime,
				TaskID: task.ID,
				Status: constants.SlotStatusAccepted,
			},
		},
	}

	if err := store.SavePlan(plan); err != nil {
		t.Fatalf("failed to save plan: %v", err)
	}

	ctx := &cli.Context{Store: store}
	cmd := &NotifyCmd{DryRun: true}

	if err := cmd.Run(ctx); err != nil {
		t.Fatalf("notify run failed: %v", err)
	}

	retrievedPlan, err := store.GetPlan(plan.Date)
	if err != nil {
		t.Fatalf("failed to retrieve plan: %v", err)
	}

	if len(retrievedPlan.Slots) == 0 {
		t.Fatal("no slots in retrieved plan")
	}

	slot := retrievedPlan.Slots[0]

	// Start notification should have been skipped (outside grace period)
	if slot.LastNotifiedStart != nil {
		t.Error("expected start notification to be skipped (outside grace period)")
	}

	// End notification should have been sent (within grace period)
	if slot.LastNotifiedEnd == nil {
		t.Error("expected end notification to be sent (within grace period)")
	}
}

func TestNotifyCmd_OnlyAcceptedOrDoneSlots(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// Create tasks
	task1 := models.Task{
		ID:          "task-status-1",
		Name:        "Accepted Task",
		Kind:        constants.TaskKindFlexible,
		DurationMin: 30,
		Recurrence: models.Recurrence{
			Type: constants.RecurrenceDaily,
		},
		Priority: 1,
		Active:   true,
	}
	task2 := models.Task{
		ID:          "task-status-2",
		Name:        "Planned Task",
		Kind:        constants.TaskKindFlexible,
		DurationMin: 30,
		Recurrence: models.Recurrence{
			Type: constants.RecurrenceDaily,
		},
		Priority: 1,
		Active:   true,
	}

	if err := store.AddTask(task1); err != nil {
		t.Fatalf("failed to add task1: %v", err)
	}
	if err := store.AddTask(task2); err != nil {
		t.Fatalf("failed to add task2: %v", err)
	}

	now := time.Now()
	currentMinutes := now.Hour()*60 + now.Minute()

	// Set start time to now. With 5 min offset, notification should have happened 5 mins ago.
	// This is within the 10 min grace period.
	startMinutes := currentMinutes
	startHour := startMinutes / 60
	startMin := startMinutes % 60
	startTime := fmt.Sprintf("%02d:%02d", startHour, startMin)
	endTime := calculateEndTime(startMinutes, 30)

	nowStr := time.Now().UTC().Format(time.RFC3339)
	plan := models.DayPlan{
		Date:       now.Format("2006-01-02"),
		Revision:   0,
		AcceptedAt: &nowStr,
		Slots: []models.Slot{
			{
				Start:  startTime,
				End:    endTime,
				TaskID: task1.ID,
				Status: constants.SlotStatusAccepted,
			},
			{
				Start:  startTime,
				End:    endTime,
				TaskID: task2.ID,
				Status: constants.SlotStatusPlanned,
			},
		},
	}

	if err := store.SavePlan(plan); err != nil {
		t.Fatalf("failed to save plan: %v", err)
	}

	ctx := &cli.Context{Store: store}
	cmd := &NotifyCmd{DryRun: true}

	if err := cmd.Run(ctx); err != nil {
		t.Fatalf("notify run failed: %v", err)
	}

	retrievedPlan, err := store.GetPlan(plan.Date)
	if err != nil {
		t.Fatalf("failed to retrieve plan: %v", err)
	}

	if len(retrievedPlan.Slots) != 2 {
		t.Fatalf("expected 2 slots, got %d", len(retrievedPlan.Slots))
	}

	// Check that only the accepted slot was notified
	acceptedSlot := retrievedPlan.Slots[0]
	plannedSlot := retrievedPlan.Slots[1]

	if acceptedSlot.LastNotifiedStart == nil {
		t.Error("expected accepted slot to be notified")
	}

	if plannedSlot.LastNotifiedStart != nil {
		t.Error("expected planned slot to not be notified")
	}
}

func TestNotifyCmd_Alerts_GracePeriod(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// Set notification grace period
	settings, _ := store.GetSettings()
	settings.NotificationsEnabled = true
	settings.NotificationGracePeriodMin = 5
	store.SaveSettings(settings)

	// Create an alert for 10:00
	alert := models.Alert{
		ID:      "alert-1",
		Message: "Test alert",
		Time:    "10:00",
		Recurrence: models.Recurrence{
			Type: constants.RecurrenceDaily,
		},
		Active:    true,
		CreatedAt: time.Now(),
	}
	store.AddAlert(alert)

	ctx := &cli.Context{Store: store}
	cmd := &NotifyCmd{DryRun: true}

	// Test within grace period (10:03)
	now := time.Date(2026, 1, 5, 10, 3, 0, 0, time.UTC)
	err := cmd.checkAndSendAlerts(ctx, now, nil)
	if err != nil {
		t.Fatalf("checkAndSendAlerts failed: %v", err)
	}

	// Verify alert was marked as sent
	updated, _ := store.GetAlert("alert-1")
	if updated.LastSent == nil {
		t.Error("expected alert to be marked as sent")
	}

	// Test beyond grace period (10:10) - should not fire
	alert2 := models.Alert{
		ID:      "alert-2",
		Message: "Test alert 2",
		Time:    "10:00",
		Recurrence: models.Recurrence{
			Type: constants.RecurrenceDaily,
		},
		Active:    true,
		CreatedAt: time.Now(),
	}
	store.AddAlert(alert2)

	now2 := time.Date(2026, 1, 5, 10, 10, 0, 0, time.UTC)
	err = cmd.checkAndSendAlerts(ctx, now2, nil)
	if err != nil {
		t.Fatalf("checkAndSendAlerts failed: %v", err)
	}

	// Verify alert was NOT marked as sent (beyond grace period)
	updated2, _ := store.GetAlert("alert-2")
	if updated2.LastSent != nil {
		t.Error("expected alert not to fire beyond grace period")
	}
}

func TestNotifyCmd_Alerts_DuplicatePrevention(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	settings, _ := store.GetSettings()
	settings.NotificationsEnabled = true
	settings.NotificationGracePeriodMin = 10
	store.SaveSettings(settings)

	// Create an alert
	alert := models.Alert{
		ID:      "alert-dup",
		Message: "Duplicate test",
		Time:    "10:00",
		Recurrence: models.Recurrence{
			Type: constants.RecurrenceDaily,
		},
		Active:    true,
		CreatedAt: time.Now(),
	}
	store.AddAlert(alert)

	ctx := &cli.Context{Store: store}
	cmd := &NotifyCmd{DryRun: true}

	// First notification
	now := time.Date(2026, 1, 5, 10, 0, 0, 0, time.UTC)
	err := cmd.checkAndSendAlerts(ctx, now, nil)
	if err != nil {
		t.Fatalf("checkAndSendAlerts failed: %v", err)
	}

	// Verify alert was sent
	updated, _ := store.GetAlert("alert-dup")
	if updated.LastSent == nil {
		t.Error("expected alert to be sent")
	}
	firstSent := *updated.LastSent

	// Try to send again on the same day
	now2 := time.Date(2026, 1, 5, 10, 5, 0, 0, time.UTC)
	err = cmd.checkAndSendAlerts(ctx, now2, nil)
	if err != nil {
		t.Fatalf("checkAndSendAlerts failed: %v", err)
	}

	// Verify LastSent didn't change (duplicate prevention)
	updated2, _ := store.GetAlert("alert-dup")
	if updated2.LastSent == nil {
		t.Fatal("expected LastSent to be set")
	}
	if !updated2.LastSent.Equal(firstSent) {
		t.Error("expected LastSent to remain unchanged (duplicate prevention)")
	}
}

func TestNotifyCmd_Alerts_OneTimeDeactivation(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	settings, _ := store.GetSettings()
	settings.NotificationsEnabled = true
	settings.NotificationGracePeriodMin = 10
	store.SaveSettings(settings)

	// Create a one-time alert for the test date
	testDate := "2026-01-05"
	alert := models.Alert{
		ID:        "alert-onetime",
		Message:   "One-time alert",
		Time:      "10:00",
		Date:      testDate,
		Active:    true,
		CreatedAt: time.Now(),
	}
	store.AddAlert(alert)

	ctx := &cli.Context{Store: store}
	cmd := &NotifyCmd{DryRun: true}

	// Send the alert
	now := time.Date(2026, 1, 5, 10, 0, 0, 0, time.UTC)
	err := cmd.checkAndSendAlerts(ctx, now, nil)
	if err != nil {
		t.Fatalf("checkAndSendAlerts failed: %v", err)
	}

	// Verify alert was deactivated
	updated, _ := store.GetAlert("alert-onetime")
	if updated.Active {
		t.Error("expected one-time alert to be deactivated after firing")
	}
	if updated.LastSent == nil {
		t.Error("expected LastSent to be set")
	}
}

func TestNotifyCmd_Alerts_WeeklyRecurrence(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	settings, _ := store.GetSettings()
	settings.NotificationsEnabled = true
	settings.NotificationGracePeriodMin = 10
	store.SaveSettings(settings)

	// Create a weekly alert for Monday and Friday
	alert := models.Alert{
		ID:      "alert-weekly",
		Message: "Weekly alert",
		Time:    "10:00",
		Recurrence: models.Recurrence{
			Type:        constants.RecurrenceWeekly,
			WeekdayMask: []time.Weekday{time.Monday, time.Friday},
		},
		Active:    true,
		CreatedAt: time.Now(),
	}
	store.AddAlert(alert)

	ctx := &cli.Context{Store: store}
	cmd := &NotifyCmd{DryRun: true}

	// Test on Monday (2026-01-05 is a Monday)
	monday := time.Date(2026, 1, 5, 10, 0, 0, 0, time.UTC)
	err := cmd.checkAndSendAlerts(ctx, monday, nil)
	if err != nil {
		t.Fatalf("checkAndSendAlerts failed: %v", err)
	}

	// Verify alert was sent on Monday
	updated, _ := store.GetAlert("alert-weekly")
	if updated.LastSent == nil {
		t.Error("expected alert to fire on Monday")
	}

	// Reset for next test
	alert.LastSent = nil
	store.UpdateAlert(alert)

	// Test on Tuesday (should not fire)
	tuesday := time.Date(2026, 1, 6, 10, 0, 0, 0, time.UTC)
	err = cmd.checkAndSendAlerts(ctx, tuesday, nil)
	if err != nil {
		t.Fatalf("checkAndSendAlerts failed: %v", err)
	}

	updated2, _ := store.GetAlert("alert-weekly")
	if updated2.LastSent != nil {
		t.Error("expected alert not to fire on Tuesday")
	}
}

func TestNotifyCmd_Alerts_NDaysRecurrence(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	settings, _ := store.GetSettings()
	settings.NotificationsEnabled = true
	settings.NotificationGracePeriodMin = 10
	store.SaveSettings(settings)

	// Create an alert for every 3 days
	createdAt := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	alert := models.Alert{
		ID:      "alert-ndays",
		Message: "Every 3 days",
		Time:    "10:00",
		Recurrence: models.Recurrence{
			Type:         constants.RecurrenceNDays,
			IntervalDays: 3,
		},
		Active:    true,
		CreatedAt: createdAt,
	}
	store.AddAlert(alert)

	ctx := &cli.Context{Store: store}
	cmd := &NotifyCmd{DryRun: true}

	// Test on day 1 (should fire)
	day1 := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	err := cmd.checkAndSendAlerts(ctx, day1, nil)
	if err != nil {
		t.Fatalf("checkAndSendAlerts failed: %v", err)
	}

	updated, _ := store.GetAlert("alert-ndays")
	if updated.LastSent == nil {
		t.Error("expected alert to fire on day 1")
	}

	// Test on day 2 (should not fire)
	day2 := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)
	err = cmd.checkAndSendAlerts(ctx, day2, nil)
	if err != nil {
		t.Fatalf("checkAndSendAlerts failed: %v", err)
	}

	// Test on day 4 (3 days after day 1, should fire)
	day4 := time.Date(2026, 1, 4, 10, 0, 0, 0, time.UTC)
	err = cmd.checkAndSendAlerts(ctx, day4, nil)
	if err != nil {
		t.Fatalf("checkAndSendAlerts failed: %v", err)
	}

	updated2, _ := store.GetAlert("alert-ndays")
	day4Date := day4.Format("2006-01-02")
	if updated2.LastSent != nil {
		lastSentDate := updated2.LastSent.Format("2006-01-02")
		if lastSentDate != day4Date {
			t.Errorf("expected alert to fire on day 4, last sent: %s", lastSentDate)
		}
	}
}

func TestNotifyCmd_Alerts_InactiveSkipped(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	settings, _ := store.GetSettings()
	settings.NotificationsEnabled = true
	settings.NotificationGracePeriodMin = 10
	store.SaveSettings(settings)

	// Create an inactive alert
	alert := models.Alert{
		ID:      "alert-inactive",
		Message: "Inactive alert",
		Time:    "10:00",
		Recurrence: models.Recurrence{
			Type: constants.RecurrenceDaily,
		},
		Active:    false,
		CreatedAt: time.Now(),
	}
	store.AddAlert(alert)

	ctx := &cli.Context{Store: store}
	cmd := &NotifyCmd{DryRun: true}

	// Try to send
	now := time.Date(2026, 1, 5, 10, 0, 0, 0, time.UTC)
	err := cmd.checkAndSendAlerts(ctx, now, nil)
	if err != nil {
		t.Fatalf("checkAndSendAlerts failed: %v", err)
	}

	// Verify alert was not sent (inactive)
	updated, _ := store.GetAlert("alert-inactive")
	if updated.LastSent != nil {
		t.Error("expected inactive alert not to fire")
	}
}
