package models

import (
	"netpala/common"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ModelErrorType struct {
	err 	error
}

func ModelError( err error ) ModelErrorType {
	return ModelErrorType{ err }
}

func (m ModelErrorType) Init() tea.Cmd {
	return nil
}

func (m ModelErrorType) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m ModelErrorType) View() string {
	size := common.WindowDimensions()

	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#a7abca")).
		AlignHorizontal(lipgloss.Center).
		AlignVertical(lipgloss.Center).
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#ff0000")).
		Foreground(lipgloss.Color("#aa0000")).
		Width(size.Width-2).
		Height(size.Height-4).
		Padding(2, 4)

	return style.Render(m.err.Error())
}