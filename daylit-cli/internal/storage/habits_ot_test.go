package storage

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/julianstephens/daylit/daylit-cli/internal/models"
)

func TestHabitCRUD(t *testing.T) {
	store, cleanup := setupTestSQLiteStore(t)
	defer cleanup()

	// Create a habit
	habit := models.Habit{
		ID:        uuid.New().String(),
		Name:      "Morning meditation",
		CreatedAt: time.Now(),
	}

	// Add habit
	if err := store.AddHabit(habit); err != nil {
		t.Fatalf("failed to add habit: %v", err)
	}

	// Get habit by ID
	retrieved, err := store.GetHabit(habit.ID)
	if err != nil {
		t.Fatalf("failed to get habit: %v", err)
	}
	if retrieved.Name != habit.Name {
		t.Errorf("expected name %q, got %q", habit.Name, retrieved.Name)
	}

	// Get habit by name
	byName, err := store.GetHabitByName(habit.Name)
	if err != nil {
		t.Fatalf("failed to get habit by name: %v", err)
	}
	if byName.ID != habit.ID {
		t.Errorf("expected ID %q, got %q", habit.ID, byName.ID)
	}

	// Update habit
	habit.Name = "Updated meditation"
	if err := store.UpdateHabit(habit); err != nil {
		t.Fatalf("failed to update habit: %v", err)
	}

	// Verify update
	updated, err := store.GetHabit(habit.ID)
	if err != nil {
		t.Fatalf("failed to get updated habit: %v", err)
	}
	if updated.Name != "Updated meditation" {
		t.Errorf("expected updated name, got %q", updated.Name)
	}
}

func TestHabitArchive(t *testing.T) {
	store, cleanup := setupTestSQLiteStore(t)
	defer cleanup()

	habit := models.Habit{
		ID:        uuid.New().String(),
		Name:      "Test habit",
		CreatedAt: time.Now(),
	}

	if err := store.AddHabit(habit); err != nil {
		t.Fatalf("failed to add habit: %v", err)
	}

	// Archive habit
	if err := store.ArchiveHabit(habit.ID); err != nil {
		t.Fatalf("failed to archive habit: %v", err)
	}

	// Verify it's not in default list
	habits, err := store.GetAllHabits(false, false)
	if err != nil {
		t.Fatalf("failed to get habits: %v", err)
	}
	for _, h := range habits {
		if h.ID == habit.ID {
			t.Error("archived habit should not appear in default list")
		}
	}

	// Verify it's in archived list
	archived, err := store.GetAllHabits(true, false)
	if err != nil {
		t.Fatalf("failed to get archived habits: %v", err)
	}
	found := false
	for _, h := range archived {
		if h.ID == habit.ID && h.ArchivedAt != nil {
			found = true
			break
		}
	}
	if !found {
		t.Error("archived habit not found in archived list")
	}

	// Unarchive habit
	if err := store.UnarchiveHabit(habit.ID); err != nil {
		t.Fatalf("failed to unarchive habit: %v", err)
	}

	// Verify it's back in default list
	habits, err = store.GetAllHabits(false, false)
	if err != nil {
		t.Fatalf("failed to get habits after unarchive: %v", err)
	}
	found = false
	for _, h := range habits {
		if h.ID == habit.ID && h.ArchivedAt == nil {
			found = true
			break
		}
	}
	if !found {
		t.Error("unarchived habit not found in default list")
	}
}

func TestHabitSoftDelete(t *testing.T) {
	store, cleanup := setupTestSQLiteStore(t)
	defer cleanup()

	habit := models.Habit{
		ID:        uuid.New().String(),
		Name:      "Test delete",
		CreatedAt: time.Now(),
	}

	if err := store.AddHabit(habit); err != nil {
		t.Fatalf("failed to add habit: %v", err)
	}

	// Delete habit
	if err := store.DeleteHabit(habit.ID); err != nil {
		t.Fatalf("failed to delete habit: %v", err)
	}

	// Verify can't get it normally
	_, err := store.GetHabit(habit.ID)
	if err == nil {
		t.Error("expected error getting deleted habit")
	}

	// Restore habit
	if err := store.RestoreHabit(habit.ID); err != nil {
		t.Fatalf("failed to restore habit: %v", err)
	}

	// Verify we can get it again
	restored, err := store.GetHabit(habit.ID)
	if err != nil {
		t.Fatalf("failed to get restored habit: %v", err)
	}
	if restored.Name != habit.Name {
		t.Errorf("expected name %q, got %q", habit.Name, restored.Name)
	}
}

