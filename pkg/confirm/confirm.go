package confirm

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	Prompt  string
	Confirm bool
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
			return m, tea.Quit
		case "n", "N":
			m.Confirm = false
			return m, tea.Quit
		case "esc", "enter":
			return m, tea.Quit
		case "ctrl+c":
			os.Exit(1)
		}
	}
	return m, cmd
}

func (m Model) View() string {
	return fmt.Sprintf("? %s (y/n)\n\n", m.Prompt)
}

func Run(i *Input) (*Model, error) {
	program := tea.NewProgram(initialModel(i))

	m, err := program.StartReturningModel()
	if err != nil {
		return nil, err
	}

	if m, ok := m.(Model); ok {
		return &m, nil
	}

	return nil, err
}
