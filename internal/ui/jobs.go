package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/zhenchaochen/kronos/internal/scheduler"
	"github.com/zhenchaochen/kronos/internal/store"
)

// JobRow holds display data for a single job.
type JobRow struct {
	Name     string
	Schedule string
	Enabled  bool
	LastRun  string
	NextRun  string
	Status   string // "running", "idle", "disabled"
	LastOK   *bool  // nil=no runs, true=success, false=failure
}

// JobsModel manages the jobs tab.
type JobsModel struct {
	rows   []JobRow
	cursor int
}

// NewJobsModel creates an empty jobs model.
func NewJobsModel() JobsModel {
	return JobsModel{}
}

// Refresh rebuilds the job rows from the scheduler and store.
func (m *JobsModel) Refresh(sched *scheduler.Scheduler, st *store.Store) {
	entries := sched.GetEntries()
	m.rows = make([]JobRow, 0, len(entries))

	for _, e := range entries {
		row := JobRow{
			Name:     e.Name,
			Schedule: e.Schedule,
			Enabled:  e.Enabled,
		}

		if !e.Enabled {
			row.Status = "disabled"
		} else {
			row.Status = "idle"
		}

		if !e.NextRun.IsZero() {
			row.NextRun = e.NextRun.Format("15:04:05")
		} else {
			row.NextRun = "—"
		}

		if last, err := st.GetLastRun(e.Name); err == nil && last != nil {
			row.LastRun = last.EndTime.Format("15:04:05")
			row.LastOK = &last.Success
		} else {
			row.LastRun = "—"
		}

		m.rows = append(m.rows, row)
	}

	if m.cursor >= len(m.rows) {
		m.cursor = max(0, len(m.rows)-1)
	}
}

// SelectedJobName returns the name of the currently selected job.
func (m *JobsModel) SelectedJobName() string {
	if len(m.rows) == 0 {
		return ""
	}
	return m.rows[m.cursor].Name
}

// Update handles key events for the jobs tab.
func (m JobsModel) Update(msg tea.Msg) (JobsModel, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.rows)-1 {
				m.cursor++
			}
		}
	}
	return m, nil
}

// View renders the jobs table.
func (m JobsModel) View(width, height int) string {
	if len(m.rows) == 0 {
		return MutedStyle.Render("  No jobs configured.")
	}

	// Column widths.
	const (
		colName     = 20
		colSchedule = 15
		colStatus   = 10
		colLastRun  = 10
		colNextRun  = 10
		colResult   = 10
	)

	header := fmt.Sprintf("  %-*s %-*s %-*s %-*s %-*s %-*s",
		colName, "NAME",
		colSchedule, "SCHEDULE",
		colStatus, "STATUS",
		colLastRun, "LAST RUN",
		colNextRun, "NEXT RUN",
		colResult, "RESULT",
	)

	var b strings.Builder
	b.WriteString(HeaderStyle.Render(header))
	b.WriteString("\n")

	visibleRows := max(1, height-2) // subtract header + padding
	startIdx := 0
	if m.cursor >= visibleRows {
		startIdx = m.cursor - visibleRows + 1
	}
	endIdx := min(startIdx+visibleRows, len(m.rows))

	for i := startIdx; i < endIdx; i++ {
		row := m.rows[i]
		status := renderStatus(row)
		result := renderResult(row)

		line := fmt.Sprintf("  %-*s %-*s %-*s %-*s %-*s %s",
			colName, truncate(row.Name, colName),
			colSchedule, truncate(row.Schedule, colSchedule),
			colStatus, status,
			colLastRun, row.LastRun,
			colNextRun, row.NextRun,
			result,
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

func renderStatus(row JobRow) string {
	switch row.Status {
	case "running":
		return SuccessStyle.Render("running")
	case "disabled":
		return MutedStyle.Render("disabled")
	default:
		return "idle"
	}
}

func renderResult(row JobRow) string {
	if row.LastOK == nil {
		return MutedStyle.Render("—")
	}
	if *row.LastOK {
		return SuccessStyle.Render("OK")
	}
	return ErrorStyle.Render("FAIL")
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

// formatDuration formats a duration for display.
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}
