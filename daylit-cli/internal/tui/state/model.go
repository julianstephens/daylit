package state

import (
	"time"

	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/julianstephens/daylit/daylit-cli/internal/constants"
	"github.com/julianstephens/daylit/daylit-cli/internal/models"
	"github.com/julianstephens/daylit/daylit-cli/internal/scheduler"
	"github.com/julianstephens/daylit/daylit-cli/internal/storage"
	"github.com/julianstephens/daylit/daylit-cli/internal/tui/components/alerts"
	"github.com/julianstephens/daylit/daylit-cli/internal/tui/components/habits"
	"github.com/julianstephens/daylit/daylit-cli/internal/tui/components/now"
	"github.com/julianstephens/daylit/daylit-cli/internal/tui/components/ot"
	"github.com/julianstephens/daylit/daylit-cli/internal/tui/components/plan"
	"github.com/julianstephens/daylit/daylit-cli/internal/tui/components/settings"
	"github.com/julianstephens/daylit/daylit-cli/internal/tui/components/tasklist"
	"github.com/julianstephens/daylit/daylit-cli/internal/validation"
)

// TaskFormModel represents the form model for task editing
type TaskFormModel struct {
	Name       string
	Duration   string
	Recurrence constants.RecurrenceType
	Interval   string
	Priority   string
	Active     bool
}

// HabitFormModel represents the form model for habit creation
type HabitFormModel struct {
	Name string
}

// SettingsFormModel represents the form model for settings
type SettingsFormModel struct {
	DayStart             string
	DayEnd               string
	DefaultBlockMin      string
	Timezone             string
	PromptOnEmpty        bool
	StrictMode           bool
	DefaultLogDays       string
	NotificationsEnabled bool
	NotifyBlockStart     bool
	NotifyBlockEnd       bool
	BlockStartOffsetMin  string
	BlockEndOffsetMin    string
}

// OTFormModel represents the form model for One Thing
type OTFormModel struct {
	Title string
	Note  string
}

// AlertFormModel represents the form model for alerts
type AlertFormModel struct {
	Message    string
	Time       string
	Date       string
	Recurrence constants.RecurrenceType
	Interval   string
	Weekdays   string
}

// Model represents the shared state for the TUI
type Model struct {
	Store               storage.Provider
	Scheduler           *scheduler.Scheduler
	State               constants.SessionState
	PreviousState       constants.SessionState
	Keys                KeyMap
	Help                help.Model
	TaskList            tasklist.Model
	PlanModel           plan.Model
	NowModel            now.Model
	HabitsModel         habits.Model
	OTModel             ot.Model
	AlertsModel         alerts.Model
	SettingsModel       settings.Model
	Form                *huh.Form
	TaskForm            *TaskFormModel
	HabitForm           *HabitFormModel
	OTForm              *OTFormModel
	AlertForm           *AlertFormModel
	SettingsForm        *SettingsFormModel
	FeedbackForm        *FeedbackFormModel
	ConfirmationForm    *ConfirmationFormModel
	PendingAction       func() tea.Cmd
	EditingTask         *models.Task
	Quitting            bool
	Width               int
	Height              int
	FeedbackSlotID      int // Index of the slot being rated
	TaskToDeleteID      string
	TaskToRestoreID     string
	HabitToArchiveID    string
	ValidationWarning   string                // Validation warning message to display
	ValidationConflicts []validation.Conflict // Detailed conflict information
	PlanToDeleteDate    string
	PlanToRestoreDate   string
	PlanToOverwriteDate string
	FormError           string // Error message to display for form operations
}

// New creates a new state Model
func New(store storage.Provider, sched *scheduler.Scheduler) Model {
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
	om := ot.New(nil, 0, 0)
	if otEntry.ID != "" {
		om = ot.New(&otEntry, 0, 0)
	}

	// Initialize settings
	currentSettings, _ := store.GetSettings()
	otSettings, _ := store.GetOTSettings()
	sm := settings.New(currentSettings, otSettings, 0, 0)

	// Initialize alerts
	alertsList, _ := store.GetAllAlerts()
	am := alerts.New(alertsList, 0, 0)

	return Model{
		Store:         store,
		Scheduler:     sched,
		State:         constants.StateNow,
		Keys:          DefaultKeyMap(),
		Help:          help.New(),
		TaskList:      tasklist.New(tasks, 0, 0),
		PlanModel:     pm,
		NowModel:      nm,
		HabitsModel:   hm,
		OTModel:       om,
		AlertsModel:   am,
		SettingsModel: sm,
	}
}
