package postgres

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/julianstephens/daylit/daylit-cli/internal/models"
)

func (s *Store) SavePlan(plan models.DayPlan) error {
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

func (s *Store) GetPlan(date string) (models.DayPlan, error) {
	// Get latest revision by default
	return s.GetLatestPlanRevision(date)
}

func (s *Store) GetLatestPlanRevision(date string) (models.DayPlan, error) {
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

func (s *Store) GetPlanRevision(date string, revision int) (models.DayPlan, error) {
	// Get a specific revision
	var acceptedAt, deletedAt sql.NullString
	err := s.db.QueryRow(
		"SELECT accepted_at, deleted_at FROM plans WHERE date = $1 AND revision = $2",
		date, revision,
	).Scan(&acceptedAt, &deletedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return models.DayPlan{}, fmt.Errorf("plan revision not found: %s rev %d", date, revision)
		}
		return models.DayPlan{}, err
	}

	return s.getPlanByRevision(date, revision, acceptedAt, deletedAt)
}

func (s *Store) getPlanByRevision(date string, revision int, acceptedAt, deletedAt sql.NullString) (models.DayPlan, error) {
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

func (s *Store) DeletePlan(date string) error {
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

func (s *Store) RestorePlan(date string) error {
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

// UpdateSlotNotificationTimestamp updates the notification timestamp for a specific slot
func (s *Store) UpdateSlotNotificationTimestamp(date string, revision int, startTime string, taskID string, notificationType string, timestamp string) error {
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
func (s *Store) GetAllPlans() ([]models.DayPlan, error) {
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

		// Get slots for this plan
		// Note: This is N+1 query, but acceptable for this specific admin/debug function
		// In a real high-load scenario, we would fetch all slots and map them in memory
		slotsRows, err := s.db.Query(`
			SELECT start_time, end_time, task_id, status, feedback_rating, feedback_note, last_notified_start, last_notified_end, deleted_at
			FROM slots WHERE plan_date = $1 AND plan_revision = $2 ORDER BY start_time`,
			plan.Date, plan.Revision)
		if err != nil {
			return nil, err
		}
		defer slotsRows.Close()

		for slotsRows.Next() {
			var slot models.Slot
			var rating, note string
			var lastNotifiedStart, lastNotifiedEnd, slotDeletedAt sql.NullString
			err := slotsRows.Scan(
				&slot.Start, &slot.End, &slot.TaskID, &slot.Status, &rating, &note, &lastNotifiedStart, &lastNotifiedEnd, &slotDeletedAt,
			)
			if err != nil {
				return nil, err
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
			if slotDeletedAt.Valid {
				slot.DeletedAt = &slotDeletedAt.String
			}
			plan.Slots = append(plan.Slots, slot)
		}

		plans = append(plans, plan)
	}

	return plans, nil
}
