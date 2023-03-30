package confirm

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/deta/space/pkg/components/styles"
)

type Model struct {
	Prompt    string
	Hidden    bool
	Confirm   bool
	Cancelled bool
}

type Input struct {
	Prompt string
}

func initialModel(i *Input) Model {
	return Model{
		Prompt: i.Prompt,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y":
			m.Confirm = true
			m.Hidden = true
			return m, tea.Quit
		case "n", "N":
			m.Confirm = false
			m.Hidden = true
			return m, tea.Quit
		case "enter":
			m.Confirm = true
			m.Hidden = true
			return m, tea.Quit
		case "ctrl+c":
			m.Cancelled = true
			m.Hidden = true
			return m, tea.Quit
		}
	}
	return m, cmd
}

func (m Model) View() string {
	if m.Hidden {
		return ""
	}
	return fmt.Sprintf("\n%s %s\n\n", styles.Question, styles.Bold(m.Prompt))
}

func Run(i *Input) (bool, error) {
	program := tea.NewProgram(initialModel(i))

	m, err := program.Run()
	if err != nil {
		return false, err
	}

	if m, ok := m.(Model); ok {
		if m.Cancelled {
			return false, fmt.Errorf("cancelled")
		}
		return m.Confirm, nil
	}

	return false, err
}
