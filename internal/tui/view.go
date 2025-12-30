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
	case StateFeedback:
		content = m.viewFeedback()
	case StateEditing:
		content = m.form.View()
	case StateConfirmDelete:
		content = m.viewConfirmDelete()
	}

	ui := lipgloss.JoinVertical(
		lipgloss.Left,
		m.viewTabs(),
		content,
		m.help.View(m),
	)

	// If we are already filling the screen (which we are, because components are sized to full width/height),
	// lipgloss.Place won't do much if we pass full width/height.
	// However, if we want to ensure centering if the terminal is huge, we might want to constrain the max width.
	// For now, let's just return ui as components are handling their own sizing/centering if needed.
	return ui
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
	return m.nowModel.View()
}

func (m Model) viewPlan() string {
	return docStyle.Render(m.planModel.View())
}

func (m Model) viewTasks() string {
	return docStyle.Render(m.taskList.View())
}

func (m Model) viewFeedback() string {
	return lipgloss.Place(m.width, m.height-4,
		lipgloss.Center, lipgloss.Center,
		lipgloss.JoinVertical(lipgloss.Center,
			"Rate the last completed task:",
			"",
			"[1] On Track",
			"[2] Too Much",
			"[3] Unnecessary",
			"",
			"[q] Cancel",
		),
	)
}

func (m Model) viewConfirmDelete() string {
	return lipgloss.Place(m.width, m.height-4,
		lipgloss.Center, lipgloss.Center,
		lipgloss.JoinVertical(lipgloss.Center,
			dangerStyle.Render("Are you sure you want to delete this task?"),
			"",
			"[y] Yes",
			"[n] No",
		),
	)
}
