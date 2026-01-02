package handlers

import (
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/julianstephens/daylit/daylit-cli/internal/constants"
	"github.com/julianstephens/daylit/daylit-cli/internal/models"
	"github.com/julianstephens/daylit/daylit-cli/internal/tui/components/settings"
	"github.com/julianstephens/daylit/daylit-cli/internal/tui/state"
)

// HandleEditSettingsState handles the edit settings state
func HandleEditSettingsState(m *state.Model, msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd

	if msg, ok := msg.(tea.KeyMsg); ok && msg.Type == tea.KeyEsc {
		m.FormError = "" // Clear error on cancel
		m.State = constants.StateSettings
		return nil
	}

	form, cmd := m.Form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.Form = f
	}
	cmds = append(cmds, cmd)

	switch m.Form.State {
	case huh.StateCompleted:
		// Save settings
		// Convert form model to models.Settings
		newSettings := models.Settings{
			DayStart:             m.SettingsForm.DayStart,
			DayEnd:               m.SettingsForm.DayEnd,
			Timezone:             m.SettingsForm.Timezone,
			NotificationsEnabled: m.SettingsForm.NotificationsEnabled,
			NotifyBlockStart:     m.SettingsForm.NotifyBlockStart,
			NotifyBlockEnd:       m.SettingsForm.NotifyBlockEnd,
		}

		if val, err := strconv.Atoi(m.SettingsForm.DefaultBlockMin); err == nil {
			newSettings.DefaultBlockMin = val
		}
		if val, err := strconv.Atoi(m.SettingsForm.BlockStartOffsetMin); err == nil {
			newSettings.BlockStartOffsetMin = val
		}
		if val, err := strconv.Atoi(m.SettingsForm.BlockEndOffsetMin); err == nil {
			newSettings.BlockEndOffsetMin = val
		}

		if err := m.Store.SaveSettings(newSettings); err != nil {
			// Store error and stay in form state to allow retry
			m.FormError = "Failed to update settings: " + err.Error()
			m.Form.State = huh.StateNormal
			return tea.Batch(cmds...)
		}
		m.FormError = "" // Clear any previous errors
		m.State = constants.StateSettings
	case huh.StateAborted:
		m.FormError = "" // Clear error on abort
		m.State = constants.StateSettings
	}
	return tea.Batch(cmds...)
}

// HandleSettingsMessages handles messages from the settings component
func HandleSettingsMessages(m *state.Model, msg tea.Msg) (bool, tea.Cmd) {
	switch msg.(type) {
	case settings.EditSettingsMsg:
		currentSettings, err := m.Store.GetSettings()
		if err != nil {
			m.FormError = "Failed to load settings: " + err.Error()
			// Initialize with defaults if loading fails
			currentSettings = models.Settings{
				DayStart:             "08:00",
				DayEnd:               "18:00",
				DefaultBlockMin:      30,
				NotificationsEnabled: true,
				NotifyBlockStart:     true,
				NotifyBlockEnd:       true,
				BlockStartOffsetMin:  5,
				BlockEndOffsetMin:    0,
				Timezone:             "Local",
			}
		} else {
			m.FormError = ""
		}

		m.SettingsForm = &state.SettingsFormModel{
			DayStart:             currentSettings.DayStart,
			DayEnd:               currentSettings.DayEnd,
			DefaultBlockMin:      strconv.Itoa(currentSettings.DefaultBlockMin),
			Timezone:             currentSettings.Timezone,
			NotificationsEnabled: currentSettings.NotificationsEnabled,
			NotifyBlockStart:     currentSettings.NotifyBlockStart,
			NotifyBlockEnd:       currentSettings.NotifyBlockEnd,
			BlockStartOffsetMin:  strconv.Itoa(currentSettings.BlockStartOffsetMin),
			BlockEndOffsetMin:    strconv.Itoa(currentSettings.BlockEndOffsetMin),
		}
		m.Form = NewSettingsForm(m.SettingsForm)
		m.State = constants.StateEditSettings
		return true, m.Form.Init()
	}
	return false, nil
}
