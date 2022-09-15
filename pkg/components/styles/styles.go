package styles

import "github.com/charmbracelet/lipgloss"

func BoldStyle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true)
}

func ColorStyle(str string) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(str))
}

func Subtle(str string) string {
	return ColorStyle("#383838").Render(str)
}

func Green(str string) string {
	return ColorStyle("#16E58A").Render(str)
}

func Blue(str string) string {
	return ColorStyle("#4D73E0").Render(str)
}

func Pink(str string) string {
	return ColorStyle("#F26DAA").Render(str)
}

func Error(str string) string {
	return ColorStyle("#FFA7A7").Render(str)
}

func Bold(str string) string {
	return BoldStyle().Render(str)
}

func Code(str string) string {
	return BoldStyle().Render(Blue(str))
}

func Highlight(str string) string {
	return BoldStyle().Background(lipgloss.Color("#F26DAA")).Render(str)
}

var (
	Question = BoldStyle().Render(Pink("?"))
	SelectTag = BoldStyle().Render(Pink(">"))
	CheckMark = BoldStyle().Render(Green("✓"))
	Info = BoldStyle().Render(Blue("i"))
)

