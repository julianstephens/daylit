package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"

	"github.com/julianstephens/daylit/daylit-cli/internal/migration"
	"github.com/julianstephens/daylit/daylit-cli/internal/models"
	"github.com/julianstephens/daylit/daylit-cli/migrations"
)

type PostgresStore struct {
	connStr string
	db      *sql.DB
}

func NewPostgresStore(connStr string) *PostgresStore {
	s := &PostgresStore{
		connStr: connStr,
	}
	s.ensureSearchPath()
	return s
}

func (s *PostgresStore) ensureSearchPath() {
	// Ensure search_path is set to daylit in the connection string
	if strings.HasPrefix(s.connStr, "postgres://") || strings.HasPrefix(s.connStr, "postgresql://") {
		u, err := url.Parse(s.connStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to parse Postgres connection string %q: %v\n", s.connStr, err)
			return
		}
		q := u.Query()
		// Only set search_path if it's not already present
		if q.Get("search_path") == "" {
			q.Set("search_path", "daylit")
			u.RawQuery = q.Encode()
			s.connStr = u.String()
		}
	} else {
		// Assume DSN format - only append if search_path is not already present
		if !hasSearchPathParam(s.connStr) {
			s.connStr = strings.TrimSpace(s.connStr) + " search_path=daylit"
		}
	}
}

// hasSearchPathParam returns true if the given DSN-style connection string
// contains a search_path parameter key (case-insensitive).
func hasSearchPathParam(connStr string) bool {
	// DSN format is typically space-separated key=value pairs.
	parts := strings.Fields(connStr)
	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		if strings.EqualFold(kv[0], "search_path") {
			return true
		}
	}
	return false
}

// hasSSLMode checks if the connection string contains an sslmode parameter key (case-insensitive).
// It supports both URL-style and DSN-style connection strings.
func hasSSLMode(connStr string) bool {
	// First, try to interpret the connection string as a URL (e.g. postgres://...?sslmode=disable).
	if u, err := url.Parse(connStr); err == nil && u.Scheme != "" {
		q := u.Query()
		for key := range q {
			if strings.EqualFold(key, "sslmode") {
				return true
			}
		}
	}

	// Fallback: treat the connection string as DSN-style space-separated key=value pairs.
	parts := strings.Fields(connStr)
	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		if strings.EqualFold(kv[0], "sslmode") {
			return true
		}
	}

	return false
}

// HasEmbeddedCredentials checks if a PostgreSQL connection string contains embedded credentials.
// This includes checking for passwords in both URL format (postgres://user:pass@host/db) and
// DSN format (user=user password=pass host=host).
func HasEmbeddedCredentials(connStr string) bool {
	// Try parsing as URL first (postgres:// or postgresql://)
	if strings.HasPrefix(connStr, "postgres://") || strings.HasPrefix(connStr, "postgresql://") {
		u, err := url.Parse(connStr)
		if err != nil {
			// If parsing fails, we log a warning but don't block.
			// This avoids false positives for valid but non-standard connection strings.
			fmt.Fprintf(os.Stderr, "Warning: Skipping embedded-credentials security check; could not parse connection string as PostgreSQL URL: %v\n", err)
			return false
		}
		// Check if password is present in the User info
		if u.User != nil {
			password, hasPassword := u.User.Password()
			// Only return true if password is both present and non-empty
			return hasPassword && password != ""
		}
		return false
	}

	// Check DSN format (space-separated key=value pairs)
	// To avoid false positives (e.g. random text containing "password="),
	// we first check if it looks like a DSN by checking for other common keys.
	isDSN := false
	hasPassword := false

	parts := strings.Fields(connStr)
	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.ToLower(kv[0])

		// Check for common DSN keys to confirm it's likely a DSN
		if key == "host" || key == "port" || key == "dbname" || key == "user" || key == "sslmode" || key == "password" {
			isDSN = true
		}

		// Check for password parameter (case-insensitive)
		if key == "password" && kv[1] != "" {
			hasPassword = true
		}
	}

	return isDSN && hasPassword
}

