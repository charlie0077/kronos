package ui

import (
	"os"

	"github.com/charmbracelet/lipgloss"
)

// Style definitions for the TUI.
var (
	ActiveTabStyle   lipgloss.Style
	InactiveTabStyle lipgloss.Style
	HeaderStyle      lipgloss.Style
	SelectedRowStyle lipgloss.Style
	StatusBarStyle   lipgloss.Style
	SuccessStyle     lipgloss.Style
	ErrorStyle       lipgloss.Style
	WarningStyle     lipgloss.Style
	MutedStyle       lipgloss.Style
	TabGapStyle      lipgloss.Style
)

// InitStyles initializes all styles, respecting NO_COLOR.
func InitStyles(noColor bool) {
	if noColor || os.Getenv("NO_COLOR") != "" {
		initPlainStyles()
		return
	}

	ActiveTabStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("230")).
		Background(lipgloss.Color("57")).
		Padding(0, 2)

	InactiveTabStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("250")).
		Padding(0, 2)

	TabGapStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	HeaderStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("75"))

	SelectedRowStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("230"))

	StatusBarStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	SuccessStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("82"))

	ErrorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("196"))

	WarningStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("214"))

	MutedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))
}

func initPlainStyles() {
	ActiveTabStyle = lipgloss.NewStyle().Bold(true).Padding(0, 2)
	InactiveTabStyle = lipgloss.NewStyle().Padding(0, 2)
	TabGapStyle = lipgloss.NewStyle()
	HeaderStyle = lipgloss.NewStyle().Bold(true)
	SelectedRowStyle = lipgloss.NewStyle().Reverse(true)
	StatusBarStyle = lipgloss.NewStyle()
	SuccessStyle = lipgloss.NewStyle()
	ErrorStyle = lipgloss.NewStyle()
	WarningStyle = lipgloss.NewStyle()
	MutedStyle = lipgloss.NewStyle()
}
