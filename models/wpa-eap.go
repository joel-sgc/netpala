package models

import (
	"fmt"
	"netpala/common"
	"netpala/config"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mritd/bubbles/selector"
)

type WpaEapForm struct {
	EapMethod      	selector.Model
	Phase2Auth     	selector.Model
	Identity       	textinput.Model
	Password       	textinput.Model
	CaCert         	textinput.Model
	focused        	int
	showPassword   	bool

	Network        	common.ScannedNetwork
	SSIDSelected		string
	EapSelected   	bool
	Phase2Selected	bool
	DisableForm   	func()
	Colors         	config.Colors
}

type EAPMethod struct {
	Type string
}

func ModelWpaEapForm(colors config.Colors) WpaEapForm {
	Identity := textinput.New()
	Identity.Placeholder = "Identity"
	Identity.Prompt = ""
	Identity.Width = 32
	Identity.CharLimit = 256

	Password := textinput.New()
	Password.Placeholder = "Password"
	Password.Prompt = ""
	Password.Width = 32
	Password.CharLimit = 256
	Password.EchoMode = textinput.EchoPassword
	Password.EchoCharacter = '*'

	CaCert := textinput.New()
	CaCert.Placeholder = "e.g. /etc/ssl/certs/ca.pem"
	CaCert.Prompt = ""
	CaCert.Width = 32
	CaCert.CharLimit = 512

	m := WpaEapForm{
		Identity:       Identity,
		Password:       Password,
		CaCert:         CaCert,
		focused:        0,
		EapSelected:    false,
		Phase2Selected: false,
		Colors:         colors,
		EapMethod: selector.Model{
			Data: []any{
				EAPMethod{Type: "PEAP"},
				EAPMethod{Type: "TTLS"},
				EAPMethod{Type: "TLS"},
				EAPMethod{Type: "PWD"},
			},
			PerPage: 4,
			HeaderFunc: emptyFunc,
			FooterFunc: emptyFunc,
		},
		Phase2Auth: selector.Model{
			Data: []any{
				EAPMethod{Type: "MSCHAPV2"},
				EAPMethod{Type: "PAP"},
				EAPMethod{Type: "CHAP"},
				EAPMethod{Type: "MSCHAP"},
				EAPMethod{Type: "NONE"},
			},
			PerPage: 5,
			HeaderFunc: emptyFunc,
			FooterFunc: emptyFunc,
		},
	}
	
	// Set the functions that need access to colors
	m.EapMethod.SelectedFunc = makeSelectedFunc(colors)
	m.EapMethod.UnSelectedFunc = makeUnselectedFunc(colors)
	m.EapMethod.FinishedFunc = makeCompletedFunc([]string{
		"PEAP",
		"TTLS",
		"TLS",
		"PWD",
	}, colors)
	
	m.Phase2Auth.SelectedFunc = makeUnselectedFunc(colors)
	m.Phase2Auth.UnSelectedFunc = makeUnselectedFunc(colors)
	m.Phase2Auth.FinishedFunc = makeCompletedFunc([]string{
		"MSCHAPV2",
		"PAP",
		"CHAP",
		"MSCHAP",
		"NONE",
	}, colors)
	
	return m
}

func (m WpaEapForm) Init() tea.Cmd {
	return textinput.Blink
}

