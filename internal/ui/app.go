package ui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/zhenchaochen/kronos/internal/config"
	"github.com/zhenchaochen/kronos/internal/logger"
	"github.com/zhenchaochen/kronos/internal/scheduler"
	"github.com/zhenchaochen/kronos/internal/store"
)

// tickMsg triggers a periodic refresh.
type tickMsg time.Time

// refreshMsg signals a job update from the scheduler callback.
type refreshMsg struct{}

// runJobDoneMsg carries the result of a manual job run.
type runJobDoneMsg struct {
	name string
	err  error
}

// Model is the root bubbletea model for the TUI.
type Model struct {
	tabs      []string
	activeTab int

	jobsModel    JobsModel
	logsModel    LogsModel
	historyModel HistoryModel

	scheduler *scheduler.Scheduler
	store     *store.Store
	logMgr    *logger.Manager
	config    *config.Config

	width      int
	height     int
	quitting   bool
	statusText string
}

// NewModel creates the root TUI model.
func NewModel(sched *scheduler.Scheduler, st *store.Store, logMgr *logger.Manager, cfg *config.Config) Model {
	m := Model{
		tabs:         []string{"Jobs", "Logs", "History"},
		scheduler:    sched,
		store:        st,
		logMgr:       logMgr,
		config:       cfg,
		jobsModel:    NewJobsModel(),
		logsModel:    NewLogsModel(),
		historyModel: NewHistoryModel(),
	}
	m.refreshAll()
	return m
}

// Init starts the tick timer.
func (m Model) Init() tea.Cmd {
	return tea.Batch(tickCmd(), tea.WindowSize())
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		m.refreshAll()
		return m, tickCmd()

	case refreshMsg:
		m.refreshAll()
		return m, nil

	case runJobDoneMsg:
		if msg.err != nil {
			m.statusText = ErrorStyle.Render(fmt.Sprintf("Run %q failed: %v", msg.name, msg.err))
		} else {
			m.statusText = SuccessStyle.Render(fmt.Sprintf("Job %q completed", msg.name))
		}
		m.refreshAll()
		return m, nil

	case tea.KeyMsg:
		m.statusText = "" // Clear status on any key press.
		// Global keys.
		switch msg.String() {
		case "ctrl+c", "q":
			if m.activeTab == TabHistory && m.historyModel.editing {
				// Let history handle it.
			} else {
				m.quitting = true
				return m, tea.Quit
			}
		case "tab":
			m.activeTab = (m.activeTab + 1) % len(m.tabs)
			m.onTabSwitch()
			return m, nil
		case "shift+tab":
			m.activeTab = (m.activeTab - 1 + len(m.tabs)) % len(m.tabs)
			m.onTabSwitch()
			return m, nil
		}

		// Tab-specific keys.
		switch m.activeTab {
		case TabJobs:
			return m.updateJobs(msg)
		case TabLogs:
			var cmd tea.Cmd
			m.logsModel, cmd = m.logsModel.Update(msg)
			return m, cmd
		case TabHistory:
			var cmd tea.Cmd
			m.historyModel, cmd = m.historyModel.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

func (m Model) updateJobs(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "r":
		// Run selected job manually.
		if name := m.jobsModel.SelectedJobName(); name != "" {
			m.statusText = MutedStyle.Render(fmt.Sprintf("Running %q...", name))
			sched := m.scheduler
			return m, func() tea.Msg {
				err := sched.RunJob(name)
				return runJobDoneMsg{name: name, err: err}
			}
		}
	case "e":
		// Enable selected job.
		if name := m.jobsModel.SelectedJobName(); name != "" {
			m.setJobEnabled(name, true)
		}
	case "d":
		// Disable selected job.
		if name := m.jobsModel.SelectedJobName(); name != "" {
			m.setJobEnabled(name, false)
		}
	case "p":
		m.scheduler.PauseAll()
	case "R":
		m.scheduler.ResumeAll()
	default:
		var cmd tea.Cmd
		m.jobsModel, cmd = m.jobsModel.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *Model) setJobEnabled(name string, enabled bool) {
	for i := range m.config.Jobs {
		if m.config.Jobs[i].Name == name {
			m.config.Jobs[i].Enabled = &enabled
			_ = m.scheduler.UpdateJobs(m.config.Jobs)
			break
		}
	}
}

func (m *Model) onTabSwitch() {
	if m.activeTab == TabLogs {
		// Sync logs tab to selected job.
		if name := m.jobsModel.SelectedJobName(); name != "" {
			m.logsModel.SetJob(name)
			m.logsModel.Refresh(m.logMgr)
		}
	}
}

func (m *Model) refreshAll() {
	m.jobsModel.Refresh(m.scheduler, m.store)
	if m.activeTab == TabLogs {
		m.logsModel.Refresh(m.logMgr)
	}
	if m.activeTab == TabHistory {
		m.historyModel.Refresh(m.store)
	}
}

// View renders the full TUI.
func (m Model) View() string {
	if m.quitting {
		return "Shutting down...\n"
	}

	tabBarHeight := 2  // tab bar + border
	statusBarHeight := 1
	contentHeight := max(1, m.height-tabBarHeight-statusBarHeight)

	tabBar := RenderTabBar(m.tabs, m.activeTab, m.width)

	var content string
	switch m.activeTab {
	case TabJobs:
		content = m.jobsModel.View(m.width, contentHeight)
	case TabLogs:
		content = m.logsModel.View(m.width, contentHeight)
	case TabHistory:
		content = m.historyModel.View(m.width, contentHeight)
	}

	var statusBar string
	if m.statusText != "" {
		statusBar = m.statusText
	} else {
		statusBar = RenderStatusBar(m.activeTab, m.width)
	}

	return tabBar + "\n" + content + "\n" + statusBar
}

// RefreshCmd returns a tea.Cmd that sends a refreshMsg (for use by scheduler callback).
func RefreshCmd() tea.Msg {
	return refreshMsg{}
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
