package handlers

import (
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/google/uuid"

	"github.com/julianstephens/daylit/daylit-cli/internal/constants"
	"github.com/julianstephens/daylit/daylit-cli/internal/models"
	"github.com/julianstephens/daylit/daylit-cli/internal/tui/components/tasklist"
	"github.com/julianstephens/daylit/daylit-cli/internal/tui/state"
)

// HandleEditingState handles the task editing state
func HandleEditingState(m *state.Model, msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd

	if msg, ok := msg.(tea.KeyMsg); ok && msg.Type == tea.KeyEsc {
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
		// Apply changes
		m.EditingTask.Name = m.TaskForm.Name
		dur, err := strconv.Atoi(m.TaskForm.Duration)
		if err == nil {
			m.EditingTask.DurationMin = dur
		}
		m.EditingTask.Recurrence.Type = m.TaskForm.Recurrence
		intervalStr := strings.TrimSpace(m.TaskForm.Interval)
		interval, err := strconv.Atoi(intervalStr)
		if err == nil {
			m.EditingTask.Recurrence.IntervalDays = interval
		}
		prio, err := strconv.Atoi(m.TaskForm.Priority)
		if err == nil {
			m.EditingTask.Priority = prio
		}
		m.EditingTask.Active = m.TaskForm.Active

		// Check if task exists to decide Add vs Update
		_, err = m.Store.GetTask(m.EditingTask.ID)
		var saveErr error
		if err != nil {
			// Task doesn't exist, add it
			saveErr = m.Store.AddTask(*m.EditingTask)
		} else {
			// Task exists, update it
			saveErr = m.Store.UpdateTask(*m.EditingTask)
		}

		// Only update task list if save was successful
		if saveErr == nil {
			tasks, err := m.Store.GetAllTasksIncludingDeleted()
			if err == nil {
				m.TaskList.SetTasks(tasks)
			}
			m.UpdateValidationStatus()
		}
		m.State = constants.StateTasks
	case huh.StateAborted:
		m.State = constants.StateTasks
	}
	return tea.Batch(cmds...)
}

// HandleTaskMessages handles messages from the task list component
func HandleTaskMessages(m *state.Model, msg tea.Msg) (bool, tea.Cmd) {
	switch msg := msg.(type) {
	case tasklist.DeleteTaskMsg:
		m.TaskToDeleteID = msg.ID
		m.State = constants.StateConfirmDelete
		return true, nil

	case tasklist.RestoreTaskMsg:
		m.TaskToRestoreID = msg.ID
		m.State = constants.StateConfirmRestore
		return true, nil

	case tasklist.AddTaskMsg:
		task := models.Task{
			ID:          uuid.New().String(),
			Name:        "New Task",
			Kind:        constants.TaskKindFlexible,
			DurationMin: 30,
			Recurrence: models.Recurrence{
				Type: constants.RecurrenceAdHoc,
			},
			Priority: 3,
			Active:   true,
		}
		m.EditingTask = &task
		m.TaskForm = &state.TaskFormModel{
			Name:       task.Name,
			Duration:   strconv.Itoa(task.DurationMin),
			Recurrence: task.Recurrence.Type,
			Interval:   strconv.Itoa(task.Recurrence.IntervalDays),
			Priority:   strconv.Itoa(task.Priority),
			Active:     task.Active,
		}
		m.Form = NewEditForm(m.TaskForm)
		m.State = constants.StateEditing
		return true, m.Form.Init()

	case tasklist.EditTaskMsg:
		m.EditingTask = &msg.Task
		m.TaskForm = &state.TaskFormModel{
			Name:       msg.Task.Name,
			Duration:   strconv.Itoa(msg.Task.DurationMin),
			Recurrence: msg.Task.Recurrence.Type,
			Interval:   strconv.Itoa(msg.Task.Recurrence.IntervalDays),
			Priority:   strconv.Itoa(msg.Task.Priority),
			Active:     msg.Task.Active,
		}
		m.Form = NewEditForm(m.TaskForm)
		m.State = constants.StateEditing
		return true, m.Form.Init()
	}
	return false, nil
}
