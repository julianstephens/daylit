package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/julianstephens/daylit/daylit-cli/internal/migration"
	"github.com/julianstephens/daylit/daylit-cli/internal/models"
	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	path string
	db   *sql.DB
}

func NewSQLiteStore(path string) *SQLiteStore {
	return &SQLiteStore{
		path: path,
	}
}

func (s *SQLiteStore) Init() error {
	// Create config directory if it doesn't exist
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Open database
	db, err := sql.Open("sqlite", s.path)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	s.db = db

	// Run migrations
	if err := s.runMigrations(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Initialize default settings if not present
	if _, err := s.GetSettings(); err != nil {
		defaultSettings := Settings{
			DayStart:                   "07:00",
			DayEnd:                     "22:00",
			DefaultBlockMin:            30,
			NotificationsEnabled:       true,
			NotifyBlockStart:           true,
			NotifyBlockEnd:             true,
			BlockStartOffsetMin:        5,
			BlockEndOffsetMin:          5,
			NotificationGracePeriodMin: 10,
		}
		if err := s.SaveSettings(defaultSettings); err != nil {
			return fmt.Errorf("failed to save default settings: %w", err)
		}
	}

	return nil
}

func (s *SQLiteStore) Load() error {
	if s.db != nil {
		return nil
	}

	if _, err := os.Stat(s.path); os.IsNotExist(err) {
		return fmt.Errorf("storage not initialized, run 'daylit init' first")
	}

	db, err := sql.Open("sqlite", s.path)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	s.db = db

	// Validate schema version if migrations directory is available.
	// If the migrations directory is missing (e.g. not shipped in production),
	// skip validation but still allow using the already-initialized database.
	migrationsPath := s.getMigrationsPath()
	if infoErr := func() error {
		_, statErr := os.Stat(migrationsPath)
		if statErr == nil {
			return nil
		}
		if os.IsNotExist(statErr) {
			// Migrations directory is not present; skip schema validation.
			return nil
		}
		// Any other error accessing the directory is treated as fatal.
		return fmt.Errorf("failed to access migrations directory %q: %w", migrationsPath, statErr)
	}(); infoErr != nil {
		return infoErr
	} else if migrationsPath != "" {
		// Only attempt validation when migrations are accessible.
		if err := s.validateSchemaVersion(); err != nil {
			return err
		}
	}

	return nil
}

func (s *SQLiteStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *SQLiteStore) runMigrations() error {
	// Get the migrations directory path (relative to the binary or in the repository)
	migrationsPath := s.getMigrationsPath()

	// Create migration runner
	runner := migration.NewRunner(s.db, migrationsPath)

	// Apply all pending migrations
	_, err := runner.ApplyMigrations(func(msg string) {
		fmt.Println(msg)
	})
	return err
}

func (s *SQLiteStore) validateSchemaVersion() error {
	migrationsPath := s.getMigrationsPath()
	runner := migration.NewRunner(s.db, migrationsPath)
	return runner.ValidateVersion()
}

func (s *SQLiteStore) getMigrationsPath() string {
	// Check if environment variable is set
	if envPath := os.Getenv("DAYLIT_MIGRATIONS_PATH"); envPath != "" {
		if absPath, err := filepath.Abs(envPath); err == nil {
			if _, err := os.Stat(absPath); err == nil {
				return absPath
			}
		}
	}

	// Try to find migrations directory relative to the executable or in common paths
	paths := []string{
		"migrations/sqlite",
		"./migrations/sqlite",
		"../migrations/sqlite",
		"../../migrations/sqlite",
		"../../../migrations/sqlite",
		"../../../../migrations/sqlite",
		filepath.Join(filepath.Dir(os.Args[0]), "migrations", "sqlite"),
		filepath.Join(filepath.Dir(os.Args[0]), "..", "migrations", "sqlite"),
	}

	for _, path := range paths {
		if absPath, err := filepath.Abs(path); err == nil {
			if _, err := os.Stat(absPath); err == nil {
				return absPath
			}
		}
	}

	// Default to "migrations/sqlite" in current directory (will fail gracefully if not found)
	return "migrations/sqlite"
}

func (s *SQLiteStore) GetSettings() (Settings, error) {
	rows, err := s.db.Query("SELECT key, value FROM settings")
	if err != nil {
		return Settings{}, err
	}
	defer rows.Close()

	settings := Settings{}
	count := 0
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return Settings{}, err
		}
		switch key {
		case "day_start":
			settings.DayStart = value
		case "day_end":
			settings.DayEnd = value
		case "default_block_min":
			if _, err := fmt.Sscanf(value, "%d", &settings.DefaultBlockMin); err != nil {
				return Settings{}, fmt.Errorf("parsing default_block_min: %w", err)
			}
		case "notifications_enabled":
			settings.NotificationsEnabled = value == "true"
		case "notify_block_start":
			settings.NotifyBlockStart = value == "true"
		case "notify_block_end":
			settings.NotifyBlockEnd = value == "true"
		case "block_start_offset_min":
			if _, err := fmt.Sscanf(value, "%d", &settings.BlockStartOffsetMin); err != nil {
				return Settings{}, fmt.Errorf("parsing block_start_offset_min: %w", err)
			}
		case "block_end_offset_min":
			if _, err := fmt.Sscanf(value, "%d", &settings.BlockEndOffsetMin); err != nil {
				return Settings{}, fmt.Errorf("parsing block_end_offset_min: %w", err)
			}
		case "notification_grace_period_min":
			if _, err := fmt.Sscanf(value, "%d", &settings.NotificationGracePeriodMin); err != nil {
				return Settings{}, fmt.Errorf("parsing notification_grace_period_min: %w", err)
			}
		}
		count++
	}

	if count == 0 {
		return Settings{}, fmt.Errorf("settings not found")
	}

	return settings, nil
}

