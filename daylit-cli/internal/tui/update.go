package tui

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/google/uuid"

	"github.com/julianstephens/daylit/daylit-cli/internal/cli"
	"github.com/julianstephens/daylit/daylit-cli/internal/constants"
	"github.com/julianstephens/daylit/daylit-cli/internal/models"
	"github.com/julianstephens/daylit/daylit-cli/internal/tui/components/alerts"
	"github.com/julianstephens/daylit/daylit-cli/internal/tui/components/habits"
	"github.com/julianstephens/daylit/daylit-cli/internal/tui/components/ot"
	"github.com/julianstephens/daylit/daylit-cli/internal/tui/components/settings"
	"github.com/julianstephens/daylit/daylit-cli/internal/tui/components/tasklist"
	"github.com/julianstephens/daylit/daylit-cli/internal/tui/state"
	"github.com/julianstephens/daylit/daylit-cli/internal/utils"
)

func newEditForm(fm *state.TaskFormModel) *huh.Form {
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
			huh.NewSelect[constants.RecurrenceType]().
				Title("Recurrence").
				Options(
					huh.NewOption("Ad-hoc", constants.RecurrenceAdHoc),
					huh.NewOption("Daily", constants.RecurrenceDaily),
					huh.NewOption("Weekly", constants.RecurrenceWeekly),
					huh.NewOption("Every N Days", constants.RecurrenceNDays),
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

func newHabitForm(fm *state.HabitFormModel) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Habit Name").
				Value(&fm.Name).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("habit name cannot be empty")
					}
					return nil
				}),
		),
	).WithTheme(huh.ThemeDracula())
}

func newAlertForm(fm *state.AlertFormModel) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Message").
				Value(&fm.Message).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("message cannot be empty")
					}
					return nil
				}),
			huh.NewInput().
				Title("Time (HH:MM)").
				Value(&fm.Time).
				Validate(func(s string) error {
					_, err := time.Parse(constants.TimeFormat, s)
					if err != nil {
						return fmt.Errorf("invalid time format, use HH:MM")
					}
					return nil
				}),
			huh.NewInput().
				Title("Date (YYYY-MM-DD)").
				Description("Leave empty for recurring alert").
				Value(&fm.Date).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return nil
					}
					_, err := time.Parse("2006-01-02", s)
					if err != nil {
						return fmt.Errorf("invalid date format, use YYYY-MM-DD")
					}
					return nil
				}),
			huh.NewSelect[constants.RecurrenceType]().
				Title("Recurrence").
				Description("Only for recurring alerts (no date)").
				Options(
					huh.NewOption("Daily", constants.RecurrenceDaily),
					huh.NewOption("Weekly", constants.RecurrenceWeekly),
					huh.NewOption("Every N Days", constants.RecurrenceNDays),
				).
				Value(&fm.Recurrence).
				Validate(func(r constants.RecurrenceType) error {
					// When Date is empty (recurring alert), a recurrence type must be selected
					if strings.TrimSpace(fm.Date) == "" && r == "" {
						return fmt.Errorf("recurrence is required when date is empty")
					}
					return nil
				}),
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
				Title("Weekdays").
				Description("For weekly: comma-separated (mon,wed,fri)").
				Value(&fm.Weekdays),
		),
	).WithTheme(huh.ThemeDracula())
}

