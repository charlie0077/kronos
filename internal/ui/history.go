package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/zhenchaochen/kronos/internal/store"
)

// HistoryModel manages the history tab.
type HistoryModel struct {
	runs     []store.RunRecord
	cursor   int
	filter   string // job name filter, empty = all
	filtered []store.RunRecord
	editing  bool   // whether we're typing a filter
	input    string // current filter input
}

// NewHistoryModel creates an empty history model.
func NewHistoryModel() HistoryModel {
	return HistoryModel{}
}

// Refresh reloads run records from the store.
func (m *HistoryModel) Refresh(st *store.Store) {
	const maxRecords = 200
	runs, err := st.GetAllRuns(maxRecords)
	if err != nil {
		m.runs = nil
		return
	}
	m.runs = runs
	m.applyFilter()
}

func (m *HistoryModel) applyFilter() {
	if m.filter == "" {
		m.filtered = m.runs
	} else {
		m.filtered = nil
		for _, r := range m.runs {
			if strings.Contains(r.JobName, m.filter) {
				m.filtered = append(m.filtered, r)
			}
		}
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
}

// Update handles key events for the history tab.
func (m HistoryModel) Update(msg tea.Msg) (HistoryModel, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		if m.editing {
			switch km.String() {
			case "enter":
				m.filter = m.input
				m.editing = false
				m.applyFilter()
			case "esc":
				m.editing = false
				m.input = m.filter
			case "backspace":
				if len(m.input) > 0 {
					m.input = m.input[:len(m.input)-1]
				}
			default:
				if len(km.String()) == 1 {
					m.input += km.String()
				}
			}
			return m, nil
		}

		switch km.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
		case "/":
			m.editing = true
			m.input = m.filter
		case "esc":
			if m.filter != "" {
				m.filter = ""
				m.input = ""
				m.applyFilter()
			}
		}
	}
	return m, nil
}

// View renders the history table.
func (m HistoryModel) View(width, height int) string {
	var b strings.Builder

	// Filter display.
	if m.editing {
		b.WriteString(HeaderStyle.Render("  Filter: "))
		b.WriteString(m.input)
		b.WriteString("▏\n")
	} else if m.filter != "" {
		b.WriteString(MutedStyle.Render(fmt.Sprintf("  Filter: %s (esc to clear)\n", m.filter)))
	}

	if len(m.filtered) == 0 {
		b.WriteString(MutedStyle.Render("  No run history."))
		return b.String()
	}

	const (
		colJob     = 20
		colTime    = 20
		colDur     = 10
		colResult  = 8
		colTrigger = 10
	)

	header := fmt.Sprintf("  %-*s %-*s %-*s %-*s %-*s",
		colJob, "JOB",
		colTime, "START TIME",
		colDur, "DURATION",
		colResult, "RESULT",
		colTrigger, "TRIGGER",
	)
	b.WriteString(HeaderStyle.Render(header))
	b.WriteString("\n")

	visibleRows := max(1, height-4)
	startIdx := 0
	if m.cursor >= visibleRows {
		startIdx = m.cursor - visibleRows + 1
	}
	endIdx := min(startIdx+visibleRows, len(m.filtered))

	for i := startIdx; i < endIdx; i++ {
		run := m.filtered[i]
		dur := formatDuration(run.EndTime.Sub(run.StartTime))

		var result string
		if run.Success {
			result = SuccessStyle.Render("OK")
		} else {
			result = ErrorStyle.Render(fmt.Sprintf("exit %d", run.ExitCode))
		}

		trigger := run.Trigger
		if trigger == "manual" {
			trigger = WarningStyle.Render("manual")
		}

		line := fmt.Sprintf("  %-*s %-*s %-*s %-*s %s",
			colJob, truncate(run.JobName, colJob),
			colTime, run.StartTime.Format("2006-01-02 15:04:05"),
			colDur, dur,
			colResult, result,
			trigger,
		)

		if i == m.cursor {
			b.WriteString(SelectedRowStyle.Render(padRight(line, width)))
		} else {
			b.WriteString(line)
		}
		b.WriteString("\n")
	}

	return b.String()
}