func (s *SQLiteStore) SaveSettings(settings Settings) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	if _, err := stmt.Exec("day_start", settings.DayStart); err != nil {
		return err
	}
	if _, err := stmt.Exec("day_end", settings.DayEnd); err != nil {
		return err
	}
	if _, err := stmt.Exec("default_block_min", fmt.Sprintf("%d", settings.DefaultBlockMin)); err != nil {
		return err
	}
	if _, err := stmt.Exec("notifications_enabled", fmt.Sprintf("%v", settings.NotificationsEnabled)); err != nil {
		return err
	}
	if _, err := stmt.Exec("notify_block_start", fmt.Sprintf("%v", settings.NotifyBlockStart)); err != nil {
		return err
	}
	if _, err := stmt.Exec("notify_block_end", fmt.Sprintf("%v", settings.NotifyBlockEnd)); err != nil {
		return err
	}
	if _, err := stmt.Exec("block_start_offset_min", fmt.Sprintf("%d", settings.BlockStartOffsetMin)); err != nil {
		return err
	}
	if _, err := stmt.Exec("block_end_offset_min", fmt.Sprintf("%d", settings.BlockEndOffsetMin)); err != nil {
		return err
	}
	if _, err := stmt.Exec("notification_grace_period_min", fmt.Sprintf("%d", settings.NotificationGracePeriodMin)); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *SQLiteStore) AddTask(task models.Task) error {
	return s.UpdateTask(task)
}

func (s *SQLiteStore) GetTask(id string) (models.Task, error) {
	row := s.db.QueryRow(`
		SELECT id, name, kind, duration_min, earliest_start, latest_end, fixed_start, fixed_end,
		       recurrence_type, recurrence_interval, recurrence_weekdays, priority, energy_band,
		       active, last_done, success_streak, avg_actual_duration, deleted_at
		FROM tasks WHERE id = ? AND deleted_at IS NULL`, id)

	var t models.Task
	var recType, recWeekdays, energyBand string
	var active bool
	var deletedAt sql.NullString

	err := row.Scan(
		&t.ID, &t.Name, &t.Kind, &t.DurationMin, &t.EarliestStart, &t.LatestEnd, &t.FixedStart, &t.FixedEnd,
		&recType, &t.Recurrence.IntervalDays, &recWeekdays, &t.Priority, &energyBand,
		&active, &t.LastDone, &t.SuccessStreak, &t.AvgActualDurationMin, &deletedAt,
	)
	if err != nil {
		return models.Task{}, err
	}

	t.Recurrence.Type = models.RecurrenceType(recType)
	t.EnergyBand = models.EnergyBand(energyBand)
	t.Active = active

	if deletedAt.Valid {
		t.DeletedAt = &deletedAt.String
	}

	if recWeekdays != "" {
		var weekdays []int
		if err := json.Unmarshal([]byte(recWeekdays), &weekdays); err == nil {
			for _, w := range weekdays {
				t.Recurrence.WeekdayMask = append(t.Recurrence.WeekdayMask, time.Weekday(w))
			}
		}
	}

	return t, nil
}

func (s *SQLiteStore) GetAllTasks() ([]models.Task, error) {
	rows, err := s.db.Query(`
		SELECT id, name, kind, duration_min, earliest_start, latest_end, fixed_start, fixed_end,
		       recurrence_type, recurrence_interval, recurrence_weekdays, priority, energy_band,
		       active, last_done, success_streak, avg_actual_duration, deleted_at
		FROM tasks WHERE deleted_at IS NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []models.Task
	for rows.Next() {
		var t models.Task
		var recType, recWeekdays, energyBand string
		var active bool
		var deletedAt sql.NullString

		err := rows.Scan(
			&t.ID, &t.Name, &t.Kind, &t.DurationMin, &t.EarliestStart, &t.LatestEnd, &t.FixedStart, &t.FixedEnd,
			&recType, &t.Recurrence.IntervalDays, &recWeekdays, &t.Priority, &energyBand,
			&active, &t.LastDone, &t.SuccessStreak, &t.AvgActualDurationMin, &deletedAt,
		)
		if err != nil {
			return nil, err
		}

		t.Recurrence.Type = models.RecurrenceType(recType)
		t.EnergyBand = models.EnergyBand(energyBand)
		t.Active = active

		if deletedAt.Valid {
			t.DeletedAt = &deletedAt.String
		}

		if recWeekdays != "" {
			var weekdays []int
			if err := json.Unmarshal([]byte(recWeekdays), &weekdays); err == nil {
				for _, w := range weekdays {
					t.Recurrence.WeekdayMask = append(t.Recurrence.WeekdayMask, time.Weekday(w))
				}
			}
		}
		tasks = append(tasks, t)
	}

	return tasks, nil
}

func (s *SQLiteStore) GetAllTasksIncludingDeleted() ([]models.Task, error) {
	rows, err := s.db.Query(`
		SELECT id, name, kind, duration_min, earliest_start, latest_end, fixed_start, fixed_end,
		       recurrence_type, recurrence_interval, recurrence_weekdays, priority, energy_band,
		       active, last_done, success_streak, avg_actual_duration, deleted_at
		FROM tasks`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []models.Task
	for rows.Next() {
		var t models.Task
		var recType, recWeekdays, energyBand sql.NullString
		var earliestStart, latestEnd, fixedStart, fixedEnd, lastDone sql.NullString
		var durationMin, recurrenceInterval, priority, successStreak sql.NullInt64
		var avgActualDuration sql.NullFloat64
		var active bool
		var deletedAt sql.NullString

		err := rows.Scan(
			&t.ID, &t.Name, &t.Kind, &durationMin, &earliestStart, &latestEnd, &fixedStart, &fixedEnd,
			&recType, &recurrenceInterval, &recWeekdays, &priority, &energyBand,
			&active, &lastDone, &successStreak, &avgActualDuration, &deletedAt,
		)
		if err != nil {
			return nil, err
		}

		if durationMin.Valid {
			t.DurationMin = int(durationMin.Int64)
		}
		if recurrenceInterval.Valid {
			t.Recurrence.IntervalDays = int(recurrenceInterval.Int64)
		}
		if priority.Valid {
			t.Priority = int(priority.Int64)
		}
		if successStreak.Valid {
			t.SuccessStreak = int(successStreak.Int64)
		}
		if avgActualDuration.Valid {
			t.AvgActualDurationMin = avgActualDuration.Float64
		}
		if recType.Valid {
			t.Recurrence.Type = models.RecurrenceType(recType.String)
		}
		if energyBand.Valid {
			t.EnergyBand = models.EnergyBand(energyBand.String)
		}
		if earliestStart.Valid {
			t.EarliestStart = earliestStart.String
		}
		if latestEnd.Valid {
			t.LatestEnd = latestEnd.String
		}
		if fixedStart.Valid {
			t.FixedStart = fixedStart.String
		}
		if fixedEnd.Valid {
			t.FixedEnd = fixedEnd.String
		}
		if lastDone.Valid {
			t.LastDone = lastDone.String
		}
		t.Active = active

		if deletedAt.Valid {
			t.DeletedAt = &deletedAt.String
		}

		if recWeekdays.Valid && recWeekdays.String != "" {
			var weekdays []int
			if err := json.Unmarshal([]byte(recWeekdays.String), &weekdays); err == nil {
				for _, w := range weekdays {
					t.Recurrence.WeekdayMask = append(t.Recurrence.WeekdayMask, time.Weekday(w))
				}
			}
		}
		tasks = append(tasks, t)
	}

	return tasks, nil
}

func (s *SQLiteStore) UpdateTask(task models.Task) error {
	weekdaysJSON, err := json.Marshal(task.Recurrence.WeekdayMask)
	if err != nil {
		return fmt.Errorf("failed to marshal recurrence weekday mask: %w", err)
	}

	var deletedAt sql.NullString
	if task.DeletedAt != nil {
		deletedAt = sql.NullString{String: *task.DeletedAt, Valid: true}
	}

	_, err = s.db.Exec(`
		INSERT OR REPLACE INTO tasks (
			id, name, kind, duration_min, earliest_start, latest_end, fixed_start, fixed_end,
			recurrence_type, recurrence_interval, recurrence_weekdays, priority, energy_band,
			active, last_done, success_streak, avg_actual_duration, deleted_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		task.ID, task.Name, task.Kind, task.DurationMin, task.EarliestStart, task.LatestEnd, task.FixedStart, task.FixedEnd,
		task.Recurrence.Type, task.Recurrence.IntervalDays, string(weekdaysJSON), task.Priority, task.EnergyBand,
		task.Active, task.LastDone, task.SuccessStreak, task.AvgActualDurationMin, deletedAt,
	)
	return err
}

func (s *SQLiteStore) DeleteTask(id string) error {
	// Soft delete: set deleted_at timestamp instead of removing the record
	var deletedAt sql.NullString
	err := s.db.QueryRow("SELECT deleted_at FROM tasks WHERE id = ?", id).Scan(&deletedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("task with id %s not found", id)
		}
		return fmt.Errorf("failed to check task existence: %w", err)
	}

	if deletedAt.Valid {
		return fmt.Errorf("task with id %s is already deleted", id)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_, err = s.db.Exec("UPDATE tasks SET deleted_at = ? WHERE id = ?", now, id)
	return err
}

func (s *SQLiteStore) RestoreTask(id string) error {
	// Restore a soft-deleted task by clearing deleted_at
	var deletedAt sql.NullString
	err := s.db.QueryRow("SELECT deleted_at FROM tasks WHERE id = ?", id).Scan(&deletedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("task with id %s not found", id)
		}
		return fmt.Errorf("failed to check task existence: %w", err)
	}

	if !deletedAt.Valid {
		return fmt.Errorf("cannot restore a task that is not deleted: %s", id)
	}

	_, err = s.db.Exec("UPDATE tasks SET deleted_at = NULL WHERE id = ?", id)
	return err
}

