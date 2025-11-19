package models

import (
	"fmt"
	"netpala/common"
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
	
	SSIDSelected		string
	EapSelected   	bool
	Phase2Selected	bool
	DisableForm   	func()
}

type EAPMethod struct {
	Type string
}

func ModelWpaEapForm() WpaEapForm {
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

	return WpaEapForm{
		EapMethod: selector.Model{
			Data: []any{
				EAPMethod{Type: "PEAP"},
				EAPMethod{Type: "TTLS"},
				EAPMethod{Type: "TLS"},
				EAPMethod{Type: "PWD"},
			},
			PerPage: 4,
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
		Phase2Auth: selector.Model{
			Data: []any{
				EAPMethod{Type: "MSCHAPV2"},
				EAPMethod{Type: "PAP"},
				EAPMethod{Type: "CHAP"},
				EAPMethod{Type: "MSCHAP"},
				EAPMethod{Type: "NONE"},
			},
			PerPage: 5,
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
		Identity:       Identity,
		Password:       Password,
		CaCert:         CaCert,
		focused:        0,
		EapSelected:    false,
		Phase2Selected: false,
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
				m.EapSelected = true
			case 1:
				m.Phase2Selected = true
			}
		// --- focus switching ---
		case "tab", "shift+tab":
			if key.String() == "shift+tab" {
				m.focused = (m.focused + 5) % 6
			} else {
				m.focused = (m.focused + 1) % 6
			}

			// Update focus state for text inputs
			m.Identity.Blur()
			m.Password.Blur()
			m.CaCert.Blur()

			switch m.focused {
			case 0:
				m.EapMethod.SelectedFunc = selectedFunc
				m.Phase2Auth.SelectedFunc = unselectedFunc
			case 1:
				m.EapMethod.SelectedFunc = unselectedFunc
				m.Phase2Auth.SelectedFunc = selectedFunc
			case 2:
				m.Identity.Focus()
				m.EapMethod.SelectedFunc = unselectedFunc
				m.Phase2Auth.SelectedFunc = unselectedFunc
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
		case "esc", "ctrl+c":
			return m, func() tea.Msg { return common.ExitFormMsg{} }
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
				m.EapMethod.SelectedFunc = unselectedFunc
				m.Phase2Auth.SelectedFunc = selectedFunc
			}
		case 1:
			sm, cmd = m.Phase2Auth.Update(msg)
			m.Phase2Auth = *sm
			cmds = append(cmds, cmd)

			if msg.(tea.KeyMsg).String() == "enter" {
				m.EapMethod.SelectedFunc = unselectedFunc
				m.Phase2Auth.SelectedFunc = unselectedFunc
				m.Identity.Focus()
				m.focused++
			}
		case 5:
			// Submit form
			config := map[string]string{
				"ssid":				 m.SSIDSelected,
				"eap":         m.EapMethod.Selected().(EAPMethod).Type,
				"phase2-auth": m.Phase2Auth.Selected().(EAPMethod).Type,
				"identity":    m.Identity.Value(),
				"password":    m.Password.Value(),
				"ca_cert":     m.CaCert.Value(),
			}
			
			// Send the message with the data back to the parent
			return m, func() tea.Msg {
				return common.SubmitEapFormMsg{Config: config}
			}
		}
	}

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
	EapMethodLabel := inactiveLabelStyle.Render("EAP Method:")
	phase2Label := inactiveLabelStyle.Render("Phase 2 (inner-auth):")

	IdentityLabel := inactiveLabelStyle.Render("Identity:")
	IdentityBox := inactiveBorderStyle.Render(m.Identity.View())

	PasswordLabel := inactiveLabelStyle.Render("\nPassword:")
	PasswordBox := inactiveBorderStyle.Render(m.Password.View())

	CaCertLabel := inactiveLabelStyle.Render("\nCA Certificate:")
	CaCertBox := inactiveBorderStyle.Render(m.CaCert.View())

	submitLabel := inactiveBorderStyle.
		Width(36).
		Align(lipgloss.Center).
		Render("Connect")

	switch m.focused {
	case 0:
		EapMethodLabel = activeLabelStyle.Render("EAP Method:")
	case 1:
		phase2Label = activeLabelStyle.Render("Phase 2 (inner-auth):")
	case 2:
		IdentityLabel = activeLabelStyle.Render("Identity:")
		IdentityBox = activeBorderStyle.Render(m.Identity.View())
	case 3:
		PasswordLabel = activeLabelStyle.Render("\nPassword:")
		PasswordBox = activeBorderStyle.Render(m.Password.View())
	case 4:
		CaCertLabel = activeLabelStyle.Render("\nCA Certificate:")
		CaCertBox = activeBorderStyle.Render(m.CaCert.View())
	case 5:
		submitLabel = activeBorderStyle.
			Width(36).
			Bold(true).
			Align(lipgloss.Center).
			BorderForeground(lipgloss.Color("#cda162")).
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

	// We perform the string alterations below the remove the spacing reserved for the header and footer of the selector
	content := lipgloss.JoinVertical(lipgloss.Left,
		EapMethodLabel,
		eapStr,

		phase2Label,
		phase2Str,

		IdentityLabel,
		IdentityBox,

		PasswordLabel,
		PasswordBox,

		CaCertLabel,
		CaCertBox,

		submitLabel,
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
				str += fmt.Sprintf("  %s", unselectedFunc(selector.Model{}, EAPMethod{Type: option}, i)) + "\n"
			}
		}

		return str
	}
}
