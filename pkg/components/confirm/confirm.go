package confirm

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/deta/pc-cli/pkg/components/styles"
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
		case "enter":
			return m, tea.Quit
		case "ctrl+c":
			os.Exit(1)
		}
	}
	return m, cmd
}

func (m Model) View() string {
	return fmt.Sprintf("%s %s %s\n\n", styles.Question, styles.Bold(m.Prompt), styles.Subtle("(y/n)"))
}

func Run(i *Input) (bool, error) {
	program := tea.NewProgram(initialModel(i))

	m, err := program.StartReturningModel()
	if err != nil {
		return false, err
	}

	if m, ok := m.(Model); ok {
		return m.Confirm, nil
	}

	return false, err
}
