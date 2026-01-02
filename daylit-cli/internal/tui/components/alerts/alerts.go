package alerts

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/julianstephens/daylit/daylit-cli/internal/models"
)

type AddAlertMsg struct{}

type DeleteAlertMsg struct {
	ID string
}

type Item struct {
	Alert models.Alert
}

func (i Item) Title() string {
	title := fmt.Sprintf("⏰ %s at %s", i.Alert.Message, i.Alert.Time)
	if !i.Alert.Active {
		title = "[INACTIVE] " + title
	}
	return title
}

func (i Item) Description() string {
	if i.Alert.Date != "" {
		return fmt.Sprintf("One-time: %s", i.Alert.Date)
	}
	return i.Alert.FormatRecurrence()
}

func (i Item) FilterValue() string { return i.Alert.Message }

type KeyMap struct {
	Add    key.Binding
	Delete key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Add: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "add"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete"),
		),
	}
}

type Model struct {
	list list.Model
	keys KeyMap
}

func New(alerts []models.Alert, width, height int) Model {
	items := make([]list.Item, len(alerts))
	for i, a := range alerts {
		items[i] = Item{Alert: a}
	}

	l := list.New(items, list.NewDefaultDelegate(), width, height)
	l.Title = "Alerts"
	l.SetShowTitle(false)
	l.SetShowHelp(false)

	keys := DefaultKeyMap()
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{keys.Add, keys.Delete}
	}
	l.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{keys.Add, keys.Delete}
	}

	return Model{
		list: l,
		keys: keys,
	}
}

func (m *Model) SetAlerts(alerts []models.Alert) {
	items := make([]list.Item, len(alerts))
	for i, a := range alerts {
		items[i] = Item{Alert: a}
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
		// Don't match if we're filtering
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch {
		case key.Matches(msg, m.keys.Add):
			return m, func() tea.Msg { return AddAlertMsg{} }
		case key.Matches(msg, m.keys.Delete):
			if item, ok := m.list.SelectedItem().(Item); ok {
				return m, func() tea.Msg {
					return DeleteAlertMsg{ID: item.Alert.ID}
				}
			}
		}
	}

	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	return m.list.View()
}

func (m *Model) SetSize(width, height int) {
	m.list.SetSize(width, height)
}
