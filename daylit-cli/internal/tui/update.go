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
	"github.com/julianstephens/daylit/daylit-cli/internal/utils"
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

func newHabitForm(fm *HabitFormModel) *huh.Form {
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

func newAlertForm(fm *AlertFormModel) *huh.Form {
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

func newSettingsForm(fm *SettingsFormModel) *huh.Form {
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
					if s == "" || s == "Local" {
						return nil
					}
					_, err := time.LoadLocation(s)
					if err != nil {
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

func newOTForm(fm *OTFormModel) *huh.Form {
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
	if m.state == constants.StateEditing {
		if msg, ok := msg.(tea.KeyMsg); ok && msg.Type == tea.KeyEsc {
			m.state = constants.StateTasks
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
				tasks, err := m.store.GetAllTasksIncludingDeleted()
				if err == nil {
					m.taskList.SetTasks(tasks)
				}
				m.updateValidationStatus()
			}
			m.state = constants.StateTasks
		case huh.StateAborted:
			m.state = constants.StateTasks
		}
		return m, tea.Batch(cmds...)
	}

	// Handle Add Habit State
	if m.state == constants.StateAddHabit {
		if msg, ok := msg.(tea.KeyMsg); ok && msg.Type == tea.KeyEsc {
			m.state = constants.StateHabits
			return m, nil
		}

		form, cmd := m.form.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.form = f
		}
		cmds = append(cmds, cmd)

		switch m.form.State {
		case huh.StateCompleted:
			// Create new habit
			habit := models.Habit{
				ID:        uuid.New().String(),
				Name:      m.habitForm.Name,
				CreatedAt: time.Now(),
			}
			if err := m.store.AddHabit(habit); err == nil {
				// Refresh habits list only if add succeeded
				today := time.Now().Format(constants.DateFormat)
				habitsList, _ := m.store.GetAllHabits(false, true)
				habitEntries, _ := m.store.GetHabitEntriesForDay(today)
				m.habitsModel.SetHabits(habitsList, habitEntries)
				m.state = constants.StateHabits
			} else {
				// Stay in form state on error to allow retry
				// The form will display, user can cancel with ESC or retry
				m.form.State = huh.StateNormal
			}
		case huh.StateAborted:
			m.state = constants.StateHabits
		}
		return m, tea.Batch(cmds...)
	}

	// Handle Add Alert State
	if m.state == constants.StateAddAlert {
		if msg, ok := msg.(tea.KeyMsg); ok && msg.Type == tea.KeyEsc {
			m.state = constants.StateAlerts
			return m, nil
		}

		form, cmd := m.form.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.form = f
		}
		cmds = append(cmds, cmd)

		switch m.form.State {
		case huh.StateCompleted:
			// Validate and create new alert
			alert := models.Alert{
				ID:        uuid.New().String(),
				Message:   m.alertForm.Message,
				Time:      m.alertForm.Time,
				Date:      m.alertForm.Date,
				Active:    true,
				CreatedAt: time.Now(),
			}

			// Set recurrence if not one-time
			if alert.Date == "" {
				alert.Recurrence.Type = m.alertForm.Recurrence
				if m.alertForm.Interval != "" {
					interval, err := strconv.Atoi(m.alertForm.Interval)
					if err != nil || interval < 1 {
						// Invalid interval; keep user in the form to correct the value
						m.formError = "Invalid interval: must be a positive number"
						m.form.State = huh.StateNormal
						return m, tea.Batch(cmds...)
					}
					alert.Recurrence.IntervalDays = interval
				}

				// Parse weekdays for weekly recurrence
				if m.alertForm.Recurrence == constants.RecurrenceWeekly && m.alertForm.Weekdays != "" {
					weekdays, err := cli.ParseWeekdays(m.alertForm.Weekdays)
					if err != nil {
						// Invalid weekdays; keep user in the form to correct the value
						m.formError = fmt.Sprintf("Invalid weekdays: %v", err)
						m.form.State = huh.StateNormal
						return m, tea.Batch(cmds...)
					}
					alert.Recurrence.WeekdayMask = weekdays
				}
			}

			if err := m.store.AddAlert(alert); err == nil {
				// Refresh alerts list only if add succeeded
				alertsList, _ := m.store.GetAllAlerts()
				m.alertsModel.SetAlerts(alertsList)
				m.formError = "" // Clear any previous errors
				m.state = constants.StateAlerts
			} else {
				// Store error and stay in form state to allow retry
				m.formError = fmt.Sprintf("Failed to add alert: %v", err)
				m.form.State = huh.StateNormal
			}
		case huh.StateAborted:
			m.state = constants.StateAlerts
		}
		return m, tea.Batch(cmds...)
	}

	// Handle Edit OT State
	if m.state == constants.StateEditOT {
		if msg, ok := msg.(tea.KeyMsg); ok && msg.Type == tea.KeyEsc {
			m.formError = "" // Clear error on cancel
			m.state = constants.StateOT
			return m, nil
		}

		form, cmd := m.form.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.form = f
		}
		cmds = append(cmds, cmd)

		switch m.form.State {
		case huh.StateCompleted:
			// Save or update OT entry
			today := time.Now().Format(constants.DateFormat)

			// Trim whitespace from title and note
			title := strings.TrimSpace(m.otForm.Title)
			note := strings.TrimSpace(m.otForm.Note)

			// Check if entry exists for today
			existingEntry, err := m.store.GetOTEntry(today)
			if err == nil {
				// Update existing entry
				existingEntry.Title = title
				existingEntry.Note = note
				existingEntry.UpdatedAt = time.Now()
				if err := m.store.UpdateOTEntry(existingEntry); err != nil {
					// Store error and stay in form state to allow retry
					m.formError = fmt.Sprintf("Failed to update OT: %v", err)
					m.form.State = huh.StateNormal
					return m, tea.Batch(cmds...)
				}
				// Reload entry from storage to get a fresh copy
				updatedEntry, err := m.store.GetOTEntry(today)
				if err != nil {
					// Fallback to using the data we just saved if reload fails
					m.otModel.SetEntry(&existingEntry)
				} else {
					m.otModel.SetEntry(&updatedEntry)
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
				if err := m.store.AddOTEntry(newEntry); err != nil {
					// Store error and stay in form state to allow retry
					m.formError = fmt.Sprintf("Failed to create OT: %v", err)
					m.form.State = huh.StateNormal
					return m, tea.Batch(cmds...)
				}
				// Reload entry from storage to get a fresh copy
				savedEntry, err := m.store.GetOTEntry(today)
				if err != nil {
					// Fallback to using the data we just created if reload fails
					m.otModel.SetEntry(&newEntry)
				} else {
					m.otModel.SetEntry(&savedEntry)
				}
			}
			m.formError = "" // Clear any previous errors
			m.state = constants.StateOT
		case huh.StateAborted:
			m.formError = "" // Clear error on abort
			m.state = constants.StateOT
		}
		return m, tea.Batch(cmds...)
	}

	// Handle Edit Settings State
	if m.state == constants.StateEditSettings {
		if msg, ok := msg.(tea.KeyMsg); ok && msg.Type == tea.KeyEsc {
			m.state = constants.StateSettings
			return m, nil
		}

		form, cmd := m.form.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.form = f
		}
		cmds = append(cmds, cmd)

		switch m.form.State {
		case huh.StateCompleted:
			// Apply settings changes
			settings, _ := m.store.GetSettings()
			settings.DayStart = m.settingsForm.DayStart
			settings.DayEnd = m.settingsForm.DayEnd

			// Parse and validate DefaultBlockMin
			blockMin, err := strconv.Atoi(m.settingsForm.DefaultBlockMin)
			if err != nil {
				// Stay in form state on conversion error
				m.form.State = huh.StateNormal
				return m, tea.Batch(cmds...)
			}
			settings.DefaultBlockMin = blockMin

			// Timezone setting
			settings.Timezone = m.settingsForm.Timezone

			// Notification settings
			settings.NotificationsEnabled = m.settingsForm.NotificationsEnabled
			settings.NotifyBlockStart = m.settingsForm.NotifyBlockStart
			settings.NotifyBlockEnd = m.settingsForm.NotifyBlockEnd

			startOffset, err := strconv.Atoi(m.settingsForm.BlockStartOffsetMin)
			if err == nil {
				settings.BlockStartOffsetMin = startOffset
			}

			endOffset, err := strconv.Atoi(m.settingsForm.BlockEndOffsetMin)
			if err == nil {
				settings.BlockEndOffsetMin = endOffset
			}

			// Apply OT settings changes
			otSettings, _ := m.store.GetOTSettings()
			otSettings.PromptOnEmpty = m.settingsForm.PromptOnEmpty
			otSettings.StrictMode = m.settingsForm.StrictMode

			// Parse and validate DefaultLogDays
			logDays, err := strconv.Atoi(m.settingsForm.DefaultLogDays)
			if err != nil {
				// Stay in form state on conversion error
				m.form.State = huh.StateNormal
				return m, tea.Batch(cmds...)
			}
			otSettings.DefaultLogDays = logDays

			// Save settings and check for errors
			if err := m.store.SaveSettings(settings); err != nil {
				// Stay in form state on save error
				m.form.State = huh.StateNormal
				return m, tea.Batch(cmds...)
			}

			if err := m.store.SaveOTSettings(otSettings); err != nil {
				// Stay in form state on save error
				m.form.State = huh.StateNormal
				return m, tea.Batch(cmds...)
			}

			// Refresh settings view only after successful save
			m.settingsModel.SetSettings(settings, otSettings)
			m.state = constants.StateSettings
		case huh.StateAborted:
			m.state = constants.StateSettings
		}
		return m, tea.Batch(cmds...)
	}

	// Handle Feedback State
	if m.state == constants.StateFeedback {
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
				m.state = m.previousState
				return m, nil
			default:
				return m, nil
			}

			// Apply feedback
			today := time.Now().Format(constants.DateFormat)
			plan, err := m.store.GetPlan(today)
			if err == nil && m.feedbackSlotID >= 0 && m.feedbackSlotID < len(plan.Slots) {
				slot := &plan.Slots[m.feedbackSlotID]
				slot.Feedback = &models.Feedback{
					Rating: rating,
				}
				slot.Status = constants.SlotStatusDone

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
					m.store.UpdateTask(task)
				}

				// Refresh views
				tasks, err := m.store.GetAllTasks()
				if err != nil {
					// On error, revert to previous state
					m.state = m.previousState
					return m, nil
				}
				tasksIncludingDeleted, _ := m.store.GetAllTasksIncludingDeleted()
				m.planModel.SetPlan(plan, tasks)
				m.nowModel.SetPlan(plan, tasks)
				m.taskList.SetTasks(tasksIncludingDeleted)
				m.updateValidationStatus()
			}

			m.state = m.previousState
			return m, nil
		}
		return m, nil
	}

	// Handle Confirm Delete State
	if m.state == constants.StateConfirmDelete {
		if msg, ok := msg.(tea.KeyMsg); ok {
			switch msg.String() {
			case "y", "Y":
				if err := m.store.DeleteTask(m.taskToDeleteID); err != nil {
					// On error, silently return to tasks view
					m.state = constants.StateTasks
					m.taskToDeleteID = ""
					return m, nil
				}
				// Deletion succeeded - always refresh and clear state
				tasks, err := m.store.GetAllTasksIncludingDeleted()
				if err == nil {
					m.taskList.SetTasks(tasks)
				}
				m.updateValidationStatus()
				m.state = constants.StateTasks
				m.taskToDeleteID = ""
			case "n", "N", "esc", "q":
				m.state = constants.StateTasks
				m.taskToDeleteID = ""
			}
		}
		return m, nil
	}

	// Handle Confirm Restore State
	if m.state == constants.StateConfirmRestore {
		if msg, ok := msg.(tea.KeyMsg); ok {
			switch msg.String() {
			case "y", "Y":
				if m.taskToRestoreID != "" {
					if err := m.store.RestoreTask(m.taskToRestoreID); err != nil {
						// On error, silently return to tasks view
						m.state = constants.StateTasks
						m.taskToRestoreID = ""
						return m, nil
					}
					// Restore succeeded - refresh and clear state
					tasks, err := m.store.GetAllTasksIncludingDeleted()
					if err == nil {
						m.taskList.SetTasks(tasks)
					}
					m.updateValidationStatus()
					m.state = constants.StateTasks
					m.taskToRestoreID = ""
				} else if m.planToRestoreDate != "" {
					if err := m.store.RestorePlan(m.planToRestoreDate); err != nil {
						// On error, silently return to plan view
						m.state = constants.StatePlan
						m.planToRestoreDate = ""
						return m, nil
					}
					// Restore succeeded - refresh plan
					today := time.Now().Format(constants.DateFormat)
					plan, err := m.store.GetPlan(today)
					tasks, _ := m.store.GetAllTasksIncludingDeleted()
					if err == nil {
						m.planModel.SetPlan(plan, tasks)
						m.nowModel.SetPlan(plan, tasks)
					}
					m.updateValidationStatus()
					m.state = constants.StatePlan
					m.planToRestoreDate = ""
				}
			case "n", "N", "esc", "q":
				if m.planToRestoreDate != "" {
					m.state = constants.StatePlan
				} else {
					m.state = constants.StateTasks
				}
				m.taskToRestoreID = ""
				m.planToRestoreDate = ""
			}
		}
		return m, nil
	}

	// Handle Confirm Overwrite State
	if m.state == constants.StateConfirmOverwrite {
		if msg, ok := msg.(tea.KeyMsg); ok {
			switch msg.String() {
			case "y", "Y":
				// Generate new plan (creates new revision)
				if m.planToOverwriteDate != "" {
					settings, _ := m.store.GetSettings()
					dayStart := settings.DayStart
					if dayStart == "" {
						dayStart = "08:00"
					}
					dayEnd := settings.DayEnd
					if dayEnd == "" {
						dayEnd = "18:00"
					}

					tasks, _ := m.store.GetAllTasks()
					plan, err := m.scheduler.GeneratePlan(m.planToOverwriteDate, tasks, dayStart, dayEnd)
					if err == nil {
						m.store.SavePlan(plan)
						allTasks, _ := m.store.GetAllTasksIncludingDeleted()
						m.planModel.SetPlan(plan, allTasks)
						m.nowModel.SetPlan(plan, allTasks)
						m.taskList.SetTasks(allTasks)
						m.updateValidationStatus()
					}
				}
				m.state = constants.StatePlan
				m.planToOverwriteDate = ""
			case "n", "N", "esc", "q":
				m.state = constants.StatePlan
				m.planToOverwriteDate = ""
			}
		}
		return m, nil
	}

	// Handle Confirm Archive State
	if m.state == constants.StateConfirmArchive {
		if msg, ok := msg.(tea.KeyMsg); ok {
			switch msg.String() {
			case "y", "Y":
				if err := m.store.ArchiveHabit(m.habitToArchiveID); err != nil {
					m.state = constants.StateHabits
					m.habitToArchiveID = ""
					return m, nil
				}
				// Refresh habits list
				today := time.Now().Format(constants.DateFormat)
				habitsList, _ := m.store.GetAllHabits(false, true)
				habitEntries, _ := m.store.GetHabitEntriesForDay(today)
				m.habitsModel.SetHabits(habitsList, habitEntries)
				m.state = constants.StateHabits
				m.habitToArchiveID = ""
			case "n", "N", "esc", "q":
				m.state = constants.StateHabits
				m.habitToArchiveID = ""
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
		m.habitsModel.SetSize(msg.Width-h, listHeight-v)
		m.otModel.SetSize(msg.Width-h, listHeight-v)
		m.alertsModel.SetSize(msg.Width-h, listHeight-v)
		m.settingsModel.SetSize(msg.Width-h, listHeight-v)

	case tasklist.DeleteTaskMsg:
		m.taskToDeleteID = msg.ID
		m.state = constants.StateConfirmDelete
		return m, nil

	case tasklist.RestoreTaskMsg:
		m.taskToRestoreID = msg.ID
		m.state = constants.StateConfirmRestore
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
		m.state = constants.StateEditing
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
		m.state = constants.StateEditing
		return m, m.form.Init()

	// Handle habit messages
	case habits.AddHabitMsg:
		m.habitForm = &HabitFormModel{
			Name: "",
		}
		m.form = newHabitForm(m.habitForm)
		m.state = constants.StateAddHabit
		return m, m.form.Init()

	case habits.MarkHabitMsg:
		today := time.Now().Format(constants.DateFormat)
		entry := models.HabitEntry{
			ID:        uuid.New().String(),
			HabitID:   msg.ID,
			Day:       today,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := m.store.AddHabitEntry(entry); err == nil {
			habitsList, _ := m.store.GetAllHabits(false, true)
			habitEntries, _ := m.store.GetHabitEntriesForDay(today)
			m.habitsModel.SetHabits(habitsList, habitEntries)
		}
		return m, nil

	case habits.UnmarkHabitMsg:
		today := time.Now().Format(constants.DateFormat)
		entry, err := m.store.GetHabitEntry(msg.ID, today)
		if err == nil {
			if err := m.store.DeleteHabitEntry(entry.ID); err == nil {
				habitsList, _ := m.store.GetAllHabits(false, true)
				habitEntries, _ := m.store.GetHabitEntriesForDay(today)
				m.habitsModel.SetHabits(habitsList, habitEntries)
			}
		}
		return m, nil

	case habits.ArchiveHabitMsg:
		m.habitToArchiveID = msg.ID
		m.state = constants.StateConfirmArchive
		return m, nil

	case habits.DeleteHabitMsg:
		if err := m.store.DeleteHabit(msg.ID); err == nil {
			today := time.Now().Format(constants.DateFormat)
			habitsList, _ := m.store.GetAllHabits(false, true)
			habitEntries, _ := m.store.GetHabitEntriesForDay(today)
			m.habitsModel.SetHabits(habitsList, habitEntries)
		}
		return m, nil

	case habits.RestoreHabitMsg:
		if err := m.store.RestoreHabit(msg.ID); err == nil {
			today := time.Now().Format(constants.DateFormat)
			habitsList, _ := m.store.GetAllHabits(false, true)
			habitEntries, _ := m.store.GetHabitEntriesForDay(today)
			m.habitsModel.SetHabits(habitsList, habitEntries)
		}
		return m, nil

	// Handle alert messages
	case alerts.AddAlertMsg:
		m.alertForm = &AlertFormModel{
			Message:    "",
			Time:       "",
			Date:       "",
			Recurrence: constants.RecurrenceDaily,
			Interval:   "1",
			Weekdays:   "",
		}
		m.form = newAlertForm(m.alertForm)
		m.state = constants.StateAddAlert
		return m, m.form.Init()

	case alerts.DeleteAlertMsg:
		if err := m.store.DeleteAlert(msg.ID); err == nil {
			alertsList, _ := m.store.GetAllAlerts()
			m.alertsModel.SetAlerts(alertsList)
		}
		return m, nil

	// Handle settings messages
	case settings.EditSettingsMsg:
		currentSettings, _ := m.store.GetSettings()
		currentOTSettings, _ := m.store.GetOTSettings()
		m.settingsForm = &SettingsFormModel{
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
		m.form = newSettingsForm(m.settingsForm)
		m.state = constants.StateEditSettings
		return m, m.form.Init()

	// Handle OT messages
	case ot.EditOTMsg:
		today := time.Now().Format(constants.DateFormat)
		existingEntry, err := m.store.GetOTEntry(today)

		// Handle database errors differently from "not found"
		if err != nil {
			// Check if it's a "not found" error (sql.ErrNoRows)
			if err == sql.ErrNoRows {
				// Entry not found - initialize with empty values
				existingEntry = models.OTEntry{}
			} else {
				// Actual database error - show error to user
				m.formError = fmt.Sprintf("Error loading OT: %v", err)
				// Still allow editing with empty form
				existingEntry = models.OTEntry{}
			}
		} else {
			// Clear any previous form errors only if no error occurred
			m.formError = ""
		}

		m.otForm = &OTFormModel{
			Title: existingEntry.Title,
			Note:  existingEntry.Note,
		}
		m.form = newOTForm(m.otForm)
		m.state = constants.StateEditOT
		return m, m.form.Init()

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			m.quitting = true
			return m, tea.Quit
		case key.Matches(msg, m.keys.Tab, m.keys.Right):
			m.state = (m.state + 1) % constants.NumMainTabs
			return m, nil
		case key.Matches(msg, m.keys.ShiftTab, m.keys.Left):
			m.state = (m.state - 1 + constants.NumMainTabs) % constants.NumMainTabs
			return m, nil
		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
			return m, nil
		case key.Matches(msg, m.keys.Feedback):
			// Find slot for feedback
			today := time.Now().Format(constants.DateFormat)
			plan, err := m.store.GetPlan(today)
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
					m.previousState = m.state
					m.state = constants.StateFeedback
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
	case constants.StateTasks:
		m.taskList, cmd = m.taskList.Update(msg)
		cmds = append(cmds, cmd)
	case constants.StatePlan:
		if msg, ok := msg.(tea.KeyMsg); ok && key.Matches(msg, m.keys.Generate) {
			// Generate plan
			today := time.Now().Format(constants.DateFormat)

			// Check if plan already exists
			_, err := m.store.GetPlan(today)
			if err == nil {
				// Plan exists, ask for confirmation
				m.planToOverwriteDate = today
				m.state = constants.StateConfirmOverwrite
				return m, nil
			}

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
				m.updateValidationStatus()
			}
		}
		m.planModel, cmd = m.planModel.Update(msg)
		cmds = append(cmds, cmd)
	case constants.StateHabits:
		m.habitsModel, cmd = m.habitsModel.Update(msg)
		cmds = append(cmds, cmd)
	case constants.StateOT:
		m.otModel, cmd = m.otModel.Update(msg)
		cmds = append(cmds, cmd)
	case constants.StateAlerts:
		m.alertsModel, cmd = m.alertsModel.Update(msg)
		cmds = append(cmds, cmd)
	case constants.StateSettings:
		m.settingsModel, cmd = m.settingsModel.Update(msg)
		cmds = append(cmds, cmd)
	case constants.StateNow:
		// nowModel is already updated above, but if we add specific keys for Now view, handle them here
	}

	return m, tea.Batch(cmds...)
}
