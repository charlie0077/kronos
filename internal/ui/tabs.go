package ui

import "strings"

// RenderTabBar renders a horizontal tab bar with the active tab highlighted.
func RenderTabBar(tabs []string, active int, width int) string {
	var parts []string
	for i, t := range tabs {
		if i == active {
			parts = append(parts, ActiveTabStyle.Render(t))
		} else {
			parts = append(parts, InactiveTabStyle.Render(t))
		}
	}
	bar := strings.Join(parts, TabGapStyle.Render(" │ "))
	border := strings.Repeat("─", max(0, width-1))
	return bar + "\n" + MutedStyle.Render(border)
}
