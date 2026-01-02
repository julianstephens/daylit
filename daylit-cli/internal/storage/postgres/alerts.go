package postgres

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

	_, err = s.db.Exec(`
		INSERT INTO alerts (
			id, message, time, date, 
			recurrence_type, recurrence_interval, recurrence_weekdays,
			active, last_sent, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`,
		alert.ID, alert.Message, alert.Time, alert.Date,
		string(alert.Recurrence.Type), alert.Recurrence.IntervalDays, string(weekdaysJSON),
		alert.Active, alert.LastSent, alert.CreatedAt,
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
	var lastSent *time.Time

	err := s.db.QueryRow(`
		SELECT id, message, time, date,
			recurrence_type, recurrence_interval, recurrence_weekdays,
			active, last_sent, created_at
		FROM alerts
		WHERE id = $1
	`, id).Scan(
		&alert.ID, &alert.Message, &alert.Time, &alert.Date,
		&recurrenceType, &alert.Recurrence.IntervalDays, &weekdaysJSON,
		&alert.Active, &lastSent, &alert.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return models.Alert{}, fmt.Errorf("alert not found")
	}
	if err != nil {
		return models.Alert{}, fmt.Errorf("failed to get alert: %w", err)
	}

	alert.Recurrence.Type = models.RecurrenceType(recurrenceType)
	alert.LastSent = lastSent

	if err := json.Unmarshal([]byte(weekdaysJSON), &alert.Recurrence.WeekdayMask); err != nil {
		return models.Alert{}, fmt.Errorf("failed to unmarshal weekdays: %w", err)
	}

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
		var lastSent *time.Time

		err := rows.Scan(
			&alert.ID, &alert.Message, &alert.Time, &alert.Date,
			&recurrenceType, &alert.Recurrence.IntervalDays, &weekdaysJSON,
			&alert.Active, &lastSent, &alert.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan alert: %w", err)
		}

		alert.Recurrence.Type = models.RecurrenceType(recurrenceType)
		alert.LastSent = lastSent

		if err := json.Unmarshal([]byte(weekdaysJSON), &alert.Recurrence.WeekdayMask); err != nil {
			return nil, fmt.Errorf("failed to unmarshal weekdays: %w", err)
		}

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

	result, err := s.db.Exec(`
		UPDATE alerts SET
			message = $1, time = $2, date = $3,
			recurrence_type = $4, recurrence_interval = $5, recurrence_weekdays = $6,
			active = $7, last_sent = $8
		WHERE id = $9
	`,
		alert.Message, alert.Time, alert.Date,
		string(alert.Recurrence.Type), alert.Recurrence.IntervalDays, string(weekdaysJSON),
		alert.Active, alert.LastSent, alert.ID,
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
	result, err := s.db.Exec(`DELETE FROM alerts WHERE id = $1`, id)
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
