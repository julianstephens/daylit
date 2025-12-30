package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
		// Adjust height for tabs and help
		listHeight := msg.Height - 4 // Approximate height for tabs + help
		m.taskList.SetSize(msg.Width, listHeight)
		m.planModel.SetSize(msg.Width, listHeight)
		m.nowModel.SetSize(msg.Width, listHeight)

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			m.quitting = true
			return m, tea.Quit
		case key.Matches(msg, m.keys.Tab, m.keys.Right):
			m.state = (m.state + 1) % 3
			return m, nil
		case key.Matches(msg, m.keys.ShiftTab, m.keys.Left):
			m.state = (m.state - 1 + 3) % 3
			return m, nil
		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
			return m, nil
		}
	}

	// Always update nowModel for time ticks
	var cmd tea.Cmd
	m.nowModel, cmd = m.nowModel.Update(msg)
	cmds = append(cmds, cmd)

	switch m.state {
	case StateTasks:
		m.taskList, cmd = m.taskList.Update(msg)
		cmds = append(cmds, cmd)
	case StatePlan:
		if msg, ok := msg.(tea.KeyMsg); ok && key.Matches(msg, m.keys.Generate) {
			// Generate plan
			today := time.Now().Format("2006-01-02")
			settings := m.store.GetSettings()

			// Default settings if not set
			dayStart := settings.DayStart
			if dayStart == "" {
				dayStart = "08:00"
			}
			dayEnd := settings.DayEnd
			if dayEnd == "" {
				dayEnd = "18:00"
			}

			plan, err := m.scheduler.GeneratePlan(today, m.store.GetAllTasks(), dayStart, dayEnd)
			if err == nil {
				m.store.SavePlan(plan)
				m.planModel.SetPlan(plan, m.store.GetAllTasks())
				m.nowModel.SetPlan(plan, m.store.GetAllTasks())
			}
		}
		m.planModel, cmd = m.planModel.Update(msg)
		cmds = append(cmds, cmd)
	case StateNow:
		// nowModel is already updated above, but if we add specific keys for Now view, handle them here
	}

	return m, tea.Batch(cmds...)
}
