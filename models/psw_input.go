package models

import (
	"fmt"
	"netpala/common"
	"netpala/config"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type PasswordInput struct {
	Network      common.ScannedNetwork
	Password     textinput.Model
	ConfirmValue bool
	showPassword bool
	Colors       config.Colors
}

func ModelPasswordInput(colors config.Colors) PasswordInput {
	Input := textinput.New()
	Input.Placeholder = "Enter Password..."
	Input.Prompt = ""
	Input.Width = 31
	Input.CharLimit = 64
	Input.EchoMode = textinput.EchoPassword
	Input.EchoCharacter = '*'

	return PasswordInput{
		Password:     Input,
		ConfirmValue: false,
		Colors:       colors,
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
		case "ctrl+p":
			m.showPassword = !m.showPassword
			if m.showPassword {
				m.Password.EchoMode = textinput.EchoNormal
			} else {
				m.Password.EchoMode = textinput.EchoPassword
			}
			return m, nil
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
		BorderForeground(lipgloss.Color(m.Colors.Active)).
		Foreground(lipgloss.Color(m.Colors.Primary)).
		Align(lipgloss.Center).
		Padding(0, 1).
		Width(50)

	inactiveBorderStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(m.Colors.Inactive)).
		Align(lipgloss.Center).
		Padding(0, 3).
		Width(18)

	activeBorderStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(m.Colors.ActiveText)).
		Align(lipgloss.Center).
		Padding(0, 3).
		Width(18)

	confirmButton := inactiveBorderStyle.Render("Confirm")
	cancelButton := activeBorderStyle.Render("Cancel")

	if m.ConfirmValue {
		confirmButton = activeBorderStyle.Render("Confirm")
		cancelButton = inactiveBorderStyle.Render("Cancel")
	}

	hint := "[ctrl+p: show]"
	if m.showPassword {
		hint = "[ctrl+p: hide]"
	}
	hintLabel := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.Colors.ActiveText)).
		Width(40).
		Align(lipgloss.Center).
		Render(hint)

	// Network info header
	ssid := m.Network.SSID
	if lipgloss.Width(ssid) > 40 {
		ssid = ssid[:37] + "..."
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(m.Colors.ActiveText)).
		Width(44).
		Align(lipgloss.Center)

	dividerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.Colors.Inactive)).
		Width(44)

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.Colors.Active)).
		Bold(true)

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.Colors.Primary))

	macRow := lipgloss.JoinHorizontal(lipgloss.Left,
		labelStyle.Width(10).Render("MAC"),
		valueStyle.Render(m.Network.BSSID),
	)

	securityRow := lipgloss.JoinHorizontal(lipgloss.Left,
		labelStyle.Width(10).Render("Security"),
		valueStyle.Width(34).Render(m.Network.Security),
	)

	signalRow := lipgloss.JoinHorizontal(lipgloss.Left,
		labelStyle.Width(10).Render("Signal"),
		valueStyle.Render(fmt.Sprintf("%d%%", m.Network.Signal)),
	)

	return containerStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			titleStyle.Render(ssid),
			dividerStyle.Render(strings.Repeat("─", 44)),
			macRow,
			securityRow,
			signalRow,
			"",
			hintLabel,
			activeBorderStyle.Width(38).BorderForeground(lipgloss.Color(m.Colors.Active)).Render(m.Password.View()),
			lipgloss.JoinHorizontal(lipgloss.Center,
				cancelButton, confirmButton,
			),
		),
	)
}