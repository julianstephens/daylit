package sqlite

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/julianstephens/daylit/daylit-cli/internal/models"
)

// GetAllPlans retrieves all plans (all dates, all revisions) including deleted ones
func (s *Store) GetAllPlans() ([]models.DayPlan, error) {
	// Check if notification columns exist (for backward compatibility with older DBs during migration)
	var hasNotificationCols bool
	checkRows, err := s.db.Query("SELECT count(*) FROM pragma_table_info('slots') WHERE name='last_notified_start'")
	if err == nil {
		defer checkRows.Close()
		var count int
		if checkRows.Next() {
			if err := checkRows.Scan(&count); err == nil {
				hasNotificationCols = count > 0
			}
		}
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
func (s *Store) GetAllHabitEntries() ([]models.HabitEntry, error) {
	// Check if table exists (for backward compatibility)
	exists, err := s.tableExists("habit_entries")
	if err != nil || !exists {
		// If we can't confirm the table exists, or it does not exist,
		// behave as if it does not.
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
func (s *Store) GetAllOTEntries() ([]models.OTEntry, error) {
	// Check if table exists (for backward compatibility)
	exists, err := s.tableExists("ot_entries")
	if err != nil || !exists {
		// If we can't confirm the table exists, or it does not exist,
		// behave as if it does not.
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
