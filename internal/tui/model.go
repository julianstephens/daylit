package tui

import (
	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/julianstephens/daylit/internal/scheduler"
	"github.com/julianstephens/daylit/internal/storage"
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
	quitting  bool
	width     int
	height    int
}

func NewModel(store *storage.Storage, sched *scheduler.Scheduler) Model {
	return Model{
		store:     store,
		scheduler: sched,
		state:     StateNow,
		keys:      DefaultKeyMap(),
		help:      help.New(),
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}
