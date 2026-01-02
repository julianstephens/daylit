package settings

import (
	"fmt"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/julianstephens/daylit/daylit-cli/internal/models"
	"github.com/julianstephens/daylit/daylit-cli/internal/storage"
)

type EditSettingsMsg struct{}

type Model struct {
	settings   storage.Settings
	otSettings models.OTSettings
	width      int
	height     int
	viewport   viewport.Model
}

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			MarginBottom(1)

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Width(25)

	valueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")).
			Bold(true)

	sectionStyle = lipgloss.NewStyle().
			MarginTop(1).
			MarginBottom(1)
)

func New(settings storage.Settings, otSettings models.OTSettings, width, height int) Model {
	m := Model{
		settings:   settings,
		otSettings: otSettings,
		width:      width,
		height:     height,
		viewport:   viewport.New(width, height),
	}
	m.updateViewportContent()
	return m
}

func (m *Model) SetSettings(settings storage.Settings, otSettings models.OTSettings) {
	m.settings = settings
	m.otSettings = otSettings
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
		case "e":
			return m, func() tea.Msg { return EditSettingsMsg{} }
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

	// General Settings
	generalTitle := titleStyle.Render("General Settings")
	generalContent := lipgloss.JoinVertical(
		lipgloss.Left,
		fmt.Sprintf("%s %s", labelStyle.Render("Day Start:"), valueStyle.Render(m.settings.DayStart)),
		fmt.Sprintf("%s %s", labelStyle.Render("Day End:"), valueStyle.Render(m.settings.DayEnd)),
		fmt.Sprintf("%s %s", labelStyle.Render("Default Block (min):"), valueStyle.Render(fmt.Sprintf("%d", m.settings.DefaultBlockMin))),
		fmt.Sprintf("%s %s", labelStyle.Render("Timezone:"), valueStyle.Render(m.settings.Timezone)),
	)
	sections = append(sections, sectionStyle.Render(generalTitle+"\n"+generalContent))

	// Once Today Settings
	otTitle := titleStyle.Render("Once Today Settings")
	otContent := lipgloss.JoinVertical(
		lipgloss.Left,
		fmt.Sprintf("%s %s", labelStyle.Render("Prompt On Empty:"), valueStyle.Render(fmt.Sprintf("%t", m.otSettings.PromptOnEmpty))),
		fmt.Sprintf("%s %s", labelStyle.Render("Strict Mode:"), valueStyle.Render(fmt.Sprintf("%t", m.otSettings.StrictMode))),
		fmt.Sprintf("%s %s", labelStyle.Render("Default Log Days:"), valueStyle.Render(fmt.Sprintf("%d", m.otSettings.DefaultLogDays))),
	)
	sections = append(sections, sectionStyle.Render(otTitle+"\n"+otContent))

	// Notification Settings
	notifTitle := titleStyle.Render("Notification Settings")
	notifContent := lipgloss.JoinVertical(
		lipgloss.Left,
		fmt.Sprintf("%s %s", labelStyle.Render("Enabled:"), valueStyle.Render(fmt.Sprintf("%t", m.settings.NotificationsEnabled))),
		fmt.Sprintf("%s %s", labelStyle.Render("Notify Block Start:"), valueStyle.Render(fmt.Sprintf("%t", m.settings.NotifyBlockStart))),
		fmt.Sprintf("%s %s", labelStyle.Render("Start Offset (min):"), valueStyle.Render(fmt.Sprintf("%d", m.settings.BlockStartOffsetMin))),
		fmt.Sprintf("%s %s", labelStyle.Render("Notify Block End:"), valueStyle.Render(fmt.Sprintf("%t", m.settings.NotifyBlockEnd))),
		fmt.Sprintf("%s %s", labelStyle.Render("End Offset (min):"), valueStyle.Render(fmt.Sprintf("%d", m.settings.BlockEndOffsetMin))),
	)
	sections = append(sections, sectionStyle.Render(notifTitle+"\n"+notifContent))

	// Help text
	helpText := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Italic(true).
		MarginTop(2).
		Render("Press 'e' to edit settings")

	sections = append(sections, helpText)

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)
	m.viewport.SetContent(lipgloss.NewStyle().Padding(0, 2).Render(content))
}
