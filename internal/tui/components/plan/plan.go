package plan

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/julianstephens/daylit/internal/models"
)

var (
	timeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Width(12)

	taskStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Bold(true)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true)
)

type Model struct {
	viewport viewport.Model
	Plan     *models.DayPlan
	Tasks    map[string]models.Task
	width    int
	height   int
}

func New(width, height int) Model {
	vp := viewport.New(width, height)
	return Model{
		viewport: vp,
		Tasks:    make(map[string]models.Task),
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	if m.Plan == nil {
		return "No plan for today. Press 'g' to generate."
	}
	return m.viewport.View()
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.viewport.Width = width
	m.viewport.Height = height
	m.Render()
}

func (m *Model) SetPlan(plan models.DayPlan, tasks []models.Task) {
	m.Plan = &plan
	for _, t := range tasks {
		m.Tasks[t.ID] = t
	}
	m.Render()
}

func (m *Model) Render() {
	if m.Plan == nil {
		m.viewport.SetContent("No plan loaded.")
		return
	}

	var b strings.Builder
	for _, slot := range m.Plan.Slots {
		taskName := "Unknown Task"
		if t, ok := m.Tasks[slot.TaskID]; ok {
			taskName = t.Name
		}

		timeStr := fmt.Sprintf("%s - %s", slot.Start, slot.End)

		line := fmt.Sprintf("%s %s %s\n",
			timeStyle.Render(timeStr),
			taskStyle.Render(taskName),
			statusStyle.Render(string(slot.Status)),
		)
		b.WriteString(line)
	}
	m.viewport.SetContent(b.String())
}
