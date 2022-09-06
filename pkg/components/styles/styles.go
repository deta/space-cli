package styles

import "github.com/charmbracelet/lipgloss"

var (
	// colors
	Subtle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#383838"))
	Green     = lipgloss.NewStyle().Foreground(lipgloss.Color("#16E58A"))
	Highlight = lipgloss.NewStyle().Foreground(lipgloss.Color("#F26DAA"))
	Success   = lipgloss.NewStyle().Foreground(lipgloss.Color("#95CAA3"))
	Error     = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFA7A7"))

	Question = Green.Copy().Bold(true).Render("?")

	SelectTag = Green.Copy().Bold(true).Render(">")
)