func (s *PostgresStore) Init() error {
	// Open database connection
	db, err := sql.Open("postgres", s.connStr)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool parameters to avoid connection exhaustion
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Create schema if it doesn't exist (before assigning to s.db to maintain consistency)
	if _, err := db.Exec("CREATE SCHEMA IF NOT EXISTS daylit"); err != nil {
		db.Close()
		return fmt.Errorf("failed to create schema: %w", err)
	}

	// Assign to s.db only after schema creation succeeds
	s.db = db

	// Test connection
	if err := s.db.Ping(); err != nil {
		if strings.Contains(err.Error(), "SSL is not enabled on the server") && !hasSSLMode(s.connStr) {
			return fmt.Errorf("failed to connect to database: %w (hint: try adding ?sslmode=disable to your connection string)", err)
		}
		return fmt.Errorf("failed to connect to database: %w", err)
	}

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

func (s *PostgresStore) Load() error {
	if s.db != nil {
		return nil
	}

	db, err := sql.Open("postgres", s.connStr)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	s.db = db

	// Configure connection pool parameters to avoid connection exhaustion
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	if err := s.db.Ping(); err != nil {
		if strings.Contains(err.Error(), "SSL is not enabled on the server") && !hasSSLMode(s.connStr) {
			return fmt.Errorf("failed to connect to database: %w (hint: try adding ?sslmode=disable to your connection string)", err)
		}
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Validate schema version using embedded migrations
	if err := s.validateSchemaVersion(); err != nil {
		return err
	}

	return nil
}

func (s *PostgresStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *PostgresStore) runMigrations() error {
	// Get the embedded PostgreSQL migrations sub-filesystem
	subFS, err := fs.Sub(migrations.FS, "postgres")
	if err != nil {
		return fmt.Errorf("failed to access postgres migrations: %w", err)
	}

	runner := migration.NewRunner(s.db, subFS)
	_, err = runner.ApplyMigrations(func(msg string) {
		fmt.Println(msg)
	})
	return err
}

func (s *PostgresStore) validateSchemaVersion() error {
	subFS, err := fs.Sub(migrations.FS, "postgres")
	if err != nil {
		return fmt.Errorf("failed to access postgres migrations: %w", err)
	}

	runner := migration.NewRunner(s.db, subFS)
	return runner.ValidateVersion()
}

func (s *PostgresStore) GetSettings() (Settings, error) {
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

func (s *PostgresStore) SaveSettings(settings Settings) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// PostgreSQL uses INSERT ... ON CONFLICT for upsert
	stmt, err := tx.Prepare(`
		INSERT INTO settings (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value
	`)
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

func (s *PostgresStore) GetConfigPath() string {
	// Return a non-sensitive identifier instead of the full connection string
	return "postgresql"
}

// Task methods

func (s *PostgresStore) AddTask(task models.Task) error {
	return s.UpdateTask(task)
}

func (s *PostgresStore) GetTask(id string) (models.Task, error) {
	row := s.db.QueryRow(`
SELECT id, name, kind, duration_min, earliest_start, latest_end, fixed_start, fixed_end,
       recurrence_type, recurrence_interval, recurrence_weekdays, priority, energy_band,
       active, last_done, success_streak, avg_actual_duration, deleted_at
FROM tasks WHERE id = $1 AND deleted_at IS NULL`, id)

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

func (s *PostgresStore) GetAllTasks() ([]models.Task, error) {
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

func (s *PostgresStore) GetAllTasksIncludingDeleted() ([]models.Task, error) {
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

func (s *PostgresStore) UpdateTask(task models.Task) error {
	weekdaysJSON, err := json.Marshal(task.Recurrence.WeekdayMask)
	if err != nil {
		return fmt.Errorf("failed to marshal recurrence weekday mask: %w", err)
	}

	var deletedAt sql.NullString
	if task.DeletedAt != nil {
		deletedAt = sql.NullString{String: *task.DeletedAt, Valid: true}
	}

	// PostgreSQL uses INSERT ... ON CONFLICT for upsert
	_, err = s.db.Exec(`
INSERT INTO tasks (
id, name, kind, duration_min, earliest_start, latest_end, fixed_start, fixed_end,
recurrence_type, recurrence_interval, recurrence_weekdays, priority, energy_band,
active, last_done, success_streak, avg_actual_duration, deleted_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
ON CONFLICT (id) DO UPDATE SET
name = EXCLUDED.name,
kind = EXCLUDED.kind,
duration_min = EXCLUDED.duration_min,
earliest_start = EXCLUDED.earliest_start,
latest_end = EXCLUDED.latest_end,
fixed_start = EXCLUDED.fixed_start,
fixed_end = EXCLUDED.fixed_end,
recurrence_type = EXCLUDED.recurrence_type,
recurrence_interval = EXCLUDED.recurrence_interval,
recurrence_weekdays = EXCLUDED.recurrence_weekdays,
priority = EXCLUDED.priority,
energy_band = EXCLUDED.energy_band,
active = EXCLUDED.active,
last_done = EXCLUDED.last_done,
success_streak = EXCLUDED.success_streak,
avg_actual_duration = EXCLUDED.avg_actual_duration,
deleted_at = EXCLUDED.deleted_at`,
		task.ID, task.Name, task.Kind, task.DurationMin, task.EarliestStart, task.LatestEnd, task.FixedStart, task.FixedEnd,
		task.Recurrence.Type, task.Recurrence.IntervalDays, string(weekdaysJSON), task.Priority, task.EnergyBand,
		task.Active, task.LastDone, task.SuccessStreak, task.AvgActualDurationMin, deletedAt,
	)
	return err
}

func (s *PostgresStore) DeleteTask(id string) error {
	var deletedAt sql.NullString
	err := s.db.QueryRow("SELECT deleted_at FROM tasks WHERE id = $1", id).Scan(&deletedAt)
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
	_, err = s.db.Exec("UPDATE tasks SET deleted_at = $1 WHERE id = $2", now, id)
	return err
}

func (s *PostgresStore) RestoreTask(id string) error {
	var deletedAt sql.NullString
	err := s.db.QueryRow("SELECT deleted_at FROM tasks WHERE id = $1", id).Scan(&deletedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("task with id %s not found", id)
		}
		return fmt.Errorf("failed to check task existence: %w", err)
	}

	if !deletedAt.Valid {
		return fmt.Errorf("cannot restore a task that is not deleted: %s", id)
	}

	_, err = s.db.Exec("UPDATE tasks SET deleted_at = NULL WHERE id = $1", id)
	return err
}
func (s *PostgresStore) SavePlan(plan models.DayPlan) error {
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
			"SELECT revision, accepted_at FROM plans WHERE date = $1 AND deleted_at IS NULL ORDER BY revision DESC LIMIT 1",
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
				_, err = tx.Exec("DELETE FROM slots WHERE plan_date = $1 AND plan_revision = $2 AND deleted_at IS NULL", plan.Date, plan.Revision)
				if err != nil {
					return err
				}
				_, err = tx.Exec("DELETE FROM plans WHERE date = $1 AND revision = $2", plan.Date, plan.Revision)
				if err != nil {
					return err
				}
			}
		}
	} else {
		// If revision is manually set, validate that it doesn't overwrite an accepted plan
		// unless it's the same plan being updated (same accepted_at timestamp)
		var existingAcceptedAt sql.NullString
		err = tx.QueryRow("SELECT accepted_at FROM plans WHERE date = $1 AND revision = $2 AND deleted_at IS NULL", plan.Date, plan.Revision).Scan(&existingAcceptedAt)
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
	err = tx.QueryRow("SELECT deleted_at FROM plans WHERE date = $1 AND revision = $2", plan.Date, plan.Revision).Scan(&deletedAt)
	if err == nil && deletedAt.Valid {
		return fmt.Errorf("cannot save slots to a deleted plan: %s revision %d", plan.Date, plan.Revision)
	}

	// Prepare accepted_at value
	var acceptedAtVal sql.NullString
	if plan.AcceptedAt != nil {
		acceptedAtVal = sql.NullString{String: *plan.AcceptedAt, Valid: true}
	}

	// Insert or replace plan
	_, err = tx.Exec(`
		INSERT INTO plans (date, revision, accepted_at, deleted_at) VALUES ($1, $2, $3, NULL)
		ON CONFLICT (date, revision) DO UPDATE SET
			accepted_at = EXCLUDED.accepted_at,
			deleted_at = EXCLUDED.deleted_at`,
		plan.Date, plan.Revision, acceptedAtVal,
	)
	if err != nil {
		return err
	}

	// Delete existing non-soft-deleted slots for this plan revision
	_, err = tx.Exec("DELETE FROM slots WHERE plan_date = $1 AND plan_revision = $2 AND deleted_at IS NULL", plan.Date, plan.Revision)
	if err != nil {
		return err
	}

	// Insert slots
	stmt, err := tx.Prepare(`
		INSERT INTO slots (
			plan_date, plan_revision, start_time, end_time, task_id, status, feedback_rating, feedback_note, deleted_at, last_notified_start, last_notified_end
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`)
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

func (s *PostgresStore) GetPlan(date string) (models.DayPlan, error) {
	// Get latest revision by default
	return s.GetLatestPlanRevision(date)
}

func (s *PostgresStore) GetLatestPlanRevision(date string) (models.DayPlan, error) {
	// Get the latest non-deleted revision for this date
	var revision int
	var acceptedAt sql.NullString
	err := s.db.QueryRow(
		"SELECT revision, accepted_at FROM plans WHERE date = $1 AND deleted_at IS NULL ORDER BY revision DESC LIMIT 1",
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

func (s *PostgresStore) GetPlanRevision(date string, revision int) (models.DayPlan, error) {
	// Get a specific revision
	var acceptedAt, deletedAt sql.NullString
	err := s.db.QueryRow(
		"SELECT accepted_at, deleted_at FROM plans WHERE date = $1 AND revision = $2",
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

func (s *PostgresStore) getPlanByRevision(date string, revision int, acceptedAt, deletedAt sql.NullString) (models.DayPlan, error) {
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
		FROM slots WHERE plan_date = $1 AND plan_revision = $2 AND deleted_at IS NULL ORDER BY start_time`,
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

func (s *PostgresStore) DeletePlan(date string) error {
	// Soft delete: set deleted_at timestamp for all revisions of the plan and their slots
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Check if any non-deleted plans exist for this date
	var count int
	err = tx.QueryRow("SELECT COUNT(*) FROM plans WHERE date = $1 AND deleted_at IS NULL", date).Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		return fmt.Errorf("no active plans found for date: %s", date)
	}

	now := time.Now().UTC().Format(time.RFC3339)

	// Soft delete all revisions of the plan
	if _, err := tx.Exec("UPDATE plans SET deleted_at = $1 WHERE date = $2 AND deleted_at IS NULL", now, date); err != nil {
		return err
	}

	// Soft delete associated slots that are not already soft-deleted
	if _, err := tx.Exec("UPDATE slots SET deleted_at = $1 WHERE plan_date = $2 AND deleted_at IS NULL", now, date); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *PostgresStore) RestorePlan(date string) error {
	// Restore soft-deleted plans (all revisions and their slots) by clearing deleted_at
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Get the most recent deleted_at timestamp for plans on this date
	var planDeletedAt sql.NullString
	err = tx.QueryRow(
		"SELECT deleted_at FROM plans WHERE date = $1 AND deleted_at IS NOT NULL ORDER BY deleted_at DESC LIMIT 1",
		date,
	).Scan(&planDeletedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("no deleted plans found for date: %s", date)
		}
		return err
	}

	// Restore all plan revisions that were deleted at the same time
	if _, err := tx.Exec("UPDATE plans SET deleted_at = NULL WHERE date = $1 AND deleted_at = $2", date, planDeletedAt.String); err != nil {
		return err
	}

	// Restore only slots that share the same deleted_at timestamp as the plans
	if _, err := tx.Exec(
		"UPDATE slots SET deleted_at = NULL WHERE plan_date = $1 AND deleted_at = $2",
		date, planDeletedAt.String,
	); err != nil {
		return err
	}

	return tx.Commit()
}

// Habits

func (s *PostgresStore) AddHabit(habit models.Habit) error {
	return s.UpdateHabit(habit)
}

func (s *PostgresStore) GetHabit(id string) (models.Habit, error) {
	row := s.db.QueryRow(`
		SELECT id, name, created_at, archived_at, deleted_at
		FROM habits WHERE id = $1 AND deleted_at IS NULL`, id)

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

func (s *PostgresStore) GetHabitByName(name string) (models.Habit, error) {
	row := s.db.QueryRow(`
		SELECT id, name, created_at, archived_at, deleted_at
		FROM habits WHERE name = $1 AND deleted_at IS NULL`, name)

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

func (s *PostgresStore) GetAllHabits(includeArchived, includeDeleted bool) ([]models.Habit, error) {
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

func (s *PostgresStore) UpdateHabit(habit models.Habit) error {
	var archivedAt, deletedAt sql.NullString
	if habit.ArchivedAt != nil {
		archivedAt = sql.NullString{String: habit.ArchivedAt.Format(time.RFC3339), Valid: true}
	}
	if habit.DeletedAt != nil {
		deletedAt = sql.NullString{String: habit.DeletedAt.Format(time.RFC3339), Valid: true}
	}

	_, err := s.db.Exec(`
		INSERT INTO habits (id, name, created_at, archived_at, deleted_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT(id) DO UPDATE SET
			name = EXCLUDED.name,
			archived_at = EXCLUDED.archived_at,
			deleted_at = EXCLUDED.deleted_at`,
		habit.ID, habit.Name, habit.CreatedAt.Format(time.RFC3339), archivedAt, deletedAt)

	return err
}

func (s *PostgresStore) ArchiveHabit(id string) error {
	result, err := s.db.Exec(`
		UPDATE habits SET archived_at = $1 WHERE id = $2 AND deleted_at IS NULL AND archived_at IS NULL`,
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

func (s *PostgresStore) UnarchiveHabit(id string) error {
	result, err := s.db.Exec(`
		UPDATE habits SET archived_at = NULL WHERE id = $1 AND deleted_at IS NULL AND archived_at IS NOT NULL`,
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

func (s *PostgresStore) DeleteHabit(id string) error {
	result, err := s.db.Exec(`
		UPDATE habits SET deleted_at = $1 WHERE id = $2 AND deleted_at IS NULL`,
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

func (s *PostgresStore) RestoreHabit(id string) error {
	result, err := s.db.Exec(`
		UPDATE habits SET deleted_at = NULL WHERE id = $1 AND deleted_at IS NOT NULL`,
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

func (s *PostgresStore) AddHabitEntry(entry models.HabitEntry) error {
	return s.UpdateHabitEntry(entry)
}

func (s *PostgresStore) GetHabitEntry(habitID, day string) (models.HabitEntry, error) {
	row := s.db.QueryRow(`
		SELECT id, habit_id, day, note, created_at, updated_at, deleted_at
		FROM habit_entries WHERE habit_id = $1 AND day = $2 AND deleted_at IS NULL`,
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

func (s *PostgresStore) GetHabitEntriesForDay(day string) ([]models.HabitEntry, error) {
	rows, err := s.db.Query(`
		SELECT id, habit_id, day, note, created_at, updated_at, deleted_at
		FROM habit_entries WHERE day = $1 AND deleted_at IS NULL
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

func (s *PostgresStore) GetHabitEntriesForHabit(habitID string, startDay, endDay string) ([]models.HabitEntry, error) {
	rows, err := s.db.Query(`
		SELECT id, habit_id, day, note, created_at, updated_at, deleted_at
		FROM habit_entries
		WHERE habit_id = $1 AND day >= $2 AND day <= $3 AND deleted_at IS NULL
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

func (s *PostgresStore) UpdateHabitEntry(entry models.HabitEntry) error {
	var deletedAt sql.NullString
	if entry.DeletedAt != nil {
		deletedAt = sql.NullString{String: entry.DeletedAt.Format(time.RFC3339), Valid: true}
	}

	_, err := s.db.Exec(`
		INSERT INTO habit_entries (id, habit_id, day, note, created_at, updated_at, deleted_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT(habit_id, day) DO UPDATE SET
			note = EXCLUDED.note,
			updated_at = EXCLUDED.updated_at,
			deleted_at = EXCLUDED.deleted_at`,
		entry.ID, entry.HabitID, entry.Day, entry.Note,
		entry.CreatedAt.Format(time.RFC3339), entry.UpdatedAt.Format(time.RFC3339), deletedAt)

	return err
}

func (s *PostgresStore) DeleteHabitEntry(id string) error {
	result, err := s.db.Exec(`
		UPDATE habit_entries SET deleted_at = $1 WHERE id = $2 AND deleted_at IS NULL`,
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

func (s *PostgresStore) RestoreHabitEntry(id string) error {
	result, err := s.db.Exec(`
		UPDATE habit_entries SET deleted_at = NULL WHERE id = $1 AND deleted_at IS NOT NULL`,
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

func (s *PostgresStore) GetOTSettings() (models.OTSettings, error) {
	rows, err := s.db.Query("SELECT key, value FROM settings WHERE key LIKE 'ot_%'")
	if err != nil {
		return models.OTSettings{}, err
	}
	defer rows.Close()

	settings := models.OTSettings{
		PromptOnEmpty:  true,
		StrictMode:     true,
		DefaultLogDays: 14,
	}

	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return models.OTSettings{}, err
		}
		switch key {
		case "ot_prompt_on_empty":
			settings.PromptOnEmpty = value == "true"
		case "ot_strict_mode":
			settings.StrictMode = value == "true"
		case "ot_default_log_days":
			if _, err := fmt.Sscanf(value, "%d", &settings.DefaultLogDays); err != nil {
				return models.OTSettings{}, fmt.Errorf("parsing ot_default_log_days: %w", err)
			}
		}
	}

	return settings, rows.Err()
}

func (s *PostgresStore) SaveOTSettings(settings models.OTSettings) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT INTO settings (key, value) VALUES ($1, $2) ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value")
	if err != nil {
		return err
	}
	defer stmt.Close()

	if _, err := stmt.Exec("ot_prompt_on_empty", fmt.Sprintf("%v", settings.PromptOnEmpty)); err != nil {
		return err
	}
	if _, err := stmt.Exec("ot_strict_mode", fmt.Sprintf("%v", settings.StrictMode)); err != nil {
		return err
	}
	if _, err := stmt.Exec("ot_default_log_days", fmt.Sprintf("%d", settings.DefaultLogDays)); err != nil {
		return err
	}

	return tx.Commit()
}

// OT Entries

func (s *PostgresStore) AddOTEntry(entry models.OTEntry) error {
	return s.UpdateOTEntry(entry)
}

func (s *PostgresStore) GetOTEntry(day string) (models.OTEntry, error) {
	row := s.db.QueryRow(`
		SELECT id, day, title, note, created_at, updated_at, deleted_at
		FROM ot_entries WHERE day = $1 AND deleted_at IS NULL`, day)

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

func (s *PostgresStore) GetOTEntries(startDay, endDay string, includeDeleted bool) ([]models.OTEntry, error) {
	query := `
		SELECT id, day, title, note, created_at, updated_at, deleted_at
		FROM ot_entries WHERE day >= $1 AND day <= $2`
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

func (s *PostgresStore) UpdateOTEntry(entry models.OTEntry) error {
	var deletedAt sql.NullString
	if entry.DeletedAt != nil {
		deletedAt = sql.NullString{String: entry.DeletedAt.Format(time.RFC3339), Valid: true}
	}

	_, err := s.db.Exec(`
		INSERT INTO ot_entries (id, day, title, note, created_at, updated_at, deleted_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT(day) DO UPDATE SET
			title = EXCLUDED.title,
			note = EXCLUDED.note,
			updated_at = EXCLUDED.updated_at,
			deleted_at = EXCLUDED.deleted_at`,
		entry.ID, entry.Day, entry.Title, entry.Note,
		entry.CreatedAt.Format(time.RFC3339), entry.UpdatedAt.Format(time.RFC3339), deletedAt)

	return err
}

func (s *PostgresStore) DeleteOTEntry(day string) error {
	result, err := s.db.Exec(`
		UPDATE ot_entries SET deleted_at = $1 WHERE day = $2 AND deleted_at IS NULL`,
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

func (s *PostgresStore) RestoreOTEntry(day string) error {
	result, err := s.db.Exec(`
		UPDATE ot_entries SET deleted_at = NULL WHERE day = $1 AND deleted_at IS NOT NULL`,
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
func (s *PostgresStore) UpdateSlotNotificationTimestamp(date string, revision int, startTime string, taskID string, notificationType string, timestamp string) error {
	var query string
	switch notificationType {
	case "start":
		query = "UPDATE slots SET last_notified_start = $1 WHERE plan_date = $2 AND plan_revision = $3 AND start_time = $4 AND task_id = $5 AND deleted_at IS NULL"
	case "end":
		query = "UPDATE slots SET last_notified_end = $1 WHERE plan_date = $2 AND plan_revision = $3 AND start_time = $4 AND task_id = $5 AND deleted_at IS NULL"
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
func (s *PostgresStore) GetAllPlans() ([]models.DayPlan, error) {
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
		slotRows, err := s.db.Query(`
SELECT start_time, end_time, task_id, status, feedback_rating, feedback_note,
       deleted_at, last_notified_start, last_notified_end
FROM slots WHERE plan_date = $1 AND plan_revision = $2
ORDER BY start_time`,
			plan.Date, plan.Revision)
		if err != nil {
			return nil, err
		}

		for slotRows.Next() {
			var slot models.Slot
			var rating, note string
			var slotDeletedAt, lastNotifiedStart, lastNotifiedEnd sql.NullString
			err := slotRows.Scan(
				&slot.Start, &slot.End, &slot.TaskID, &slot.Status,
				&rating, &note, &slotDeletedAt, &lastNotifiedStart, &lastNotifiedEnd,
			)
			if err != nil {
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
			if lastNotifiedStart.Valid {
				slot.LastNotifiedStart = &lastNotifiedStart.String
			}
			if lastNotifiedEnd.Valid {
				slot.LastNotifiedEnd = &lastNotifiedEnd.String
			}

			plan.Slots = append(plan.Slots, slot)
		}
		slotRows.Close()

		plans = append(plans, plan)
	}

	return plans, rows.Err()
}

// GetAllHabitEntries retrieves all habit entries including deleted ones
func (s *PostgresStore) GetAllHabitEntries() ([]models.HabitEntry, error) {
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
func (s *PostgresStore) GetAllOTEntries() ([]models.OTEntry, error) {
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
