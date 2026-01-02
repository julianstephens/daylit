package handlers

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/julianstephens/daylit/daylit-cli/internal/constants"
	"github.com/julianstephens/daylit/daylit-cli/internal/tui/state"
)

// HandleGlobalKeys handles global key presses
func HandleGlobalKeys(m *state.Model, msg tea.KeyMsg) (bool, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		m.Quitting = true
		return true, tea.Quit
	case "tab", "l":
		// Cycle through main views - all 7 tabs
		m.State = (m.State + 1) % constants.NumMainTabs
		return true, nil
	case "shift+tab", "h":
		// Cycle backwards through main views
		m.State = (m.State - 1 + constants.NumMainTabs) % constants.NumMainTabs
		return true, nil
	case "?":
		// Toggle help
		m.Help.ShowAll = !m.Help.ShowAll
		return true, nil
	}
	return false, nil
}
