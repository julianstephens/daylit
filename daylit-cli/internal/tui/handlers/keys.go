package handlers

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/julianstephens/daylit/daylit-cli/internal/constants"
	"github.com/julianstephens/daylit/daylit-cli/internal/tui/state"
)

// HandleGlobalKeys handles global key presses
func HandleGlobalKeys(m *state.Model, msg tea.KeyMsg) (bool, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		m.Quitting = true
		return true, tea.Quit
	case "tab":
		// Cycle through main views
		switch m.State {
		case constants.StateTasks:
			m.State = constants.StateHabits
		case constants.StateHabits:
			m.State = constants.StateOT
		case constants.StateOT:
			m.State = constants.StateSettings
		case constants.StateSettings:
			m.State = constants.StateTasks
		default:
			// If in a sub-state (like editing), don't switch views with tab
			// unless we want to force exit the sub-state
		}
		return true, nil
	case "shift+tab":
		// Cycle backwards through main views
		switch m.State {
		case constants.StateTasks:
			m.State = constants.StateSettings
		case constants.StateHabits:
			m.State = constants.StateTasks
		case constants.StateOT:
			m.State = constants.StateHabits
		case constants.StateSettings:
			m.State = constants.StateOT
		default:
			// If in a sub-state, don't switch
		}
		return true, nil
	}
	return false, nil
}
