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

// Stage is a single step in a deployment process. Only one stage can be running at one time,
// And the entire process exits if any stage fails along the way

// The Action is the function that is run to complete the stage's work
// IsComplete
type Stage struct {
	Name           string
	Action         func() error
	Error          error
	IsComplete     bool
	IsCompleteFunc func() bool
	Reset          func() error
}

var stageIndex = 0

var stages = []Stage{
	{
		Name: "One",
		Action: func() error {
			time.Sleep(3 * time.Second)
			return nil
		},
		IsCompleteFunc: func() bool { return false },
		IsComplete:     false,
	},
	{
		Name: "Two",
		Action: func() error {
			time.Sleep(3 * time.Second)
			return errors.New("This one errored")
		},
		IsCompleteFunc: func() bool { return false },
		IsComplete:     false,
	},
	{
		Name: "Three",
		Action: func() error {
			time.Sleep(3 * time.Second)
			return nil
		},
		IsCompleteFunc: func() bool { return false },
		IsComplete:     false,
	},
}

type model struct {
	status  int
	Error   error
	spinner spinner.Model
}

type startDeployMsg struct{}

func startDeployCmd() tea.Msg {
	return startDeployMsg{}
}

func runStage() tea.Msg {
	if !stages[stageIndex].IsCompleteFunc() {
		// Run the current stage, and record its result status
		stages[stageIndex].Error = stages[stageIndex].Action()
	}
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
		// If we have an error, then set the error so that the views can properly update
		if stages[stageIndex].Error != nil {
			m.Error = stages[stageIndex].Error
			writeCommandLogFile()
			return m, tea.Quit
		}
		// Otherwise, mark the current stage as complete and move to the next stage
		stages[stageIndex].IsComplete = true
		// If we've reached the end of the defined stages, we're done
		if stageIndex+1 >= len(stages) {
			return m, tea.Quit
		}
		stageIndex++
		return m, runStage

	case errMsg:
		m.Error = msg
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

// commandLog is rendered when the deployment encounters an error. It retains a log of all the "commands" that were run in the course of deploying the example
// "commands" are intentionally in air-quotes here because this also includes things like checking for the existence of environment variables, and is not yet
// implemented in a truly re-windable cross-platform way, but it's a start, and it's better than asking someone over an email what failed
var commandLog = []string{}

func logCommand(s string) {
	commandLog = append(commandLog, s)
}

func writeCommandLogFile() {
	//Write the entire command log to a file on the filesystem so that the user has the option of sending it to Gruntwork for debugging purposes
	// We currently write the file to ./gruntwork-examples-debug.log in the same directory as the executable was run in

	// Create the file
	f, err := os.Create("bubbletea-debug.log")
	if err != nil {
		fmt.Println(err)
		return
	}
	// Write to the file, first writing the UTC timestamp as the first line, then looping through the command log to write each command on a new line
	f.WriteString("Ran at: " + time.Now().UTC().String() + "\n")
	f.WriteString("******************************************************************************\n")
	f.WriteString("Human legible log of steps taken and commands run up to the point of failure:\n")
	f.WriteString("******************************************************************************\n")
	for _, cmd := range commandLog {
		f.WriteString(cmd + "\n")
	}
	f.WriteString("^ The above command is likely the one that caused the error!\n")
	f.WriteString("\n\n")
	f.WriteString("******************************************************************************\n")
	f.WriteString("Complete log of the error that halted the deployment:\n")
	f.WriteString("******************************************************************************\n")
	f.WriteString("\n\n")
	f.WriteString(stages[stageIndex].Error.Error() + "\n")
}

func main() {
	if _, err := tea.NewProgram(initialModel()).Run(); err != nil {
		fmt.Printf("Uh oh, there was an error: %v\n", err)
		os.Exit(1)
	}
}
