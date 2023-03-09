package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Stage struct {
	Name       string
	Action     func() error
	Error      error
	IsComplete bool
	Reset      func() error
}

var stageIndex = 0

var stages = []Stage{
	{
		Name: "One",
		Action: func() error {
			time.Sleep(3 * time.Second)
			return errors.New("This one errored")
		},
		IsComplete: false,
	},
	{
		Name: "Two",
		Action: func() error {
			time.Sleep(3 * time.Second)
			return nil
		},
		IsComplete: false,
	},
}

type model struct {
	status  int
	err     error
	spinner spinner.Model
}

type startDeployMsg struct{}

func startDeployCmd() tea.Msg {
	return startDeployMsg{}
}

func runStage() tea.Msg {
	currentStage := stages[stageIndex]
	// Run the current stage, and record its result status
	currentStage.Error = currentStage.Action()
	return stageCompleteMsg{}
}

type stageCompleteMsg struct{}

type errMsg struct{ err error }

// For messages that contain errors it's often handy to also implement the
// error interface on the message.
func (e errMsg) Error() string { return e.err.Error() }

func initialModel() model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	return model{
		spinner: s,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, startDeployCmd)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case stageCompleteMsg:
		// Mark the current stage as complete and move to the next stage
		stages[stageIndex].IsComplete = true

		if stageIndex+1 >= len(stages) {
			return m, tea.Quit
		}

		stageIndex++
		return m, runStage

	case errMsg:
		m.err = msg
		return m, tea.Quit

	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}

	case startDeployMsg:
		return m, runStage
	}

	var spinnerCmd tea.Cmd
	m.spinner, spinnerCmd = m.spinner.Update(msg)
	return m, spinnerCmd
}

func renderCheckbox(s Stage) string {
	sb := strings.Builder{}
	if s.Error != nil {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(" ❌ "))
	} else if s.IsComplete {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("170")).Render(" ✅ "))
	} else {
		sb.WriteString(" 🔲 ")
	}
	return sb.String()
}

func renderWorkingStatus(m model, s Stage) string {
	sb := strings.Builder{}
	if !s.IsComplete {
		sb.WriteString(m.spinner.View())
	} else {
		sb.WriteString(" ")
	}
	sb.WriteString(" ")
	sb.WriteString(s.Name)
	return sb.String()
}

func (m model) View() string {
	sb := strings.Builder{}

	sb.WriteString(fmt.Sprintf("Current stage: %s\n", stages[stageIndex].Name))

	for _, stage := range stages {
		sb.WriteString(renderCheckbox(stage) + " " + renderWorkingStatus(m, stage) + "\n")
	}
	return sb.String()
}

func main() {
	if _, err := tea.NewProgram(initialModel()).Run(); err != nil {
		fmt.Printf("Uh oh, there was an error: %v\n", err)
		os.Exit(1)
	}
}
