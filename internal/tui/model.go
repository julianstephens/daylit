package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/julianstephens/daylit/internal/models"
	"github.com/julianstephens/daylit/internal/scheduler"
	"github.com/julianstephens/daylit/internal/storage"
	"github.com/julianstephens/daylit/internal/tui/components/now"
	"github.com/julianstephens/daylit/internal/tui/components/plan"
	"github.com/julianstephens/daylit/internal/tui/components/tasklist"
	"github.com/julianstephens/daylit/internal/validation"
)

type SessionState int

const (
	StateNow SessionState = iota
	StatePlan
	StateTasks
	StateFeedback
	StateEditing
	StateConfirmDelete
)

type TaskFormModel struct {
	Name       string
	Duration   string
	Recurrence models.RecurrenceType
	Interval   string
	Priority   string
	Active     bool
}

type Model struct {
	store             storage.Provider
	scheduler         *scheduler.Scheduler
	state             SessionState
	previousState     SessionState
	keys              KeyMap
	help              help.Model
	taskList          tasklist.Model
	planModel         plan.Model
	nowModel          now.Model
	form              *huh.Form
	taskForm          *TaskFormModel
	editingTask       *models.Task
	quitting          bool
	width             int
	height            int
	feedbackSlotID    int // Index of the slot being rated
	taskToDeleteID    string
	validationWarning string // Validation warning message to display
}

func NewModel(store storage.Provider, sched *scheduler.Scheduler) Model {
	today := time.Now().Format("2006-01-02")
	planData, planErr := store.GetPlan(today)
	pm := plan.New(0, 0)
	nm := now.New()
	tasks, taskErr := store.GetAllTasks()
	if taskErr != nil {
		// Initialize with empty task list on error
		tasks = []models.Task{}
	}
	if planErr == nil {
		pm.SetPlan(planData, tasks)
		nm.SetPlan(planData, tasks)
	}

	m := Model{
		store:     store,
		scheduler: sched,
		state:     StateNow,
		keys:      DefaultKeyMap(),
		help:      help.New(),
		taskList:  tasklist.New(tasks, 0, 0),
		planModel: pm,
		nowModel:  nm,
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
		return
	}

	// Get settings
	settings, err := m.store.GetSettings()
	if err != nil {
		// Store errors prevent validation - show generic message
		m.validationWarning = "⚠ Validation unavailable"
		return
	}

	// Get today's plan
	today := time.Now().Format("2006-01-02")
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

	if len(allConflicts) > 0 {
		// Show count of conflicts
		m.validationWarning = fmt.Sprintf("⚠ %d validation warning(s)", len(allConflicts))
	} else {
		m.validationWarning = ""
	}
}