func (s *SQLiteStore) SavePlan(plan models.DayPlan) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Prevent bypassing the delete/restore workflow by ensuring plans cannot be saved
	// with DeletedAt manually set. Use DeletePlan/RestorePlan for managing deletion state.
	if plan.DeletedAt != nil {
		return fmt.Errorf("cannot save a plan with deleted_at set; use DeletePlan to soft-delete or RestorePlan to restore")
	}

	// Determine the revision number
	// If plan.Revision is 0, auto-assign it
	if plan.Revision == 0 {
		// Check if there's an existing accepted plan for this date
		var existingRevision int
		var acceptedAt sql.NullString
		err = tx.QueryRow(
			"SELECT revision, accepted_at FROM plans WHERE date = ? AND deleted_at IS NULL ORDER BY revision DESC LIMIT 1",
			plan.Date,
		).Scan(&existingRevision, &acceptedAt)

		if err == sql.ErrNoRows {
			// No existing plan, start with revision 1
			plan.Revision = 1
		} else if err != nil {
			return fmt.Errorf("failed to check existing plan: %w", err)
		} else {
			// Existing plan found
			if acceptedAt.Valid {
				// Plan is accepted - must create a new revision
				plan.Revision = existingRevision + 1
			} else {
				// Plan exists but not accepted - can overwrite
				plan.Revision = existingRevision
				// Delete the old plan and its slots first
				_, err = tx.Exec("DELETE FROM slots WHERE plan_date = ? AND plan_revision = ? AND deleted_at IS NULL", plan.Date, plan.Revision)
				if err != nil {
					return err
				}
				_, err = tx.Exec("DELETE FROM plans WHERE date = ? AND revision = ?", plan.Date, plan.Revision)
				if err != nil {
					return err
				}
			}
		}
	} else {
		// If revision is manually set, validate that it doesn't overwrite an accepted plan
		// unless it's the same plan being updated (same accepted_at timestamp)
		var existingAcceptedAt sql.NullString
		err = tx.QueryRow("SELECT accepted_at FROM plans WHERE date = ? AND revision = ? AND deleted_at IS NULL", plan.Date, plan.Revision).Scan(&existingAcceptedAt)
		if err == nil && existingAcceptedAt.Valid {
			// Check if we're updating the same plan (same accepted_at timestamp)
			planAcceptedAtStr := ""
			if plan.AcceptedAt != nil {
				planAcceptedAtStr = *plan.AcceptedAt
			}
			if planAcceptedAtStr != existingAcceptedAt.String {
				return fmt.Errorf("cannot overwrite accepted plan: %s revision %d", plan.Date, plan.Revision)
			}
		}
		// If the query returns no rows or accepted_at is NULL, it's safe to proceed
	}

	// Check if plan is deleted - forbid adding slots to deleted plans
	var deletedAt sql.NullString
	err = tx.QueryRow("SELECT deleted_at FROM plans WHERE date = ? AND revision = ?", plan.Date, plan.Revision).Scan(&deletedAt)
	if err == nil && deletedAt.Valid {
		return fmt.Errorf("cannot save slots to a deleted plan: %s revision %d", plan.Date, plan.Revision)
	}

	// Prepare accepted_at value
	var acceptedAtVal sql.NullString
	if plan.AcceptedAt != nil {
		acceptedAtVal = sql.NullString{String: *plan.AcceptedAt, Valid: true}
	}

	// Insert or replace plan
	_, err = tx.Exec(
		"INSERT OR REPLACE INTO plans (date, revision, accepted_at, deleted_at) VALUES (?, ?, ?, NULL)",
		plan.Date, plan.Revision, acceptedAtVal,
	)
	if err != nil {
		return err
	}

	// Delete existing non-soft-deleted slots for this plan revision
	_, err = tx.Exec("DELETE FROM slots WHERE plan_date = ? AND plan_revision = ? AND deleted_at IS NULL", plan.Date, plan.Revision)
	if err != nil {
		return err
	}

	// Insert slots
	stmt, err := tx.Prepare(`
		INSERT INTO slots (
			plan_date, plan_revision, start_time, end_time, task_id, status, feedback_rating, feedback_note, deleted_at, last_notified_start, last_notified_end
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, slot := range plan.Slots {
		var rating, note string
		if slot.Feedback != nil {
			rating = string(slot.Feedback.Rating)
			note = slot.Feedback.Note
		}
		var slotDeletedAt sql.NullString
		if slot.DeletedAt != nil {
			slotDeletedAt = sql.NullString{String: *slot.DeletedAt, Valid: true}
		}
		var lastNotifiedStart, lastNotifiedEnd sql.NullString
		if slot.LastNotifiedStart != nil {
			lastNotifiedStart = sql.NullString{String: *slot.LastNotifiedStart, Valid: true}
		}
		if slot.LastNotifiedEnd != nil {
			lastNotifiedEnd = sql.NullString{String: *slot.LastNotifiedEnd, Valid: true}
		}
		_, err = stmt.Exec(
			plan.Date, plan.Revision, slot.Start, slot.End, slot.TaskID, slot.Status, rating, note, slotDeletedAt, lastNotifiedStart, lastNotifiedEnd,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *SQLiteStore) GetPlan(date string) (models.DayPlan, error) {
	// Get latest revision by default
	return s.GetLatestPlanRevision(date)
}

func (s *SQLiteStore) GetLatestPlanRevision(date string) (models.DayPlan, error) {
	// Get the latest non-deleted revision for this date
	var revision int
	var acceptedAt sql.NullString
	err := s.db.QueryRow(
		"SELECT revision, accepted_at FROM plans WHERE date = ? AND deleted_at IS NULL ORDER BY revision DESC LIMIT 1",
		date,
	).Scan(&revision, &acceptedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return models.DayPlan{}, fmt.Errorf("no plan found for date: %s", date)
		}
		return models.DayPlan{}, err
	}

	return s.getPlanByRevision(date, revision, acceptedAt, sql.NullString{})
}

func (s *SQLiteStore) GetPlanRevision(date string, revision int) (models.DayPlan, error) {
	// Get a specific revision
	var acceptedAt, deletedAt sql.NullString
	err := s.db.QueryRow(
		"SELECT accepted_at, deleted_at FROM plans WHERE date = ? AND revision = ?",
		date, revision,
	).Scan(&acceptedAt, &deletedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return models.DayPlan{}, fmt.Errorf("no plan found for date: %s revision: %d", date, revision)
		}
		return models.DayPlan{}, err
	}

	if deletedAt.Valid {
		return models.DayPlan{}, fmt.Errorf("plan for date %s revision %d has been deleted; use 'daylit restore plan %s' to restore it", date, revision, date)
	}

	return s.getPlanByRevision(date, revision, acceptedAt, deletedAt)
}

func (s *SQLiteStore) getPlanByRevision(date string, revision int, acceptedAt, deletedAt sql.NullString) (models.DayPlan, error) {
	plan := models.DayPlan{
		Date:     date,
		Revision: revision,
	}

	if acceptedAt.Valid {
		plan.AcceptedAt = &acceptedAt.String
	}

	// Get slots (exclude soft-deleted slots)
	rows, err := s.db.Query(`
		SELECT start_time, end_time, task_id, status, feedback_rating, feedback_note, last_notified_start, last_notified_end
		FROM slots WHERE plan_date = ? AND plan_revision = ? AND deleted_at IS NULL ORDER BY start_time`,
		date, revision)
	if err != nil {
		return models.DayPlan{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var slot models.Slot
		var rating, note string
		var lastNotifiedStart, lastNotifiedEnd sql.NullString
		err := rows.Scan(
			&slot.Start, &slot.End, &slot.TaskID, &slot.Status, &rating, &note, &lastNotifiedStart, &lastNotifiedEnd,
		)
		if err != nil {
			return models.DayPlan{}, err
		}

		if rating != "" {
			slot.Feedback = &models.Feedback{
				Rating: models.FeedbackRating(rating),
				Note:   note,
			}
		}
		if lastNotifiedStart.Valid {
			slot.LastNotifiedStart = &lastNotifiedStart.String
		}
		if lastNotifiedEnd.Valid {
			slot.LastNotifiedEnd = &lastNotifiedEnd.String
		}
		plan.Slots = append(plan.Slots, slot)
	}

	return plan, nil
}

func (s *SQLiteStore) DeletePlan(date string) error {
	// Soft delete: set deleted_at timestamp for all revisions of the plan and their slots
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Check if any non-deleted plans exist for this date
	var count int
	err = tx.QueryRow("SELECT COUNT(*) FROM plans WHERE date = ? AND deleted_at IS NULL", date).Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		return fmt.Errorf("no active plans found for date: %s", date)
	}

	now := time.Now().UTC().Format(time.RFC3339)

	// Soft delete all revisions of the plan
	if _, err := tx.Exec("UPDATE plans SET deleted_at = ? WHERE date = ? AND deleted_at IS NULL", now, date); err != nil {
		return err
	}

	// Soft delete associated slots that are not already soft-deleted
	if _, err := tx.Exec("UPDATE slots SET deleted_at = ? WHERE plan_date = ? AND deleted_at IS NULL", now, date); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *SQLiteStore) RestorePlan(date string) error {
	// Restore soft-deleted plans (all revisions and their slots) by clearing deleted_at
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Get the most recent deleted_at timestamp for plans on this date
	var planDeletedAt sql.NullString
	err = tx.QueryRow(
		"SELECT deleted_at FROM plans WHERE date = ? AND deleted_at IS NOT NULL ORDER BY deleted_at DESC LIMIT 1",
		date,
	).Scan(&planDeletedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("no deleted plans found for date: %s", date)
		}
		return err
	}

	// Restore all plan revisions that were deleted at the same time
	if _, err := tx.Exec("UPDATE plans SET deleted_at = NULL WHERE date = ? AND deleted_at = ?", date, planDeletedAt.String); err != nil {
		return err
	}

	// Restore only slots that share the same deleted_at timestamp as the plans
	if _, err := tx.Exec(
		"UPDATE slots SET deleted_at = NULL WHERE plan_date = ? AND deleted_at = ?",
		date, planDeletedAt.String,
	); err != nil {
		return err
	}

	return tx.Commit()
}

// Habits

func (s *SQLiteStore) AddHabit(habit models.Habit) error {
	return s.UpdateHabit(habit)
}

func (s *SQLiteStore) GetHabit(id string) (models.Habit, error) {
	row := s.db.QueryRow(`
		SELECT id, name, created_at, archived_at, deleted_at
		FROM habits WHERE id = ? AND deleted_at IS NULL`, id)

	var h models.Habit
	var createdAt string
	var archivedAt, deletedAt sql.NullString

	err := row.Scan(&h.ID, &h.Name, &createdAt, &archivedAt, &deletedAt)
	if err != nil {
		return models.Habit{}, err
	}

	h.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return models.Habit{}, fmt.Errorf("failed to parse created_at: %w", err)
	}
	if archivedAt.Valid {
		t, err := time.Parse(time.RFC3339, archivedAt.String)
		if err != nil {
			return models.Habit{}, fmt.Errorf("failed to parse archived_at: %w", err)
		}
		h.ArchivedAt = &t
	}
	if deletedAt.Valid {
		t, err := time.Parse(time.RFC3339, deletedAt.String)
		if err != nil {
			return models.Habit{}, fmt.Errorf("failed to parse deleted_at: %w", err)
		}
		h.DeletedAt = &t
	}

	return h, nil
}

func (s *SQLiteStore) GetHabitByName(name string) (models.Habit, error) {
	row := s.db.QueryRow(`
		SELECT id, name, created_at, archived_at, deleted_at
		FROM habits WHERE name = ? AND deleted_at IS NULL`, name)

	var h models.Habit
	var createdAt string
	var archivedAt, deletedAt sql.NullString

	err := row.Scan(&h.ID, &h.Name, &createdAt, &archivedAt, &deletedAt)
	if err != nil {
		return models.Habit{}, err
	}

	h.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return models.Habit{}, fmt.Errorf("failed to parse created_at: %w", err)
	}
	if archivedAt.Valid {
		t, err := time.Parse(time.RFC3339, archivedAt.String)
		if err != nil {
			return models.Habit{}, fmt.Errorf("failed to parse archived_at: %w", err)
		}
		h.ArchivedAt = &t
	}
	if deletedAt.Valid {
		t, err := time.Parse(time.RFC3339, deletedAt.String)
		if err != nil {
			return models.Habit{}, fmt.Errorf("failed to parse deleted_at: %w", err)
		}
		h.DeletedAt = &t
	}

	return h, nil
}

func (s *SQLiteStore) GetAllHabits(includeArchived, includeDeleted bool) ([]models.Habit, error) {
	// Check if table exists (for backward compatibility)
	var tableExists bool
	checkRows, err := s.db.Query("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='habits'")
	if err == nil {
		var count int
		if checkRows.Next() {
			checkRows.Scan(&count)
		}
		checkRows.Close()
		tableExists = count > 0
	}
	if !tableExists {
		return []models.Habit{}, nil
	}

	query := "SELECT id, name, created_at, archived_at, deleted_at FROM habits WHERE 1=1"
	if !includeDeleted {
		query += " AND deleted_at IS NULL"
	}
	if !includeArchived {
		query += " AND archived_at IS NULL"
	}
	query += " ORDER BY created_at"

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var habits []models.Habit
	for rows.Next() {
		var h models.Habit
		var createdAt string
		var archivedAt, deletedAt sql.NullString

		err := rows.Scan(&h.ID, &h.Name, &createdAt, &archivedAt, &deletedAt)
		if err != nil {
			return nil, err
		}

		h.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse created_at for habit %s: %w", h.ID, err)
		}
		if archivedAt.Valid {
			t, err := time.Parse(time.RFC3339, archivedAt.String)
			if err != nil {
				return nil, fmt.Errorf("failed to parse archived_at for habit %s: %w", h.ID, err)
			}
			h.ArchivedAt = &t
		}
		if deletedAt.Valid {
			t, err := time.Parse(time.RFC3339, deletedAt.String)
			if err != nil {
				return nil, fmt.Errorf("failed to parse deleted_at for habit %s: %w", h.ID, err)
			}
			h.DeletedAt = &t
		}

		habits = append(habits, h)
	}

	return habits, nil
}

func (s *SQLiteStore) UpdateHabit(habit models.Habit) error {
	var archivedAt, deletedAt sql.NullString
	if habit.ArchivedAt != nil {
		archivedAt = sql.NullString{String: habit.ArchivedAt.Format(time.RFC3339), Valid: true}
	}
	if habit.DeletedAt != nil {
		deletedAt = sql.NullString{String: habit.DeletedAt.Format(time.RFC3339), Valid: true}
	}

	_, err := s.db.Exec(`
		INSERT INTO habits (id, name, created_at, archived_at, deleted_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			archived_at = excluded.archived_at,
			deleted_at = excluded.deleted_at`,
		habit.ID, habit.Name, habit.CreatedAt.Format(time.RFC3339), archivedAt, deletedAt)

	return err
}

func (s *SQLiteStore) ArchiveHabit(id string) error {
	result, err := s.db.Exec(`
		UPDATE habits SET archived_at = ? WHERE id = ? AND deleted_at IS NULL AND archived_at IS NULL`,
		time.Now().Format(time.RFC3339), id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("habit not found or already archived/deleted")
	}

	return nil
}

func (s *SQLiteStore) UnarchiveHabit(id string) error {
	result, err := s.db.Exec(`
		UPDATE habits SET archived_at = NULL WHERE id = ? AND deleted_at IS NULL AND archived_at IS NOT NULL`,
		id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("habit not found or not archived")
	}

	return nil
}

func (s *SQLiteStore) DeleteHabit(id string) error {
	result, err := s.db.Exec(`
		UPDATE habits SET deleted_at = ? WHERE id = ? AND deleted_at IS NULL`,
		time.Now().Format(time.RFC3339), id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("habit not found or already deleted")
	}

	return nil
}

func (s *SQLiteStore) RestoreHabit(id string) error {
	result, err := s.db.Exec(`
		UPDATE habits SET deleted_at = NULL WHERE id = ? AND deleted_at IS NOT NULL`,
		id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("habit not found or not deleted")
	}

	return nil
}

// Habit Entries

func (s *SQLiteStore) AddHabitEntry(entry models.HabitEntry) error {
	return s.UpdateHabitEntry(entry)
}

func (s *SQLiteStore) GetHabitEntry(habitID, day string) (models.HabitEntry, error) {
	row := s.db.QueryRow(`
		SELECT id, habit_id, day, note, created_at, updated_at, deleted_at
		FROM habit_entries WHERE habit_id = ? AND day = ? AND deleted_at IS NULL`,
		habitID, day)

	var e models.HabitEntry
	var createdAt, updatedAt string
	var deletedAt sql.NullString

	err := row.Scan(&e.ID, &e.HabitID, &e.Day, &e.Note, &createdAt, &updatedAt, &deletedAt)
	if err != nil {
		return models.HabitEntry{}, err
	}

	e.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return models.HabitEntry{}, fmt.Errorf("failed to parse created_at: %w", err)
	}
	e.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		return models.HabitEntry{}, fmt.Errorf("failed to parse updated_at: %w", err)
	}
	if deletedAt.Valid {
		t, err := time.Parse(time.RFC3339, deletedAt.String)
		if err != nil {
			return models.HabitEntry{}, fmt.Errorf("failed to parse deleted_at: %w", err)
		}
		e.DeletedAt = &t
	}

	return e, nil
}

func (s *SQLiteStore) GetHabitEntriesForDay(day string) ([]models.HabitEntry, error) {
	rows, err := s.db.Query(`
		SELECT id, habit_id, day, note, created_at, updated_at, deleted_at
		FROM habit_entries WHERE day = ? AND deleted_at IS NULL
		ORDER BY created_at`, day)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []models.HabitEntry
	for rows.Next() {
		var e models.HabitEntry
		var createdAt, updatedAt string
		var deletedAt sql.NullString

		err := rows.Scan(&e.ID, &e.HabitID, &e.Day, &e.Note, &createdAt, &updatedAt, &deletedAt)
		if err != nil {
			return nil, err
		}

		e.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse created_at for entry %s: %w", e.ID, err)
		}
		e.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse updated_at for entry %s: %w", e.ID, err)
		}
		if deletedAt.Valid {
			t, err := time.Parse(time.RFC3339, deletedAt.String)
			if err != nil {
				return nil, fmt.Errorf("failed to parse deleted_at for entry %s: %w", e.ID, err)
			}
			e.DeletedAt = &t
		}

		entries = append(entries, e)
	}

	return entries, nil
}

func (s *SQLiteStore) GetHabitEntriesForHabit(habitID string, startDay, endDay string) ([]models.HabitEntry, error) {
	rows, err := s.db.Query(`
		SELECT id, habit_id, day, note, created_at, updated_at, deleted_at
		FROM habit_entries
		WHERE habit_id = ? AND day >= ? AND day <= ? AND deleted_at IS NULL
		ORDER BY day DESC`, habitID, startDay, endDay)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []models.HabitEntry
	for rows.Next() {
		var e models.HabitEntry
		var createdAt, updatedAt string
		var deletedAt sql.NullString

		err := rows.Scan(&e.ID, &e.HabitID, &e.Day, &e.Note, &createdAt, &updatedAt, &deletedAt)
		if err != nil {
			return nil, err
		}

		e.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse created_at for entry %s: %w", e.ID, err)
		}
		e.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse updated_at for entry %s: %w", e.ID, err)
		}
		if deletedAt.Valid {
			t, err := time.Parse(time.RFC3339, deletedAt.String)
			if err != nil {
				return nil, fmt.Errorf("failed to parse deleted_at for entry %s: %w", e.ID, err)
			}
			e.DeletedAt = &t
		}

		entries = append(entries, e)
	}

	return entries, nil
}

func (s *SQLiteStore) UpdateHabitEntry(entry models.HabitEntry) error {
	var deletedAt sql.NullString
	if entry.DeletedAt != nil {
		deletedAt = sql.NullString{String: entry.DeletedAt.Format(time.RFC3339), Valid: true}
	}

	_, err := s.db.Exec(`
		INSERT INTO habit_entries (id, habit_id, day, note, created_at, updated_at, deleted_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(habit_id, day) DO UPDATE SET
			note = excluded.note,
			updated_at = excluded.updated_at,
			deleted_at = excluded.deleted_at`,
		entry.ID, entry.HabitID, entry.Day, entry.Note,
		entry.CreatedAt.Format(time.RFC3339), entry.UpdatedAt.Format(time.RFC3339), deletedAt)

	return err
}

func (s *SQLiteStore) DeleteHabitEntry(id string) error {
	result, err := s.db.Exec(`
		UPDATE habit_entries SET deleted_at = ? WHERE id = ? AND deleted_at IS NULL`,
		time.Now().Format(time.RFC3339), id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("habit entry not found or already deleted")
	}

	return nil
}

func (s *SQLiteStore) RestoreHabitEntry(id string) error {
	result, err := s.db.Exec(`
		UPDATE habit_entries SET deleted_at = NULL WHERE id = ? AND deleted_at IS NOT NULL`,
		id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("habit entry not found or not deleted")
	}

	return nil
}

// OT Settings

func (s *SQLiteStore) GetOTSettings() (models.OTSettings, error) {
	// Check if table exists (for backward compatibility)
	var tableExists bool
	checkRows, err := s.db.Query("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='ot_settings'")
	if err == nil {
		var count int
		if checkRows.Next() {
			checkRows.Scan(&count)
		}
		checkRows.Close()
		tableExists = count > 0
	}
	if !tableExists {
		return models.OTSettings{}, nil
	}

	row := s.db.QueryRow(`
		SELECT id, prompt_on_empty, strict_mode, default_log_days
		FROM ot_settings WHERE id = 1`)

	var settings models.OTSettings
	var promptOnEmpty, strictMode int

	err = row.Scan(&settings.ID, &promptOnEmpty, &strictMode, &settings.DefaultLogDays)
	if err != nil {
		return models.OTSettings{}, err
	}

	settings.PromptOnEmpty = promptOnEmpty == 1
	settings.StrictMode = strictMode == 1

	return settings, nil
}

func (s *SQLiteStore) SaveOTSettings(settings models.OTSettings) error {
	var promptOnEmpty, strictMode int
	if settings.PromptOnEmpty {
		promptOnEmpty = 1
	}
	if settings.StrictMode {
		strictMode = 1
	}

	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO ot_settings (id, prompt_on_empty, strict_mode, default_log_days)
		VALUES (1, ?, ?, ?)`,
		promptOnEmpty, strictMode, settings.DefaultLogDays)

	return err
}

// OT Entries

func (s *SQLiteStore) AddOTEntry(entry models.OTEntry) error {
	return s.UpdateOTEntry(entry)
}

func (s *SQLiteStore) GetOTEntry(day string) (models.OTEntry, error) {
	row := s.db.QueryRow(`
		SELECT id, day, title, note, created_at, updated_at, deleted_at
		FROM ot_entries WHERE day = ? AND deleted_at IS NULL`, day)

	var e models.OTEntry
	var createdAt, updatedAt string
	var deletedAt sql.NullString

	err := row.Scan(&e.ID, &e.Day, &e.Title, &e.Note, &createdAt, &updatedAt, &deletedAt)
	if err != nil {
		return models.OTEntry{}, err
	}

	e.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return models.OTEntry{}, fmt.Errorf("failed to parse created_at: %w", err)
	}
	e.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		return models.OTEntry{}, fmt.Errorf("failed to parse updated_at: %w", err)
	}
	if deletedAt.Valid {
		t, err := time.Parse(time.RFC3339, deletedAt.String)
		if err != nil {
			return models.OTEntry{}, fmt.Errorf("failed to parse deleted_at: %w", err)
		}
		e.DeletedAt = &t
	}

	return e, nil
}