func newSettingsForm(fm *state.SettingsFormModel) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Day Start (HH:MM)").
				Value(&fm.DayStart).
				Validate(func(s string) error {
					_, err := time.Parse(constants.TimeFormat, s)
					if err != nil {
						return fmt.Errorf("invalid time format, use HH:MM")
					}
					return nil
				}),
			huh.NewInput().
				Title("Day End (HH:MM)").
				Value(&fm.DayEnd).
				Validate(func(s string) error {
					endTime, err := time.Parse(constants.TimeFormat, s)
					if err != nil {
						return fmt.Errorf("invalid time format, use HH:MM")
					}
					// Cross-field validation: ensure Day End is after Day Start
					startTime, err := time.Parse(constants.TimeFormat, fm.DayStart)
					if err == nil && !endTime.After(startTime) {
						return fmt.Errorf("day end must be after day start")
					}
					return nil
				}),
			huh.NewInput().
				Title("Default Block (minutes)").
				Value(&fm.DefaultBlockMin).
				Validate(func(s string) error {
					i, err := strconv.Atoi(s)
					if err != nil {
						return err
					}
					if i <= 0 {
						return fmt.Errorf("must be a positive number")
					}
					return nil
				}),
			huh.NewInput().
				Title("Timezone (IANA name or 'Local')").
				Description("Examples: Local, UTC, America/New_York, Europe/London, Asia/Tokyo").
				Value(&fm.Timezone).
				Validate(func(s string) error {
					if !utils.ValidateTimezone(s) {
						return fmt.Errorf("invalid timezone name")
					}
					return nil
				}),
			huh.NewConfirm().
				Title("Prompt On Empty").
				Value(&fm.PromptOnEmpty),
			huh.NewConfirm().
				Title("Strict Mode").
				Value(&fm.StrictMode),
			huh.NewInput().
				Title("Default Log Days").
				Value(&fm.DefaultLogDays).
				Validate(func(s string) error {
					i, err := strconv.Atoi(s)
					if err != nil {
						return err
					}
					if i < 0 {
						return fmt.Errorf("must be a non-negative number")
					}
					return nil
				}),
		),
		huh.NewGroup(
			huh.NewConfirm().
				Title("Enable Notifications").
				Value(&fm.NotificationsEnabled),
			huh.NewConfirm().
				Title("Notify on Block Start").
				Value(&fm.NotifyBlockStart),
			huh.NewInput().
				Title("Start Offset (minutes)").
				Value(&fm.BlockStartOffsetMin).
				Validate(func(s string) error {
					_, err := strconv.Atoi(s)
					return err
				}),
			huh.NewConfirm().
				Title("Notify on Block End").
				Value(&fm.NotifyBlockEnd),
			huh.NewInput().
				Title("End Offset (minutes)").
				Value(&fm.BlockEndOffsetMin).
				Validate(func(s string) error {
					_, err := strconv.Atoi(s)
					return err
				}),
		),
	).WithTheme(huh.ThemeDracula())
}

func newOTForm(fm *state.OTFormModel) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("One Thing Title").
				Value(&fm.Title).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("title cannot be empty")
					}
					return nil
				}),
			huh.NewText().
				Title("Note (optional)").
				Value(&fm.Note),
		),
	).WithTheme(huh.ThemeDracula())
}

