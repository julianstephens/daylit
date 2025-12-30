package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/google/uuid"

	"github.com/julianstephens/daylit/internal/constants"
	"github.com/julianstephens/daylit/internal/models"
	"github.com/julianstephens/daylit/internal/tui/components/tasklist"
)

func newEditForm(fm *TaskFormModel) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Name").
				Value(&fm.Name),
			huh.NewInput().
				Title("Duration (min)").
				Value(&fm.Duration).
				Validate(func(s string) error {
					i, err := strconv.Atoi(s)
					if err != nil {
						return err
					}
					if i <= 0 {
						return fmt.Errorf("duration must be a positive number of minutes")
					}
					return nil
				}),
			huh.NewSelect[models.RecurrenceType]().
				Title("Recurrence").
				Options(
					huh.NewOption("Ad-hoc", models.RecurrenceAdHoc),
					huh.NewOption("Daily", models.RecurrenceDaily),
					huh.NewOption("Weekly", models.RecurrenceWeekly),
					huh.NewOption("Every N Days", models.RecurrenceNDays),
				).
				Value(&fm.Recurrence),
			huh.NewInput().
				Title("Interval (days)").
				Description("For 'Every N Days' recurrence").
				Value(&fm.Interval).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return nil
					}
					i, err := strconv.Atoi(s)
					if err != nil {
						return err
					}
					if i <= 0 {
						return fmt.Errorf("interval must be a positive number of days")
					}
					return nil
				}),
			huh.NewInput().
				Title("Priority (1-5)").
				Value(&fm.Priority).
				Validate(func(s string) error {
					i, err := strconv.Atoi(s)
					if err != nil {
						return err
					}
					if i < 1 || i > 5 {
						return fmt.Errorf("priority must be 1-5")
					}
					return nil
				}),
			huh.NewConfirm().
				Title("Active").
				Value(&fm.Active),
		),
	).WithTheme(huh.ThemeDracula())
}

func parseTimeToMinutes(timeStr string) (int, error) {
	parts := strings.Split(timeStr, ":")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid time format: %q", timeStr)
	}
	hour, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("invalid hour in %q: %w", timeStr, err)
	}
	minute, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, fmt.Errorf("invalid minute in %q: %w", timeStr, err)
	}
	return hour*60 + minute, nil
}

