package text

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/deta/pc-cli/pkg/components/styles"
)

type Model struct {
	TextInput     textinput.Model
	Hidden        bool
	Cancelled     bool
	Prompt        string
	ValidationMsg string
	Validator     func(value string) error
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
		Validator: i.Validator,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Value() string {
	if m.TextInput.Value() == "" {
		return m.TextInput.Placeholder
	}
	return m.TextInput.Value()
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
					m.ValidationMsg = fmt.Sprintf("❗ Error: %s", err.Error())
					return m, nil
				}
			}
			m.Hidden = true
			return m, tea.Quit

		case tea.KeyCtrlC:
			m.Cancelled = true
			m.Hidden = true
			return m, tea.Quit
		}
	}

	m.TextInput, cmd = m.TextInput.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	if m.Hidden {
		return ""
	}
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
	if m.ValidationMsg != "" {
		s += m.ValidationMsg
	}
	return s
}

func Run(i *Input) (string, error) {
	program := tea.NewProgram(initialModel(i))

	m, err := program.Run()
	if err != nil {
		return "", err
	}

	model, ok := m.(Model)
	if !ok {
		return "", fmt.Errorf("invalid model type")
	}

	if model.Cancelled {
		return "", fmt.Errorf("cancelled")
	}

	return model.Value(), nil
}