func (s *SQLiteStore) GetOTEntries(startDay, endDay string, includeDeleted bool) ([]models.OTEntry, error) {
	query := `
		SELECT id, day, title, note, created_at, updated_at, deleted_at
		FROM ot_entries WHERE day >= ? AND day <= ?`
	if !includeDeleted {
		query += " AND deleted_at IS NULL"
	}
	query += " ORDER BY day DESC"

	rows, err := s.db.Query(query, startDay, endDay)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []models.OTEntry
	for rows.Next() {
		var e models.OTEntry
		var createdAt, updatedAt string
		var deletedAt sql.NullString

		err := rows.Scan(&e.ID, &e.Day, &e.Title, &e.Note, &createdAt, &updatedAt, &deletedAt)
		if err != nil {
			return nil, err
		}

		e.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse created_at for entry %s: %w", e.ID, err)
		}
		e.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse updated_at for entry %s: %w", e.ID, err)
		}
		if deletedAt.Valid {
			t, err := time.Parse(time.RFC3339, deletedAt.String)
			if err != nil {
				return nil, fmt.Errorf("failed to parse deleted_at for entry %s: %w", e.ID, err)
			}
			e.DeletedAt = &t
		}

		entries = append(entries, e)
	}

	return entries, nil
}

func (s *SQLiteStore) UpdateOTEntry(entry models.OTEntry) error {
	var deletedAt sql.NullString
	if entry.DeletedAt != nil {
		deletedAt = sql.NullString{String: entry.DeletedAt.Format(time.RFC3339), Valid: true}
	}

	_, err := s.db.Exec(`
		INSERT INTO ot_entries (id, day, title, note, created_at, updated_at, deleted_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(day) DO UPDATE SET
			title = excluded.title,
			note = excluded.note,
			updated_at = excluded.updated_at,
			deleted_at = excluded.deleted_at`,
		entry.ID, entry.Day, entry.Title, entry.Note,
		entry.CreatedAt.Format(time.RFC3339), entry.UpdatedAt.Format(time.RFC3339), deletedAt)

	return err
}