func calculateSlotDuration(slot models.Slot) int {
	start, err := time.Parse("15:04", slot.Start)
	if err != nil {
		return 0
	}
	end, err := time.Parse("15:04", slot.End)
	if err != nil {
		return 0
	}
	return int(end.Sub(start).Minutes())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Handle Editing State
	if m.state == StateEditing {
		if msg, ok := msg.(tea.KeyMsg); ok && msg.Type == tea.KeyEsc {
			m.state = StateTasks
			return m, nil
		}

		form, cmd := m.form.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.form = f
		}
		cmds = append(cmds, cmd)

		switch m.form.State {
		case huh.StateCompleted:
			// Apply changes
			m.editingTask.Name = m.taskForm.Name
			dur, err := strconv.Atoi(m.taskForm.Duration)
			if err == nil {
				m.editingTask.DurationMin = dur
			}
			m.editingTask.Recurrence.Type = m.taskForm.Recurrence
			intervalStr := strings.TrimSpace(m.taskForm.Interval)
			interval, err := strconv.Atoi(intervalStr)
			if err == nil {
				m.editingTask.Recurrence.IntervalDays = interval
			}
			prio, err := strconv.Atoi(m.taskForm.Priority)
			if err == nil {
				m.editingTask.Priority = prio
			}
			m.editingTask.Active = m.taskForm.Active

			// Check if task exists to decide Add vs Update
			_, err = m.store.GetTask(m.editingTask.ID)
			var saveErr error
			if err != nil {
				// Task doesn't exist, add it
				saveErr = m.store.AddTask(*m.editingTask)
			} else {
				// Task exists, update it
				saveErr = m.store.UpdateTask(*m.editingTask)
			}

			// Only update task list if save was successful
			if saveErr == nil {
				tasks, err := m.store.GetAllTasks()
				if err == nil {
					m.taskList.SetTasks(tasks)
				}
			}
			m.state = StateTasks
		case huh.StateAborted:
			m.state = StateTasks
		}
		return m, tea.Batch(cmds...)
	}

	// Handle Feedback State
	if m.state == StateFeedback {
		if msg, ok := msg.(tea.KeyMsg); ok {
			var rating models.FeedbackRating
			switch msg.String() {
			case "1":
				rating = models.FeedbackOnTrack
			case "2":
				rating = models.FeedbackTooMuch
			case "3":
				rating = models.FeedbackUnnecessary
			case "q", "esc":
				m.state = m.previousState
				return m, nil
			default:
				return m, nil
			}

			// Apply feedback
			today := time.Now().Format("2006-01-02")
			plan, err := m.store.GetPlan(today)
			if err == nil && m.feedbackSlotID >= 0 && m.feedbackSlotID < len(plan.Slots) {
				slot := &plan.Slots[m.feedbackSlotID]
				slot.Feedback = &models.Feedback{
					Rating: rating,
				}
				slot.Status = models.SlotStatusDone

				// Save plan first to ensure feedback is persisted
				if err := m.store.SavePlan(plan); err != nil {
					// On error, revert to previous state
					m.state = m.previousState
					return m, nil
				}

				// Update task stats only after plan is saved
				task, err := m.store.GetTask(slot.TaskID)
				if err == nil {
					switch rating {
					case models.FeedbackOnTrack:
						slotDuration := calculateSlotDuration(*slot)
						if slotDuration > 0 {
							if task.AvgActualDurationMin <= 0 {
								task.AvgActualDurationMin = float64(slotDuration)
							} else {
								task.AvgActualDurationMin = (task.AvgActualDurationMin * constants.FeedbackExistingWeight) + (float64(slotDuration) * constants.FeedbackNewWeight)
							}
						}
					case models.FeedbackTooMuch:
						task.DurationMin = int(float64(task.DurationMin) * constants.FeedbackTooMuchReductionFactor)
						if task.DurationMin < constants.MinTaskDurationMin {
							task.DurationMin = constants.MinTaskDurationMin
						}
					case models.FeedbackUnnecessary:
						if task.Recurrence.Type == models.RecurrenceNDays {
							task.Recurrence.IntervalDays++
						}
					}
					task.LastDone = today
					task.SuccessStreak++
					// Ignore task update errors to avoid inconsistency if it fails after plan save
					m.store.UpdateTask(task)
				}

				// Refresh views
				tasks, err := m.store.GetAllTasks()
				if err != nil {
					// On error, revert to previous state
					m.state = m.previousState
					return m, nil
				}
				m.planModel.SetPlan(plan, tasks)
				m.nowModel.SetPlan(plan, tasks)
				m.taskList.SetTasks(tasks)
			}

			m.state = m.previousState
			return m, nil
		}
		return m, nil
	}

	// Handle Confirm Delete State
	if m.state == StateConfirmDelete {
		if msg, ok := msg.(tea.KeyMsg); ok {
			switch msg.String() {
			case "y", "Y":
				if err := m.store.DeleteTask(m.taskToDeleteID); err != nil {
					// On error, silently return to tasks view
					m.state = StateTasks
					m.taskToDeleteID = ""
					return m, nil
				}
				// Deletion succeeded - always refresh and clear state
				tasks, err := m.store.GetAllTasks()
				if err == nil {
					m.taskList.SetTasks(tasks)
				}
				m.state = StateTasks
				m.taskToDeleteID = ""
			case "n", "N", "esc", "q":
				m.state = StateTasks
				m.taskToDeleteID = ""
			}
		}
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
		// Adjust height for tabs and help
		listHeight := msg.Height - 4 // Approximate height for tabs + help

		h, v := docStyle.GetFrameSize()
		m.taskList.SetSize(msg.Width-h, listHeight-v)
		m.planModel.SetSize(msg.Width-h, listHeight-v)
		m.nowModel.SetSize(msg.Width, listHeight)

	case tasklist.DeleteTaskMsg:
		m.taskToDeleteID = msg.ID
		m.state = StateConfirmDelete
		return m, nil

	case tasklist.AddTaskMsg:
		task := models.Task{
			ID:          uuid.New().String(),
			Name:        "New Task",
			Kind:        models.TaskKindFlexible,
			DurationMin: 30,
			Recurrence: models.Recurrence{
				Type: models.RecurrenceAdHoc,
			},
			Priority: 3,
			Active:   true,
		}
		m.editingTask = &task
		m.taskForm = &TaskFormModel{
			Name:       task.Name,
			Duration:   strconv.Itoa(task.DurationMin),
			Recurrence: task.Recurrence.Type,
			Interval:   strconv.Itoa(task.Recurrence.IntervalDays),
			Priority:   strconv.Itoa(task.Priority),
			Active:     task.Active,
		}
		m.form = newEditForm(m.taskForm)
		m.state = StateEditing
		return m, m.form.Init()

	case tasklist.EditTaskMsg:
		m.editingTask = &msg.Task
		m.taskForm = &TaskFormModel{
			Name:       msg.Task.Name,
			Duration:   strconv.Itoa(msg.Task.DurationMin),
			Recurrence: msg.Task.Recurrence.Type,
			Interval:   strconv.Itoa(msg.Task.Recurrence.IntervalDays),
			Priority:   strconv.Itoa(msg.Task.Priority),
			Active:     msg.Task.Active,
		}
		m.form = newEditForm(m.taskForm)
		m.state = StateEditing
		return m, m.form.Init()

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			m.quitting = true
			return m, tea.Quit
		case key.Matches(msg, m.keys.Tab, m.keys.Right):
			m.state = (m.state + 1) % 3 // Only cycle through main 3 tabs
			return m, nil
		case key.Matches(msg, m.keys.ShiftTab, m.keys.Left):
			m.state = (m.state - 1 + 3) % 3
			return m, nil
		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
			return m, nil
		case key.Matches(msg, m.keys.Feedback):
			// Find slot for feedback
			today := time.Now().Format("2006-01-02")
			plan, err := m.store.GetPlan(today)
			if err == nil {
				now := time.Now()
				currentMinutes := now.Hour()*60 + now.Minute()
				targetSlotIdx := -1

				for i := len(plan.Slots) - 1; i >= 0; i-- {
					slot := &plan.Slots[i]
					if (slot.Status == models.SlotStatusAccepted || slot.Status == models.SlotStatusDone) &&
						slot.Feedback == nil {
						endMinutes, err := parseTimeToMinutes(slot.End)
						if err == nil && endMinutes <= currentMinutes {
							targetSlotIdx = i
							break
						}
					}
				}

				if targetSlotIdx != -1 {
					m.previousState = m.state
					m.state = StateFeedback
					m.feedbackSlotID = targetSlotIdx
					return m, nil
				}
			}
		}
	}

	// Always update nowModel for time ticks
	var cmd tea.Cmd
	m.nowModel, cmd = m.nowModel.Update(msg)
	cmds = append(cmds, cmd)

	switch m.state {
	case StateTasks:
		m.taskList, cmd = m.taskList.Update(msg)
		cmds = append(cmds, cmd)
	case StatePlan:
		if msg, ok := msg.(tea.KeyMsg); ok && key.Matches(msg, m.keys.Generate) {
			// Generate plan
			today := time.Now().Format("2006-01-02")
			settings, _ := m.store.GetSettings()

			// Default settings if not set
			dayStart := settings.DayStart
			if dayStart == "" {
				dayStart = "08:00"
			}
			dayEnd := settings.DayEnd
			if dayEnd == "" {
				dayEnd = "18:00"
			}

			tasks, _ := m.store.GetAllTasks()
			plan, err := m.scheduler.GeneratePlan(today, tasks, dayStart, dayEnd)
			if err == nil {
				m.store.SavePlan(plan)
				m.planModel.SetPlan(plan, tasks)
				m.nowModel.SetPlan(plan, tasks)
			}
		}
		m.planModel, cmd = m.planModel.Update(msg)
		cmds = append(cmds, cmd)
	case StateNow:
		// nowModel is already updated above, but if we add specific keys for Now view, handle them here
	}

	return m, tea.Batch(cmds...)
}
