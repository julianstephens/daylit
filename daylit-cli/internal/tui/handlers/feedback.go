package handlers

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/julianstephens/daylit/daylit-cli/internal/constants"
	"github.com/julianstephens/daylit/daylit-cli/internal/models"
	"github.com/julianstephens/daylit/daylit-cli/internal/tui/state"
	"github.com/julianstephens/daylit/daylit-cli/internal/utils"
)

// HandleFeedbackState handles the feedback state using key-based rating system
func HandleFeedbackState(m *state.Model, msg tea.Msg) tea.Cmd {
	if msg, ok := msg.(tea.KeyMsg); ok {
		var rating models.FeedbackRating
		switch msg.String() {
		case "1":
			rating = constants.FeedbackOnTrack
		case "2":
			rating = constants.FeedbackTooMuch
		case "3":
			rating = constants.FeedbackUnnecessary
		case "q", "esc":
			m.State = m.PreviousState
			return nil
		default:
			return nil
		}

		// Apply feedback
		today := time.Now().Format(constants.DateFormat)
		plan, err := m.Store.GetPlan(today)
		if err == nil && m.FeedbackSlotID >= 0 && m.FeedbackSlotID < len(plan.Slots) {
			slot := &plan.Slots[m.FeedbackSlotID]
			slot.Feedback = &models.Feedback{
				Rating: rating,
			}
			slot.Status = constants.SlotStatusDone

			// Save plan first to ensure feedback is persisted
			if err := m.Store.SavePlan(plan); err != nil {
				// On error, revert to previous state
				m.State = m.PreviousState
				return nil
			}

			// Update task stats only after plan is saved
			task, err := m.Store.GetTask(slot.TaskID)
			if err == nil {
				switch rating {
				case constants.FeedbackOnTrack:
					slotDuration := CalculateSlotDuration(*slot)
					if slotDuration > 0 {
						if task.AvgActualDurationMin <= 0 {
							task.AvgActualDurationMin = float64(slotDuration)
						} else {
							task.AvgActualDurationMin = (task.AvgActualDurationMin * constants.FeedbackExistingWeight) + (float64(slotDuration) * constants.FeedbackNewWeight)
						}
					}
				case constants.FeedbackTooMuch:
					task.DurationMin = int(float64(task.DurationMin) * constants.FeedbackTooMuchReductionFactor)
					if task.DurationMin < constants.MinTaskDurationMin {
						task.DurationMin = constants.MinTaskDurationMin
					}
				case constants.FeedbackUnnecessary:
					if task.Recurrence.Type == constants.RecurrenceNDays {
						task.Recurrence.IntervalDays++
					}
				}
				task.LastDone = today
				task.SuccessStreak++
				// Ignore task update errors to avoid inconsistency if it fails after plan save
				m.Store.UpdateTask(task)
			}

			// Refresh views
			tasks, err := m.Store.GetAllTasks()
			if err != nil {
				// On error, revert to previous state
				m.State = m.PreviousState
				return nil
			}
			tasksIncludingDeleted, _ := m.Store.GetAllTasksIncludingDeleted()
			m.PlanModel.SetPlan(plan, tasks)
			m.NowModel.SetPlan(plan, tasks)
			m.TaskList.SetTasks(tasksIncludingDeleted)
			m.UpdateValidationStatus()
		}

		m.State = m.PreviousState
		return nil
	}
	return nil
}

// HandleFeedbackMessages handles messages related to feedback
func HandleFeedbackMessages(m *state.Model, msg tea.Msg) (bool, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		if msg.String() == "f" {
			// Find slot for feedback
			today := time.Now().Format(constants.DateFormat)
			plan, err := m.Store.GetPlan(today)
			if err == nil {
				now := time.Now()
				currentMinutes := now.Hour()*60 + now.Minute()
				targetSlotIdx := -1

				for i := len(plan.Slots) - 1; i >= 0; i-- {
					slot := &plan.Slots[i]
					if (slot.Status == constants.SlotStatusAccepted || slot.Status == constants.SlotStatusDone) &&
						slot.Feedback == nil {
						endMinutes, err := utils.ParseTimeToMinutes(slot.End)
						if err == nil && endMinutes <= currentMinutes {
							targetSlotIdx = i
							break
						}
					}
				}

				if targetSlotIdx != -1 {
					m.PreviousState = m.State
					m.State = constants.StateFeedback
					m.FeedbackSlotID = targetSlotIdx
					return true, nil
				}
			}
		}
	}
	return false, nil
}
