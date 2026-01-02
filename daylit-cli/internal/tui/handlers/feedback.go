package handlers

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/julianstephens/daylit/daylit-cli/internal/constants"
	"github.com/julianstephens/daylit/daylit-cli/internal/tui/state"
)

// HandleFeedbackState handles the feedback state
func HandleFeedbackState(m *state.Model, msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd

	if msg, ok := msg.(tea.KeyMsg); ok && msg.Type == tea.KeyEsc {
		m.FormError = "" // Clear error on cancel
		m.State = constants.StateTasks
		return nil
	}

	form, cmd := m.Form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.Form = f
	}
	cmds = append(cmds, cmd)

	switch m.Form.State {
	case huh.StateCompleted:
		// Save feedback
		if m.PlanModel.Plan != nil && m.FeedbackSlotID >= 0 && m.FeedbackSlotID < len(m.PlanModel.Plan.Slots) {
			slot := m.PlanModel.Plan.Slots[m.FeedbackSlotID]
			err := m.Store.UpdateSlotFeedback(
				m.PlanModel.Plan.Date,
				m.PlanModel.Plan.Revision,
				slot.Start,
				slot.TaskID,
				m.FeedbackForm.Rating,
				m.FeedbackForm.Comment,
			)
			if err != nil {
				// Store error and stay in form state to allow retry
				m.FormError = "Failed to save feedback: " + err.Error()
				m.Form.State = huh.StateNormal
				return tea.Batch(cmds...)
			}
		}
		m.FormError = "" // Clear any previous errors
		m.State = constants.StateTasks
	case huh.StateAborted:
		m.FormError = "" // Clear error on abort
		m.State = constants.StateTasks
	}
	return tea.Batch(cmds...)
}

// HandleFeedbackMessages handles messages related to feedback
func HandleFeedbackMessages(m *state.Model, msg tea.Msg) (bool, tea.Cmd) {
	switch msg.(type) {
	case constants.FeedbackMsg:
		m.FeedbackForm = &state.FeedbackFormModel{}
		m.Form = NewFeedbackForm(m.FeedbackForm)
		m.State = constants.StateFeedback
		return true, m.Form.Init()
	}
	return false, nil
}
