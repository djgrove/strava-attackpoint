package tui

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF6B35")).
			MarginBottom(1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			MarginBottom(1)

	menuItemStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	selectedItemStyle = lipgloss.NewStyle().
				PaddingLeft(2).
				Foreground(lipgloss.Color("#FF6B35")).
				Bold(true)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			MarginTop(1)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00CC66"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF3333"))

	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFAA00"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			MarginTop(1)
)
