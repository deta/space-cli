package text

import (
	"fmt"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"os"
)

type errMsg error

type Model struct {
	TextInput textinput.Model
	Prompt    string
	quitting  bool
	err       error
}

type Input struct {
	Prompt      string
	Placeholder string
}

func initialModel(i *Input) Model {
	ti := textinput.New()
	ti.Placeholder = i.Placeholder
	ti.Focus()
	ti.CharLimit = 20
	ti.Width = 20

	return Model{
		TextInput: ti,
		Prompt:    i.Prompt,
		err:       nil,
	}
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter, tea.KeyEsc:
			m.quitting = true
			return m, tea.Quit
		case tea.KeyCtrlC:
			os.Exit(1)
		}
	case errMsg:
		m.err = msg
		return m, nil
	}

	m.TextInput, cmd = m.TextInput.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	return fmt.Sprintf(
		"? %s %s\n\n",
		m.Prompt,
		m.TextInput.View(),
	)
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
