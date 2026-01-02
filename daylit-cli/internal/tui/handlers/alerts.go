package handlers

import (
	"fmt"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/google/uuid"

	"github.com/julianstephens/daylit/daylit-cli/internal/cli"
	"github.com/julianstephens/daylit/daylit-cli/internal/constants"
	"github.com/julianstephens/daylit/daylit-cli/internal/models"
	"github.com/julianstephens/daylit/daylit-cli/internal/tui/components/alerts"
	"github.com/julianstephens/daylit/daylit-cli/internal/tui/state"
)

// HandleAddAlertState handles the add alert state
func HandleAddAlertState(m *state.Model, msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd

	if msg, ok := msg.(tea.KeyMsg); ok && msg.Type == tea.KeyEsc {
		m.State = constants.StateAlerts
		return nil
	}

	form, cmd := m.Form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.Form = f
	}
	cmds = append(cmds, cmd)

	switch m.Form.State {
	case huh.StateCompleted:
		// Validate and create new alert
		alert := models.Alert{
			ID:        uuid.New().String(),
			Message:   m.AlertForm.Message,
			Time:      m.AlertForm.Time,
			Date:      m.AlertForm.Date,
			Active:    true,
			CreatedAt: time.Now(),
		}

		// Set recurrence if not one-time
		if alert.Date == "" {
			alert.Recurrence.Type = m.AlertForm.Recurrence
			if m.AlertForm.Interval != "" {
				interval, err := strconv.Atoi(m.AlertForm.Interval)
				if err != nil || interval < 1 {
					// Invalid interval; keep user in the form to correct the value
					m.FormError = "Invalid interval: must be a positive number"
					m.Form.State = huh.StateNormal
					return tea.Batch(cmds...)
				}
				alert.Recurrence.IntervalDays = interval
			}

			// Parse weekdays for weekly recurrence
			if m.AlertForm.Recurrence == constants.RecurrenceWeekly && m.AlertForm.Weekdays != "" {
				weekdays, err := cli.ParseWeekdays(m.AlertForm.Weekdays)
				if err != nil {
					// Invalid weekdays; keep user in the form to correct the value
					m.FormError = fmt.Sprintf("Invalid weekdays: %v", err)
					m.Form.State = huh.StateNormal
					return tea.Batch(cmds...)
				}
				alert.Recurrence.WeekdayMask = weekdays
			}
		}

		if err := m.Store.AddAlert(alert); err == nil {
			// Refresh alerts list only if add succeeded
			alertsList, _ := m.Store.GetAllAlerts()
			m.AlertsModel.SetAlerts(alertsList)
			m.FormError = "" // Clear any previous errors
			m.State = constants.StateAlerts
		} else {
			// Store error and stay in form state to allow retry
			m.FormError = fmt.Sprintf("Failed to add alert: %v", err)
			m.Form.State = huh.StateNormal
		}
	case huh.StateAborted:
		m.State = constants.StateAlerts
	}
	return tea.Batch(cmds...)
}

// HandleAlertMessages handles messages from the alerts component
func HandleAlertMessages(m *state.Model, msg tea.Msg) (bool, tea.Cmd) {
	switch msg := msg.(type) {
	case alerts.AddAlertMsg:
		m.AlertForm = &state.AlertFormModel{
			Message:    "",
			Time:       "",
			Date:       "",
			Recurrence: constants.RecurrenceDaily,
			Interval:   "1",
			Weekdays:   "",
		}
		m.Form = NewAlertForm(m.AlertForm)
		m.State = constants.StateAddAlert
		return true, m.Form.Init()

	case alerts.DeleteAlertMsg:
		if err := m.Store.DeleteAlert(msg.ID); err == nil {
			alertsList, _ := m.Store.GetAllAlerts()
			m.AlertsModel.SetAlerts(alertsList)
		}
		return true, nil
	}
	return false, nil
}
