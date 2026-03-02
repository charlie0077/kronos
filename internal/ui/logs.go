package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/zhenchaochen/kronos/internal/logger"
)

// LogsModel manages the logs tab displaying live tail of a job's log.
type LogsModel struct {
	jobName string
	lines   []string
	offset  int // scroll offset from bottom
}

// NewLogsModel creates an empty logs model.
func NewLogsModel() LogsModel {
	return LogsModel{}
}

// SetJob switches the viewed job and resets scroll.
func (m *LogsModel) SetJob(name string) {
	if m.jobName != name {
		m.jobName = name
		m.lines = nil
		m.offset = 0
	}
}

// Refresh reloads log lines from the logger.
func (m *LogsModel) Refresh(logMgr *logger.Manager) {
	if m.jobName == "" {
		m.lines = nil
		return
	}
	const maxLines = 500
	l := logMgr.GetLogger(m.jobName)
	lines, err := l.Tail(maxLines)
	if err != nil {
		m.lines = []string{fmt.Sprintf("Error reading log: %v", err)}
		return
	}
	m.lines = lines
}

// Update handles key events for the logs tab.
func (m LogsModel) Update(msg tea.Msg) (LogsModel, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.String() {
		case "up", "k":
			if m.offset < len(m.lines)-1 {
				m.offset++
			}
		case "down", "j":
			if m.offset > 0 {
				m.offset--
			}
		case "home", "g":
			m.offset = max(0, len(m.lines)-1)
		case "end", "G":
			m.offset = 0
		}
	}
	return m, nil
}

// View renders the logs.
func (m LogsModel) View(width, height int) string {
	if m.jobName == "" {
		return MutedStyle.Render("  Select a job from the Jobs tab to view logs.")
	}

	header := HeaderStyle.Render(fmt.Sprintf("  Logs: %s", m.jobName))

	if len(m.lines) == 0 {
		return header + "\n" + MutedStyle.Render("  No log output yet.")
	}

	visibleLines := max(1, height-2)
	endIdx := len(m.lines) - m.offset
	startIdx := max(0, endIdx-visibleLines)
	if endIdx < 0 {
		endIdx = 0
	}
	if startIdx >= endIdx {
		startIdx = max(0, endIdx-1)
	}

	var b strings.Builder
	b.WriteString(header)
	b.WriteString("\n")

	for i := startIdx; i < endIdx; i++ {
		line := m.lines[i]
		if len(line) > width-2 {
			line = line[:width-2]
		}
		b.WriteString("  ")
		b.WriteString(line)
		b.WriteString("\n")
	}

	// Scroll indicator.
	if m.offset > 0 {
		indicator := MutedStyle.Render(fmt.Sprintf("  ↓ %d more lines below", m.offset))
		b.WriteString(indicator)
	}

	return b.String()
}
