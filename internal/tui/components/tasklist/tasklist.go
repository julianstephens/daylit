package tasklist

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/julianstephens/daylit/internal/models"
)

type Item struct {
	Task models.Task
}

func (i Item) Title() string { return i.Task.Name }
func (i Item) Description() string {
	return fmt.Sprintf("%d min | %s", i.Task.DurationMin, i.Task.Recurrence.Type)
}
func (i Item) FilterValue() string { return i.Task.Name }

type Model struct {
	list list.Model
}

func New(tasks []models.Task, width, height int) Model {
	items := make([]list.Item, len(tasks))
	for i, t := range tasks {
		items[i] = Item{Task: t}
	}

	l := list.New(items, list.NewDefaultDelegate(), width, height)
	l.Title = "Tasks"
	l.SetShowHelp(false) // We handle help globally in the main model

	return Model{list: l}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	return m.list.View()
}

func (m *Model) SetSize(width, height int) {
	m.list.SetSize(width, height)
}
