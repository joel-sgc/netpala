package main

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mritd/bubbles/common"
	"github.com/mritd/bubbles/selector"
)

type wpa_eap_form struct {
	eap_method 		selector.Model
	phase_2_auth 	selector.Model
	identity 			textinput.Model
	password 			textinput.Model
	ca_cert 			textinput.Model
	focused  			int
}

type EAPMethod struct {
	Type string
}

func WpaEapForm() wpa_eap_form {
	identity := textinput.New()
	identity.Placeholder = "Identity"
	identity.Prompt = ""
	identity.Width = 32
	identity.CharLimit = 256

	password := textinput.New()
	password.Placeholder = "Password"
	password.Prompt = ""
	password.Width = 32
	password.CharLimit = 256
	password.EchoMode = textinput.EchoPassword
	password.EchoCharacter = '*'

	ca_cert := textinput.New()
	ca_cert.Placeholder = "Path to CA certificate (e.g. /etc/ssl/certs/ca.pem)"
	ca_cert.Prompt = ""
	ca_cert.Width = 32
	ca_cert.CharLimit = 256

	return wpa_eap_form{
		eap_method: selector.Model{
			Data: []any{
				EAPMethod{ Type: "PEAP" },
				EAPMethod{ Type: "TTLS" },
				EAPMethod{ Type: "TLS" },
				EAPMethod{ Type: "PWD" },
			},
			PerPage: 4,
			SelectedFunc: selectedFunc,
			UnSelectedFunc: unselectedFunc,
		},
		phase_2_auth: selector.Model{
			Data: []any{
				EAPMethod{ Type: "MSCHAPV2" },
				EAPMethod{ Type: "PAP" },
				EAPMethod{ Type: "CHAP" },
				EAPMethod{ Type: "MSCHAP" },
				EAPMethod{ Type: "NONE" },
			},
			PerPage: 5,
			SelectedFunc: selectedFunc,
			UnSelectedFunc: unselectedFunc,
		},
		identity: identity,
		password: password,
		ca_cert: ca_cert,
		focused:  0,
	}
}
func (m wpa_eap_form) Init() tea.Cmd {
	return textinput.Blink
}

func (m wpa_eap_form) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	// Handle global key presses for focus switching and quitting first.
	switch key := msg.(type) {
	case tea.KeyMsg:
		switch key.String() {
		// --- focus switching ---
		case "tab", "shift+tab":
			if key.String() == "shift+tab" {
				m.focused--
			} else {
				m.focused++
			}
			if m.focused < 0 {
				m.focused = 4
			} else if m.focused > 4 {
				m.focused = 0
			}

			// Update focus state for text inputs
			m.identity.Blur()
			m.password.Blur()
			m.ca_cert.Blur()

			switch m.focused {
			case 2:
				m.identity.Focus()
			case 3:
				m.password.Focus()
			case 4:
				m.ca_cert.Focus()
			}
			// Don't pass the tab key to the component itself
			return m, nil

		// --- select all (Ctrl+A) ---
		case "ctrl+a":
			switch m.focused {
			case 2:
				ti := m.identity
				ti.SetCursor(len(ti.Value()))
				m.identity = ti
			case 3:
				ti := m.password
				ti.SetCursor(len(ti.Value()))
				m.password = ti
			case 4:
				ti := m.ca_cert
				ti.SetCursor(len(ti.Value()))
				m.ca_cert = ti
			}
			return m, nil

		// --- quit ---
		case "ctrl+c", "esc":
			return m, tea.Quit
		}
	}

	// --- THIS IS THE NEW LOGIC ---
	var sm *selector.Model

	// 1. Pass non-key messages (like WindowSizeMsg) to selectors so they can render.
	if _, ok := msg.(tea.KeyMsg); !ok {
		sm, cmd = m.eap_method.Update(msg)
		m.eap_method = *sm
		cmds = append(cmds, cmd)

		sm, cmd = m.phase_2_auth.Update(msg)
		m.phase_2_auth = *sm
		cmds = append(cmds, cmd)
	}

	// 2. Pass *all* messages to text inputs; they handle focus internally.
	m.identity, cmd = m.identity.Update(msg)
	cmds = append(cmds, cmd)
	m.password, cmd = m.password.Update(msg)
	cmds = append(cmds, cmd)
	m.ca_cert, cmd = m.ca_cert.Update(msg)
	cmds = append(cmds, cmd)

	// 3. Only pass key-press messages to the *focused* selector.
	if _, ok := msg.(tea.KeyMsg); ok {
		if m.focused == 0 {
			sm, cmd = m.eap_method.Update(msg)
			m.eap_method = *sm
			cmds = append(cmds, cmd)
		}
		if m.focused == 1 {
			sm, cmd = m.phase_2_auth.Update(msg)
			m.phase_2_auth = *sm
			cmds = append(cmds, cmd)
		}
	}
	// --- END NEW LOGIC ---

	return m, tea.Batch(cmds...)
}

func (m wpa_eap_form) View() string {
	activeBorder := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(0, 1)

	inactiveBorder := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1)

	var formStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder(), true).
		BorderForeground(lipgloss.Color("63")).
		Padding(0, 1)

	// Always render all text boxes, just change the style.
	identityBox := inactiveBorder.Render(m.identity.View())
	passwordBox := inactiveBorder.Render(m.password.View())
	ca_certBox := inactiveBorder.Render(m.ca_cert.View())

	switch m.focused {
	case 2:
		identityBox = activeBorder.Render(m.identity.View())
	case 3:
		passwordBox = activeBorder.Render(m.password.View())
	case 4:
		ca_certBox = activeBorder.Render(m.ca_cert.View())
	}
	// --- END NEW LOGIC ---

	content := lipgloss.JoinVertical(lipgloss.Left,
		"EAP Method:",
		m.eap_method.View(),
		"Phase 2 (inner-auth):",
		m.phase_2_auth.View(),
		"Identity:",
		identityBox,
		"\nPassword:",
		passwordBox,
		"\nCA Certificate:",
		ca_certBox,
	)

	return formStyle.Render(content)
}

func selectedFunc(m selector.Model, obj any, gdIndex int) string {
	t := obj.(EAPMethod)
	return common.FontColor(fmt.Sprintf("[%d] %s", gdIndex+1, t.Type), selector.ColorSelected)
}

func unselectedFunc(m selector.Model, obj any, gdIndex int) string {
	t := obj.(EAPMethod)
	return common.FontColor(fmt.Sprintf(" %d. %s", gdIndex+1, t.Type), selector.ColorUnSelected)
}
