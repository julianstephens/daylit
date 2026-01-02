package sqlite

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/julianstephens/daylit/daylit-cli/internal/models"
)

func (s *Store) AddAlert(alert models.Alert) error {
	if err := alert.Validate(); err != nil {
		return err
	}

	weekdaysJSON, err := json.Marshal(alert.Recurrence.WeekdayMask)
	if err != nil {
		return fmt.Errorf("failed to marshal weekdays: %w", err)
	}

	var lastSentStr *string
	if alert.LastSent != nil {
		str := alert.LastSent.Format(time.RFC3339)
		lastSentStr = &str
	}

	createdAtStr := alert.CreatedAt.Format(time.RFC3339)

	_, err = s.db.Exec(`
		INSERT INTO alerts (
			id, message, time, date, 
			recurrence_type, recurrence_interval, recurrence_weekdays,
			active, last_sent, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		alert.ID, alert.Message, alert.Time, alert.Date,
		string(alert.Recurrence.Type), alert.Recurrence.IntervalDays, string(weekdaysJSON),
		alert.Active, lastSentStr, createdAtStr,
	)

	if err != nil {
		return fmt.Errorf("failed to insert alert: %w", err)
	}

	return nil
}

func (s *Store) GetAlert(id string) (models.Alert, error) {
	var alert models.Alert
	var weekdaysJSON string
	var recurrenceType string
	var lastSentStr *string
	var createdAtStr string

	err := s.db.QueryRow(`
		SELECT id, message, time, date,
			recurrence_type, recurrence_interval, recurrence_weekdays,
			active, last_sent, created_at
		FROM alerts
		WHERE id = ?
	`, id).Scan(
		&alert.ID, &alert.Message, &alert.Time, &alert.Date,
		&recurrenceType, &alert.Recurrence.IntervalDays, &weekdaysJSON,
		&alert.Active, &lastSentStr, &createdAtStr,
	)

	if err == sql.ErrNoRows {
		return models.Alert{}, fmt.Errorf("alert not found")
	}
	if err != nil {
		return models.Alert{}, fmt.Errorf("failed to get alert: %w", err)
	}

	alert.Recurrence.Type = models.RecurrenceType(recurrenceType)

	if err := json.Unmarshal([]byte(weekdaysJSON), &alert.Recurrence.WeekdayMask); err != nil {
		return models.Alert{}, fmt.Errorf("failed to unmarshal weekdays: %w", err)
	}

	if lastSentStr != nil {
		t, err := time.Parse(time.RFC3339, *lastSentStr)
		if err != nil {
			return models.Alert{}, fmt.Errorf("failed to parse last_sent: %w", err)
		}
		alert.LastSent = &t
	}

	createdAt, err := time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return models.Alert{}, fmt.Errorf("failed to parse created_at: %w", err)
	}
	alert.CreatedAt = createdAt

	return alert, nil
}

func (s *Store) GetAllAlerts() ([]models.Alert, error) {
	rows, err := s.db.Query(`
		SELECT id, message, time, date,
			recurrence_type, recurrence_interval, recurrence_weekdays,
			active, last_sent, created_at
		FROM alerts
		ORDER BY time ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query alerts: %w", err)
	}
	defer rows.Close()

	var alerts []models.Alert
	for rows.Next() {
		var alert models.Alert
		var weekdaysJSON string
		var recurrenceType string
		var lastSentStr *string
		var createdAtStr string

		err := rows.Scan(
			&alert.ID, &alert.Message, &alert.Time, &alert.Date,
			&recurrenceType, &alert.Recurrence.IntervalDays, &weekdaysJSON,
			&alert.Active, &lastSentStr, &createdAtStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan alert: %w", err)
		}

		alert.Recurrence.Type = models.RecurrenceType(recurrenceType)

		if err := json.Unmarshal([]byte(weekdaysJSON), &alert.Recurrence.WeekdayMask); err != nil {
			return nil, fmt.Errorf("failed to unmarshal weekdays: %w", err)
		}

		if lastSentStr != nil {
			t, err := time.Parse(time.RFC3339, *lastSentStr)
			if err != nil {
				return nil, fmt.Errorf("failed to parse last_sent: %w", err)
			}
			alert.LastSent = &t
		}

		createdAt, err := time.Parse(time.RFC3339, createdAtStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse created_at: %w", err)
		}
		alert.CreatedAt = createdAt

		alerts = append(alerts, alert)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating alerts: %w", err)
	}

	return alerts, nil
}

func (s *Store) UpdateAlert(alert models.Alert) error {
	if err := alert.Validate(); err != nil {
		return err
	}

	weekdaysJSON, err := json.Marshal(alert.Recurrence.WeekdayMask)
	if err != nil {
		return fmt.Errorf("failed to marshal weekdays: %w", err)
	}

	var lastSentStr *string
	if alert.LastSent != nil {
		str := alert.LastSent.Format(time.RFC3339)
		lastSentStr = &str
	}

	result, err := s.db.Exec(`
		UPDATE alerts SET
			message = ?, time = ?, date = ?,
			recurrence_type = ?, recurrence_interval = ?, recurrence_weekdays = ?,
			active = ?, last_sent = ?
		WHERE id = ?
	`,
		alert.Message, alert.Time, alert.Date,
		string(alert.Recurrence.Type), alert.Recurrence.IntervalDays, string(weekdaysJSON),
		alert.Active, lastSentStr, alert.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update alert: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("alert not found")
	}

	return nil
}

func (s *Store) DeleteAlert(id string) error {
	result, err := s.db.Exec(`DELETE FROM alerts WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete alert: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("alert not found")
	}

	return nil
}
