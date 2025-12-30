package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/julianstephens/daylit/internal/migration"
	"github.com/julianstephens/daylit/internal/models"
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
			DayStart:        "07:00",
			DayEnd:          "22:00",
			DefaultBlockMin: 30,
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

	// Validate schema version
	if err := s.validateSchemaVersion(); err != nil {
		return err
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
	// Try to find migrations directory relative to the executable or in common paths
	paths := []string{
		"migrations",
		"./migrations",
		"../migrations",
		"../../migrations",
		filepath.Join(filepath.Dir(os.Args[0]), "migrations"),
		filepath.Join(filepath.Dir(os.Args[0]), "..", "migrations"),
	}

	for _, path := range paths {
		if absPath, err := filepath.Abs(path); err == nil {
			if _, err := os.Stat(absPath); err == nil {
				return absPath
			}
		}
	}

	// Default to "migrations" in current directory
	return "migrations"
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

	return tx.Commit()
}

func (s *SQLiteStore) AddTask(task models.Task) error {
	return s.UpdateTask(task)
}

func (s *SQLiteStore) GetTask(id string) (models.Task, error) {
	row := s.db.QueryRow(`
		SELECT id, name, kind, duration_min, earliest_start, latest_end, fixed_start, fixed_end,
		       recurrence_type, recurrence_interval, recurrence_weekdays, priority, energy_band,
		       active, last_done, success_streak, avg_actual_duration
		FROM tasks WHERE id = ?`, id)

	var t models.Task
	var recType, recWeekdays, energyBand string
	var active bool

	err := row.Scan(
		&t.ID, &t.Name, &t.Kind, &t.DurationMin, &t.EarliestStart, &t.LatestEnd, &t.FixedStart, &t.FixedEnd,
		&recType, &t.Recurrence.IntervalDays, &recWeekdays, &t.Priority, &energyBand,
		&active, &t.LastDone, &t.SuccessStreak, &t.AvgActualDurationMin,
	)
	if err != nil {
		return models.Task{}, err
	}

	t.Recurrence.Type = models.RecurrenceType(recType)
	t.EnergyBand = models.EnergyBand(energyBand)
	t.Active = active

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
		       active, last_done, success_streak, avg_actual_duration
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

		err := rows.Scan(
			&t.ID, &t.Name, &t.Kind, &t.DurationMin, &t.EarliestStart, &t.LatestEnd, &t.FixedStart, &t.FixedEnd,
			&recType, &t.Recurrence.IntervalDays, &recWeekdays, &t.Priority, &energyBand,
			&active, &t.LastDone, &t.SuccessStreak, &t.AvgActualDurationMin,
		)
		if err != nil {
			return nil, err
		}

		t.Recurrence.Type = models.RecurrenceType(recType)
		t.EnergyBand = models.EnergyBand(energyBand)
		t.Active = active

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

func (s *SQLiteStore) UpdateTask(task models.Task) error {
	weekdaysJSON, err := json.Marshal(task.Recurrence.WeekdayMask)
	if err != nil {
		return fmt.Errorf("failed to marshal recurrence weekday mask: %w", err)
	}

	_, err = s.db.Exec(`
		INSERT OR REPLACE INTO tasks (
			id, name, kind, duration_min, earliest_start, latest_end, fixed_start, fixed_end,
			recurrence_type, recurrence_interval, recurrence_weekdays, priority, energy_band,
			active, last_done, success_streak, avg_actual_duration
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		task.ID, task.Name, task.Kind, task.DurationMin, task.EarliestStart, task.LatestEnd, task.FixedStart, task.FixedEnd,
		task.Recurrence.Type, task.Recurrence.IntervalDays, string(weekdaysJSON), task.Priority, task.EnergyBand,
		task.Active, task.LastDone, task.SuccessStreak, task.AvgActualDurationMin,
	)
	return err
}

func (s *SQLiteStore) DeleteTask(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	// Delete any slots that reference this task to maintain referential integrity.
	if _, err := tx.Exec("DELETE FROM slots WHERE task_id = ?", id); err != nil {
		tx.Rollback()
		return err
	}

	// Now delete the task itself.
	if _, err := tx.Exec("DELETE FROM tasks WHERE id = ?", id); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (s *SQLiteStore) SavePlan(plan models.DayPlan) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Insert plan
	_, err = tx.Exec("INSERT OR IGNORE INTO plans (date) VALUES (?)", plan.Date)
	if err != nil {
		return err
	}

	// Delete existing slots for this plan
	_, err = tx.Exec("DELETE FROM slots WHERE plan_date = ?", plan.Date)
	if err != nil {
		return err
	}

	// Insert slots
	stmt, err := tx.Prepare(`
		INSERT INTO slots (
			plan_date, start_time, end_time, task_id, status, feedback_rating, feedback_note
		) VALUES (?, ?, ?, ?, ?, ?, ?)`)
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
		_, err = stmt.Exec(
			plan.Date, slot.Start, slot.End, slot.TaskID, slot.Status, rating, note,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *SQLiteStore) GetPlan(date string) (models.DayPlan, error) {
	// Check if plan exists
	var exists int
	err := s.db.QueryRow("SELECT 1 FROM plans WHERE date = ?", date).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.DayPlan{}, fmt.Errorf("no plan found for date: %s", date)
		}
		return models.DayPlan{}, err
	}

	plan := models.DayPlan{
		Date: date,
	}

	// Get slots
	rows, err := s.db.Query(`
		SELECT start_time, end_time, task_id, status, feedback_rating, feedback_note
		FROM slots WHERE plan_date = ? ORDER BY start_time`, date)
	if err != nil {
		return models.DayPlan{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var slot models.Slot
		var rating, note string
		err := rows.Scan(
			&slot.Start, &slot.End, &slot.TaskID, &slot.Status, &rating, &note,
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
		plan.Slots = append(plan.Slots, slot)
	}

	return plan, nil
}

func (s *SQLiteStore) GetConfigPath() string {
	return s.path
}

// GetDB returns the underlying database connection
func (s *SQLiteStore) GetDB() (*sql.DB, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	return s.db, nil
}

// GetMigrationsPath returns the path to the migrations directory
func (s *SQLiteStore) GetMigrationsPath() string {
	return s.getMigrationsPath()
}
