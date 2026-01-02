package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/julianstephens/daylit/daylit-cli/internal/constants"
	"github.com/julianstephens/daylit/daylit-cli/internal/models"
	"github.com/julianstephens/daylit/daylit-cli/internal/scheduler"
	"github.com/julianstephens/daylit/daylit-cli/internal/storage"
	"github.com/julianstephens/daylit/daylit-cli/internal/tui/components/habits"
	"github.com/julianstephens/daylit/daylit-cli/internal/tui/components/now"
	"github.com/julianstephens/daylit/daylit-cli/internal/tui/components/ot"
	"github.com/julianstephens/daylit/daylit-cli/internal/tui/components/plan"
	"github.com/julianstephens/daylit/daylit-cli/internal/tui/components/settings"
	"github.com/julianstephens/daylit/daylit-cli/internal/tui/components/tasklist"
	"github.com/julianstephens/daylit/daylit-cli/internal/validation"
)

type SessionState int

const (
	StateNow SessionState = iota
	StatePlan
	StateTasks
	StateHabits
	StateOT
	StateSettings
	StateFeedback
	StateEditing
	StateConfirmDelete
	StateConfirmRestore
	StateConfirmOverwrite
	StateConfirmArchive
	StateAddHabit
	StateEditOT
	StateEditSettings
)

type TaskFormModel struct {
	Name       string
	Duration   string
	Recurrence models.RecurrenceType
	Interval   string
	Priority   string
	Active     bool
}

type HabitFormModel struct {
	Name string
}

type SettingsFormModel struct {
	DayStart             string
	DayEnd               string
	DefaultBlockMin      string
	PromptOnEmpty        bool
	StrictMode           bool
	DefaultLogDays       string
	NotificationsEnabled bool
	NotifyBlockStart     bool
	NotifyBlockEnd       bool
	BlockStartOffsetMin  string
	BlockEndOffsetMin    string
}

type OTFormModel struct {
	Title string
	Note  string
}

type Model struct {
	store               storage.Provider
	scheduler           *scheduler.Scheduler
	state               SessionState
	previousState       SessionState
	keys                KeyMap
	help                help.Model
	taskList            tasklist.Model
	planModel           plan.Model
	nowModel            now.Model
	habitsModel         habits.Model
	otModel             ot.Model
	settingsModel       settings.Model
	form                *huh.Form
	taskForm            *TaskFormModel
	habitForm           *HabitFormModel
	otForm              *OTFormModel
	settingsForm        *SettingsFormModel
	editingTask         *models.Task
	quitting            bool
	width               int
	height              int
	feedbackSlotID      int // Index of the slot being rated
	taskToDeleteID      string
	taskToRestoreID     string
	habitToArchiveID    string
	validationWarning   string                // Validation warning message to display
	validationConflicts []validation.Conflict // Detailed conflict information
	planToDeleteDate    string
	planToRestoreDate   string
	planToOverwriteDate string
}

func NewModel(store storage.Provider, sched *scheduler.Scheduler) Model {
	today := time.Now().Format(constants.DateFormat)
	planData, planErr := store.GetPlan(today)
	pm := plan.New(0, 0)
	nm := now.New()
	tasks, taskErr := store.GetAllTasksIncludingDeleted()
	if taskErr != nil {
		// Initialize with empty task list on error
		tasks = []models.Task{}
	}
	if planErr == nil {
		pm.SetPlan(planData, tasks)
		nm.SetPlan(planData, tasks)
	}

	// Initialize habits
	habitsList, _ := store.GetAllHabits(false, true) // includeArchived=false, includeDeleted=true
	habitEntries, _ := store.GetHabitEntriesForDay(today)
	hm := habits.New(habitsList, habitEntries, 0, 0)

	// Initialize OT
	otEntry, _ := store.GetOTEntry(today)
	var otEntryPtr *models.OTEntry
	if otEntry.ID != "" {
		otEntryPtr = &otEntry
	}
	om := ot.New(otEntryPtr, 0, 0)

	// Initialize settings
	currentSettings, _ := store.GetSettings()
	otSettings, _ := store.GetOTSettings()
	sm := settings.New(currentSettings, otSettings, 0, 0)

	m := Model{
		store:         store,
		scheduler:     sched,
		state:         StateNow,
		keys:          DefaultKeyMap(),
		help:          help.New(),
		taskList:      tasklist.New(tasks, 0, 0),
		planModel:     pm,
		nowModel:      nm,
		habitsModel:   hm,
		otModel:       om,
		settingsModel: sm,
	}

	// Run validation on initialization
	m.updateValidationStatus()

	return m
}

func (m Model) ShortHelp() []key.Binding {
	keys := []key.Binding{m.keys.Tab, m.keys.Quit, m.keys.Help}
	switch m.state {
	case StateTasks:
		keys = append(keys, m.keys.Add, m.keys.Edit, m.keys.Delete)
	case StatePlan:
		keys = append(keys, m.keys.Generate)
	case StateHabits:
		keys = append(keys, m.keys.Add)
	}
	keys = append(keys, m.keys.Feedback)
	return keys
}

func (m Model) FullHelp() [][]key.Binding {
	global := []key.Binding{m.keys.Tab, m.keys.ShiftTab, m.keys.Quit, m.keys.Help, m.keys.Feedback}
	navigation := []key.Binding{m.keys.Up, m.keys.Down, m.keys.Left, m.keys.Right, m.keys.Enter}

	var actions []key.Binding
	switch m.state {
	case StateTasks:
		actions = []key.Binding{m.keys.Add, m.keys.Edit, m.keys.Delete}
	case StatePlan:
		actions = []key.Binding{m.keys.Generate}
	case StateHabits:
		actions = []key.Binding{m.keys.Add}
	}

	return [][]key.Binding{global, navigation, actions}
}

func (m Model) Init() tea.Cmd {
	return m.nowModel.Init()
}

// updateValidationStatus runs validation and updates the warning message
func (m *Model) updateValidationStatus() {
	// Get all tasks
	tasks, err := m.store.GetAllTasks()
	if err != nil {
		// Store errors prevent validation - show generic message
		m.validationWarning = "⚠ Validation unavailable"
		m.validationConflicts = nil
		return
	}

	// Get settings
	settings, err := m.store.GetSettings()
	if err != nil {
		// Store errors prevent validation - show generic message
		m.validationWarning = "⚠ Validation unavailable"
		m.validationConflicts = nil
		return
	}

	// Get today's plan
	today := time.Now().Format(constants.DateFormat)
	plan, err := m.store.GetPlan(today)

	validator := validation.New()

	// Validate tasks first
	taskResult := validator.ValidateTasks(tasks)

	// Validate plan if it exists
	var planResult validation.ValidationResult
	if err == nil && len(plan.Slots) > 0 {
		planResult = validator.ValidatePlan(plan, tasks, settings.DayStart, settings.DayEnd)
	}

	// Combine conflicts
	allConflicts := append(taskResult.Conflicts, planResult.Conflicts...)
	m.validationConflicts = allConflicts

	if len(allConflicts) > 0 {
		// Show count of conflicts
		m.validationWarning = fmt.Sprintf("⚠ %d validation warning(s)", len(allConflicts))
	} else {
		m.validationWarning = ""
	}
}
