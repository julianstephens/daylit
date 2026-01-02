package handlers

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/google/uuid"

	"github.com/julianstephens/daylit/daylit-cli/internal/constants"
	"github.com/julianstephens/daylit/daylit-cli/internal/models"
	"github.com/julianstephens/daylit/daylit-cli/internal/tui/components/ot"
	"github.com/julianstephens/daylit/daylit-cli/internal/tui/state"
)

// HandleEditOTState handles the edit OT state
func HandleEditOTState(m *state.Model, msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd

	if msg, ok := msg.(tea.KeyMsg); ok && msg.Type == tea.KeyEsc {
		m.FormError = "" // Clear error on cancel
		m.State = constants.StateOT
		return nil
	}

	form, cmd := m.Form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.Form = f
	}
	cmds = append(cmds, cmd)

	switch m.Form.State {
	case huh.StateCompleted:
		// Save or update OT entry
		today := time.Now().Format(constants.DateFormat)

		// Trim whitespace from title and note
		title := strings.TrimSpace(m.OTForm.Title)
		note := strings.TrimSpace(m.OTForm.Note)

		// Check if entry exists for today
		existingEntry, err := m.Store.GetOTEntry(today)
		if err == nil {
			// Update existing entry
			existingEntry.Title = title
			existingEntry.Note = note
			existingEntry.UpdatedAt = time.Now()
			if err := m.Store.UpdateOTEntry(existingEntry); err != nil {
				// Store error and stay in form state to allow retry
				m.FormError = fmt.Sprintf("Failed to update OT: %v", err)
				m.Form.State = huh.StateNormal
				return tea.Batch(cmds...)
			}
			// Reload entry from storage to get a fresh copy
			updatedEntry, err := m.Store.GetOTEntry(today)
			if err != nil {
				// Fallback to using the data we just saved if reload fails
				m.OTModel.SetEntry(&existingEntry)
			} else {
				m.OTModel.SetEntry(&updatedEntry)
			}
		} else {
			// Create new entry
			newEntry := models.OTEntry{
				ID:        uuid.New().String(),
				Day:       today,
				Title:     title,
				Note:      note,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			if err := m.Store.AddOTEntry(newEntry); err != nil {
				// Store error and stay in form state to allow retry
				m.FormError = fmt.Sprintf("Failed to create OT: %v", err)
				m.Form.State = huh.StateNormal
				return tea.Batch(cmds...)
			}
			// Reload entry from storage to get a fresh copy
			savedEntry, err := m.Store.GetOTEntry(today)
			if err != nil {
				// Fallback to using the data we just created if reload fails
				m.OTModel.SetEntry(&newEntry)
			} else {
				m.OTModel.SetEntry(&savedEntry)
			}
		}
		m.FormError = "" // Clear any previous errors
		m.State = constants.StateOT
	case huh.StateAborted:
		m.FormError = "" // Clear error on abort
		m.State = constants.StateOT
	}
	return tea.Batch(cmds...)
}

// HandleOTMessages handles messages from the OT component
func HandleOTMessages(m *state.Model, msg tea.Msg) (bool, tea.Cmd) {
	switch msg.(type) {
	case ot.EditOTMsg:
		today := time.Now().Format(constants.DateFormat)
		existingEntry, err := m.Store.GetOTEntry(today)

		// Handle database errors differently from "not found"
		if err != nil {
			// Check if it's a "not found" error (sql.ErrNoRows)
			if err == sql.ErrNoRows {
				// Entry not found - initialize with empty values
				existingEntry = models.OTEntry{}
			} else {
				// Actual database error - show error to user
				m.FormError = fmt.Sprintf("Error loading OT: %v", err)
				// Still allow editing with empty form
				existingEntry = models.OTEntry{}
			}
		} else {
			// Clear any previous form errors only if no error occurred
			m.FormError = ""
		}

		m.OTForm = &state.OTFormModel{
			Title: existingEntry.Title,
			Note:  existingEntry.Note,
		}
		m.Form = NewOTForm(m.OTForm)
		m.State = constants.StateEditOT
		return true, m.Form.Init()
	}
	return false, nil
}
