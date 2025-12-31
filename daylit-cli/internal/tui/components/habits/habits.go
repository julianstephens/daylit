package habits

import (
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/julianstephens/daylit/daylit-cli/internal/models"
)

type AddHabitMsg struct{}

type MarkHabitMsg struct {
	ID string
}

type UnmarkHabitMsg struct {
	ID string
}

type ArchiveHabitMsg struct {
	ID string
}

type DeleteHabitMsg struct {
	ID string
}

type RestoreHabitMsg struct {
	ID string
}

type Item struct {
	Habit     models.Habit
	IsMarked  bool
	IsDeleted bool
}

func (i Item) Title() string {
	title := i.Habit.Name
	if i.IsDeleted {
		title = "[DELETED] " + title
	} else if i.Habit.ArchivedAt != nil {
		title = "[ARCHIVED] " + title
	} else if i.IsMarked {
		title = "✓ " + title
	} else {
		title = "○ " + title
	}
	return title
}

func (i Item) Description() string {
	if i.IsDeleted {
		return "can restore with 'r'"
	}
	if i.Habit.ArchivedAt != nil {
		return "archived"
	}
	if i.IsMarked {
		return "completed today"
	}
	return "not completed today"
}

func (i Item) FilterValue() string { return i.Habit.Name }

type KeyMap struct {
	Add     key.Binding
	Mark    key.Binding
	Unmark  key.Binding
	Archive key.Binding
	Delete  key.Binding
	Restore key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Add: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "add"),
		),
		Mark: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "mark done"),
		),
		Unmark: key.NewBinding(
			key.WithKeys("u"),
			key.WithHelp("u", "unmark"),
		),
		Archive: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "archive"),
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
	list         list.Model
	keys         KeyMap
	markedHabits map[string]bool // habitID -> isMarked
	today        string
}

func New(habits []models.Habit, entries []models.HabitEntry, width, height int) Model {
	today := time.Now().Format("2006-01-02")
	markedHabits := make(map[string]bool)
	for _, entry := range entries {
		markedHabits[entry.HabitID] = true
	}

	items := make([]list.Item, len(habits))
	for i, h := range habits {
		isDeleted := h.DeletedAt != nil
		isMarked := markedHabits[h.ID] && !isDeleted && h.ArchivedAt == nil
		items[i] = Item{
			Habit:     h,
			IsMarked:  isMarked,
			IsDeleted: isDeleted,
		}
	}

	l := list.New(items, list.NewDefaultDelegate(), width, height)
	l.Title = "Habits"
	l.SetShowTitle(false)
	l.SetShowHelp(false)

	keys := DefaultKeyMap()
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{keys.Add, keys.Mark, keys.Unmark, keys.Archive, keys.Delete, keys.Restore}
	}
	l.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{keys.Add, keys.Mark, keys.Unmark, keys.Archive, keys.Delete, keys.Restore}
	}

	return Model{
		list:         l,
		keys:         keys,
		markedHabits: markedHabits,
		today:        today,
	}
}

func (m *Model) SetHabits(habits []models.Habit, entries []models.HabitEntry) {
	m.today = time.Now().Format("2006-01-02")
	m.markedHabits = make(map[string]bool)
	for _, entry := range entries {
		m.markedHabits[entry.HabitID] = true
	}

	items := make([]list.Item, len(habits))
	for i, h := range habits {
		isDeleted := h.DeletedAt != nil
		isMarked := m.markedHabits[h.ID] && !isDeleted && h.ArchivedAt == nil
		items[i] = Item{
			Habit:     h,
			IsMarked:  isMarked,
			IsDeleted: isDeleted,
		}
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
			return m, func() tea.Msg { return AddHabitMsg{} }
		case key.Matches(msg, m.keys.Mark):
			if i, ok := m.list.SelectedItem().(Item); ok {
				if !i.IsDeleted && i.Habit.ArchivedAt == nil && !i.IsMarked {
					return m, func() tea.Msg { return MarkHabitMsg{ID: i.Habit.ID} }
				}
			}
		case key.Matches(msg, m.keys.Unmark):
			if i, ok := m.list.SelectedItem().(Item); ok {
				if !i.IsDeleted && i.Habit.ArchivedAt == nil && i.IsMarked {
					return m, func() tea.Msg { return UnmarkHabitMsg{ID: i.Habit.ID} }
				}
			}
		case key.Matches(msg, m.keys.Archive):
			if i, ok := m.list.SelectedItem().(Item); ok {
				if !i.IsDeleted && i.Habit.ArchivedAt == nil {
					return m, func() tea.Msg { return ArchiveHabitMsg{ID: i.Habit.ID} }
				}
			}
		case key.Matches(msg, m.keys.Delete):
			if i, ok := m.list.SelectedItem().(Item); ok {
				if !i.IsDeleted {
					return m, func() tea.Msg { return DeleteHabitMsg{ID: i.Habit.ID} }
				}
			}
		case key.Matches(msg, m.keys.Restore):
			if i, ok := m.list.SelectedItem().(Item); ok {
				if i.IsDeleted {
					return m, func() tea.Msg { return RestoreHabitMsg{ID: i.Habit.ID} }
				}
			}
		}
	}

	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	if len(m.list.Items()) == 0 && m.list.FilterState() != list.Filtering {
		return "\n  No habits yet.\n  Press 'a' to add one."
	}
	return m.list.View()
}

func (m *Model) SetSize(width, height int) {
	m.list.SetSize(width, height)
}
