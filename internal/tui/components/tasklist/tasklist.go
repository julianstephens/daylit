package tasklist

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/julianstephens/daylit/internal/models"
)

type AddTaskMsg struct{}

type DeleteTaskMsg struct {
	ID string
}

type EditTaskMsg struct {
	Task models.Task
}

type RestoreTaskMsg struct {
	ID string
}

type Item struct {
	Task models.Task
}

func (i Item) Title() string {
	if i.Task.DeletedAt != nil {
		return "👻 " + i.Task.Name + " (deleted)"
	}
	return i.Task.Name
}
func (i Item) Description() string {
	desc := fmt.Sprintf("%d min | %s", i.Task.DurationMin, i.Task.Recurrence.Type)
	if i.Task.DeletedAt != nil {
		desc += " | can restore with 'r'"
	}
	return desc
}
func (i Item) FilterValue() string { return i.Task.Name }

type KeyMap struct {
	Add     key.Binding
	Edit    key.Binding
	Delete  key.Binding
	Restore key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Add: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "add"),
		),
		Edit: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete"),
		),
		Restore: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "restore"),
		),
	}
}

type Model struct {
	list list.Model
	keys KeyMap
}

func New(tasks []models.Task, width, height int) Model {
	items := make([]list.Item, len(tasks))
	for i, t := range tasks {
		items[i] = Item{Task: t}
	}

	l := list.New(items, list.NewDefaultDelegate(), width, height)
	l.Title = "Tasks"
	l.SetShowTitle(false)
	l.SetShowHelp(false) // We handle help globally in the main model

	// Add custom keys to list additional short help
	keys := DefaultKeyMap()
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{keys.Add, keys.Edit, keys.Delete, keys.Restore}
	}
	l.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{keys.Add, keys.Edit, keys.Delete, keys.Restore}
	}

	return Model{list: l, keys: keys}
}

func (m *Model) SetTasks(tasks []models.Task) {
	items := make([]list.Item, len(tasks))
	for i, t := range tasks {
		items[i] = Item{Task: t}
	}
	m.list.SetItems(items)
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.list.FilterState() == list.Filtering {
			break
		}
		switch {
		case key.Matches(msg, m.keys.Add):
			return m, func() tea.Msg { return AddTaskMsg{} }
		case key.Matches(msg, m.keys.Edit):
			if i, ok := m.list.SelectedItem().(Item); ok {
				return m, func() tea.Msg { return EditTaskMsg(i) }
			}
		case key.Matches(msg, m.keys.Delete):
			if i, ok := m.list.SelectedItem().(Item); ok {
				return m, func() tea.Msg { return DeleteTaskMsg{ID: i.Task.ID} }
			}
		case key.Matches(msg, m.keys.Restore):
			if i, ok := m.list.SelectedItem().(Item); ok {
				if i.Task.DeletedAt != nil {
					return m, func() tea.Msg { return RestoreTaskMsg{ID: i.Task.ID} }
				}
			}
		}
	}

	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	if len(m.list.Items()) == 0 && m.list.FilterState() != list.Filtering {
		return "\n  No tasks yet.\n  Press 'a' to add one."
	}
	return m.list.View()
}

func (m *Model) SetSize(width, height int) {
	m.list.SetSize(width, height)
}
