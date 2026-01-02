package handlers

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/julianstephens/daylit/daylit-cli/internal/constants"
	"github.com/julianstephens/daylit/daylit-cli/internal/tui/state"
)

// HandleConfirmationState handles the generic confirmation state
func HandleConfirmationState(m *state.Model, msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd

	if msg, ok := msg.(tea.KeyMsg); ok && msg.Type == tea.KeyEsc {
		m.FormError = "" // Clear error on cancel
		m.State = constants.StateTasks
		return nil
	}

	form, cmd := m.Form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.Form = f
	}
	cmds = append(cmds, cmd)

	switch m.Form.State {
	case huh.StateCompleted:
		if m.ConfirmationForm.Confirmed {
			// Execute the confirmed action
			if m.PendingAction != nil {
				cmd := m.PendingAction()
				cmds = append(cmds, cmd)
			}
		}
		m.PendingAction = nil
		m.FormError = "" // Clear any previous errors
		m.State = constants.StateTasks
	case huh.StateAborted:
		m.PendingAction = nil
		m.FormError = "" // Clear error on abort
		m.State = constants.StateTasks
	}
	return tea.Batch(cmds...)
}

// HandleConfirmDeleteState handles the delete confirmation state
func HandleConfirmDeleteState(m *state.Model, msg tea.Msg) tea.Cmd {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "y", "Y":
			if m.TaskToDeleteID != "" {
				if err := m.Store.DeleteTask(m.TaskToDeleteID); err == nil {
					tasks, _ := m.Store.GetAllTasksIncludingDeleted()
					m.TaskList.SetTasks(tasks)
					m.UpdateValidationStatus()
				}
				m.TaskToDeleteID = ""
			}
			m.State = constants.StateTasks
		case "n", "N", "esc":
			m.TaskToDeleteID = ""
			m.State = constants.StateTasks
		}
	}
	return nil
}

// HandleConfirmRestoreState handles the restore confirmation state
func HandleConfirmRestoreState(m *state.Model, msg tea.Msg) tea.Cmd {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "y", "Y":
			if m.TaskToRestoreID != "" {
				if err := m.Store.RestoreTask(m.TaskToRestoreID); err == nil {
					tasks, _ := m.Store.GetAllTasksIncludingDeleted()
					m.TaskList.SetTasks(tasks)
					m.UpdateValidationStatus()
				}
				m.TaskToRestoreID = ""
			}
			m.State = constants.StateTasks
		case "n", "N", "esc":
			m.TaskToRestoreID = ""
			m.State = constants.StateTasks
		}
	}
	return nil
}

// HandleConfirmOverwriteState handles the overwrite confirmation state
func HandleConfirmOverwriteState(m *state.Model, msg tea.Msg) tea.Cmd {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "y", "Y":
			if m.PlanToOverwriteDate != "" {
				settings, _ := m.Store.GetSettings()
				dayStart := settings.DayStart
				if dayStart == "" {
					dayStart = "08:00"
				}
				dayEnd := settings.DayEnd
				if dayEnd == "" {
					dayEnd = "18:00"
				}

				tasks, _ := m.Store.GetAllTasks()
				plan, err := m.Scheduler.GeneratePlan(m.PlanToOverwriteDate, tasks, dayStart, dayEnd)
				if err == nil {
					m.Store.SavePlan(plan)
					m.PlanModel.SetPlan(plan, tasks)
					m.NowModel.SetPlan(plan, tasks)
					m.UpdateValidationStatus()
				}
				m.PlanToOverwriteDate = ""
			}
			m.State = constants.StatePlan
		case "n", "N", "esc":
			m.PlanToOverwriteDate = ""
			m.State = constants.StatePlan
		}
	}
	return nil
}

// HandleConfirmArchiveState handles the archive confirmation state
func HandleConfirmArchiveState(m *state.Model, msg tea.Msg) tea.Cmd {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "y", "Y":
			today := time.Now().Format(constants.DateFormat)
			if err := m.Store.ArchivePlan(today); err == nil {
				// Refresh plan view
				plan, err := m.Store.GetPlan(today)
				if err == nil {
					tasks, _ := m.Store.GetAllTasks()
					m.PlanModel.SetPlan(plan, tasks)
					m.NowModel.SetPlan(plan, tasks)
				}
			}
			m.State = constants.StatePlan
		case "n", "N", "esc":
			m.State = constants.StatePlan
		}
	}
	return nil
}

// HandleConfirmationMessages handles messages related to confirmations
func HandleConfirmationMessages(m *state.Model, msg tea.Msg) (bool, tea.Cmd) {
	switch msg := msg.(type) {
	case constants.ConfirmationMsg:
		m.ConfirmationForm = &state.ConfirmationFormModel{
			Message: msg.Message,
		}
		m.PendingAction = msg.Action
		m.Form = NewConfirmationForm(m.ConfirmationForm)
		m.State = constants.StateConfirmation
		return true, m.Form.Init()
	}
	return false, nil
}