func TestHabitEntryCRUD(t *testing.T) {
	store, cleanup := setupTestSQLiteStore(t)
	defer cleanup()

	// Create a habit first
	habit := models.Habit{
		ID:        uuid.New().String(),
		Name:      "Test habit",
		CreatedAt: time.Now(),
	}
	if err := store.AddHabit(habit); err != nil {
		t.Fatalf("failed to add habit: %v", err)
	}

	// Create entry
	entry := models.HabitEntry{
		ID:        uuid.New().String(),
		HabitID:   habit.ID,
		Day:       "2025-12-31",
		Note:      "Morning session",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := store.AddHabitEntry(entry); err != nil {
		t.Fatalf("failed to add habit entry: %v", err)
	}

	// Get entry
	retrieved, err := store.GetHabitEntry(habit.ID, "2025-12-31")
	if err != nil {
		t.Fatalf("failed to get habit entry: %v", err)
	}
	if retrieved.Note != "Morning session" {
		t.Errorf("expected note %q, got %q", "Morning session", retrieved.Note)
	}

	// Get entries for day
	dayEntries, err := store.GetHabitEntriesForDay("2025-12-31")
	if err != nil {
		t.Fatalf("failed to get entries for day: %v", err)
	}
	if len(dayEntries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(dayEntries))
	}

	// Get entries for habit
	habitEntries, err := store.GetHabitEntriesForHabit(habit.ID, "2025-12-01", "2025-12-31")
	if err != nil {
		t.Fatalf("failed to get entries for habit: %v", err)
	}
	if len(habitEntries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(habitEntries))
	}
}

