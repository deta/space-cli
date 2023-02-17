package textarea

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/deta/pc-cli/pkg/components/styles"
)

type errMsg error

type Model struct {
	TextArea textarea.Model
	Prompt   string
	exitCode int
	Err      error
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
		case tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyCtrlC:
			m.exitCode = 1
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
	return fmt.Sprintf(
		"%s\n\n%s\n\n%s\n\n",
		m.Prompt,
		m.TextArea.View(),
		styles.Subtle("Newline (enter) Submit (esc)"),
	)
}

func Run(i *Input) (string, error) {
	program := tea.NewProgram(initialModel(i))

	m, err := program.Run()
	if err != nil {
		return "", err
	}

	if m, ok := m.(Model); ok {
		if m.exitCode != 0 {
			os.Exit(m.exitCode)
		}
		return m.TextArea.Value(), nil
	}

	return "", err
}
