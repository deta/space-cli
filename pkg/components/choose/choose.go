package choose

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/deta/space/pkg/components/styles"
)

type Model struct {
	Cursor    int
	Chosen    bool
	Hidden    bool
	Cancelled bool
	Prompt    string
	Choices   []string
}

type Input struct {
	Prompt  string
	Choices []string
}

func initialModel(i *Input) Model {
	return Model{
		Cursor:  0,
		Chosen:  false,
		Prompt:  i.Prompt,
		Choices: i.Choices,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			m.Hidden = true
			return m, tea.Quit
		case tea.KeyCtrlC:
			m.Hidden = true
			m.Cancelled = true
			return m, tea.Quit
		}
	}

	return updateChoices(msg, m)
}

func (m Model) View() string {
	if m.Hidden {
		return ""
	}

	c := m.Cursor

	tpl := fmt.Sprintf("\n%s %s  \n\n", styles.Question, styles.Bold(m.Prompt))

	tpl += "%s\n"
	choices := ""
	for i, choice := range m.Choices {
		if i == len(m.Choices)-1 {
			choices += RenderChoice(choice, c == i)
		} else {
			choices += fmt.Sprintf("%s\n", RenderChoice(choice, c == i))
		}
	}

	return fmt.Sprintf(tpl, choices)
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

func RenderChoice(choice string, chosen bool) string {
	if chosen {
		return fmt.Sprintf("%s %s", styles.SelectTag, choice)
	}
	return fmt.Sprintf("  %s", choice)
}

func Run(i *Input) (*Model, error) {
	program := tea.NewProgram(initialModel(i))

	m, err := program.Run()
	if err != nil {
		return nil, err
	}

	if m, ok := m.(Model); ok {
		if m.Cancelled {
			return nil, fmt.Errorf("cancelled")
		}
		return &m, nil
	}

	return nil, err
}
