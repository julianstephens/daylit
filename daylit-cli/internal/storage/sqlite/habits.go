package sqlite

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/julianstephens/daylit/daylit-cli/internal/models"
)

func (s *Store) AddHabit(habit models.Habit) error {
	return s.UpdateHabit(habit)
}

func (s *Store) GetHabit(id string) (models.Habit, error) {
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

func (s *Store) GetHabitByName(name string) (models.Habit, error) {
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

func (s *Store) GetAllHabits(includeArchived, includeDeleted bool) ([]models.Habit, error) {
	// Check if table exists (for backward compatibility)
	exists, err := s.tableExists("habits")
	if err != nil || !exists {
		// If we can't confirm the table exists, or it does not exist,
		// behave as if it does not.
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

func (s *Store) UpdateHabit(habit models.Habit) error {
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

func (s *Store) ArchiveHabit(id string) error {
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

func (s *Store) UnarchiveHabit(id string) error {
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

func (s *Store) DeleteHabit(id string) error {
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

func (s *Store) RestoreHabit(id string) error {
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

func (s *Store) AddHabitEntry(entry models.HabitEntry) error {
	return s.UpdateHabitEntry(entry)
}

func (s *Store) GetHabitEntry(habitID, day string) (models.HabitEntry, error) {
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

func (s *Store) GetHabitEntriesForDay(day string) ([]models.HabitEntry, error) {
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

func (s *Store) GetHabitEntriesForHabit(habitID string, startDay, endDay string) ([]models.HabitEntry, error) {
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

func (s *Store) UpdateHabitEntry(entry models.HabitEntry) error {
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

func (s *Store) DeleteHabitEntry(id string) error {
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

func (s *Store) RestoreHabitEntry(id string) error {
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
