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
	eap_method   selector.Model
	phase_2_auth selector.Model
	identity     textinput.Model
	password     textinput.Model
	ca_cert      textinput.Model
	focused      int
}

type EAPMethod struct {
	Type string
}

func WpaEapForm() wpa_eap_form {
	identity := textinput.New()
	identity.Placeholder = "Identity"
	identity.Prompt = ""
	identity.Width = 32

	password := textinput.New()
	password.Placeholder = "Password"
	password.Prompt = ""
	password.Width = 32
	password.EchoMode = textinput.EchoPassword
	password.EchoCharacter = '*'

	ca_cert := textinput.New()
	ca_cert.Placeholder = "Certificate path (e.g. /etc/ssl/certs/ca.pem)"
	ca_cert.Prompt = ""
	ca_cert.Width = 32

	return wpa_eap_form{
		eap_method: selector.Model{
			Data: []any{
				EAPMethod{"PEAP"},
				EAPMethod{"TTLS"},
				EAPMethod{"TLS"},
				EAPMethod{"PWD"},
			},
			PerPage:       4,
			SelectedFunc:  selectedFunc,
			UnSelectedFunc: unselectedFunc,
		},
		phase_2_auth: selector.Model{
			Data: []any{
				EAPMethod{"MSCHAPV2"},
				EAPMethod{"PAP"},
				EAPMethod{"CHAP"},
				EAPMethod{"MSCHAP"},
				EAPMethod{"NONE"},
			},
			PerPage:       5,
			SelectedFunc:  selectedFunc,
			UnSelectedFunc: unselectedFunc,
		},
		identity: identity,
		password: password,
		ca_cert:  ca_cert,
	}
}

func (m wpa_eap_form) Init() tea.Cmd {
	return textinput.Blink
}

func (m wpa_eap_form) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch key := msg.(type) {
	case tea.KeyMsg:
		switch key.String() {
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

		case "ctrl+c", "esc":
			return m, tea.Quit
		}
	}

	if m.focused == 0 {
		sm, cmd := m.eap_method.Update(msg)
		m.eap_method = *sm
		cmds = append(cmds, cmd)
	}
	if m.focused == 1 {
		sm, cmd := m.phase_2_auth.Update(msg)
		m.phase_2_auth = *sm
		cmds = append(cmds, cmd)
	}

	m.identity, cmd = m.identity.Update(msg)
	cmds = append(cmds, cmd)
	m.password, cmd = m.password.Update(msg)
	cmds = append(cmds, cmd)
	m.ca_cert, cmd = m.ca_cert.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m wpa_eap_form) View() string {
	active := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("63"))
	inactive := lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("240"))

	var identityBox, passwordBox, certBox string
	switch m.focused {
	case 2:
		identityBox = active.Render(m.identity.View())
		passwordBox = inactive.Render(m.password.View())
		certBox = inactive.Render(m.ca_cert.View())
	case 3:
		identityBox = inactive.Render(m.identity.View())
		passwordBox = active.Render(m.password.View())
		certBox = inactive.Render(m.ca_cert.View())
	case 4:
		identityBox = inactive.Render(m.identity.View())
		passwordBox = inactive.Render(m.password.View())
		certBox = active.Render(m.ca_cert.View())
	default:
		identityBox = inactive.Render(m.identity.View())
		passwordBox = inactive.Render(m.password.View())
		certBox = inactive.Render(m.ca_cert.View())
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		"EAP Method:",
		m.eap_method.View(),
		"Phase 2 (inner-auth):",
		m.phase_2_auth.View(),
		"Identity:",
		identityBox,
		"\nPassword:",
		passwordBox,
		"\nCA Certificate:",
		certBox,
	)
}

func selectedFunc(m selector.Model, obj any, gdIndex int) string {
	t := obj.(EAPMethod)
	return common.FontColor(fmt.Sprintf("[%d] %s", gdIndex+1, t.Type), selector.ColorSelected)
}

func unselectedFunc(m selector.Model, obj any, gdIndex int) string {
	t := obj.(EAPMethod)
	return common.FontColor(fmt.Sprintf(" %d. %s", gdIndex+1, t.Type), selector.ColorUnSelected)
}
