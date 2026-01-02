package postgres

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/julianstephens/daylit/daylit-cli/internal/constants"
	"github.com/julianstephens/daylit/daylit-cli/internal/models"
)

func (s *Store) AddTask(task models.Task) error {
	return s.UpdateTask(task)
}

func (s *Store) GetTask(id string) (models.Task, error) {
	row := s.db.QueryRow(`
SELECT id, name, kind, duration_min, earliest_start, latest_end, fixed_start, fixed_end,
       recurrence_type, recurrence_interval, recurrence_weekdays, recurrence_month_day,
       recurrence_week_occurrence, recurrence_month, recurrence_day_of_week,
       priority, energy_band, active, last_done, success_streak, avg_actual_duration, deleted_at
FROM tasks WHERE id = $1 AND deleted_at IS NULL`, id)

	var t models.Task
	var recType, recWeekdays, energyBand string
	var active bool
	var deletedAt sql.NullString
	var recMonthDay, recWeekOccurrence, recMonth, recDayOfWeek sql.NullInt64

	err := row.Scan(
		&t.ID, &t.Name, &t.Kind, &t.DurationMin, &t.EarliestStart, &t.LatestEnd, &t.FixedStart, &t.FixedEnd,
		&recType, &t.Recurrence.IntervalDays, &recWeekdays, &recMonthDay, &recWeekOccurrence, &recMonth, &recDayOfWeek,
		&t.Priority, &energyBand, &active, &t.LastDone, &t.SuccessStreak, &t.AvgActualDurationMin, &deletedAt,
	)
	if err != nil {
		return models.Task{}, err
	}

	t.Recurrence.Type = constants.RecurrenceType(recType)
	t.EnergyBand = constants.EnergyBand(energyBand)
	t.Active = active

	if recMonthDay.Valid {
		t.Recurrence.MonthDay = int(recMonthDay.Int64)
	}
	if recWeekOccurrence.Valid {
		t.Recurrence.WeekOccurrence = int(recWeekOccurrence.Int64)
	}
	if recMonth.Valid {
		t.Recurrence.Month = int(recMonth.Int64)
	}
	if recDayOfWeek.Valid {
		t.Recurrence.DayOfWeekInMonth = time.Weekday(recDayOfWeek.Int64)
	}

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

func (s *Store) GetAllTasks() ([]models.Task, error) {
	rows, err := s.db.Query(`
SELECT id, name, kind, duration_min, earliest_start, latest_end, fixed_start, fixed_end,
       recurrence_type, recurrence_interval, recurrence_weekdays, recurrence_month_day,
       recurrence_week_occurrence, recurrence_month, recurrence_day_of_week,
       priority, energy_band, active, last_done, success_streak, avg_actual_duration, deleted_at
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
		var recMonthDay, recWeekOccurrence, recMonth, recDayOfWeek sql.NullInt64

		err := rows.Scan(
			&t.ID, &t.Name, &t.Kind, &t.DurationMin, &t.EarliestStart, &t.LatestEnd, &t.FixedStart, &t.FixedEnd,
			&recType, &t.Recurrence.IntervalDays, &recWeekdays, &recMonthDay, &recWeekOccurrence, &recMonth, &recDayOfWeek,
			&t.Priority, &energyBand, &active, &t.LastDone, &t.SuccessStreak, &t.AvgActualDurationMin, &deletedAt,
		)
		if err != nil {
			return nil, err
		}

		t.Recurrence.Type = constants.RecurrenceType(recType)
		t.EnergyBand = constants.EnergyBand(energyBand)
		t.Active = active

		if recMonthDay.Valid {
			t.Recurrence.MonthDay = int(recMonthDay.Int64)
		}
		if recWeekOccurrence.Valid {
			t.Recurrence.WeekOccurrence = int(recWeekOccurrence.Int64)
		}
		if recMonth.Valid {
			t.Recurrence.Month = int(recMonth.Int64)
		}
		if recDayOfWeek.Valid {
			t.Recurrence.DayOfWeekInMonth = time.Weekday(recDayOfWeek.Int64)
		}

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

func (s *Store) GetAllTasksIncludingDeleted() ([]models.Task, error) {
	rows, err := s.db.Query(`
SELECT id, name, kind, duration_min, earliest_start, latest_end, fixed_start, fixed_end,
       recurrence_type, recurrence_interval, recurrence_weekdays, recurrence_month_day,
       recurrence_week_occurrence, recurrence_month, recurrence_day_of_week,
       priority, energy_band, active, last_done, success_streak, avg_actual_duration, deleted_at
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
		var recMonthDay, recWeekOccurrence, recMonth, recDayOfWeek sql.NullInt64
		var avgActualDuration sql.NullFloat64
		var active bool
		var deletedAt sql.NullString

		err := rows.Scan(
			&t.ID, &t.Name, &t.Kind, &durationMin, &earliestStart, &latestEnd, &fixedStart, &fixedEnd,
			&recType, &recurrenceInterval, &recWeekdays, &recMonthDay, &recWeekOccurrence, &recMonth, &recDayOfWeek,
			&priority, &energyBand, &active, &lastDone, &successStreak, &avgActualDuration, &deletedAt,
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
		if recMonthDay.Valid {
			t.Recurrence.MonthDay = int(recMonthDay.Int64)
		}
		if recWeekOccurrence.Valid {
			t.Recurrence.WeekOccurrence = int(recWeekOccurrence.Int64)
		}
		if recMonth.Valid {
			t.Recurrence.Month = int(recMonth.Int64)
		}
		if recDayOfWeek.Valid {
			t.Recurrence.DayOfWeekInMonth = time.Weekday(recDayOfWeek.Int64)
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
			t.Recurrence.Type = constants.RecurrenceType(recType.String)
		}
		if energyBand.Valid {
			t.EnergyBand = constants.EnergyBand(energyBand.String)
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

func (s *Store) UpdateTask(task models.Task) error {
	weekdaysJSON, err := json.Marshal(task.Recurrence.WeekdayMask)
	if err != nil {
		return fmt.Errorf("failed to marshal recurrence weekday mask: %w", err)
	}

	var deletedAt sql.NullString
	if task.DeletedAt != nil {
		deletedAt = sql.NullString{String: *task.DeletedAt, Valid: true}
	}

	var recMonthDay, recWeekOccurrence, recMonth, recDayOfWeek sql.NullInt64
	if task.Recurrence.MonthDay != 0 {
		recMonthDay = sql.NullInt64{Int64: int64(task.Recurrence.MonthDay), Valid: true}
	}
	if task.Recurrence.WeekOccurrence != 0 || task.Recurrence.Type == constants.RecurrenceMonthlyDay {
		// For monthly_day, -1 is a valid value (last occurrence), so we need to check the type
		recWeekOccurrence = sql.NullInt64{Int64: int64(task.Recurrence.WeekOccurrence), Valid: true}
	}
	if task.Recurrence.Month != 0 {
		recMonth = sql.NullInt64{Int64: int64(task.Recurrence.Month), Valid: true}
	}
	if task.Recurrence.Type == constants.RecurrenceMonthlyDay {
		recDayOfWeek = sql.NullInt64{Int64: int64(task.Recurrence.DayOfWeekInMonth), Valid: true}
	}

	// PostgreSQL uses INSERT ... ON CONFLICT for upsert
	_, err = s.db.Exec(`
INSERT INTO tasks (
id, name, kind, duration_min, earliest_start, latest_end, fixed_start, fixed_end,
recurrence_type, recurrence_interval, recurrence_weekdays, recurrence_month_day,
recurrence_week_occurrence, recurrence_month, recurrence_day_of_week,
priority, energy_band, active, last_done, success_streak, avg_actual_duration, deleted_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22)
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
recurrence_month_day = EXCLUDED.recurrence_month_day,
recurrence_week_occurrence = EXCLUDED.recurrence_week_occurrence,
recurrence_month = EXCLUDED.recurrence_month,
recurrence_day_of_week = EXCLUDED.recurrence_day_of_week,
priority = EXCLUDED.priority,
energy_band = EXCLUDED.energy_band,
active = EXCLUDED.active,
last_done = EXCLUDED.last_done,
success_streak = EXCLUDED.success_streak,
avg_actual_duration = EXCLUDED.avg_actual_duration,
deleted_at = EXCLUDED.deleted_at`,
		task.ID, task.Name, task.Kind, task.DurationMin, task.EarliestStart, task.LatestEnd, task.FixedStart, task.FixedEnd,
		task.Recurrence.Type, task.Recurrence.IntervalDays, string(weekdaysJSON), recMonthDay,
		recWeekOccurrence, recMonth, recDayOfWeek,
		task.Priority, task.EnergyBand, task.Active, task.LastDone, task.SuccessStreak, task.AvgActualDurationMin, deletedAt,
	)
	return err
}

func (s *Store) DeleteTask(id string) error {
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

func (s *Store) RestoreTask(id string) error {
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
