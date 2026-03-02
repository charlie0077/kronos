package ui

// Tab index constants.
const (
	TabJobs    = 0
	TabLogs    = 1
	TabHistory = 2
)

// RenderStatusBar renders context-sensitive key hints.
func RenderStatusBar(activeTab int, width int) string {
	var hints string
	switch activeTab {
	case TabJobs:
		hints = "  [r]un  [e]nable  [d]isable  [p]ause-all  [R]esume-all  tab:switch  q:quit"
	case TabLogs:
		hints = "  [j/k]scroll  [g]top  [G]bottom  tab:switch  q:quit"
	case TabHistory:
		hints = "  [/]filter  [j/k]scroll  tab:switch  q:quit"
	}
	return StatusBarStyle.Render(padRight(hints, width))
}
