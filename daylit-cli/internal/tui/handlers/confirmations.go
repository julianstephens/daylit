package handlers

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/julianstephens/daylit/daylit-cli/internal/constants"
	"github.com/julianstephens/daylit/daylit-cli/internal/tui/state"
)

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
			if m.HabitToArchiveID != "" {
				if err := m.Store.ArchiveHabit(m.HabitToArchiveID); err == nil {
					// Refresh habits list
					today := time.Now().Format(constants.DateFormat)
					habitsList, _ := m.Store.GetAllHabits(false, true)
					habitEntries, _ := m.Store.GetHabitEntriesForDay(today)
					m.HabitsModel.SetHabits(habitsList, habitEntries)
				}
				m.HabitToArchiveID = ""
			}
			m.State = constants.StateHabits
		case "n", "N", "esc":
			m.HabitToArchiveID = ""
			m.State = constants.StateHabits
		}
	}
	return nil
}
