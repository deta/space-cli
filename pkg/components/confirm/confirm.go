package confirm

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/deta/space/pkg/components/styles"
)

type Model struct {
	Prompt    string
	Confirm   bool
	Quitting  bool
	Cancelled bool
}

type Input struct {
	Prompt string
}

func initialModel(input string) Model {
	return Model{
		Prompt: input,
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
			m.Quitting = true
			return m, tea.Quit
		case "n", "N":
			m.Confirm = false
			m.Quitting = true
			return m, tea.Quit
		case "enter":
			m.Confirm = true
			m.Quitting = true
			return m, tea.Quit
		case "ctrl+c":
			m.Cancelled = true
			m.Quitting = true
			return m, tea.Quit
		}
	}
	return m, cmd
}

func (m Model) View() string {
	input := "(Y/n)"
	if m.Quitting && m.Confirm {
		input = "y"
	} else if m.Quitting && !m.Confirm {
		input = "n"
	}

	return fmt.Sprintf("%s %s %s\n", styles.Question, styles.Bold(m.Prompt), styles.Subtle(input))
}

func Run(input string) (bool, error) {
	program := tea.NewProgram(initialModel(input))

	m, err := program.Run()
	if err != nil {
		return false, err
	}

	model, ok := m.(Model)
	if !ok {
		return false, err
	}

	if model.Cancelled {
		return false, fmt.Errorf("cancelled")
	}
	return model.Confirm, nil
}