func calculateSlotDuration(slot models.Slot) int {
	start, err := time.Parse(constants.TimeFormat, slot.Start)
	if err != nil {
		return 0
	}
	end, err := time.Parse(constants.TimeFormat, slot.End)
	if err != nil {
		return 0
	}
	return int(end.Sub(start).Minutes())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Handle Editing State
	if m.State == constants.StateEditing {
		if msg, ok := msg.(tea.KeyMsg); ok && msg.Type == tea.KeyEsc {
			m.State = constants.StateTasks
			return m, nil
		}

		form, cmd := m.Form.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.Form = f
		}
		cmds = append(cmds, cmd)

		switch m.Form.State {
		case huh.StateCompleted:
			// Apply changes
			m.EditingTask.Name = m.TaskForm.Name
			dur, err := strconv.Atoi(m.TaskForm.Duration)
			if err == nil {
				m.EditingTask.DurationMin = dur
			}
			m.EditingTask.Recurrence.Type = m.TaskForm.Recurrence
			intervalStr := strings.TrimSpace(m.TaskForm.Interval)
			interval, err := strconv.Atoi(intervalStr)
			if err == nil {
				m.EditingTask.Recurrence.IntervalDays = interval
			}
			prio, err := strconv.Atoi(m.TaskForm.Priority)
			if err == nil {
				m.EditingTask.Priority = prio
			}
			m.EditingTask.Active = m.TaskForm.Active

			// Check if task exists to decide Add vs Update
			_, err = m.Store.GetTask(m.EditingTask.ID)
			var saveErr error
			if err != nil {
				// Task doesn't exist, add it
				saveErr = m.Store.AddTask(*m.EditingTask)
			} else {
				// Task exists, update it
				saveErr = m.Store.UpdateTask(*m.EditingTask)
			}

			// Only update task list if save was successful
			if saveErr == nil {
				tasks, err := m.Store.GetAllTasksIncludingDeleted()
				if err == nil {
					m.TaskList.SetTasks(tasks)
				}
				m.updateValidationStatus()
			}
			m.State = constants.StateTasks
		case huh.StateAborted:
			m.State = constants.StateTasks
		}
		return m, tea.Batch(cmds...)
	}

	// Handle Add Habit State
	if m.State == constants.StateAddHabit {
		if msg, ok := msg.(tea.KeyMsg); ok && msg.Type == tea.KeyEsc {
			m.State = constants.StateHabits
			return m, nil
		}

		form, cmd := m.Form.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.Form = f
		}
		cmds = append(cmds, cmd)

		switch m.Form.State {
		case huh.StateCompleted:
			// Create new habit
			habit := models.Habit{
				ID:        uuid.New().String(),
				Name:      m.HabitForm.Name,
				CreatedAt: time.Now(),
			}
			if err := m.Store.AddHabit(habit); err == nil {
				// Refresh habits list only if add succeeded
				today := time.Now().Format(constants.DateFormat)
				habitsList, _ := m.Store.GetAllHabits(false, true)
				habitEntries, _ := m.Store.GetHabitEntriesForDay(today)
				m.HabitsModel.SetHabits(habitsList, habitEntries)
				m.State = constants.StateHabits
			} else {
				// Stay in form state on error to allow retry
				// The form will display, user can cancel with ESC or retry
				m.Form.State = huh.StateNormal
			}
		case huh.StateAborted:
			m.State = constants.StateHabits
		}
		return m, tea.Batch(cmds...)
	}

	// Handle Add Alert State
	if m.State == constants.StateAddAlert {
		if msg, ok := msg.(tea.KeyMsg); ok && msg.Type == tea.KeyEsc {
			m.State = constants.StateAlerts
			return m, nil
		}

		form, cmd := m.Form.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.Form = f
		}
		cmds = append(cmds, cmd)

		switch m.Form.State {
		case huh.StateCompleted:
			// Validate and create new alert
			alert := models.Alert{
				ID:        uuid.New().String(),
				Message:   m.AlertForm.Message,
				Time:      m.AlertForm.Time,
				Date:      m.AlertForm.Date,
				Active:    true,
				CreatedAt: time.Now(),
			}

			// Set recurrence if not one-time
			if alert.Date == "" {
				alert.Recurrence.Type = m.AlertForm.Recurrence
				if m.AlertForm.Interval != "" {
					interval, err := strconv.Atoi(m.AlertForm.Interval)
					if err != nil || interval < 1 {
						// Invalid interval; keep user in the form to correct the value
						m.FormError = "Invalid interval: must be a positive number"
						m.Form.State = huh.StateNormal
						return m, tea.Batch(cmds...)
					}
					alert.Recurrence.IntervalDays = interval
				}

				// Parse weekdays for weekly recurrence
				if m.AlertForm.Recurrence == constants.RecurrenceWeekly && m.AlertForm.Weekdays != "" {
					weekdays, err := cli.ParseWeekdays(m.AlertForm.Weekdays)
					if err != nil {
						// Invalid weekdays; keep user in the form to correct the value
						m.FormError = fmt.Sprintf("Invalid weekdays: %v", err)
						m.Form.State = huh.StateNormal
						return m, tea.Batch(cmds...)
					}
					alert.Recurrence.WeekdayMask = weekdays
				}
			}

			if err := m.Store.AddAlert(alert); err == nil {
				// Refresh alerts list only if add succeeded
				alertsList, _ := m.Store.GetAllAlerts()
				m.AlertsModel.SetAlerts(alertsList)
				m.FormError = "" // Clear any previous errors
				m.State = constants.StateAlerts
			} else {
				// Store error and stay in form state to allow retry
				m.FormError = fmt.Sprintf("Failed to add alert: %v", err)
				m.Form.State = huh.StateNormal
			}
		case huh.StateAborted:
			m.State = constants.StateAlerts
		}
		return m, tea.Batch(cmds...)
	}

	// Handle Edit OT State
	if m.State == constants.StateEditOT {
		if msg, ok := msg.(tea.KeyMsg); ok && msg.Type == tea.KeyEsc {
			m.FormError = "" // Clear error on cancel
			m.State = constants.StateOT
			return m, nil
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
					return m, tea.Batch(cmds...)
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
					return m, tea.Batch(cmds...)
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
		return m, tea.Batch(cmds...)
	}

	// Handle Edit Settings State
	if m.State == constants.StateEditSettings {
		if msg, ok := msg.(tea.KeyMsg); ok && msg.Type == tea.KeyEsc {
			m.State = constants.StateSettings
			return m, nil
		}

		form, cmd := m.Form.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.Form = f
		}
		cmds = append(cmds, cmd)

		switch m.Form.State {
		case huh.StateCompleted:
			// Apply settings changes
			settings, _ := m.Store.GetSettings()
			settings.DayStart = m.SettingsForm.DayStart
			settings.DayEnd = m.SettingsForm.DayEnd

			// Parse and validate DefaultBlockMin
			blockMin, err := strconv.Atoi(m.SettingsForm.DefaultBlockMin)
			if err != nil {
				// Stay in form state on conversion error
				m.Form.State = huh.StateNormal
				return m, tea.Batch(cmds...)
			}
			settings.DefaultBlockMin = blockMin

			// Timezone setting
			settings.Timezone = m.SettingsForm.Timezone

			// Notification settings
			settings.NotificationsEnabled = m.SettingsForm.NotificationsEnabled
			settings.NotifyBlockStart = m.SettingsForm.NotifyBlockStart
			settings.NotifyBlockEnd = m.SettingsForm.NotifyBlockEnd

			startOffset, err := strconv.Atoi(m.SettingsForm.BlockStartOffsetMin)
			if err == nil {
				settings.BlockStartOffsetMin = startOffset
			}

			endOffset, err := strconv.Atoi(m.SettingsForm.BlockEndOffsetMin)
			if err == nil {
				settings.BlockEndOffsetMin = endOffset
			}

			// Apply OT settings changes
			otSettings, _ := m.Store.GetOTSettings()
			otSettings.PromptOnEmpty = m.SettingsForm.PromptOnEmpty
			otSettings.StrictMode = m.SettingsForm.StrictMode

			// Parse and validate DefaultLogDays
			logDays, err := strconv.Atoi(m.SettingsForm.DefaultLogDays)
			if err != nil {
				// Stay in form state on conversion error
				m.Form.State = huh.StateNormal
				return m, tea.Batch(cmds...)
			}
			otSettings.DefaultLogDays = logDays

			// Save settings and check for errors
			if err := m.Store.SaveSettings(settings); err != nil {
				// Stay in form state on save error
				m.Form.State = huh.StateNormal
				return m, tea.Batch(cmds...)
			}

			if err := m.Store.SaveOTSettings(otSettings); err != nil {
				// Stay in form state on save error
				m.Form.State = huh.StateNormal
				return m, tea.Batch(cmds...)
			}

			// Refresh settings view only after successful save
			m.SettingsModel.SetSettings(settings, otSettings)
			m.State = constants.StateSettings
		case huh.StateAborted:
			m.State = constants.StateSettings
		}
		return m, tea.Batch(cmds...)
	}

	// Handle Feedback State
	if m.State == constants.StateFeedback {
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
				return m, nil
			default:
				return m, nil
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
					return m, nil
				}

				// Update task stats only after plan is saved
				task, err := m.Store.GetTask(slot.TaskID)
				if err == nil {
					switch rating {
					case constants.FeedbackOnTrack:
						slotDuration := calculateSlotDuration(*slot)
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
					return m, nil
				}
				tasksIncludingDeleted, _ := m.Store.GetAllTasksIncludingDeleted()
				m.PlanModel.SetPlan(plan, tasks)
				m.NowModel.SetPlan(plan, tasks)
				m.TaskList.SetTasks(tasksIncludingDeleted)
				m.updateValidationStatus()
			}

			m.State = m.PreviousState
			return m, nil
		}
		return m, nil
	}

	// Handle Confirm Delete State
	if m.State == constants.StateConfirmDelete {
		if msg, ok := msg.(tea.KeyMsg); ok {
			switch msg.String() {
			case "y", "Y":
				if err := m.Store.DeleteTask(m.TaskToDeleteID); err != nil {
					// On error, silently return to tasks view
					m.State = constants.StateTasks
					m.TaskToDeleteID = ""
					return m, nil
				}
				// Deletion succeeded - always refresh and clear state
				tasks, err := m.Store.GetAllTasksIncludingDeleted()
				if err == nil {
					m.TaskList.SetTasks(tasks)
				}
				m.updateValidationStatus()
				m.State = constants.StateTasks
				m.TaskToDeleteID = ""
			case "n", "N", "esc", "q":
				m.State = constants.StateTasks
				m.TaskToDeleteID = ""
			}
		}
		return m, nil
	}

	// Handle Confirm Restore State
	if m.State == constants.StateConfirmRestore {
		if msg, ok := msg.(tea.KeyMsg); ok {
			switch msg.String() {
			case "y", "Y":
				if m.TaskToRestoreID != "" {
					if err := m.Store.RestoreTask(m.TaskToRestoreID); err != nil {
						// On error, silently return to tasks view
						m.State = constants.StateTasks
						m.TaskToRestoreID = ""
						return m, nil
					}
					// Restore succeeded - refresh and clear state
					tasks, err := m.Store.GetAllTasksIncludingDeleted()
					if err == nil {
						m.TaskList.SetTasks(tasks)
					}
					m.updateValidationStatus()
					m.State = constants.StateTasks
					m.TaskToRestoreID = ""
				} else if m.PlanToRestoreDate != "" {
					if err := m.Store.RestorePlan(m.PlanToRestoreDate); err != nil {
						// On error, silently return to plan view
						m.State = constants.StatePlan
						m.PlanToRestoreDate = ""
						return m, nil
					}
					// Restore succeeded - refresh plan
					today := time.Now().Format(constants.DateFormat)
					plan, err := m.Store.GetPlan(today)
					tasks, _ := m.Store.GetAllTasksIncludingDeleted()
					if err == nil {
						m.PlanModel.SetPlan(plan, tasks)
						m.NowModel.SetPlan(plan, tasks)
					}
					m.updateValidationStatus()
					m.State = constants.StatePlan
					m.PlanToRestoreDate = ""
				}
			case "n", "N", "esc", "q":
				if m.PlanToRestoreDate != "" {
					m.State = constants.StatePlan
				} else {
					m.State = constants.StateTasks
				}
				m.TaskToRestoreID = ""
				m.PlanToRestoreDate = ""
			}
		}
		return m, nil
	}

	// Handle Confirm Overwrite State
	if m.State == constants.StateConfirmOverwrite {
		if msg, ok := msg.(tea.KeyMsg); ok {
			switch msg.String() {
			case "y", "Y":
				// Generate new plan (creates new revision)
				if m.PlanToOverwriteDate != "" {
					settings, _ := m.Store.GetSettings()
					dayStart := settings.DayStart
					if dayStart == "" {
						dayStart = "08:00"
					}
					dayEnd := settings.DayEnd
					if dayEnd == "" {
						dayEnd = "18:00"
					}

					tasks, _ := m.Store.GetAllTasks()
					plan, err := m.Scheduler.GeneratePlan(m.PlanToOverwriteDate, tasks, dayStart, dayEnd)
					if err == nil {
						m.Store.SavePlan(plan)
						allTasks, _ := m.Store.GetAllTasksIncludingDeleted()
						m.PlanModel.SetPlan(plan, allTasks)
						m.NowModel.SetPlan(plan, allTasks)
						m.TaskList.SetTasks(allTasks)
						m.updateValidationStatus()
					}
				}
				m.State = constants.StatePlan
				m.PlanToOverwriteDate = ""
			case "n", "N", "esc", "q":
				m.State = constants.StatePlan
				m.PlanToOverwriteDate = ""
			}
		}
		return m, nil
	}

	// Handle Confirm Archive State
	if m.State == constants.StateConfirmArchive {
		if msg, ok := msg.(tea.KeyMsg); ok {
			switch msg.String() {
			case "y", "Y":
				if err := m.Store.ArchiveHabit(m.HabitToArchiveID); err != nil {
					m.State = constants.StateHabits
					m.HabitToArchiveID = ""
					return m, nil
				}
				// Refresh habits list
				today := time.Now().Format(constants.DateFormat)
				habitsList, _ := m.Store.GetAllHabits(false, true)
				habitEntries, _ := m.Store.GetHabitEntriesForDay(today)
				m.HabitsModel.SetHabits(habitsList, habitEntries)
				m.State = constants.StateHabits
				m.HabitToArchiveID = ""
			case "n", "N", "esc", "q":
				m.State = constants.StateHabits
				m.HabitToArchiveID = ""
			}
		}
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.Help.Width = msg.Width
		// Adjust height for tabs and help
		listHeight := msg.Height - 4 // Approximate height for tabs + help

		h, v := docStyle.GetFrameSize()
		m.TaskList.SetSize(msg.Width-h, listHeight-v)
		m.PlanModel.SetSize(msg.Width-h, listHeight-v)
		m.NowModel.SetSize(msg.Width, listHeight)
		m.HabitsModel.SetSize(msg.Width-h, listHeight-v)
		m.OTModel.SetSize(msg.Width-h, listHeight-v)
		m.AlertsModel.SetSize(msg.Width-h, listHeight-v)
		m.SettingsModel.SetSize(msg.Width-h, listHeight-v)

	case tasklist.DeleteTaskMsg:
		m.TaskToDeleteID = msg.ID
		m.State = constants.StateConfirmDelete
		return m, nil

	case tasklist.RestoreTaskMsg:
		m.TaskToRestoreID = msg.ID
		m.State = constants.StateConfirmRestore
		return m, nil

	case tasklist.AddTaskMsg:
		task := models.Task{
			ID:          uuid.New().String(),
			Name:        "New Task",
			Kind:        constants.TaskKindFlexible,
			DurationMin: 30,
			Recurrence: models.Recurrence{
				Type: constants.RecurrenceAdHoc,
			},
			Priority: 3,
			Active:   true,
		}
		m.EditingTask = &task
		m.TaskForm = &state.TaskFormModel{
			Name:       task.Name,
			Duration:   strconv.Itoa(task.DurationMin),
			Recurrence: task.Recurrence.Type,
			Interval:   strconv.Itoa(task.Recurrence.IntervalDays),
			Priority:   strconv.Itoa(task.Priority),
			Active:     task.Active,
		}
		m.Form = newEditForm(m.TaskForm)
		m.State = constants.StateEditing
		return m, m.Form.Init()

	case tasklist.EditTaskMsg:
		m.EditingTask = &msg.Task
		m.TaskForm = &state.TaskFormModel{
			Name:       msg.Task.Name,
			Duration:   strconv.Itoa(msg.Task.DurationMin),
			Recurrence: msg.Task.Recurrence.Type,
			Interval:   strconv.Itoa(msg.Task.Recurrence.IntervalDays),
			Priority:   strconv.Itoa(msg.Task.Priority),
			Active:     msg.Task.Active,
		}
		m.Form = newEditForm(m.TaskForm)
		m.State = constants.StateEditing
		return m, m.Form.Init()

	// Handle habit messages
	case habits.AddHabitMsg:
		m.HabitForm = &state.HabitFormModel{
			Name: "",
		}
		m.Form = newHabitForm(m.HabitForm)
		m.State = constants.StateAddHabit
		return m, m.Form.Init()

	case habits.MarkHabitMsg:
		today := time.Now().Format(constants.DateFormat)
		entry := models.HabitEntry{
			ID:        uuid.New().String(),
			HabitID:   msg.ID,
			Day:       today,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := m.Store.AddHabitEntry(entry); err == nil {
			habitsList, _ := m.Store.GetAllHabits(false, true)
			habitEntries, _ := m.Store.GetHabitEntriesForDay(today)
			m.HabitsModel.SetHabits(habitsList, habitEntries)
		}
		return m, nil

	case habits.UnmarkHabitMsg:
		today := time.Now().Format(constants.DateFormat)
		entry, err := m.Store.GetHabitEntry(msg.ID, today)
		if err == nil {
			if err := m.Store.DeleteHabitEntry(entry.ID); err == nil {
				habitsList, _ := m.Store.GetAllHabits(false, true)
				habitEntries, _ := m.Store.GetHabitEntriesForDay(today)
				m.HabitsModel.SetHabits(habitsList, habitEntries)
			}
		}
		return m, nil

	case habits.ArchiveHabitMsg:
		m.HabitToArchiveID = msg.ID
		m.State = constants.StateConfirmArchive
		return m, nil

	case habits.DeleteHabitMsg:
		if err := m.Store.DeleteHabit(msg.ID); err == nil {
			today := time.Now().Format(constants.DateFormat)
			habitsList, _ := m.Store.GetAllHabits(false, true)
			habitEntries, _ := m.Store.GetHabitEntriesForDay(today)
			m.HabitsModel.SetHabits(habitsList, habitEntries)
		}
		return m, nil

	case habits.RestoreHabitMsg:
		if err := m.Store.RestoreHabit(msg.ID); err == nil {
			today := time.Now().Format(constants.DateFormat)
			habitsList, _ := m.Store.GetAllHabits(false, true)
			habitEntries, _ := m.Store.GetHabitEntriesForDay(today)
			m.HabitsModel.SetHabits(habitsList, habitEntries)
		}
		return m, nil

	// Handle alert messages
	case alerts.AddAlertMsg:
		m.AlertForm = &state.AlertFormModel{
			Message:    "",
			Time:       "",
			Date:       "",
			Recurrence: constants.RecurrenceDaily,
			Interval:   "1",
			Weekdays:   "",
		}
		m.Form = newAlertForm(m.AlertForm)
		m.State = constants.StateAddAlert
		return m, m.Form.Init()

	case alerts.DeleteAlertMsg:
		if err := m.Store.DeleteAlert(msg.ID); err == nil {
			alertsList, _ := m.Store.GetAllAlerts()
			m.AlertsModel.SetAlerts(alertsList)
		}
		return m, nil

	// Handle settings messages
	case settings.EditSettingsMsg:
		currentSettings, _ := m.Store.GetSettings()
		currentOTSettings, _ := m.Store.GetOTSettings()
		m.SettingsForm = &state.SettingsFormModel{
			DayStart:             currentSettings.DayStart,
			DayEnd:               currentSettings.DayEnd,
			DefaultBlockMin:      strconv.Itoa(currentSettings.DefaultBlockMin),
			Timezone:             currentSettings.Timezone,
			PromptOnEmpty:        currentOTSettings.PromptOnEmpty,
			StrictMode:           currentOTSettings.StrictMode,
			DefaultLogDays:       strconv.Itoa(currentOTSettings.DefaultLogDays),
			NotificationsEnabled: currentSettings.NotificationsEnabled,
			NotifyBlockStart:     currentSettings.NotifyBlockStart,
			NotifyBlockEnd:       currentSettings.NotifyBlockEnd,
			BlockStartOffsetMin:  strconv.Itoa(currentSettings.BlockStartOffsetMin),
			BlockEndOffsetMin:    strconv.Itoa(currentSettings.BlockEndOffsetMin),
		}
		m.Form = newSettingsForm(m.SettingsForm)
		m.State = constants.StateEditSettings
		return m, m.Form.Init()

	// Handle OT messages
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
		m.Form = newOTForm(m.OTForm)
		m.State = constants.StateEditOT
		return m, m.Form.Init()

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.Keys.Quit):
			m.Quitting = true
			return m, tea.Quit
		case key.Matches(msg, m.Keys.Tab, m.Keys.Right):
			m.State = (m.State + 1) % constants.NumMainTabs
			return m, nil
		case key.Matches(msg, m.Keys.ShiftTab, m.Keys.Left):
			m.State = (m.State - 1 + constants.NumMainTabs) % constants.NumMainTabs
			return m, nil
		case key.Matches(msg, m.Keys.Help):
			m.Help.ShowAll = !m.Help.ShowAll
			return m, nil
		case key.Matches(msg, m.Keys.Feedback):
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
					return m, nil
				}
			}
		}
	}

	// Always update nowModel for time ticks
	var cmd tea.Cmd
	m.NowModel, cmd = m.NowModel.Update(msg)
	cmds = append(cmds, cmd)

	switch m.State {
	case constants.StateTasks:
		m.TaskList, cmd = m.TaskList.Update(msg)
		cmds = append(cmds, cmd)
	case constants.StatePlan:
		if msg, ok := msg.(tea.KeyMsg); ok && key.Matches(msg, m.Keys.Generate) {
			// Generate plan
			today := time.Now().Format(constants.DateFormat)

			// Check if plan already exists
			_, err := m.Store.GetPlan(today)
			if err == nil {
				// Plan exists, ask for confirmation
				m.PlanToOverwriteDate = today
				m.State = constants.StateConfirmOverwrite
				return m, nil
			}

			settings, _ := m.Store.GetSettings()

			// Default settings if not set
			dayStart := settings.DayStart
			if dayStart == "" {
				dayStart = "08:00"
			}
			dayEnd := settings.DayEnd
			if dayEnd == "" {
				dayEnd = "18:00"
			}

			tasks, _ := m.Store.GetAllTasks()
			plan, err := m.Scheduler.GeneratePlan(today, tasks, dayStart, dayEnd)
			if err == nil {
				m.Store.SavePlan(plan)
				m.PlanModel.SetPlan(plan, tasks)
				m.NowModel.SetPlan(plan, tasks)
				m.updateValidationStatus()
			}
		}
		m.PlanModel, cmd = m.PlanModel.Update(msg)
		cmds = append(cmds, cmd)
	case constants.StateHabits:
		m.HabitsModel, cmd = m.HabitsModel.Update(msg)
		cmds = append(cmds, cmd)
	case constants.StateOT:
		m.OTModel, cmd = m.OTModel.Update(msg)
		cmds = append(cmds, cmd)
	case constants.StateAlerts:
		m.AlertsModel, cmd = m.AlertsModel.Update(msg)
		cmds = append(cmds, cmd)
	case constants.StateSettings:
		m.SettingsModel, cmd = m.SettingsModel.Update(msg)
		cmds = append(cmds, cmd)
	case constants.StateNow:
		// nowModel is already updated above, but if we add specific keys for Now view, handle them here
	}

	return m, tea.Batch(cmds...)
}
