package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/julianstephens/daylit/daylit-cli/internal/constants"
	"github.com/julianstephens/daylit/daylit-cli/internal/scheduler"
	"github.com/julianstephens/daylit/daylit-cli/internal/storage"
	"github.com/julianstephens/daylit/daylit-cli/internal/tui/state"
)

// Model wraps the state.Model and adds TUI-specific methods
type Model struct {
	state.Model
}

// NewModel creates a new TUI Model
func NewModel(store storage.Provider, sched *scheduler.Scheduler) Model {
	m := Model{
		Model: state.New(store, sched),
	}

	// Run validation on initialization
	m.UpdateValidationStatus()

	return m
}

// ShortHelp returns the short help key bindings
func (m Model) ShortHelp() []key.Binding {
	keys := []key.Binding{m.Keys.Tab, m.Keys.Quit, m.Keys.Help}
	switch m.State {
	case constants.StateTasks:
		keys = append(keys, m.Keys.Add, m.Keys.Edit, m.Keys.Delete)
	case constants.StatePlan:
		keys = append(keys, m.Keys.Generate)
	case constants.StateHabits:
		keys = append(keys, m.Keys.Add)
	}
	keys = append(keys, m.Keys.Feedback)
	return keys
}

// FullHelp returns the full help key bindings
func (m Model) FullHelp() [][]key.Binding {
	global := []key.Binding{m.Keys.Tab, m.Keys.ShiftTab, m.Keys.Quit, m.Keys.Help, m.Keys.Feedback}
	navigation := []key.Binding{m.Keys.Up, m.Keys.Down, m.Keys.Left, m.Keys.Right, m.Keys.Enter}

	var actions []key.Binding
	switch m.State {
	case constants.StateTasks:
		actions = []key.Binding{m.Keys.Add, m.Keys.Edit, m.Keys.Delete}
	case constants.StatePlan:
		actions = []key.Binding{m.Keys.Generate}
	case constants.StateHabits:
		actions = []key.Binding{m.Keys.Add}
	}

	return [][]key.Binding{global, navigation, actions}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return m.NowModel.Init()
}
