package now

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/julianstephens/daylit/daylit-cli/internal/models"
)

var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true).
			Padding(1, 2).
			Align(lipgloss.Center)

	timeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Padding(0, 1)

	taskNameStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Bold(true).
			Padding(1, 0).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Width(40).
			Align(lipgloss.Center)
)

type Model struct {
	Plan   *models.DayPlan
	Tasks  map[string]models.Task
	Time   time.Time
	width  int
	height int
}

func New() Model {
	return Model{
		Tasks: make(map[string]models.Task),
		Time:  time.Now(),
	}
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

type TickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

func (m Model) Init() tea.Cmd {
	return tick()
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case TickMsg:
		m.Time = time.Time(msg)
		return m, tick()
	}
	return m, nil
}

func (m Model) View() string {
	if m.Plan == nil {
		return titleStyle.Render("No plan for today.")
	}

	currentSlot := m.getCurrentSlot()

	var content string
	if currentSlot == nil {
		content = "Free time"
	} else {
		taskName := "Unknown Task"
		if t, ok := m.Tasks[currentSlot.TaskID]; ok {
			taskName = t.Name
		}

		content = lipgloss.JoinVertical(lipgloss.Center,
			timeStyle.Render(fmt.Sprintf("%s - %s", currentSlot.Start, currentSlot.End)),
			taskNameStyle.Render(taskName),
			lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(string(currentSlot.Status)),
		)
	}

	content = lipgloss.JoinVertical(lipgloss.Center,
		titleStyle.Render(fmt.Sprintf("Now: %02d:%02d", m.Time.Hour(), m.Time.Minute())),
		content,
	)

	if m.width > 0 && m.height > 0 {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
	}
	return content
}

func (m *Model) SetPlan(plan models.DayPlan, tasks []models.Task) {
	m.Plan = &plan
	for _, t := range tasks {
		m.Tasks[t.ID] = t
	}
}

func (m Model) getCurrentSlot() *models.Slot {
	if m.Plan == nil {
		return nil
	}

	currentMinutes := m.Time.Hour()*60 + m.Time.Minute()

	for i := range m.Plan.Slots {
		slot := &m.Plan.Slots[i]

		startMinutes, err := parseTimeToMinutes(slot.Start)
		if err != nil {
			continue
		}
		endMinutes, err := parseTimeToMinutes(slot.End)
		if err != nil {
			continue
		}

		if startMinutes <= currentMinutes && currentMinutes < endMinutes {
			return slot
		}
	}
	return nil
}

func parseTimeToMinutes(t string) (int, error) {
	parts := strings.Split(t, ":")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid time format")
	}
	h, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, err
	}
	m, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, err
	}
	return h*60 + m, nil
}
