package ot

import (
	"fmt"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/julianstephens/daylit/daylit-cli/internal/models"
)

type EditOTMsg struct{}

type Model struct {
	entry    *models.OTEntry
	width    int
	height   int
	viewport viewport.Model
}

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			MarginBottom(1)

	otTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Bold(true).
			MarginBottom(1)

	noteStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true).
			MarginTop(1)

	emptyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true)

	sectionStyle = lipgloss.NewStyle().
			MarginTop(1).
			MarginBottom(1)
)

func New(entry *models.OTEntry, width, height int) Model {
	m := Model{
		entry:    entry,
		width:    width,
		height:   height,
		viewport: viewport.New(width, height),
	}
	m.updateViewportContent()
	return m
}

func (m *Model) SetEntry(entry *models.OTEntry) {
	m.entry = entry
	m.updateViewportContent()
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "e", "s":
			return m, func() tea.Msg { return EditOTMsg{} }
		}
	}
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	if m.width == 0 {
		return ""
	}
	return m.viewport.View()
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.viewport.Width = width
	m.viewport.Height = height
	m.updateViewportContent()
}

func (m *Model) updateViewportContent() {
	var sections []string

	// Title
	headerTitle := titleStyle.Render("One Thing (OT)")
	sections = append(sections, headerTitle)

	// OT Content
	if m.entry == nil {
		emptyMessage := emptyStyle.Render("No One Thing set for today.")
		sections = append(sections, sectionStyle.Render(emptyMessage))
	} else {
		otTitle := otTitleStyle.Render(m.entry.Title)
		sections = append(sections, sectionStyle.Render(otTitle))

		if m.entry.Note != "" {
			note := noteStyle.Render(fmt.Sprintf("Note: %s", m.entry.Note))
			sections = append(sections, note)
		}
	}

	// Help text
	helpText := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Italic(true).
		MarginTop(2).
		Render("Press 'e' or 's' to set/edit your One Thing")

	sections = append(sections, helpText)

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)
	m.viewport.SetContent(lipgloss.NewStyle().Padding(0, 2).Render(content))
}
