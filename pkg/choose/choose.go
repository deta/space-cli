package choose

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	Cursor   int
	Chosen   bool
	Quitting bool
	Prompt   string
	Choices  []string
}

type Input struct {
	Prompt  string
	Choices []string
}

func initialModel(i *Input) Model {
	return Model{
		Cursor:   0,
		Chosen:   false,
		Quitting: false,
		Prompt:   i.Prompt,
		Choices:  i.Choices,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter, tea.KeyEsc:
			m.Chosen = true
			m.Quitting = true
			return m, tea.Quit
		case tea.KeyCtrlC:
			os.Exit(1)
		}
	}

	return updateChoices(msg, m)
}

func (m Model) View() string {

	return choicesView(m)
}

// Update loop for the first view where you're choosing a task.
func updateChoices(msg tea.Msg, m Model) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			m.Cursor += 1
			if m.Cursor >= len(m.Choices) {
				m.Cursor = 0
			}
		case "k", "up":
			m.Cursor -= 1
			if m.Cursor < 0 {
				m.Cursor = len(m.Choices) - 1
			}
		}
	}

	return m, nil
}

// sub-view functions
func choicesView(m Model) string {
	c := m.Cursor

	tpl := fmt.Sprintf("? %s\n", m.Prompt)

	tpl += "%s\n"
	choices := ""
	for i, choice := range m.Choices {
		if i == len(m.Choices)-1 {
			choices += RenderChoice(choice, c == i)
		} else {
			choices += fmt.Sprintf("%s\n", RenderChoice(choice, c == i))
		}
	}

	if m.Quitting && m.Chosen {
		tpl += fmt.Sprintf("\n> selected %s\n\n", m.Choices[m.Cursor])
	}
	return fmt.Sprintf(tpl, choices)
}

func RenderChoice(choice string, chosen bool) string {
	if chosen {
		return fmt.Sprintf("- %s", choice)
	}
	return fmt.Sprintf("  %s", choice)
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