func (m WpaEapForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	// Handle global key presses for focus switching and quitting first.
	switch key := msg.(type) {
	case tea.KeyMsg:
		switch key.String() {
		case "enter":
			switch m.focused {
			case 0:
				m.EapSelected = true
			case 1:
				m.Phase2Selected = true
			}
		// --- focus switching ---
		case "tab", "shift+tab":
			if key.String() == "shift+tab" {
				m.focused = (m.focused + 6) % 7
			} else {
				m.focused = (m.focused + 1) % 7
			}

			// Update focus state for text inputs
			m.Identity.Blur()
			m.Password.Blur()
			m.CaCert.Blur()

			switch m.focused {
			case 0:
				m.EapMethod.SelectedFunc = makeSelectedFunc(m.Colors)
				m.Phase2Auth.SelectedFunc = makeUnselectedFunc(m.Colors)
			case 1:
				m.EapMethod.SelectedFunc = makeUnselectedFunc(m.Colors)
				m.Phase2Auth.SelectedFunc = makeSelectedFunc(m.Colors)
			case 2:
				m.Identity.Focus()
				m.EapMethod.SelectedFunc = makeUnselectedFunc(m.Colors)
				m.Phase2Auth.SelectedFunc = makeUnselectedFunc(m.Colors)
			case 3:
				m.Password.Focus()
			case 4:
				m.CaCert.Focus()
			}
			// Don't pass the tab key to the component itself
			return m, nil

		// --- select all (Ctrl+A) ---
		case "ctrl+a":
			switch m.focused {
			case 2:
				ti := m.Identity
				ti.SetCursor(len(ti.Value()))
				m.Identity = ti
			case 3:
				ti := m.Password
				ti.SetCursor(len(ti.Value()))
				m.Password = ti
			case 4:
				ti := m.CaCert
				ti.SetCursor(len(ti.Value()))
				m.CaCert = ti
			}
			return m, nil
		// --- toggle password visibility (Ctrl+P) ---
		case "ctrl+p":
			if m.focused == 3 {
				m.showPassword = !m.showPassword
				if m.showPassword {
					m.Password.EchoMode = textinput.EchoNormal
				} else {
					m.Password.EchoMode = textinput.EchoPassword
				}
			}
			return m, nil
		case "esc", "ctrl+c":
			return m, func() tea.Msg { return common.ExitFormMsg{} }
		case "left", "right":
			switch m.focused {
			case 5:
				m.focused = 6
			case 6:
				m.focused = 5
			}
		}
	}

	var sm *selector.Model

	// 1. Pass non-key messages (like WindowSizeMsg) to selectors so they can render.
	if _, ok := msg.(tea.KeyMsg); !ok {
		sm, cmd = m.EapMethod.Update(msg)
		m.EapMethod = *sm
		cmds = append(cmds, cmd)

		sm, cmd = m.Phase2Auth.Update(msg)
		m.Phase2Auth = *sm
		cmds = append(cmds, cmd)
	}

	// 2. Pass *all* messages to text inputs; they handle focus internally.
	m.Identity, cmd = m.Identity.Update(msg)
	cmds = append(cmds, cmd)
	m.Password, cmd = m.Password.Update(msg)
	cmds = append(cmds, cmd)
	m.CaCert, cmd = m.CaCert.Update(msg)
	cmds = append(cmds, cmd)

	// 3. Only pass key-press messages to the *focused* selector.
	if _, ok := msg.(tea.KeyMsg); ok {
		switch m.focused {
		case 0:
			sm, cmd = m.EapMethod.Update(msg)
			m.EapMethod = *sm
			cmds = append(cmds, cmd)

			if msg.(tea.KeyMsg).String() == "enter" {
				m.focused++
				m.EapMethod.SelectedFunc = makeUnselectedFunc(m.Colors)
				m.Phase2Auth.SelectedFunc = makeSelectedFunc(m.Colors)
			}
		case 1:
			sm, cmd = m.Phase2Auth.Update(msg)
			m.Phase2Auth = *sm
			cmds = append(cmds, cmd)

			if msg.(tea.KeyMsg).String() == "enter" {
				m.EapMethod.SelectedFunc = makeUnselectedFunc(m.Colors)
				m.Phase2Auth.SelectedFunc = makeUnselectedFunc(m.Colors)
				m.Identity.Focus()
				m.focused++
			}
		case 5:
			// Cancel button
			if msg.(tea.KeyMsg).String() == "enter" {
				return m, func() tea.Msg { return common.ExitFormMsg{} }
			}
		case 6:
			// Connect button
			if msg.(tea.KeyMsg).String() == "enter" {
				config := map[string]string{
					"ssid":        m.SSIDSelected,
					"eap":         m.EapMethod.Selected().(EAPMethod).Type,
					"phase2-auth": m.Phase2Auth.Selected().(EAPMethod).Type,
					"identity":    m.Identity.Value(),
					"password":    m.Password.Value(),
					"ca_cert":     m.CaCert.Value(),
				}
				return m, func() tea.Msg {
					return common.SubmitEapFormMsg{Config: config}
				}
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m WpaEapForm) View() string {
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

	// Render-only copies with full-width sizing.
	// inputWidth = content width; visual box = inputWidth + border(2) + padding(2) = 56.
	const inputWidth = 52
	identityInput := m.Identity
	identityInput.Width = inputWidth
	passwordInput := m.Password
	passwordInput.Width = inputWidth
	caCertInput := m.CaCert
	caCertInput.Width = inputWidth

	// Always render all text boxes, just change the style.
	EapMethodLabel := inactiveLabelStyle.Render("EAP Method:")
	phase2Label := inactiveLabelStyle.Render("Phase 2 (inner-auth):")

	IdentityLabel := inactiveLabelStyle.Render("Identity:")
	IdentityBox := inactiveBorderStyle.Render(identityInput.View())

	PasswordLabel := inactiveLabelStyle.Render("\nPassword:")
	PasswordBox := inactiveBorderStyle.Render(passwordInput.View())

	CaCertLabel := inactiveLabelStyle.Render("\nCA Certificate:")
	CaCertBox := inactiveBorderStyle.Render(caCertInput.View())

	cancelButton := inactiveBorderStyle.
		Width(27).
		Align(lipgloss.Center).
		Render("Cancel")
	connectButton := inactiveBorderStyle.
		Width(26).
		Align(lipgloss.Center).
		Render("Connect")

	switch m.focused {
	case 0:
		EapMethodLabel = activeLabelStyle.Render("EAP Method:")
	case 1:
		phase2Label = activeLabelStyle.Render("Phase 2 (inner-auth):")
	case 2:
		IdentityLabel = activeLabelStyle.Render("Identity:")
		IdentityBox = activeBorderStyle.Render(identityInput.View())
	case 3:
		hint := " [ctrl+p: show]"
		if m.showPassword {
			hint = " [ctrl+p: hide]"
		}
		PasswordLabel = activeLabelStyle.Render("\nPassword:" + hint)
		PasswordBox = activeBorderStyle.Render(passwordInput.View())
	case 4:
		CaCertLabel = activeLabelStyle.Render("\nCA Certificate:")
		CaCertBox = activeBorderStyle.Render(caCertInput.View())
	case 5:
		cancelButton = activeBorderStyle.
			Width(27).
			Bold(true).
			Align(lipgloss.Center).
			BorderForeground(lipgloss.Color(m.Colors.ActiveText)).
			Render("Cancel")
	case 6:
		connectButton = activeBorderStyle.
			Width(26).
			Bold(true).
			Align(lipgloss.Center).
			BorderForeground(lipgloss.Color(m.Colors.ActiveText)).
			Render("Connect")
	}
	// --- END NEW LOGIC ---
	eapStr := strings.TrimSuffix(strings.Replace(m.EapMethod.View(), "\n", "", 2), "\n")
	phase2Str := strings.TrimSuffix(strings.Replace(m.Phase2Auth.View(), "\n", "", 2), "\n")
	if m.EapSelected {
		eapStr = m.EapMethod.View()
	}
	if m.Phase2Selected {
		phase2Str = m.Phase2Auth.View()
	}

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

	// --- selectors side by side to save vertical space ---
	eapColumn := lipgloss.NewStyle().Width(26).Render(
		lipgloss.JoinVertical(lipgloss.Left, EapMethodLabel, eapStr),
	)
	phase2Column := lipgloss.NewStyle().Width(30).Render(
		lipgloss.JoinVertical(lipgloss.Left, phase2Label, phase2Str),
	)
	selectorsRow := lipgloss.JoinHorizontal(lipgloss.Top, eapColumn, phase2Column)

	content := lipgloss.JoinVertical(lipgloss.Left,
		hdrTitleStyle.Render(ssid),
		headerInfoRow,
		headerDivider,
		"",
		selectorsRow,
		IdentityLabel,
		IdentityBox,
		PasswordLabel,
		PasswordBox,
		CaCertLabel,
		CaCertBox,
		lipgloss.JoinHorizontal(lipgloss.Top, cancelButton, connectButton),
	)
	return formStyle.Render(content)
}

func makeSelectedFunc(colors config.Colors) func(selector.Model, any, int) string {
	return func(sm selector.Model, obj any, gdIndex int) string {
		str := obj.(EAPMethod).Type
		return lipgloss.NewStyle().Bold(false).Background(lipgloss.Color(colors.SelectionBg)).Foreground(lipgloss.Color(colors.Inactive)).Render(fmt.Sprintf(" %d. %s", gdIndex+1, str))
	}
}

func makeUnselectedFunc(colors config.Colors) func(selector.Model, any, int) string {
	return func(sm selector.Model, obj any, gdIndex int) string {
		str := obj.(EAPMethod).Type
		return lipgloss.NewStyle().Bold(false).Foreground(lipgloss.Color(colors.Primary)).Render(fmt.Sprintf(" %d. %s", gdIndex+1, str))
	}
}

func emptyFunc(m selector.Model, obj any, gdIndex int) string {
	return ""
}

func makeCompletedFunc(options []string, colors config.Colors) func(any) string {
	return func(selected any) string {
		str := ""

		for i, option := range options {
			if option == selected.(EAPMethod).Type {
				str += lipgloss.NewStyle().Foreground(lipgloss.Color(colors.ActiveText)).Render(fmt.Sprintf("%s  %d. %s", "»", i+1, option)) + "\n"
			} else {
				str += fmt.Sprintf("  %s", makeUnselectedFunc(colors)(selector.Model{}, EAPMethod{Type: option}, i)) + "\n"
			}
		}

		return str
	}
}