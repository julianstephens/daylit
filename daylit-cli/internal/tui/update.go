package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/julianstephens/daylit/daylit-cli/internal/constants"
	"github.com/julianstephens/daylit/daylit-cli/internal/tui/handlers"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Handle Editing State
	if m.State == constants.StateEditing {
		cmd := handlers.HandleEditingState(&m.Model, msg)
		return m, cmd
	}

	// Handle Add Habit State
	if m.State == constants.StateAddHabit {
		cmd := handlers.HandleAddHabitState(&m.Model, msg)
		return m, cmd
	}

	// Handle Add Alert State
	if m.State == constants.StateAddAlert {
		cmd := handlers.HandleAddAlertState(&m.Model, msg)
		return m, cmd
	}

	// Handle Edit OT State
	if m.State == constants.StateEditOT {
		cmd := handlers.HandleEditOTState(&m.Model, msg)
		return m, cmd
	}

	// Handle Edit Settings State
	if m.State == constants.StateEditSettings {
		cmd := handlers.HandleEditSettingsState(&m.Model, msg)
		return m, cmd
	}

	// Handle Feedback State
	if m.State == constants.StateFeedback {
		cmd := handlers.HandleFeedbackState(&m.Model, msg)
		return m, cmd
	}

	// Handle Confirmation State
	if m.State == constants.StateConfirmation {
		cmd := handlers.HandleConfirmationState(&m.Model, msg)
		return m, cmd
	}

	// Handle Confirm Delete State
	if m.State == constants.StateConfirmDelete {
		cmd := handlers.HandleConfirmDeleteState(&m.Model, msg)
		return m, cmd
	}

	// Handle Confirm Restore State
	if m.State == constants.StateConfirmRestore {
		cmd := handlers.HandleConfirmRestoreState(&m.Model, msg)
		return m, cmd
	}

	// Handle Confirm Overwrite State
	if m.State == constants.StateConfirmOverwrite {
		cmd := handlers.HandleConfirmOverwriteState(&m.Model, msg)
		return m, cmd
	}

	// Handle Confirm Archive State
	if m.State == constants.StateConfirmArchive {
		cmd := handlers.HandleConfirmArchiveState(&m.Model, msg)
		return m, cmd
	}

	// Handle Window Size
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		m.Width = msg.Width
		m.Height = msg.Height
		m.Help.Width = msg.Width
		// Adjust height for tabs and help
		listHeight := msg.Height - 4 // Approximate height for tabs + help

		h, v := docStyle.GetFrameSize()
		m.TaskList.SetSize(msg.Width-h, listHeight-v)
		m.PlanModel.SetSize(msg.Width-h, listHeight-v)
		m.NowModel.SetSize(msg.Width, listHeight)
		m.HabitsModel.SetSize(msg.Width-h, listHeight-v)
		m.OTModel.SetSize(msg.Width-h, listHeight-v)
		m.AlertsModel.SetSize(msg.Width-h, listHeight-v)
		m.SettingsModel.SetSize(msg.Width-h, listHeight-v)
		return m, nil
	}

	// Handle Component Messages
	if handled, cmd := handlers.HandleTaskMessages(&m.Model, msg); handled {
		return m, cmd
	}

	if handled, cmd := handlers.HandleHabitMessages(&m.Model, msg); handled {
		return m, cmd
	}

	if handled, cmd := handlers.HandleAlertMessages(&m.Model, msg); handled {
		return m, cmd
	}

	if handled, cmd := handlers.HandleSettingsMessages(&m.Model, msg); handled {
		return m, cmd
	}

	if handled, cmd := handlers.HandleOTMessages(&m.Model, msg); handled {
		return m, cmd
	}

	if handled, cmd := handlers.HandleFeedbackMessages(&m.Model, msg); handled {
		return m, cmd
	}

	if handled, cmd := handlers.HandleConfirmationMessages(&m.Model, msg); handled {
		return m, cmd
	}

	// Global Keys
	if msg, ok := msg.(tea.KeyMsg); ok {
		if handled, cmd := handlers.HandleGlobalKeys(&m.Model, msg); handled {
			if m.Quitting {
				return m, tea.Quit
			}
			return m, cmd
		}
	}

	// Always update nowModel for time ticks
	var cmd tea.Cmd
	m.NowModel, cmd = m.NowModel.Update(msg)
	cmds = append(cmds, cmd)

	switch m.State {
	case constants.StateTasks:
		m.TaskList, cmd = m.TaskList.Update(msg)
		cmds = append(cmds, cmd)
	case constants.StatePlan:
		if msg, ok := msg.(tea.KeyMsg); ok && key.Matches(msg, m.Keys.Generate) {
			// Generate plan
			today := time.Now().Format(constants.DateFormat)

			// Check if plan already exists
			_, err := m.Store.GetPlan(today)
			if err == nil {
				// Plan exists, ask for confirmation
				m.PlanToOverwriteDate = today
				m.State = constants.StateConfirmOverwrite
				return m, nil
			}

			settings, _ := m.Store.GetSettings()

			// Default settings if not set
			dayStart := settings.DayStart
			if dayStart == "" {
				dayStart = "08:00"
			}
			dayEnd := settings.DayEnd
			if dayEnd == "" {
				dayEnd = "18:00"
			}

			tasks, _ := m.Store.GetAllTasks()
			plan, err := m.Scheduler.GeneratePlan(today, tasks, dayStart, dayEnd)
			if err == nil {
				m.Store.SavePlan(plan)
				m.PlanModel.SetPlan(plan, tasks)
				m.NowModel.SetPlan(plan, tasks)
				m.UpdateValidationStatus()
			}
		}
		m.PlanModel, cmd = m.PlanModel.Update(msg)
		cmds = append(cmds, cmd)
	case constants.StateHabits:
		m.HabitsModel, cmd = m.HabitsModel.Update(msg)
		cmds = append(cmds, cmd)
	case constants.StateOT:
		m.OTModel, cmd = m.OTModel.Update(msg)
		cmds = append(cmds, cmd)
	case constants.StateAlerts:
		m.AlertsModel, cmd = m.AlertsModel.Update(msg)
		cmds = append(cmds, cmd)
	case constants.StateSettings:
		m.SettingsModel, cmd = m.SettingsModel.Update(msg)
		cmds = append(cmds, cmd)
	case constants.StateNow:
		// nowModel is already updated above
	}

	return m, tea.Batch(cmds...)
}
