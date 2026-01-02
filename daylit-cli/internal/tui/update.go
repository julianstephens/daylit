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

	"github.com/julianstephens/daylit/daylit-cli/internal/constants"
	"github.com/julianstephens/daylit/daylit-cli/internal/models"
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
				tasks, err := m.store.GetAllTasksIncludingDeleted()
				if err == nil {
					m.taskList.SetTasks(tasks)
				}
				m.updateValidationStatus()
			}
			m.state = StateTasks
		case huh.StateAborted:
			m.state = StateTasks
		}
		return m, tea.Batch(cmds...)
	}

	// Handle Add Habit State
	if m.state == StateAddHabit {
		if msg, ok := msg.(tea.KeyMsg); ok && msg.Type == tea.KeyEsc {
			m.state = StateHabits
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
				m.state = StateHabits
			} else {
				// Stay in form state on error to allow retry
				// The form will display, user can cancel with ESC or retry
				m.form.State = huh.StateNormal
			}
		case huh.StateAborted:
			m.state = StateHabits
		}
		return m, tea.Batch(cmds...)
	}

	// Handle Edit OT State
	if m.state == StateEditOT {
		if msg, ok := msg.(tea.KeyMsg); ok && msg.Type == tea.KeyEsc {
			m.formError = "" // Clear error on cancel
			m.state = StateOT
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
			if err == nil && existingEntry.ID != "" {
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
				if err == nil {
					entryPtr := &updatedEntry
					m.otModel.SetEntry(entryPtr)
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
				if err == nil {
					entryPtr := &savedEntry
					m.otModel.SetEntry(entryPtr)
				}
			}
			m.formError = "" // Clear any previous errors
			m.state = StateOT
		case huh.StateAborted:
			m.formError = "" // Clear error on abort
			m.state = StateOT
		}
		return m, tea.Batch(cmds...)
	}

	// Handle Edit Settings State
	if m.state == StateEditSettings {
		if msg, ok := msg.(tea.KeyMsg); ok && msg.Type == tea.KeyEsc {
			m.state = StateSettings
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
			m.state = StateSettings
		case huh.StateAborted:
			m.state = StateSettings
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
			today := time.Now().Format(constants.DateFormat)
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
				tasks, err := m.store.GetAllTasksIncludingDeleted()
				if err == nil {
					m.taskList.SetTasks(tasks)
				}
				m.updateValidationStatus()
				m.state = StateTasks
				m.taskToDeleteID = ""
			case "n", "N", "esc", "q":
				m.state = StateTasks
				m.taskToDeleteID = ""
			}
		}
		return m, nil
	}

	// Handle Confirm Restore State
	if m.state == StateConfirmRestore {
		if msg, ok := msg.(tea.KeyMsg); ok {
			switch msg.String() {
			case "y", "Y":
				if m.taskToRestoreID != "" {
					if err := m.store.RestoreTask(m.taskToRestoreID); err != nil {
						// On error, silently return to tasks view
						m.state = StateTasks
						m.taskToRestoreID = ""
						return m, nil
					}
					// Restore succeeded - refresh and clear state
					tasks, err := m.store.GetAllTasksIncludingDeleted()
					if err == nil {
						m.taskList.SetTasks(tasks)
					}
					m.updateValidationStatus()
					m.state = StateTasks
					m.taskToRestoreID = ""
				} else if m.planToRestoreDate != "" {
					if err := m.store.RestorePlan(m.planToRestoreDate); err != nil {
						// On error, silently return to plan view
						m.state = StatePlan
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
					m.state = StatePlan
					m.planToRestoreDate = ""
				}
			case "n", "N", "esc", "q":
				if m.planToRestoreDate != "" {
					m.state = StatePlan
				} else {
					m.state = StateTasks
				}
				m.taskToRestoreID = ""
				m.planToRestoreDate = ""
			}
		}
		return m, nil
	}

	// Handle Confirm Overwrite State
	if m.state == StateConfirmOverwrite {
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
				m.state = StatePlan
				m.planToOverwriteDate = ""
			case "n", "N", "esc", "q":
				m.state = StatePlan
				m.planToOverwriteDate = ""
			}
		}
		return m, nil
	}

	// Handle Confirm Archive State
	if m.state == StateConfirmArchive {
		if msg, ok := msg.(tea.KeyMsg); ok {
			switch msg.String() {
			case "y", "Y":
				if err := m.store.ArchiveHabit(m.habitToArchiveID); err != nil {
					m.state = StateHabits
					m.habitToArchiveID = ""
					return m, nil
				}
				// Refresh habits list
				today := time.Now().Format(constants.DateFormat)
				habitsList, _ := m.store.GetAllHabits(false, true)
				habitEntries, _ := m.store.GetHabitEntriesForDay(today)
				m.habitsModel.SetHabits(habitsList, habitEntries)
				m.state = StateHabits
				m.habitToArchiveID = ""
			case "n", "N", "esc", "q":
				m.state = StateHabits
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
		m.settingsModel.SetSize(msg.Width-h, listHeight-v)

	case tasklist.DeleteTaskMsg:
		m.taskToDeleteID = msg.ID
		m.state = StateConfirmDelete
		return m, nil

	case tasklist.RestoreTaskMsg:
		m.taskToRestoreID = msg.ID
		m.state = StateConfirmRestore
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

	// Handle habit messages
	case habits.AddHabitMsg:
		m.habitForm = &HabitFormModel{
			Name: "",
		}
		m.form = newHabitForm(m.habitForm)
		m.state = StateAddHabit
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
		m.state = StateConfirmArchive
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

	// Handle settings messages
	case settings.EditSettingsMsg:
		currentSettings, _ := m.store.GetSettings()
		currentOTSettings, _ := m.store.GetOTSettings()
		m.settingsForm = &SettingsFormModel{
			DayStart:             currentSettings.DayStart,
			DayEnd:               currentSettings.DayEnd,
			DefaultBlockMin:      strconv.Itoa(currentSettings.DefaultBlockMin),
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
		m.state = StateEditSettings
		return m, m.form.Init()

	// Handle OT messages
	case ot.EditOTMsg:
		today := time.Now().Format(constants.DateFormat)
		existingEntry, err := m.store.GetOTEntry(today)

		// Handle error explicitly - distinguish between "not found" and actual errors
		if err != nil {
			// If it's not a "not found" error, we might have a database issue
			// For now, we'll initialize with empty values but could add error handling
			existingEntry = models.OTEntry{}
		}

		m.otForm = &OTFormModel{
			Title: existingEntry.Title,
			Note:  existingEntry.Note,
		}
		m.formError = "" // Clear any previous form errors
		m.form = newOTForm(m.otForm)
		m.state = StateEditOT
		return m, m.form.Init()

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			m.quitting = true
			return m, tea.Quit
		case key.Matches(msg, m.keys.Tab, m.keys.Right):
			m.state = (m.state + 1) % NumMainTabs
			return m, nil
		case key.Matches(msg, m.keys.ShiftTab, m.keys.Left):
			m.state = (m.state - 1 + NumMainTabs) % NumMainTabs
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
					if (slot.Status == models.SlotStatusAccepted || slot.Status == models.SlotStatusDone) &&
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
			today := time.Now().Format(constants.DateFormat)

			// Check if plan already exists
			_, err := m.store.GetPlan(today)
			if err == nil {
				// Plan exists, ask for confirmation
				m.planToOverwriteDate = today
				m.state = StateConfirmOverwrite
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
	case StateHabits:
		m.habitsModel, cmd = m.habitsModel.Update(msg)
		cmds = append(cmds, cmd)
	case StateOT:
		m.otModel, cmd = m.otModel.Update(msg)
		cmds = append(cmds, cmd)
	case StateSettings:
		m.settingsModel, cmd = m.settingsModel.Update(msg)
		cmds = append(cmds, cmd)
	case StateNow:
		// nowModel is already updated above, but if we add specific keys for Now view, handle them here
	}

	return m, tea.Batch(cmds...)
}
