package models

import (
	"netpala/common"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Confirmation struct {
	Message string
	Value 	bool
}

func ModelConfirmation() Confirmation {
	return Confirmation{
		Value: false,
	}
}
func (m Confirmation) Init() tea.Cmd {
	return nil
}

func (m Confirmation) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Handle global key presses for focus switching and quitting first.
	switch key := msg.(type) {
	case tea.KeyMsg:
		switch key.String() {
		case "esc", "ctrl+c", "enter":
			return m, func() tea.Msg { return common.SubmitConfirmationMsg{ Value: m.Value } }
		case "tab", "right":
			m.Value = true
		case "shift+tab", "left":
			m.Value = false
		}
	}

	return m, cmd
}

func (m Confirmation) View() string {
	containerStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#a7abca")).
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

	if m.Value {
		confirmButton = activeBorderStyle.Render("Confirm")
		cancelButton = inactiveBorderStyle.Render("Cancel")
	}

	return containerStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			m.Message,
			lipgloss.JoinHorizontal(lipgloss.Center, 
				cancelButton, confirmButton,
			),
	),
	)
}