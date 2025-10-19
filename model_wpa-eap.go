package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mritd/bubbles/selector"
)

type wpa_eap_form struct {
	eap_method   selector.Model
	phase_2_auth selector.Model
	identity     textinput.Model
	password     textinput.Model
	ca_cert      textinput.Model
	focused      int
	eap_selected bool
	phase_2_selected bool
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
	ca_cert.CharLimit = 512

	return wpa_eap_form{
		eap_method: selector.Model{
			Data: []any{
				EAPMethod{Type: "PEAP"},
				EAPMethod{Type: "TTLS"},
				EAPMethod{Type: "TLS"},
				EAPMethod{Type: "PWD"},
			},
			PerPage:        4,
			FinishedFunc: completedFuncfunc([]string{
				"PEAP",
				"TTLS",
				"TLS",
				"PWD",
			}),
			SelectedFunc:   selectedFunc,
			UnSelectedFunc: unselectedFunc,
			HeaderFunc:     emptyFunc,
			FooterFunc:     emptyFunc,
		},
		phase_2_auth: selector.Model{
			Data: []any{
				EAPMethod{Type: "MSCHAPV2"},
				EAPMethod{Type: "PAP"},
				EAPMethod{Type: "CHAP"},
				EAPMethod{Type: "MSCHAP"},
				EAPMethod{Type: "NONE"},
			},
			PerPage:        5,
			FinishedFunc: completedFuncfunc([]string{
				"MSCHAPV2",
				"PAP",
				"CHAP",
				"MSCHAP",
				"NONE",
			}),
			SelectedFunc:   unselectedFunc,
			UnSelectedFunc: unselectedFunc,
			HeaderFunc:     emptyFunc,
			FooterFunc:     emptyFunc,
		},
		identity: identity,
		password: password,
		ca_cert:  ca_cert,
		focused:  0,
		eap_selected: false,
		phase_2_selected: false,
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
		case "enter":
			switch m.focused {
			case 0:
				m.eap_selected = true
			case 1:
				m.phase_2_selected = true
			}
		// --- focus switching ---
		case "tab", "shift+tab":
			if key.String() == "shift+tab" {
				m.focused = (m.focused + 4) % 5
			} else {
				m.focused = (m.focused + 1) % 5
			}

			// Update focus state for text inputs
			m.identity.Blur()
			m.password.Blur()
			m.ca_cert.Blur()

			switch m.focused {
			case 0:
				m.eap_method.SelectedFunc = selectedFunc
				m.phase_2_auth.SelectedFunc = unselectedFunc
			case 1:
				m.eap_method.SelectedFunc = unselectedFunc
				m.phase_2_auth.SelectedFunc = selectedFunc
			case 2:
				m.identity.Focus()
				m.eap_method.SelectedFunc = unselectedFunc
				m.phase_2_auth.SelectedFunc = unselectedFunc
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
		switch m.focused {
		case 0:
			sm, cmd = m.eap_method.Update(msg)
			m.eap_method = *sm
			cmds = append(cmds, cmd)

		if (msg.(tea.KeyMsg).String() == "enter") {		
			m.focused++
			m.eap_method.SelectedFunc = unselectedFunc
			m.phase_2_auth.SelectedFunc = selectedFunc
		}
		case 1:
			sm, cmd = m.phase_2_auth.Update(msg)
			m.phase_2_auth = *sm
			cmds = append(cmds, cmd)

			if (msg.(tea.KeyMsg).String() == "enter") {
				m.eap_method.SelectedFunc = unselectedFunc
				m.phase_2_auth.SelectedFunc = unselectedFunc
				m.identity.Focus()
				m.focused++
			}
		}
	}
	// --- END NEW LOGIC ---

	return m, tea.Batch(cmds...)
}

func (m wpa_eap_form) View() string {
	inactiveBorderStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#444a66")).
		Padding(0, 1)

	activeBorderStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#a7abca")).
		Padding(0, 1)

	inactiveLabelStyle := lipgloss.NewStyle().
		Bold(false).
		Foreground(lipgloss.Color("#a7abca"))

	activeLabelStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#cda162"))

	formStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#9cca69")).
		Padding(0, 1)

	// Always render all text boxes, just change the style.
	eapMethodLabel := inactiveLabelStyle.Render("EAP Method:")
	phase2Label := inactiveLabelStyle.Render("Phase 2 (inner-auth):")

	identityLabel := inactiveLabelStyle.Render("Identity:")
	identityBox := inactiveBorderStyle.Render(m.identity.View())

	passwordLabel := inactiveLabelStyle.Render("\nPassword:")
	passwordBox := inactiveBorderStyle.Render(m.password.View())

	ca_certLabel := inactiveLabelStyle.Render("\nCA Certificate:")
	ca_certBox := inactiveBorderStyle.Render(m.ca_cert.View())
	
	
	switch m.focused {
	case 0:
		eapMethodLabel = activeLabelStyle.Render("EAP Method:")
	case 1:
		phase2Label = activeLabelStyle.Render("Phase 2 (inner-auth):")
	case 2:
		identityLabel = activeLabelStyle.Render("Identity:")
		identityBox = activeBorderStyle.Render(m.identity.View())
	case 3:
		passwordLabel = activeLabelStyle.Render("\nPassword:")
		passwordBox = activeBorderStyle.Render(m.password.View())
	case 4:
		ca_certLabel = activeLabelStyle.Render("\nCA Certificate:")
		ca_certBox = activeBorderStyle.Render(m.ca_cert.View())
	}
	// --- END NEW LOGIC ---
	eap_str := strings.TrimSuffix(strings.Replace(m.eap_method.View(), "\n", "", 2), "\n")
	phase_2_str := strings.TrimSuffix(strings.Replace(m.phase_2_auth.View(), "\n", "", 2), "\n")
	if (m.eap_selected) {
		eap_str = m.eap_method.View()
	}
	if (m.phase_2_selected) {
		phase_2_str = m.phase_2_auth.View()
	}

	// We perform the string alterations below the remove the spacing reserved for the header and footer of the selector
	content := lipgloss.JoinVertical(lipgloss.Left,
		eapMethodLabel,
		eap_str,
		
		phase2Label,
		phase_2_str,
		
		identityLabel,
		identityBox,
		
		passwordLabel,
		passwordBox,
		
		ca_certLabel,
		ca_certBox,
	)

	return formStyle.Render(content)
}

func selectedFunc(m selector.Model, obj any, gdIndex int) string {
	str := obj.(EAPMethod).Type
	return lipgloss.NewStyle().Bold(false).Background(lipgloss.Color("#a7abca")).Foreground(lipgloss.Color("#444a66")).Render(fmt.Sprintf(" %d. %s", gdIndex+1, str))
}

func unselectedFunc(m selector.Model, obj any, gdIndex int) string {
	str := obj.(EAPMethod).Type
	return lipgloss.NewStyle().Bold(false).Foreground(lipgloss.Color("#a7abca")).Render(fmt.Sprintf(" %d. %s", gdIndex+1, str))
}

func emptyFunc(m selector.Model, obj interface{}, gdIndex int) string {
	return ""
}

func completedFuncfunc(options []string) func(selected any) string {
	return func(selected any) string {
		str := ""

		for i, option := range options {
			if (option == selected.(EAPMethod).Type) {
				str += fmt.Sprintf("%s %s\n", lipgloss.NewStyle().Foreground(lipgloss.Color("#9cca69")).Render("»"), selectedFunc(selector.Model{}, selected, i))
			} else {
				str += fmt.Sprintf("  %s\n", unselectedFunc(selector.Model{}, EAPMethod{ Type: option }, i))
			}
		}
		
		return str
	}
}