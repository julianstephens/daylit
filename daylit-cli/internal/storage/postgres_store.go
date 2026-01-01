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
	_ "github.com/lib/pq"
)

type PostgresStore struct {
	connStr string
	db      *sql.DB
}

func NewPostgresStore(connStr string) *PostgresStore {
	return &PostgresStore{
		connStr: connStr,
	}
}

func (s *PostgresStore) Init() error {
	// Open database connection
	db, err := sql.Open("postgres", s.connStr)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	s.db = db

	// Test connection
	if err := s.db.Ping(); err != nil {
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

	// Test connection
	if err := s.db.Ping(); err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Validate schema version if migrations directory is available
	migrationsPath := s.getMigrationsPath()
	if migrationsPath != "" {
		if _, err := os.Stat(migrationsPath); err == nil {
			if err := s.validateSchemaVersion(); err != nil {
				return err
			}
		}
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
	migrationsPath := s.getMigrationsPath()
	runner := migration.NewRunner(s.db, migrationsPath)
	_, err := runner.ApplyMigrations(func(msg string) {
		fmt.Println(msg)
	})
	return err
}

func (s *PostgresStore) validateSchemaVersion() error {
	migrationsPath := s.getMigrationsPath()
	runner := migration.NewRunner(s.db, migrationsPath)
	return runner.ValidateVersion()
}

func (s *PostgresStore) getMigrationsPath() string {
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
		"migrations/postgres",
		"./migrations/postgres",
		"../migrations/postgres",
		"../../migrations/postgres",
		"../../../migrations/postgres",
		"../../../../migrations/postgres",
		filepath.Join(filepath.Dir(os.Args[0]), "migrations", "postgres"),
		filepath.Join(filepath.Dir(os.Args[0]), "..", "migrations", "postgres"),
	}

	for _, path := range paths {
		if absPath, err := filepath.Abs(path); err == nil {
			if _, err := os.Stat(absPath); err == nil {
				return absPath
			}
		}
	}

	// Default to "migrations/postgres" in current directory (will fail gracefully if not found)
	return "migrations/postgres"
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
	return s.connStr
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
