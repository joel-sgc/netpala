package models

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mritd/bubbles/selector"
)

type WpaEapForm struct {
	eapMethod   selector.Model
	phase2Auth selector.Model
	identity     textinput.Model
	password     textinput.Model
	caCert      textinput.Model
	focused      int
	eapSelected bool
	phase2Selected bool
}

type EAPMethod struct {
	Type string
}

func ModelWpaEapForm() WpaEapForm {
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

	caCert := textinput.New()
	caCert.Placeholder = "Path to CA certificate (e.g. /etc/ssl/certs/ca.pem)"
	caCert.Prompt = ""
	caCert.Width = 32
	caCert.CharLimit = 512

	return WpaEapForm{
		eapMethod: selector.Model{
			Data: []any{
				EAPMethod{Type: "PEAP"},
				EAPMethod{Type: "TTLS"},
				EAPMethod{Type: "TLS"},
				EAPMethod{Type: "PWD"},
			},
			PerPage:        4,
			FinishedFunc: completedFunc([]string{
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
		phase2Auth: selector.Model{
			Data: []any{
				EAPMethod{Type: "MSCHAPV2"},
				EAPMethod{Type: "PAP"},
				EAPMethod{Type: "CHAP"},
				EAPMethod{Type: "MSCHAP"},
				EAPMethod{Type: "NONE"},
			},
			PerPage:        5,
			FinishedFunc: completedFunc([]string{
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
		caCert:  caCert,
		focused:  0,
		eapSelected: false,
		phase2Selected: false,
	}
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
				m.eapSelected = true
			case 1:
				m.phase2Selected = true
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
			m.caCert.Blur()

			switch m.focused {
			case 0:
				m.eapMethod.SelectedFunc = selectedFunc
				m.phase2Auth.SelectedFunc = unselectedFunc
			case 1:
				m.eapMethod.SelectedFunc = unselectedFunc
				m.phase2Auth.SelectedFunc = selectedFunc
			case 2:
				m.identity.Focus()
				m.eapMethod.SelectedFunc = unselectedFunc
				m.phase2Auth.SelectedFunc = unselectedFunc
			case 3:
				m.password.Focus()
			case 4:
				m.caCert.Focus()
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
				ti := m.caCert
				ti.SetCursor(len(ti.Value()))
				m.caCert = ti
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
		sm, cmd = m.eapMethod.Update(msg)
		m.eapMethod = *sm
		cmds = append(cmds, cmd)

		sm, cmd = m.phase2Auth.Update(msg)
		m.phase2Auth = *sm
		cmds = append(cmds, cmd)
	}

	// 2. Pass *all* messages to text inputs; they handle focus internally.
	m.identity, cmd = m.identity.Update(msg)
	cmds = append(cmds, cmd)
	m.password, cmd = m.password.Update(msg)
	cmds = append(cmds, cmd)
	m.caCert, cmd = m.caCert.Update(msg)
	cmds = append(cmds, cmd)

	// 3. Only pass key-press messages to the *focused* selector.
	if _, ok := msg.(tea.KeyMsg); ok {
		switch m.focused {
		case 0:
			sm, cmd = m.eapMethod.Update(msg)
			m.eapMethod = *sm
			cmds = append(cmds, cmd)

			if msg.(tea.KeyMsg).String() == "enter" {
				m.focused++
				m.eapMethod.SelectedFunc = unselectedFunc
				m.phase2Auth.SelectedFunc = selectedFunc
			}
		case 1:
			sm, cmd = m.phase2Auth.Update(msg)
			m.phase2Auth = *sm
			cmds = append(cmds, cmd)

			if msg.(tea.KeyMsg).String() == "enter" {
				m.eapMethod.SelectedFunc = unselectedFunc
				m.phase2Auth.SelectedFunc = unselectedFunc
				m.identity.Focus()
				m.focused++
			}
		}
	}
	// --- END NEW LOGIC ---

	return m, tea.Batch(cmds...)
}

func (m WpaEapForm) View() string {
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

	caCertLabel := inactiveLabelStyle.Render("\nCA Certificate:")
	caCertBox := inactiveBorderStyle.Render(m.caCert.View())

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
		caCertLabel = activeLabelStyle.Render("\nCA Certificate:")
		caCertBox = activeBorderStyle.Render(m.caCert.View())
	}
	// --- END NEW LOGIC ---
	eapStr := strings.TrimSuffix(strings.Replace(m.eapMethod.View(), "\n", "", 2), "\n")
	phase2Str := strings.TrimSuffix(strings.Replace(m.phase2Auth.View(), "\n", "", 2), "\n")
	if m.eapSelected {
		eapStr = m.eapMethod.View()
	}
	if m.phase2Selected {
		phase2Str = m.phase2Auth.View()
	}

	// We perform the string alterations below the remove the spacing reserved for the header and footer of the selector
	content := lipgloss.JoinVertical(lipgloss.Left,
		eapMethodLabel,
		eapStr,

		phase2Label,
		phase2Str,

		identityLabel,
		identityBox,

		passwordLabel,
		passwordBox,

		caCertLabel,
		caCertBox,
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

func emptyFunc(m selector.Model, obj any, gdIndex int) string {
	return ""
}

func completedFunc(options []string) func(selected any) string {
	return func(selected any) string {
		str := ""

		for i, option := range options {
			if option == selected.(EAPMethod).Type {
				str += lipgloss.NewStyle().Foreground(lipgloss.Color("#cda162")).Render(fmt.Sprintf("%s  %d. %s", "»", i+1, option)) + "\n"
			} else {
				str += fmt.Sprintf("  %s\n", unselectedFunc(selector.Model{}, EAPMethod{Type: option}, i))
			}
		}

		return str
	}
}