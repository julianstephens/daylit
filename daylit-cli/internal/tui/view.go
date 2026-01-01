package tui

import (
	"fmt"

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
	case StateHabits:
		content = m.viewHabits()
	case StateSettings:
		content = m.viewSettings()
	case StateFeedback:
		content = m.viewFeedback()
	case StateEditing, StateAddHabit, StateEditSettings:
		content = m.form.View()
	case StateConfirmDelete:
		content = m.viewConfirmDelete()
	case StateConfirmRestore:
		content = m.viewConfirmRestore()
	case StateConfirmOverwrite:
		content = m.viewConfirmOverwrite()
	case StateConfirmArchive:
		content = m.viewConfirmArchive()
	}

	var banner string
	if len(m.validationConflicts) > 0 && m.state == StatePlan {
		banner = m.viewConflictBanner()
	}

	ui := lipgloss.JoinVertical(
		lipgloss.Left,
		m.viewTabs(),
		banner,
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
	tabTitles := []string{"Now", "Plan", "Tasks", "Habits", "Settings"}
	for i, title := range tabTitles {
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

func (m Model) viewHabits() string {
	return docStyle.Render(m.habitsModel.View())
}

func (m Model) viewSettings() string {
	return docStyle.Render(m.settingsModel.View())
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

func (m Model) viewConfirmRestore() string {
	itemType := "task"
	itemID := m.taskToRestoreID
	if m.planToRestoreDate != "" {
		itemType = "plan"
		itemID = m.planToRestoreDate
	}
	return lipgloss.Place(m.width, m.height-4,
		lipgloss.Center, lipgloss.Center,
		lipgloss.JoinVertical(lipgloss.Center,
			warningStyle.Render(fmt.Sprintf("Restore deleted %s: %s?", itemType, itemID)),
			"",
			"[y] Yes",
			"[n] No",
		),
	)
}

func (m Model) viewConfirmOverwrite() string {
	return lipgloss.Place(m.width, m.height-4,
		lipgloss.Center, lipgloss.Center,
		lipgloss.JoinVertical(lipgloss.Center,
			dangerStyle.Render(fmt.Sprintf("Overwrite existing plan for %s?", m.planToOverwriteDate)),
			"This will create a new revision.",
			"",
			"[y] Yes",
			"[n] No",
		),
	)
}

func (m Model) viewConflictBanner() string {
	if len(m.validationConflicts) == 0 {
		return ""
	}

	var bannerStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("0")).
		Background(lipgloss.Color("214")).
		Bold(true).
		Padding(0, 1)

	bannerText := fmt.Sprintf("⚠ %d CONFLICT(S) DETECTED", len(m.validationConflicts))
	return bannerStyle.Render(bannerText)
}

func (m Model) viewConfirmArchive() string {
	return lipgloss.Place(m.width, m.height-4,
		lipgloss.Center, lipgloss.Center,
		lipgloss.JoinVertical(lipgloss.Center,
			warningStyle.Render("Are you sure you want to archive this habit?"),
			"",
			"[y] Yes",
			"[n] No",
		),
	)
}
