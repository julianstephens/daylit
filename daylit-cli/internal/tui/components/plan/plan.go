package plan

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/julianstephens/daylit/daylit-cli/internal/models"
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

	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Bold(true)
)

type Model struct {
	viewport       viewport.Model
	Plan           *models.DayPlan
	Tasks          map[string]models.Task
	LatestRevision int // Track the latest revision number for warning display
	width          int
	height         int
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
	// By default, assume the current plan's revision is the latest known for this view.
	// Callers can override this via SetLatestRevision when they know of a newer revision.
	m.LatestRevision = plan.Revision
	for _, t := range tasks {
		m.Tasks[t.ID] = t
	}
	m.Render()
}

// SetLatestRevision updates the latest revision number for warning display
func (m *Model) SetLatestRevision(latestRev int) {
	m.LatestRevision = latestRev
	m.Render()
}

func (m *Model) Render() {
	if m.Plan == nil {
		m.viewport.SetContent("No plan loaded.")
		return
	}

	var b strings.Builder

	// Add revision badge at the top
	revisionText := fmt.Sprintf("Revision %d", m.Plan.Revision)
	if m.LatestRevision > 0 && m.Plan.Revision < m.LatestRevision {
		// Viewing an older revision - show warning
		revisionText += warningStyle.Render(fmt.Sprintf(" ⚠ Not latest (Rev %d available)", m.LatestRevision))
	}
	b.WriteString(revisionText + "\n\n")

	for _, slot := range m.Plan.Slots {
		taskName := "Unknown Task"
		taskDeleted := false
		if t, ok := m.Tasks[slot.TaskID]; ok {
			taskName = t.Name
			if t.DeletedAt != nil {
				taskDeleted = true
			}
		}

		timeStr := fmt.Sprintf("%s - %s", slot.Start, slot.End)

		// Add indicator for deleted tasks
		displayName := taskName
		if taskDeleted {
			displayName = "[DELETED] " + taskName
		}

		line := fmt.Sprintf("%s %s %s\n",
			timeStyle.Render(timeStr),
			taskStyle.Render(displayName),
			statusStyle.Render(string(slot.Status)),
		)
		b.WriteString(line)
	}
	m.viewport.SetContent(b.String())
}
