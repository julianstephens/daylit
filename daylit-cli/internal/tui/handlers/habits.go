package handlers

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/google/uuid"

	"github.com/julianstephens/daylit/daylit-cli/internal/constants"
	"github.com/julianstephens/daylit/daylit-cli/internal/models"
	"github.com/julianstephens/daylit/daylit-cli/internal/tui/components/habits"
	"github.com/julianstephens/daylit/daylit-cli/internal/tui/state"
)

// HandleAddHabitState handles the add habit state
func HandleAddHabitState(m *state.Model, msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd

	if msg, ok := msg.(tea.KeyMsg); ok && msg.Type == tea.KeyEsc {
		m.State = constants.StateHabits
		return nil
	}

	form, cmd := m.Form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.Form = f
	}
	cmds = append(cmds, cmd)

	switch m.Form.State {
	case huh.StateCompleted:
		// Create new habit
		habit := models.Habit{
			ID:        uuid.New().String(),
			Name:      m.HabitForm.Name,
			CreatedAt: time.Now(),
		}
		if err := m.Store.AddHabit(habit); err == nil {
			// Refresh habits list only if add succeeded
			today := time.Now().Format(constants.DateFormat)
			habitsList, _ := m.Store.GetAllHabits(false, true)
			habitEntries, _ := m.Store.GetHabitEntriesForDay(today)
			m.HabitsModel.SetHabits(habitsList, habitEntries)
			m.State = constants.StateHabits
		} else {
			// Stay in form state on error to allow retry
			// The form will display, user can cancel with ESC or retry
			m.Form.State = huh.StateNormal
		}
	case huh.StateAborted:
		m.State = constants.StateHabits
	}
	return tea.Batch(cmds...)
}

// HandleHabitMessages handles messages from the habits component
func HandleHabitMessages(m *state.Model, msg tea.Msg) (bool, tea.Cmd) {
	switch msg := msg.(type) {
	case habits.AddHabitMsg:
		m.HabitForm = &state.HabitFormModel{
			Name: "",
		}
		m.Form = NewHabitForm(m.HabitForm)
		m.State = constants.StateAddHabit
		return true, m.Form.Init()

	case habits.MarkHabitMsg:
		today := time.Now().Format(constants.DateFormat)
		entry := models.HabitEntry{
			ID:        uuid.New().String(),
			HabitID:   msg.ID,
			Day:       today,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := m.Store.AddHabitEntry(entry); err == nil {
			habitsList, _ := m.Store.GetAllHabits(false, true)
			habitEntries, _ := m.Store.GetHabitEntriesForDay(today)
			m.HabitsModel.SetHabits(habitsList, habitEntries)
		}
		return true, nil

	case habits.UnmarkHabitMsg:
		today := time.Now().Format(constants.DateFormat)
		entry, err := m.Store.GetHabitEntry(msg.ID, today)
		if err == nil {
			if err := m.Store.DeleteHabitEntry(entry.ID); err == nil {
				habitsList, _ := m.Store.GetAllHabits(false, true)
				habitEntries, _ := m.Store.GetHabitEntriesForDay(today)
				m.HabitsModel.SetHabits(habitsList, habitEntries)
			}
		}
		return true, nil

	case habits.ArchiveHabitMsg:
		m.HabitToArchiveID = msg.ID
		m.State = constants.StateConfirmArchive
		return true, nil

	case habits.DeleteHabitMsg:
		if err := m.Store.DeleteHabit(msg.ID); err == nil {
			today := time.Now().Format(constants.DateFormat)
			habitsList, _ := m.Store.GetAllHabits(false, true)
			habitEntries, _ := m.Store.GetHabitEntriesForDay(today)
			m.HabitsModel.SetHabits(habitsList, habitEntries)
		}
		return true, nil

	case habits.RestoreHabitMsg:
		if err := m.Store.RestoreHabit(msg.ID); err == nil {
			today := time.Now().Format(constants.DateFormat)
			habitsList, _ := m.Store.GetAllHabits(false, true)
			habitEntries, _ := m.Store.GetHabitEntriesForDay(today)
			m.HabitsModel.SetHabits(habitsList, habitEntries)
		}
		return true, nil
	}
	return false, nil
}
