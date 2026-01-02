package storage

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/julianstephens/daylit/daylit-cli/internal/models"
)

func TestAlertCRUD(t *testing.T) {
	store, cleanup := setupTestSQLiteStore(t)
	defer cleanup()

	// Create an alert
	alert := models.Alert{
		ID:      uuid.New().String(),
		Message: "Test alert",
		Time:    "10:00",
		Recurrence: models.Recurrence{
			Type: models.RecurrenceDaily,
		},
		Active:    true,
		CreatedAt: time.Now(),
	}

	// Add alert
	if err := store.AddAlert(alert); err != nil {
		t.Fatalf("failed to add alert: %v", err)
	}

	// Get alert by ID
	retrieved, err := store.GetAlert(alert.ID)
	if err != nil {
		t.Fatalf("failed to get alert: %v", err)
	}
	if retrieved.Message != alert.Message {
		t.Errorf("expected message %q, got %q", alert.Message, retrieved.Message)
	}
	if retrieved.Time != alert.Time {
		t.Errorf("expected time %q, got %q", alert.Time, retrieved.Time)
	}
	if !retrieved.Active {
		t.Errorf("expected active=true, got false")
	}

	// Update alert
	alert.Message = "Updated alert"
	alert.Active = false
	if err := store.UpdateAlert(alert); err != nil {
		t.Fatalf("failed to update alert: %v", err)
	}

	updated, err := store.GetAlert(alert.ID)
	if err != nil {
		t.Fatalf("failed to get updated alert: %v", err)
	}
	if updated.Message != "Updated alert" {
		t.Errorf("expected message %q, got %q", "Updated alert", updated.Message)
	}
	if updated.Active {
		t.Errorf("expected active=false, got true")
	}

	// Delete alert
	if err := store.DeleteAlert(alert.ID); err != nil {
		t.Fatalf("failed to delete alert: %v", err)
	}

	// Verify deletion
	_, err = store.GetAlert(alert.ID)
	if err == nil {
		t.Error("expected error when getting deleted alert, got nil")
	}
}

func TestAlertGetAll(t *testing.T) {
	store, cleanup := setupTestSQLiteStore(t)
	defer cleanup()

	// Add multiple alerts
	alerts := []models.Alert{
		{
			ID:      uuid.New().String(),
			Message: "Alert 1",
			Time:    "08:00",
			Recurrence: models.Recurrence{
				Type: models.RecurrenceDaily,
			},
			Active:    true,
			CreatedAt: time.Now(),
		},
		{
			ID:        uuid.New().String(),
			Message:   "Alert 2",
			Time:      "14:00",
			Date:      "2026-01-15",
			Active:    true,
			CreatedAt: time.Now(),
		},
		{
			ID:      uuid.New().String(),
			Message: "Alert 3",
			Time:    "18:00",
			Recurrence: models.Recurrence{
				Type:        models.RecurrenceWeekly,
				WeekdayMask: []time.Weekday{time.Monday, time.Friday},
			},
			Active:    false,
			CreatedAt: time.Now(),
		},
	}

	for _, alert := range alerts {
		if err := store.AddAlert(alert); err != nil {
			t.Fatalf("failed to add alert: %v", err)
		}
	}

	// Get all alerts
	allAlerts, err := store.GetAllAlerts()
	if err != nil {
		t.Fatalf("failed to get all alerts: %v", err)
	}

	if len(allAlerts) != len(alerts) {
		t.Errorf("expected %d alerts, got %d", len(alerts), len(allAlerts))
	}

	// Verify alerts are sorted by time
	if len(allAlerts) >= 2 {
		if allAlerts[0].Time > allAlerts[1].Time {
			t.Error("expected alerts to be sorted by time")
		}
	}
}