func (s *SQLiteStore) DeleteOTEntry(day string) error {
	result, err := s.db.Exec(`
		UPDATE ot_entries SET deleted_at = ? WHERE day = ? AND deleted_at IS NULL`,
		time.Now().Format(time.RFC3339), day)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("OT entry not found or already deleted")
	}

	return nil
}

func (s *SQLiteStore) RestoreOTEntry(day string) error {
	result, err := s.db.Exec(`
		UPDATE ot_entries SET deleted_at = NULL WHERE day = ? AND deleted_at IS NOT NULL`,
		day)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("OT entry not found or not deleted")
	}

	return nil
}

// UpdateSlotNotificationTimestamp updates the notification timestamp for a specific slot
func (s *SQLiteStore) UpdateSlotNotificationTimestamp(date string, revision int, startTime string, taskID string, notificationType string, timestamp string) error {
	var query string
	switch notificationType {
	case "start":
		query = "UPDATE slots SET last_notified_start = ? WHERE plan_date = ? AND plan_revision = ? AND start_time = ? AND task_id = ? AND deleted_at IS NULL"
	case "end":
		query = "UPDATE slots SET last_notified_end = ? WHERE plan_date = ? AND plan_revision = ? AND start_time = ? AND task_id = ? AND deleted_at IS NULL"
	default:
		return fmt.Errorf("invalid notification type: %s", notificationType)
	}

	result, err := s.db.Exec(query, timestamp, date, revision, startTime, taskID)
	if err != nil {
		return fmt.Errorf("failed to update notification timestamp: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		// This is OK - it means the slot was already notified or doesn't exist
		return nil
	}

	return nil
}

// GetAllPlans retrieves all plans (all dates, all revisions) including deleted ones
func (s *SQLiteStore) GetAllPlans() ([]models.DayPlan, error) {
	// Check if notification columns exist (for backward compatibility with older DBs during migration)
	var hasNotificationCols bool
	checkRows, err := s.db.Query("SELECT count(*) FROM pragma_table_info('slots') WHERE name='last_notified_start'")
	if err == nil {
		var count int
		if checkRows.Next() {
			checkRows.Scan(&count)
		}
		checkRows.Close()
		hasNotificationCols = count > 0
	}

	rows, err := s.db.Query(`
		SELECT date, revision, accepted_at, deleted_at
		FROM plans
		ORDER BY date, revision`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var plans []models.DayPlan
	for rows.Next() {
		var plan models.DayPlan
		var acceptedAt, deletedAt sql.NullString
		if err := rows.Scan(&plan.Date, &plan.Revision, &acceptedAt, &deletedAt); err != nil {
			return nil, err
		}

		if acceptedAt.Valid {
			plan.AcceptedAt = &acceptedAt.String
		}
		if deletedAt.Valid {
			plan.DeletedAt = &deletedAt.String
		}

		// Get slots for this plan (including deleted slots for complete migration)
		query := `SELECT start_time, end_time, task_id, status, feedback_rating, feedback_note, deleted_at`
		if hasNotificationCols {
			query += `, last_notified_start, last_notified_end`
		}
		query += ` FROM slots WHERE plan_date = ? AND plan_revision = ? ORDER BY start_time`

		slotRows, err := s.db.Query(query, plan.Date, plan.Revision)
		if err != nil {
			return nil, err
		}

		for slotRows.Next() {
			var slot models.Slot
			var rating, note string
			var slotDeletedAt, lastNotifiedStart, lastNotifiedEnd sql.NullString

			dest := []interface{}{
				&slot.Start, &slot.End, &slot.TaskID, &slot.Status,
				&rating, &note, &slotDeletedAt,
			}
			if hasNotificationCols {
				dest = append(dest, &lastNotifiedStart, &lastNotifiedEnd)
			}

			if err := slotRows.Scan(dest...); err != nil {
				slotRows.Close()
				return nil, err
			}

			if rating != "" {
				slot.Feedback = &models.Feedback{
					Rating: models.FeedbackRating(rating),
					Note:   note,
				}
			}
			if slotDeletedAt.Valid {
				slot.DeletedAt = &slotDeletedAt.String
			}
			if hasNotificationCols {
				if lastNotifiedStart.Valid {
					slot.LastNotifiedStart = &lastNotifiedStart.String
				}
				if lastNotifiedEnd.Valid {
					slot.LastNotifiedEnd = &lastNotifiedEnd.String
				}
			}

			plan.Slots = append(plan.Slots, slot)
		}
		slotRows.Close()

		plans = append(plans, plan)
	}

	return plans, rows.Err()
}

// GetAllHabitEntries retrieves all habit entries including deleted ones
func (s *SQLiteStore) GetAllHabitEntries() ([]models.HabitEntry, error) {
	// Check if table exists (for backward compatibility)
	var tableExists bool
	checkRows, err := s.db.Query("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='habit_entries'")
	if err == nil {
		defer checkRows.Close()

		var count int
		if checkRows.Next() {
			if err := checkRows.Scan(&count); err != nil {
				return nil, fmt.Errorf("failed to check existence of habit_entries table: %w", err)
			}
		}
		tableExists = count > 0
	}
	if !tableExists {
		return []models.HabitEntry{}, nil
	}

	rows, err := s.db.Query(`
		SELECT id, habit_id, day, note, created_at, updated_at, deleted_at
		FROM habit_entries
		ORDER BY day, habit_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []models.HabitEntry
	for rows.Next() {
		var entry models.HabitEntry
		var createdAt, updatedAt string
		var deletedAt sql.NullString

		if err := rows.Scan(&entry.ID, &entry.HabitID, &entry.Day, &entry.Note,
			&createdAt, &updatedAt, &deletedAt); err != nil {
			return nil, err
		}

		var err error
		entry.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse created_at for habit entry %s: %w", entry.ID, err)
		}
		entry.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse updated_at for habit entry %s: %w", entry.ID, err)
		}
		if deletedAt.Valid {
			t, err := time.Parse(time.RFC3339, deletedAt.String)
			if err != nil {
				return nil, fmt.Errorf("failed to parse deleted_at for habit entry %s: %w", entry.ID, err)
			}
			entry.DeletedAt = &t
		}

		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

// GetAllOTEntries retrieves all OT entries including deleted ones
func (s *SQLiteStore) GetAllOTEntries() ([]models.OTEntry, error) {
	// Check if table exists (for backward compatibility)
	var tableExists bool
	checkRows, err := s.db.Query("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='ot_entries'")
	if err == nil {
		var count int
		if checkRows.Next() {
			checkRows.Scan(&count)
		}
		checkRows.Close()
		tableExists = count > 0
	}
	if !tableExists {
		return []models.OTEntry{}, nil
	}

	rows, err := s.db.Query(`
		SELECT id, day, title, note, created_at, updated_at, deleted_at
		FROM ot_entries
		ORDER BY day`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []models.OTEntry
	for rows.Next() {
		var entry models.OTEntry
		var createdAt, updatedAt string
		var deletedAt sql.NullString

		if err := rows.Scan(&entry.ID, &entry.Day, &entry.Title, &entry.Note,
			&createdAt, &updatedAt, &deletedAt); err != nil {
			return nil, err
		}

		var err error
		entry.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse created_at for OT entry %s: %w", entry.ID, err)
		}
		entry.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse updated_at for OT entry %s: %w", entry.ID, err)
		}
		if deletedAt.Valid {
			t, err := time.Parse(time.RFC3339, deletedAt.String)
			if err != nil {
				return nil, fmt.Errorf("failed to parse deleted_at for OT entry %s: %w", entry.ID, err)
			}
			entry.DeletedAt = &t
		}

		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

func (s *SQLiteStore) GetConfigPath() string {
	return s.path
}

// GetDB returns the underlying database connection.
// Returns nil if the database has not been initialized or loaded.
// Callers should use Load() before calling this method.
func (s *SQLiteStore) GetDB() *sql.DB {
	return s.db
}

// GetMigrationsPath returns the path to the migrations directory.
func (s *SQLiteStore) GetMigrationsPath() string {
	return s.getMigrationsPath()
}
