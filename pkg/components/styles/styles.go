package styles

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (
	SubtleStyle = ColorStyle("#383838")
	GreenStyle  = ColorStyle("#16E58A")
	BlueStyle   = ColorStyle("#4D73E0")
	PinkStyle   = ColorStyle("#F26DAA")
	ErrorStyle  = ColorStyle("#FFA7A7")
	BoldStyle   = lipgloss.NewStyle().Bold(true)
)

func ColorStyle(str string) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(str))
}

func Subtlef(str string, a ...interface{}) string {
	return SubtleStyle.Render(fmt.Sprintf(str, a...))
}

func Subtle(str string) string {
	return SubtleStyle.Render(str)
}

func Greenf(str string, a ...interface{}) string {
	return GreenStyle.Render(fmt.Sprintf(str, a...))
}

func Green(str string) string {
	return GreenStyle.Render(str)
}

func Bluef(str string, a ...interface{}) string {
	return BlueStyle.Render(fmt.Sprintf(str, a...))
}

func Blue(str string) string {
	return BlueStyle.Render(str)
}

func Pinkf(str string, a ...interface{}) string {
	return PinkStyle.Render(fmt.Sprintf(str, a...))
}

func Pink(str string) string {
	return PinkStyle.Render(str)
}

func Errorf(str string, a ...interface{}) string {
	return ErrorStyle.Render(fmt.Sprintf(str, a...))
}

func Error(str string) string {
	return ErrorStyle.Render(str)
}

func Boldf(str string, a ...interface{}) string {
	return BoldStyle.Render(fmt.Sprintf(str, a...))
}

func Bold(str string) string {
	return BoldStyle.Render(str)
}

func Codef(str string, a ...interface{}) string {
	return BoldStyle.Render(Bluef(str, a...))
}

func Code(str string) string {
	return BoldStyle.Render(Blue(str))
}

func Highlightf(str string, a ...interface{}) string {
	return BoldStyle.Background(PinkStyle.GetForeground()).Render(fmt.Sprintf(str, a...))
}

func Highlight(str string) string {
	return BoldStyle.Background(PinkStyle.GetForeground()).Render(str)
}

var (
	Question         = BoldStyle.Render(Pink("?"))
	SelectTag        = BoldStyle.Render(Pink(">"))
	CheckMark        = BoldStyle.Render(Green("✓"))
	X                = BoldStyle.Render(ErrorStyle.Render("x"))
	ErrorExclamation = BoldStyle.Render(ErrorStyle.Render("!"))
	Info             = BoldStyle.Render(Blue("i"))
)
