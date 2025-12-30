package tui

import (
	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	var content string

	switch m.state {
	case StateNow:
		content = m.viewNow()
	case StatePlan:
		content = m.viewPlan()
	case StateTasks:
		content = m.viewTasks()
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.viewTabs(),
		content,
		m.help.View(m.keys),
	)
}

func (m Model) viewTabs() string {
	var tabs []string
	for i, title := range []string{"Now", "Plan", "Tasks"} {
		if m.state == SessionState(i) {
			tabs = append(tabs, activeTabStyle.Render(title))
		} else {
			tabs = append(tabs, inactiveTabStyle.Render(title))
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
}

func (m Model) viewNow() string {
	return "Now View (Placeholder)"
}

func (m Model) viewPlan() string {
	return "Plan View (Placeholder)"
}

func (m Model) viewTasks() string {
	return "Tasks View (Placeholder)"
}