func TestHabitEntryUniqueness(t *testing.T) {
	store, cleanup := setupTestSQLiteStore(t)
	defer cleanup()

	habit := models.Habit{
		ID:        uuid.New().String(),
		Name:      "Test habit",
		CreatedAt: time.Now(),
	}
	if err := store.AddHabit(habit); err != nil {
		t.Fatalf("failed to add habit: %v", err)
	}

	entry1 := models.HabitEntry{
		ID:        uuid.New().String(),
		HabitID:   habit.ID,
		Day:       "2025-12-31",
		Note:      "First entry",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := store.AddHabitEntry(entry1); err != nil {
		t.Fatalf("failed to add first entry: %v", err)
	}

	// Try to add another entry with same habit+day (should update, not error)
	entry2 := models.HabitEntry{
		ID:        entry1.ID, // Same ID to update
		HabitID:   habit.ID,
		Day:       "2025-12-31",
		Note:      "Updated entry",
		CreatedAt: entry1.CreatedAt,
		UpdatedAt: time.Now(),
	}

	if err := store.UpdateHabitEntry(entry2); err != nil {
		t.Fatalf("failed to update entry: %v", err)
	}

	// Verify there's still only one entry
	retrieved, err := store.GetHabitEntry(habit.ID, "2025-12-31")
	if err != nil {
		t.Fatalf("failed to get entry: %v", err)
	}
	if retrieved.Note != "Updated entry" {
		t.Errorf("expected updated note, got %q", retrieved.Note)
	}
}

func TestOTSettingsCRUD(t *testing.T) {
	store, cleanup := setupTestSQLiteStore(t)
	defer cleanup()

	// Get initial settings (should exist from migration)
	settings, err := store.GetOTSettings()
	if err != nil {
		t.Fatalf("failed to get OT settings: %v", err)
	}

	if settings.ID != 1 {
		t.Errorf("expected ID 1, got %d", settings.ID)
	}
	if !settings.PromptOnEmpty {
		t.Error("expected PromptOnEmpty to be true")
	}
	if !settings.StrictMode {
		t.Error("expected StrictMode to be true")
	}
	if settings.DefaultLogDays != 14 {
		t.Errorf("expected DefaultLogDays 14, got %d", settings.DefaultLogDays)
	}

	// Update settings
	settings.PromptOnEmpty = false
	settings.DefaultLogDays = 30
	if err := store.SaveOTSettings(settings); err != nil {
		t.Fatalf("failed to save OT settings: %v", err)
	}

	// Verify update
	updated, err := store.GetOTSettings()
	if err != nil {
		t.Fatalf("failed to get updated settings: %v", err)
	}
	if updated.PromptOnEmpty {
		t.Error("expected PromptOnEmpty to be false")
	}
	if updated.DefaultLogDays != 30 {
		t.Errorf("expected DefaultLogDays 30, got %d", updated.DefaultLogDays)
	}
}

func TestOTEntryCRUD(t *testing.T) {
	store, cleanup := setupTestSQLiteStore(t)
	defer cleanup()

	entry := models.OTEntry{
		ID:        uuid.New().String(),
		Day:       "2025-12-31",
		Title:     "Complete feature",
		Note:      "Focus on habits and OT",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Add entry
	if err := store.AddOTEntry(entry); err != nil {
		t.Fatalf("failed to add OT entry: %v", err)
	}

	// Get entry
	retrieved, err := store.GetOTEntry("2025-12-31")
	if err != nil {
		t.Fatalf("failed to get OT entry: %v", err)
	}
	if retrieved.Title != "Complete feature" {
		t.Errorf("expected title %q, got %q", "Complete feature", retrieved.Title)
	}

	// Update entry
	entry.Title = "Updated feature"
	entry.UpdatedAt = time.Now()
	if err := store.UpdateOTEntry(entry); err != nil {
		t.Fatalf("failed to update OT entry: %v", err)
	}

	// Verify update
	updated, err := store.GetOTEntry("2025-12-31")
	if err != nil {
		t.Fatalf("failed to get updated entry: %v", err)
	}
	if updated.Title != "Updated feature" {
		t.Errorf("expected updated title, got %q", updated.Title)
	}
}

func TestOTEntrySoftDelete(t *testing.T) {
	store, cleanup := setupTestSQLiteStore(t)
	defer cleanup()

	entry := models.OTEntry{
		ID:        uuid.New().String(),
		Day:       "2025-12-30",
		Title:     "Test delete",
		Note:      "",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := store.AddOTEntry(entry); err != nil {
		t.Fatalf("failed to add OT entry: %v", err)
	}

	// Delete entry
	if err := store.DeleteOTEntry("2025-12-30"); err != nil {
		t.Fatalf("failed to delete OT entry: %v", err)
	}

	// Verify can't get it normally
	_, err := store.GetOTEntry("2025-12-30")
	if err == nil {
		t.Error("expected error getting deleted OT entry")
	}

	// Restore entry
	if err := store.RestoreOTEntry("2025-12-30"); err != nil {
		t.Fatalf("failed to restore OT entry: %v", err)
	}

	// Verify we can get it again
	restored, err := store.GetOTEntry("2025-12-30")
	if err != nil {
		t.Fatalf("failed to get restored entry: %v", err)
	}
	if restored.Title != "Test delete" {
		t.Errorf("expected title %q, got %q", "Test delete", restored.Title)
	}
}

func TestOTEntriesDateRange(t *testing.T) {
	store, cleanup := setupTestSQLiteStore(t)
	defer cleanup()

	// Create multiple entries
	dates := []string{"2025-12-28", "2025-12-29", "2025-12-30"}
	for i, day := range dates {
		entry := models.OTEntry{
			ID:        uuid.New().String(),
			Day:       day,
			Title:     fmt.Sprintf("Entry %d", i+1),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := store.AddOTEntry(entry); err != nil {
			t.Fatalf("failed to add OT entry for %s: %v", day, err)
		}
	}

	// Get entries in range
	entries, err := store.GetOTEntries("2025-12-28", "2025-12-30", false)
	if err != nil {
		t.Fatalf("failed to get OT entries: %v", err)
	}

	if len(entries) != 3 {
		t.Errorf("expected 3 entries, got %d", len(entries))
	}

	// Verify they're in descending order
	if entries[0].Day != "2025-12-30" {
		t.Errorf("expected first entry to be 2025-12-30, got %s", entries[0].Day)
	}
}

func TestOTEntryUniqueness(t *testing.T) {
	store, cleanup := setupTestSQLiteStore(t)
	defer cleanup()

	entry1 := models.OTEntry{
		ID:        uuid.New().String(),
		Day:       "2025-12-31",
		Title:     "First title",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := store.AddOTEntry(entry1); err != nil {
		t.Fatalf("failed to add first entry: %v", err)
	}

	// Update with same day (should replace)
	entry2 := models.OTEntry{
		ID:        entry1.ID,
		Day:       "2025-12-31",
		Title:     "Updated title",
		CreatedAt: entry1.CreatedAt,
		UpdatedAt: time.Now(),
	}

	if err := store.UpdateOTEntry(entry2); err != nil {
		t.Fatalf("failed to update entry: %v", err)
	}

	// Verify there's still only one entry
	retrieved, err := store.GetOTEntry("2025-12-31")
	if err != nil {
		t.Fatalf("failed to get entry: %v", err)
	}
	if retrieved.Title != "Updated title" {
		t.Errorf("expected updated title, got %q", retrieved.Title)
	}
}
