package textarea

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/deta/pc-cli/pkg/components/styles"
)

type errMsg error

type Model struct {
	TextArea  textarea.Model
	Hidden    bool
	Prompt    string
	Cancelled bool
	Err       error
}

type Input struct {
	Prompt      string
	Placeholder string
}

func initialModel(i *Input) Model {
	ti := textarea.New()
	ti.Placeholder = i.Placeholder
	ti.Focus()

	return Model{
		TextArea: ti,
		Prompt:   i.Prompt,
		Err:      nil,
	}
}

func (m Model) Init() tea.Cmd {
	return textarea.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlS:
			m.Hidden = true
			return m, tea.Quit
		case tea.KeyCtrlC:
			m.Hidden = true
			m.Cancelled = true
			return m, tea.Quit
		default:
			if !m.TextArea.Focused() {
				cmd = m.TextArea.Focus()
				cmds = append(cmds, cmd)
			}
		}

	// We handle errors just like any other message
	case errMsg:
		m.Err = msg
		return m, nil
	}

	m.TextArea, cmd = m.TextArea.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if m.Hidden {
		return ""
	}
	return fmt.Sprintf(
		"%s\n\n%s\n\n%s\n\n",
		m.Prompt,
		m.TextArea.View(),
		styles.Subtle("Submit (ctrl+s), Cancel (ctrl+c)"),
	)
}

func Run(i *Input) (string, error) {
	program := tea.NewProgram(initialModel(i))

	m, err := program.Run()
	if err != nil {
		return "", err
	}

	if m, ok := m.(Model); ok {
		if m.Cancelled {
			return "", fmt.Errorf("cancelled")
		}
		return m.TextArea.Value(), nil
	}

	return "", err
}
