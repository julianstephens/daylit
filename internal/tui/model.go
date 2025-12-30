package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/julianstephens/daylit/internal/scheduler"
	"github.com/julianstephens/daylit/internal/storage"
	"github.com/julianstephens/daylit/internal/tui/components/now"
	"github.com/julianstephens/daylit/internal/tui/components/plan"
	"github.com/julianstephens/daylit/internal/tui/components/tasklist"
)

type SessionState int

const (
	StateNow SessionState = iota
	StatePlan
	StateTasks
)

type Model struct {
	store     *storage.Storage
	scheduler *scheduler.Scheduler
	state     SessionState
	keys      KeyMap
	help      help.Model
	taskList  tasklist.Model
	planModel plan.Model
	nowModel  now.Model
	quitting  bool
	width     int
	height    int
}

func NewModel(store *storage.Storage, sched *scheduler.Scheduler) Model {
	today := time.Now().Format("2006-01-02")
	planData, err := store.GetPlan(today)
	pm := plan.New(0, 0)
	nm := now.New()
	if err == nil {
		pm.SetPlan(planData, store.GetAllTasks())
		nm.SetPlan(planData, store.GetAllTasks())
	}

	return Model{
		store:     store,
		scheduler: sched,
		state:     StateNow,
		keys:      DefaultKeyMap(),
		help:      help.New(),
		taskList:  tasklist.New(store.GetAllTasks(), 0, 0),
		planModel: pm,
		nowModel:  nm,
	}
}

func (m Model) Init() tea.Cmd {
	return m.nowModel.Init()
}
