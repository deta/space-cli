package spinner

import (
	"fmt"

	"os"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/deta/pc-cli/pkg/components/styles"
)

type RequestResponse struct {
	Response interface{}
	Err      error
}

type Stop struct {
	FinishMsg       string
	RequestResponse RequestResponse
}

type errMsg error

type Model struct {
	Spinner         spinner.Model
	LoadingMsg      string
	FinishMsg       string
	RequestResponse *RequestResponse
	Request         func() tea.Msg
	Quitting        bool
	Err             error
}

type Input struct {
	LoadingMsg string
	Request    func() tea.Msg
}

func initialModel(i *Input) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = styles.PinkStyle
	return Model{
		Spinner:    s,
		LoadingMsg: i.LoadingMsg,
		Request:    i.Request,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.Spinner.Tick, m.Request)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			os.Exit(1)
			return m, tea.Quit
		default:
			return m, nil
		}
	case Stop:
		m.RequestResponse = &msg.RequestResponse
		m.FinishMsg = msg.FinishMsg
		m.Quitting = true
		return m, tea.Quit
	case errMsg:
		m.Err = msg
		return m, nil

	default:
		var cmd tea.Cmd
		m.Spinner, cmd = m.Spinner.Update(msg)
		return m, cmd
	}

}

func (m Model) View() string {
	if m.Err != nil {
		return styles.Error(m.Err.Error())
	}

	str := fmt.Sprintf("%s %s\n", m.Spinner.View(), m.LoadingMsg)
	if m.FinishMsg != "" && m.RequestResponse.Err == nil {
		str = fmt.Sprintf("%s\n", m.FinishMsg)
	}
	return str
}

func Run(i *Input) (interface{}, error) {
	program := tea.NewProgram(initialModel(i))

	m, err := program.StartReturningModel()
	if err != nil {
		return nil, err
	}

	if m, ok := m.(Model); ok {
		return m.RequestResponse.Response, m.RequestResponse.Err
	}

	return nil, err
}
