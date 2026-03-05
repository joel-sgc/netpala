package models

import (
	"fmt"
	"strings"

	"netpala/common"
	"netpala/config"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// EditForm is the TUI model for editing a saved WPA-PSK / open connection.
// Focus states:
//
//	0 = SSID textinput
//	1 = Password textinput  (skipped for non-PSK networks)
//	2 = AutoConnect toggle
//	3 = Hidden toggle
//	4 = Cancel button
//	5 = Save button
type EditForm struct {
	Network      common.KnownNetwork
	SSID         textinput.Model
	Password     textinput.Model
	AutoConnect  bool
	Hidden       bool
	showPassword bool
	focused      int
	Colors       config.Colors
}

func ModelEditForm(network common.KnownNetwork, colors config.Colors) EditForm {
	ssidInput := textinput.New()
	ssidInput.Placeholder = "Network name"
	ssidInput.Prompt = ""
	ssidInput.Width = 32
	ssidInput.CharLimit = 64
	ssidInput.SetValue(network.SSID)
	ssidInput.Focus()

	passwordInput := textinput.New()
	passwordInput.Placeholder = "Leave empty to keep current"
	passwordInput.Prompt = ""
	passwordInput.Width = 32
	passwordInput.CharLimit = 64
	passwordInput.EchoMode = textinput.EchoPassword
	passwordInput.EchoCharacter = '*'

	return EditForm{
		Network:     network,
		SSID:        ssidInput,
		Password:    passwordInput,
		AutoConnect: network.AutoConnect,
		Hidden:      network.Hidden,
		focused:     0,
		Colors:      colors,
	}
}

func (m EditForm) Init() tea.Cmd {
	return textinput.Blink
}

// isPasswordNetwork returns true for security types that use a PSK/passphrase.
func (m EditForm) isPasswordNetwork() bool {
	sec := m.Network.Security
	return sec == "wpa2-psk" || sec == "wpa3-sae" || sec == "encrypted"
}

// advanceFocus moves focus forward or backward, skipping the password field
// for non-PSK networks.
func (m EditForm) advanceFocus(backward bool) int {
	const total = 6
	var next int
	if backward {
		next = (m.focused + total - 1) % total
		if !m.isPasswordNetwork() && next == 1 {
			next = 0
		}
	} else {
		next = (m.focused + 1) % total
		if !m.isPasswordNetwork() && next == 1 {
			next = 2
		}
	}
	return next
}

func (m EditForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch key := msg.(type) {
	case tea.KeyMsg:
		switch key.String() {
		case "esc", "ctrl+c":
			return m, func() tea.Msg { return common.ExitFormMsg{} }

		case "tab", "shift+tab":
			m.SSID.Blur()
			m.Password.Blur()
			m.focused = m.advanceFocus(key.String() == "shift+tab")
			switch m.focused {
			case 0:
				m.SSID.Focus()
			case 1:
				m.Password.Focus()
			}
			return m, nil

		case "left", "right":
			if m.focused == 4 {
				m.focused = 5
			} else if m.focused == 5 {
				m.focused = 4
			}
			return m, nil

		case "ctrl+p":
			if m.focused == 1 {
				m.showPassword = !m.showPassword
				if m.showPassword {
					m.Password.EchoMode = textinput.EchoNormal
				} else {
					m.Password.EchoMode = textinput.EchoPassword
				}
			}
			return m, nil

		case " ":
			switch m.focused {
			case 2:
				m.AutoConnect = !m.AutoConnect
				return m, nil
			case 3:
				m.Hidden = !m.Hidden
				return m, nil
			}
			// for focus 0 / 1: fall through so textinput receives the space

		case "enter":
			switch m.focused {
			case 2:
				m.AutoConnect = !m.AutoConnect
				return m, nil
			case 3:
				m.Hidden = !m.Hidden
				return m, nil
			case 4: // Cancel
				return m, func() tea.Msg { return common.ExitFormMsg{} }
			case 5: // Save
				cfg := map[string]string{
					"ssid":        m.SSID.Value(),
					"password":    m.Password.Value(),
					"autoconnect": fmt.Sprintf("%t", m.AutoConnect),
					"hidden":      fmt.Sprintf("%t", m.Hidden),
				}
				connPath := m.Network.Path
				return m, func() tea.Msg {
					return common.SubmitEditFormMsg{
						ConnectionPath: connPath,
						Config:         cfg,
					}
				}
			}
			// for focus 0 / 1: fall through so textinput receives enter (no-op)
		}
	}

	m.SSID, cmd = m.SSID.Update(msg)
	cmds = append(cmds, cmd)
	m.Password, cmd = m.Password.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m EditForm) View() string {
	inactiveBorderStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(m.Colors.Inactive)).
		Padding(0, 1)

	activeBorderStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(m.Colors.Primary)).
		Padding(0, 1)

	inactiveLabelStyle := lipgloss.NewStyle().
		Bold(false).
		Foreground(lipgloss.Color(m.Colors.Primary))

	activeLabelStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(m.Colors.ActiveText))

	formStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(m.Colors.Active)).
		Padding(0, 1)

	// Render-only copies at full width (same pattern as wpa-eap.go).
	const inputWidth = 54
	ssidInput := m.SSID
	ssidInput.Width = inputWidth
	passwordInput := m.Password
	passwordInput.Width = inputWidth

	// --- compact network info header ---
	ssid := m.Network.SSID
	if lipgloss.Width(ssid) > 44 {
		ssid = ssid[:41] + "..."
	}
	hdrTitleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(m.Colors.ActiveText)).
		Width(56).
		Align(lipgloss.Center)
	hdrLabelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.Colors.Inactive)).
		Bold(true)
	hdrValueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.Colors.Primary))
	headerInfoRow := lipgloss.JoinHorizontal(lipgloss.Left,
		hdrLabelStyle.Width(5).Render("MAC"),
		hdrValueStyle.Width(20).Render(m.Network.BSSID),
		hdrLabelStyle.Width(10).Render("Security"),
		hdrValueStyle.Width(10).Render(m.Network.Security),
		hdrLabelStyle.Width(7).Render("Signal"),
		hdrValueStyle.Width(4).Render(fmt.Sprintf("%d%%", m.Network.Signal)),
	)
	headerDivider := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.Colors.Inactive)).
		Render(strings.Repeat("─", 56))

	// --- SSID input ---
	ssidLabel := inactiveLabelStyle.Render("SSID:")
	ssidBox := inactiveBorderStyle.Width(inputWidth).Render(ssidInput.View())
	if m.focused == 0 {
		ssidLabel = activeLabelStyle.Render("SSID:")
		ssidBox = activeBorderStyle.Width(inputWidth).Render(ssidInput.View())
	}

	// --- Password input (PSK networks only) ---
	var passwordRows []string
	if m.isPasswordNetwork() {
		hint := " [ctrl+p: show]"
		if m.showPassword {
			hint = " [ctrl+p: hide]"
		}
		pwLabel := inactiveLabelStyle.Render("\nPassword:")
		pwBox := inactiveBorderStyle.Width(inputWidth).Render(passwordInput.View())
		if m.focused == 1 {
			pwLabel = activeLabelStyle.Render("\nPassword:" + hint)
			pwBox = activeBorderStyle.Width(inputWidth).Render(passwordInput.View())
		}
		passwordRows = []string{pwLabel, pwBox}
	}

	// --- Toggle boxes (side by side, 26+26 = 52 content + borders = 60 visual,
	//     but we use Width(24) + border(2) + padding(2) = 28 each → 56 total) ---
	checkmark := func(on bool) string {
		if on {
			return "✓ Enabled "
		}
		return "✗ Disabled"
	}
	autoConnectBox := inactiveBorderStyle.Width(26).Render("AutoConnect: " + checkmark(m.AutoConnect))
	hiddenBox := inactiveBorderStyle.Width(26).Render("Hidden:      " + checkmark(m.Hidden))
	if m.focused == 2 {
		autoConnectBox = activeBorderStyle.
			Width(26).
			BorderForeground(lipgloss.Color(m.Colors.Primary)).
			Bold(true).
			Render("AutoConnect: " + checkmark(m.AutoConnect))
	}
	if m.focused == 3 {
		hiddenBox = activeBorderStyle.
			Width(26).
			BorderForeground(lipgloss.Color(m.Colors.Primary)).
			Bold(true).
			Render("Hidden:      " + checkmark(m.Hidden))
	}
	toggleRow := lipgloss.JoinHorizontal(lipgloss.Top, autoConnectBox, hiddenBox)

	// --- Cancel / Save buttons ---
	cancelButton := inactiveBorderStyle.Width(26).Align(lipgloss.Center).Render("Cancel")
	saveButton := inactiveBorderStyle.Width(26).Align(lipgloss.Center).Render("Save")
	if m.focused == 4 {
		cancelButton = activeBorderStyle.
			Width(26).
			Bold(true).
			Align(lipgloss.Center).
			BorderForeground(lipgloss.Color(m.Colors.ActiveText)).
			Render("Cancel")
	}
	if m.focused == 5 {
		saveButton = activeBorderStyle.
			Width(26).
			Bold(true).
			Align(lipgloss.Center).
			BorderForeground(lipgloss.Color(m.Colors.ActiveText)).
			Render("Save")
	}
	buttonRow := lipgloss.JoinHorizontal(lipgloss.Top, cancelButton, saveButton)

	// --- Assemble content ---
	rows := []string{
		hdrTitleStyle.Render(ssid),
		headerInfoRow,
		headerDivider,
		"",
		ssidLabel,
		ssidBox,
	}
	rows = append(rows, passwordRows...)
	rows = append(rows, "", toggleRow, "", buttonRow)

	content := lipgloss.JoinVertical(lipgloss.Left, rows...)
	return formStyle.Render(content)
}
