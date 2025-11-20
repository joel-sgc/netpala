package models

import (
	"netpala/common"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type PasswordInput struct {
	Message      string
	Password     textinput.Model
	ConfirmValue bool
}

func ModelPasswordInput() PasswordInput {
	Input := textinput.New()
	Input.Placeholder = "Enter Password..."
	Input.Prompt = ""
	Input.Width = 31
	Input.CharLimit = 64

	return PasswordInput{
		Password:     Input,
		ConfirmValue: false,
	}
}

func (m PasswordInput) Init() tea.Cmd {
	return textinput.Blink
}

func (m PasswordInput) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Handle global key presses for focus switching and quitting first.
	switch key := msg.(type) {
	case tea.KeyMsg:
		switch key.String() {
		case "tab", "shift+tab", "left", "right":
			m.ConfirmValue = !m.ConfirmValue
		case "esc", "ctrl+c":
			cmds = append(cmds, func() tea.Msg { return common.ExitFormMsg{} })
		case "enter":
			if m.ConfirmValue {
				cmds = append(cmds, func() tea.Msg { return common.SubmitPasswordMsg{Value: m.Password.Value()} })
			} else {
				cmds = append(cmds, func() tea.Msg { return common.ExitFormMsg{} })
			}
		}
	}

	var cmd tea.Cmd
	m.Password, cmd = m.Password.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m PasswordInput) View() string {
	containerStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#9cca69")).
		Foreground(lipgloss.Color("#a7abca")).
		Align(lipgloss.Center).
		Padding(0, 1).
		Width(50)

	inactiveBorderStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#444a66")).
		Align(lipgloss.Center).
		Padding(0, 3).
		Width(18)

	activeBorderStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#cda162")).
		Align(lipgloss.Center).
		Padding(0, 3).
		Width(18)

	confirmButton := inactiveBorderStyle.Render("Confirm")
	cancelButton := activeBorderStyle.Render("Cancel")

	if m.ConfirmValue {
		confirmButton = activeBorderStyle.Render("Confirm")
		cancelButton = inactiveBorderStyle.Render("Cancel")
	}

	return containerStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			m.Message,
			activeBorderStyle.Width(38).BorderForeground(lipgloss.Color("#9cca69")).Render(m.Password.View()),
			lipgloss.JoinHorizontal(lipgloss.Center,
				cancelButton, confirmButton,
			),
		),
	)
}
