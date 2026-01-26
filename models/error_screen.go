package models

import (
	"netpala/common"
	"netpala/config"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ModelErrorType struct {
	Description string
	Colors      config.Colors
}

func ModelError(description string, colors config.Colors) ModelErrorType {
	return ModelErrorType{
		Description: description,
		Colors:      colors,
	}
}

func (m ModelErrorType) Init() tea.Cmd {
	return nil
}

func (m ModelErrorType) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter, tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m ModelErrorType) View() string {
	size := common.WindowDimensions()
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.Colors.Primary)).
		AlignHorizontal(lipgloss.Center).
		AlignVertical(lipgloss.Center).
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(m.Colors.Error)).
		Foreground(lipgloss.Color(m.Colors.ErrorText)).
		Width(size.Width-2).
		Height(size.Height - 2)

	return style.Render(m.Description)
}