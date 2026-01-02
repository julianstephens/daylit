package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"

	"github.com/julianstephens/daylit/daylit-cli/internal/constants"
)

func (m Model) View() string {
	if m.Quitting {
		return ""
	}

	var content string

	switch m.State {
	case constants.StateNow:
		content = m.viewNow()
	case constants.StatePlan:
		content = m.viewPlan()
	case constants.StateTasks:
		content = m.viewTasks()
	case constants.StateHabits:
		content = m.viewHabits()
	case constants.StateOT:
		content = m.viewOT()
	case constants.StateAlerts:
		content = m.viewAlerts()
	case constants.StateSettings:
		content = m.viewSettings()
	case constants.StateFeedback:
		content = m.viewFeedback()
	case constants.StateEditing, constants.StateAddHabit, constants.StateAddAlert, constants.StateEditOT, constants.StateEditSettings:
		formContent := m.Form.View()
		if m.FormError != "" {
			errorStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")).
				Bold(true).
				Padding(1, 0)
			formContent = lipgloss.JoinVertical(lipgloss.Left,
				errorStyle.Render("Error: "+m.FormError),
				formContent,
			)
		}
		content = formContent
	case constants.StateConfirmDelete:
		content = m.viewConfirmDelete()
	case constants.StateConfirmRestore:
		content = m.viewConfirmRestore()
	case constants.StateConfirmOverwrite:
		content = m.viewConfirmOverwrite()
	case constants.StateConfirmArchive:
		content = m.viewConfirmArchive()
	}

	var banner string
	if len(m.ValidationConflicts) > 0 && m.State == constants.StatePlan {
		banner = m.viewConflictBanner()
	}

	ui := lipgloss.JoinVertical(
		lipgloss.Left,
		m.viewTabs(),
		banner,
		content,
		m.Help.View(m),
	)

	// If we are already filling the screen (which we are, because components are sized to full width/height),
	// lipgloss.Place won't do much if we pass full width/height.
	// However, if we want to ensure centering if the terminal is huge, we might want to constrain the max width.
	// For now, let's just return ui as components are handling their own sizing/centering if needed.
	return ui
}

func (m Model) viewTabs() string {
	var tabs []string
	tabTitles := []string{"Now", "Plan", "Tasks", "Habits", "OT", "Alerts", "Settings"}
	for i, title := range tabTitles {
		if m.State == constants.SessionState(i) {
			tabs = append(tabs, activeTabStyle.Render(title))
		} else {
			tabs = append(tabs, inactiveTabStyle.Render(title))
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
}

func (m Model) viewNow() string {
	return m.NowModel.View()
}

func (m Model) viewPlan() string {
	return docStyle.Render(m.PlanModel.View())
}

func (m Model) viewTasks() string {
	return docStyle.Render(m.TaskList.View())
}

func (m Model) viewHabits() string {
	return docStyle.Render(m.HabitsModel.View())
}

func (m Model) viewOT() string {
	return docStyle.Render(m.OTModel.View())
}

func (m Model) viewAlerts() string {
	return docStyle.Render(m.AlertsModel.View())
}

func (m Model) viewSettings() string {
	return docStyle.Render(m.SettingsModel.View())
}

func (m Model) viewFeedback() string {
	return lipgloss.Place(m.Width, m.Height-4,
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
	return lipgloss.Place(m.Width, m.Height-4,
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
	itemID := m.TaskToRestoreID
	if m.PlanToRestoreDate != "" {
		itemType = "plan"
		itemID = m.PlanToRestoreDate
	}
	return lipgloss.Place(m.Width, m.Height-4,
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
	return lipgloss.Place(m.Width, m.Height-4,
		lipgloss.Center, lipgloss.Center,
		lipgloss.JoinVertical(lipgloss.Center,
			dangerStyle.Render(fmt.Sprintf("Overwrite existing plan for %s?", m.PlanToOverwriteDate)),
			"This will create a new revision.",
			"",
			"[y] Yes",
			"[n] No",
		),
	)
}

func (m Model) viewConflictBanner() string {
	if len(m.ValidationConflicts) == 0 {
		return ""
	}

	var bannerStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("0")).
		Background(lipgloss.Color("214")).
		Bold(true).
		Padding(0, 1)

	bannerText := fmt.Sprintf("⚠ %d CONFLICT(S) DETECTED", len(m.ValidationConflicts))
	return bannerStyle.Render(bannerText)
}

func (m Model) viewConfirmArchive() string {
	return lipgloss.Place(m.Width, m.Height-4,
		lipgloss.Center, lipgloss.Center,
		lipgloss.JoinVertical(lipgloss.Center,
			warningStyle.Render("Are you sure you want to archive this habit?"),
			"",
			"[y] Yes",
			"[n] No",
		),
	)
}
