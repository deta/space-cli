package text

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/deta/space/pkg/components/styles"
)

type errMsg error

type Model struct {
	TextInput textinput.Model
	Prompt    string
	quitting  bool
	Err       error
	Validator func(value string) error
}

type Input struct {
	Prompt       string
	Placeholder  string
	Validator    func(value string) error
	PasswordMode bool
}

func initialModel(i *Input) Model {
	ti := textinput.New()
	ti.Placeholder = i.Placeholder
	ti.Focus()
	if i.PasswordMode {
		ti.EchoMode = textinput.EchoPassword
	}

	return Model{
		TextInput: ti,
		Prompt:    i.Prompt,
		Err:       nil,
		Validator: i.Validator,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			if m.Validator != nil {
				value := m.TextInput.Value()
				if value == "" {
					value = m.TextInput.Placeholder
				}

				err := m.Validator(value)
				if err != nil {
					m.Err = err
					return m, nil
				}
			}
			m.quitting = true
			m.Err = nil
			return m, tea.Quit

		case tea.KeyCtrlC:
			os.Exit(1)
		}
	case errMsg:
		m.Err = msg
		return m, nil
	}

	m.TextInput, cmd = m.TextInput.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	var s string
	if m.TextInput.EchoMode == textinput.EchoPassword {
		s = fmt.Sprintf(
			"%s %s (%s) %s\n\n",
			styles.Question,
			styles.Bold(m.Prompt),
			fmt.Sprintf("%d chars", len(m.TextInput.Value())),
			m.TextInput.View(),
		)
	} else {
		s = fmt.Sprintf(
			"%s %s %s\n\n",
			styles.Question,
			styles.Bold(m.Prompt),
			m.TextInput.View(),
		)
	}
	if m.Err != nil {
		s += styles.Error(fmt.Sprintf("❗ Error: %s", m.Err.Error()))
	}
	return s
}

func Run(i *Input) (string, error) {
	program := tea.NewProgram(initialModel(i))

	m, err := program.StartReturningModel()
	if err != nil {
		return "", err
	}

	if m, ok := m.(Model); ok {
		if m.TextInput.Value() == "" {
			return i.Placeholder, nil
		}
		return m.TextInput.Value(), nil
	}

	return "", err
}