func TestAlertValidation(t *testing.T) {
	store, cleanup := setupTestSQLiteStore(t)
	defer cleanup()

	tests := []struct {
		name    string
		alert   models.Alert
		wantErr bool
	}{
		{
			name: "valid daily alert",
			alert: models.Alert{
				ID:      uuid.New().String(),
				Message: "Valid alert",
				Time:    "10:00",
				Recurrence: models.Recurrence{
					Type: models.RecurrenceDaily,
				},
				Active:    true,
				CreatedAt: time.Now(),
			},
			wantErr: false,
		},
		{
			name: "invalid time format",
			alert: models.Alert{
				ID:        uuid.New().String(),
				Message:   "Invalid time",
				Time:      "25:00",
				Active:    true,
				CreatedAt: time.Now(),
			},
			wantErr: true,
		},
		{
			name: "empty message",
			alert: models.Alert{
				ID:        uuid.New().String(),
				Message:   "",
				Time:      "10:00",
				Active:    true,
				CreatedAt: time.Now(),
			},
			wantErr: true,
		},
		{
			name: "weekly without weekdays",
			alert: models.Alert{
				ID:      uuid.New().String(),
				Message: "Weekly alert",
				Time:    "10:00",
				Recurrence: models.Recurrence{
					Type:        models.RecurrenceWeekly,
					WeekdayMask: []time.Weekday{},
				},
				Active:    true,
				CreatedAt: time.Now(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := store.AddAlert(tt.alert)
			if (err != nil) != tt.wantErr {
				t.Errorf("AddAlert() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAlertRecurrence(t *testing.T) {
	store, cleanup := setupTestSQLiteStore(t)
	defer cleanup()

	// Test weekly alert with weekdays
	weeklyAlert := models.Alert{
		ID:      uuid.New().String(),
		Message: "Weekly alert",
		Time:    "10:00",
		Recurrence: models.Recurrence{
			Type:        models.RecurrenceWeekly,
			WeekdayMask: []time.Weekday{time.Monday, time.Wednesday, time.Friday},
		},
		Active:    true,
		CreatedAt: time.Now(),
	}

	if err := store.AddAlert(weeklyAlert); err != nil {
		t.Fatalf("failed to add weekly alert: %v", err)
	}

	retrieved, err := store.GetAlert(weeklyAlert.ID)
	if err != nil {
		t.Fatalf("failed to get weekly alert: %v", err)
	}

	if len(retrieved.Recurrence.WeekdayMask) != 3 {
		t.Errorf("expected 3 weekdays, got %d", len(retrieved.Recurrence.WeekdayMask))
	}

	// Test n_days alert with interval
	nDaysAlert := models.Alert{
		ID:      uuid.New().String(),
		Message: "Every 3 days",
		Time:    "14:00",
		Recurrence: models.Recurrence{
			Type:         models.RecurrenceNDays,
			IntervalDays: 3,
		},
		Active:    true,
		CreatedAt: time.Now(),
	}

	if err := store.AddAlert(nDaysAlert); err != nil {
		t.Fatalf("failed to add n_days alert: %v", err)
	}

	retrieved2, err := store.GetAlert(nDaysAlert.ID)
	if err != nil {
		t.Fatalf("failed to get n_days alert: %v", err)
	}

	if retrieved2.Recurrence.IntervalDays != 3 {
		t.Errorf("expected interval 3, got %d", retrieved2.Recurrence.IntervalDays)
	}
}

func TestAlertLastSent(t *testing.T) {
	store, cleanup := setupTestSQLiteStore(t)
	defer cleanup()

	alert := models.Alert{
		ID:      uuid.New().String(),
		Message: "Test alert",
		Time:    "10:00",
		Recurrence: models.Recurrence{
			Type: models.RecurrenceDaily,
		},
		Active:    true,
		CreatedAt: time.Now(),
	}

	if err := store.AddAlert(alert); err != nil {
		t.Fatalf("failed to add alert: %v", err)
	}

	// Initially LastSent should be nil
	retrieved, err := store.GetAlert(alert.ID)
	if err != nil {
		t.Fatalf("failed to get alert: %v", err)
	}
	if retrieved.LastSent != nil {
		t.Error("expected LastSent to be nil initially")
	}

	// Update with LastSent
	now := time.Now()
	alert.LastSent = &now
	if err := store.UpdateAlert(alert); err != nil {
		t.Fatalf("failed to update alert: %v", err)
	}

	updated, err := store.GetAlert(alert.ID)
	if err != nil {
		t.Fatalf("failed to get updated alert: %v", err)
	}
	if updated.LastSent == nil {
		t.Error("expected LastSent to be set")
	}
}
