package tui

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	Tab      key.Binding
	ShiftTab key.Binding
	Quit     key.Binding
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	Enter    key.Binding
	Help     key.Binding
	Generate key.Binding
	Feedback key.Binding
	Add      key.Binding
	Edit     key.Binding
	Delete   key.Binding
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Tab, k.Quit, k.Help}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Tab, k.ShiftTab, k.Quit},
		{k.Up, k.Down, k.Left, k.Right, k.Enter, k.Help, k.Generate, k.Feedback, k.Add, k.Edit, k.Delete},
	}
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next tab"),
		),
		ShiftTab: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "prev tab"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("h"),
			key.WithHelp("h", "prev tab"),
		),
		Right: key.NewBinding(
			key.WithKeys("l"),
			key.WithHelp("l", "next tab"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
		Generate: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("g", "generate plan"),
		),
		Feedback: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "feedback"),
		),
		Add: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "add task"),
		),
		Edit: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit task"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete task"),
		),
	}
}
