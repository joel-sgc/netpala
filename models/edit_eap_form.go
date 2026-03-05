package models

import (
	"fmt"
	"strings"

	"netpala/common"
	"netpala/config"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mritd/bubbles/selector"
)

// initSelectorSize pushes a WindowSizeMsg with the actual terminal dimensions
// into a selector so it renders correctly on first display — identical to the
// pattern used when re-initialising WpaEapForm in netpala.go.
func initSelectorSize(s selector.Model) selector.Model {
	dims := common.WindowDimensions()
	sm, _ := s.Update(tea.WindowSizeMsg{Width: dims.Width, Height: dims.Height})
	return *sm
}

// EditEapForm is the TUI model for editing a saved WPA-EAP connection.
// Focus states:
//
//	0 = SSID textinput
//	1 = EAP Method selector
//	2 = Phase2 Auth selector
//	3 = Identity textinput
//	4 = Password textinput
//	5 = CA Certificate textinput
//	6 = AutoConnect toggle
//	7 = Hidden toggle
//	8 = Cancel button
//	9 = Save button
type EditEapForm struct {
	Msg            common.EapEditSettingsMsg // preserves ConnectionPath for submit
	EapMethod      selector.Model
	Phase2Auth     selector.Model
	SSID           textinput.Model
	Identity       textinput.Model
	Password       textinput.Model
	CaCert         textinput.Model
	AutoConnect    bool
	Hidden         bool
	showPassword   bool
	EapSelected    bool
	Phase2Selected bool
	focused        int
	Colors         config.Colors
}

// reorderToFront moves the EAPMethod whose Type equals targetType to index 0.
// The selector.Model.index field is unexported with no setter, so we ensure
// the desired default is the first item so that Selected() returns it at open.
func reorderToFront(methods []any, targetType string) []any {
	for i, m := range methods {
		if m.(EAPMethod).Type == targetType {
			if i == 0 {
				return methods
			}
			reordered := make([]any, 0, len(methods))
			reordered = append(reordered, methods[i])
			reordered = append(reordered, methods[:i]...)
			reordered = append(reordered, methods[i+1:]...)
			return reordered
		}
	}
	return methods
}

func ModelEditEapForm(msg common.EapEditSettingsMsg, colors config.Colors) EditEapForm {
	ssidInput := textinput.New()
	ssidInput.Placeholder = "Network name"
	ssidInput.Prompt = ""
	ssidInput.Width = 32
	ssidInput.CharLimit = 64
	ssidInput.SetValue(msg.SSID)
	ssidInput.Focus()

	identityInput := textinput.New()
	identityInput.Placeholder = "Identity"
	identityInput.Prompt = ""
	identityInput.Width = 32
	identityInput.CharLimit = 256
	identityInput.SetValue(msg.Identity)

	passwordInput := textinput.New()
	passwordInput.Placeholder = "Password"
	passwordInput.Prompt = ""
	passwordInput.Width = 32
	passwordInput.CharLimit = 256
	passwordInput.EchoMode = textinput.EchoPassword
	passwordInput.EchoCharacter = '*'
	passwordInput.SetValue(msg.Password)

	caCertInput := textinput.New()
	caCertInput.Placeholder = "e.g. /etc/ssl/certs/ca.pem"
	caCertInput.Prompt = ""
	caCertInput.Width = 32
	caCertInput.CharLimit = 512
	caCertInput.SetValue(msg.CaCert)

	// Reorder EAP methods so the current one is at index 0 (selector starts at index 0).
	eapMethods := []any{
		EAPMethod{Type: "PEAP"},
		EAPMethod{Type: "TTLS"},
		EAPMethod{Type: "TLS"},
		EAPMethod{Type: "PWD"},
	}
	currentEap := strings.ToUpper(msg.EapMethod)
	if currentEap == "" {
		currentEap = "PEAP"
	}
	eapMethods = reorderToFront(eapMethods, currentEap)

	// Reorder Phase2 methods so the current one is at index 0.
	phase2Methods := []any{
		EAPMethod{Type: "MSCHAPV2"},
		EAPMethod{Type: "PAP"},
		EAPMethod{Type: "CHAP"},
		EAPMethod{Type: "MSCHAP"},
		EAPMethod{Type: "NONE"},
	}
	currentPhase2 := strings.ToUpper(msg.Phase2Auth)
	if currentPhase2 == "" {
		currentPhase2 = "MSCHAPV2"
	}
	phase2Methods = reorderToFront(phase2Methods, currentPhase2)

	m := EditEapForm{
		Msg:            msg,
		SSID:           ssidInput,
		Identity:       identityInput,
		Password:       passwordInput,
		CaCert:         caCertInput,
		AutoConnect:    msg.AutoConnect,
		Hidden:         msg.Hidden,
		focused:        0,
		EapSelected:    false,
		Phase2Selected: false,
		Colors:         colors,
		EapMethod: selector.Model{
			Data:       eapMethods,
			PerPage:    4,
			HeaderFunc: emptyFunc,
			FooterFunc: emptyFunc,
		},
		Phase2Auth: selector.Model{
			Data:       phase2Methods,
			PerPage:    5,
			HeaderFunc: emptyFunc,
			FooterFunc: emptyFunc,
		},
	}

	m.EapMethod.SelectedFunc = makeSelectedFunc(colors)
	m.EapMethod.UnSelectedFunc = makeUnselectedFunc(colors)
	m.EapMethod.FinishedFunc = makeCompletedFunc([]string{"PEAP", "TTLS", "TLS", "PWD"}, colors)

	m.Phase2Auth.SelectedFunc = makeUnselectedFunc(colors)
	m.Phase2Auth.UnSelectedFunc = makeUnselectedFunc(colors)
	m.Phase2Auth.FinishedFunc = makeCompletedFunc([]string{"MSCHAPV2", "PAP", "CHAP", "MSCHAP", "NONE"}, colors)

	m.EapMethod = initSelectorSize(m.EapMethod)
	m.Phase2Auth = initSelectorSize(m.Phase2Auth)

	// Confirm the pre-positioned selection in each selector so View() renders
	// via FinishedFunc (showing the pre-populated value) from first display.
	enterKey := tea.KeyMsg{Type: tea.KeyEnter}
	eapSm, _ := m.EapMethod.Update(enterKey)
	m.EapMethod = *eapSm
	phase2Sm, _ := m.Phase2Auth.Update(enterKey)
	m.Phase2Auth = *phase2Sm

	m.EapSelected = true
	m.Phase2Selected = true

	return m
}

func (m EditEapForm) Init() tea.Cmd {
	return textinput.Blink
}

const eapEditTotal = 10

// applyFocus updates which component is focused based on m.focused.
func (m EditEapForm) applyFocus() EditEapForm {
	m.SSID.Blur()
	m.Identity.Blur()
	m.Password.Blur()
	m.CaCert.Blur()
	m.EapMethod.SelectedFunc = makeUnselectedFunc(m.Colors)
	m.Phase2Auth.SelectedFunc = makeUnselectedFunc(m.Colors)

	switch m.focused {
	case 0:
		m.SSID.Focus()
	case 1:
		m.EapMethod.SelectedFunc = makeSelectedFunc(m.Colors)
	case 2:
		m.Phase2Auth.SelectedFunc = makeSelectedFunc(m.Colors)
	case 3:
		m.Identity.Focus()
	case 4:
		m.Password.Focus()
	case 5:
		m.CaCert.Focus()
	}
	return m
}

func (m EditEapForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch key := msg.(type) {
	case tea.KeyMsg:
		switch key.String() {
		case "esc", "ctrl+c":
			return m, func() tea.Msg { return common.ExitFormMsg{} }

		case "tab", "shift+tab":
			m.focused = (m.focused + func() int {
				if key.String() == "shift+tab" {
					return eapEditTotal - 1
				}
				return 1
			}()) % eapEditTotal
			m = m.applyFocus()
			return m, nil

		case "left", "right":
			if m.focused == 8 {
				m.focused = 9
			} else if m.focused == 9 {
				m.focused = 8
			}
			return m, nil

		case "ctrl+p":
			if m.focused == 4 {
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
			case 6:
				m.AutoConnect = !m.AutoConnect
				return m, nil
			case 7:
				m.Hidden = !m.Hidden
				return m, nil
			}
			// fall through to text inputs for other focus states

		case "enter":
			switch m.focused {
			case 6:
				m.AutoConnect = !m.AutoConnect
				return m, nil
			case 7:
				m.Hidden = !m.Hidden
				return m, nil
			case 8: // Cancel
				return m, func() tea.Msg { return common.ExitFormMsg{} }
			case 9: // Save
				cfg := map[string]string{
					"ssid":        m.SSID.Value(),
					"eap":         m.EapMethod.Selected().(EAPMethod).Type,
					"phase2-auth": m.Phase2Auth.Selected().(EAPMethod).Type,
					"identity":    m.Identity.Value(),
					"password":    m.Password.Value(),
					"ca_cert":     m.CaCert.Value(),
					"autoconnect": fmt.Sprintf("%t", m.AutoConnect),
					"hidden":      fmt.Sprintf("%t", m.Hidden),
				}
				connPath := m.Msg.ConnectionPath
				return m, func() tea.Msg {
					return common.SubmitEditEapFormMsg{
						ConnectionPath: connPath,
						Config:         cfg,
					}
				}
			}
		}
	}

	// Pass non-key messages (WindowSizeMsg etc.) to both selectors.
	if _, ok := msg.(tea.KeyMsg); !ok {
		var sm *selector.Model
		sm, cmd = m.EapMethod.Update(msg)
		m.EapMethod = *sm
		cmds = append(cmds, cmd)

		sm, cmd = m.Phase2Auth.Update(msg)
		m.Phase2Auth = *sm
		cmds = append(cmds, cmd)
	}

	// Pass all messages to text inputs (they handle focus internally).
	m.SSID, cmd = m.SSID.Update(msg)
	cmds = append(cmds, cmd)
	m.Identity, cmd = m.Identity.Update(msg)
	cmds = append(cmds, cmd)
	m.Password, cmd = m.Password.Update(msg)
	cmds = append(cmds, cmd)
	m.CaCert, cmd = m.CaCert.Update(msg)
	cmds = append(cmds, cmd)

	// Pass key messages to the focused selector only.
	if _, ok := msg.(tea.KeyMsg); ok {
		var sm *selector.Model
		switch m.focused {
		case 1:
			sm, cmd = m.EapMethod.Update(msg)
			m.EapMethod = *sm
			cmds = append(cmds, cmd)
			// After enter, advance focus to Phase2.
			if msg.(tea.KeyMsg).String() == "enter" {
				m.EapSelected = true
				m.focused = 2
				m.EapMethod.SelectedFunc = makeUnselectedFunc(m.Colors)
				m.Phase2Auth.SelectedFunc = makeSelectedFunc(m.Colors)
			}
		case 2:
			sm, cmd = m.Phase2Auth.Update(msg)
			m.Phase2Auth = *sm
			cmds = append(cmds, cmd)
			// After enter, advance focus to Identity.
			if msg.(tea.KeyMsg).String() == "enter" {
				m.Phase2Selected = true
				m.EapMethod.SelectedFunc = makeUnselectedFunc(m.Colors)
				m.Phase2Auth.SelectedFunc = makeUnselectedFunc(m.Colors)
				m.focused = 3
				m.Identity.Focus()
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m EditEapForm) View() string {
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

	// Render-only copies at full width.
	const inputWidth = 52
	ssidInput := m.SSID
	ssidInput.Width = inputWidth
	identityInput := m.Identity
	identityInput.Width = inputWidth
	passwordInput := m.Password
	passwordInput.Width = inputWidth
	caCertInput := m.CaCert
	caCertInput.Width = inputWidth

	// --- header ---
	ssid := m.Msg.SSID
	if lipgloss.Width(ssid) > 44 {
		ssid = ssid[:41] + "..."
	}
	hdrTitleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(m.Colors.ActiveText)).
		Width(56).
		Align(lipgloss.Center)
	hdrLabelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.Colors.Active)).
		Bold(true)
	hdrValueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.Colors.Primary))
	headerInfoRow := lipgloss.JoinHorizontal(lipgloss.Left,
		hdrLabelStyle.Width(8).Render("Method"),
		hdrValueStyle.Width(14).Render(strings.ToUpper(m.Msg.EapMethod)),
		hdrLabelStyle.Width(10).Render("Identity"),
		hdrValueStyle.Width(24).Render(m.Msg.Identity),
	)
	headerDivider := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.Colors.Inactive)).
		Render(strings.Repeat("─", 56))

	// --- selector labels ---
	eapMethodLabel := inactiveLabelStyle.Render("EAP Method:")
	phase2Label := inactiveLabelStyle.Render("Phase 2 (inner-auth):")
	if m.focused == 1 {
		eapMethodLabel = activeLabelStyle.Render("EAP Method:")
	}
	if m.focused == 2 {
		phase2Label = activeLabelStyle.Render("Phase 2 (inner-auth):")
	}

	// Build selector view strings.
	eapStr := strings.TrimSuffix(strings.Replace(m.EapMethod.View(), "\n", "", 2), "\n")
	phase2Str := strings.TrimSuffix(strings.Replace(m.Phase2Auth.View(), "\n", "", 2), "\n")
	if m.EapSelected {
		eapStr = strings.TrimRight(m.EapMethod.View(), "\n")
	}
	if m.Phase2Selected {
		phase2Str = strings.TrimRight(m.Phase2Auth.View(), "\n")
	}

	eapColumn := lipgloss.NewStyle().Width(26).Render(
		lipgloss.JoinVertical(lipgloss.Left, eapMethodLabel, eapStr),
	)
	phase2Column := lipgloss.NewStyle().Width(30).Render(
		lipgloss.JoinVertical(lipgloss.Left, phase2Label, phase2Str),
	)
	selectorsRow := lipgloss.JoinHorizontal(lipgloss.Top, eapColumn, phase2Column)

	// --- SSID input ---
	ssidLabel := inactiveLabelStyle.Render("SSID:")
	ssidBox := inactiveBorderStyle.Width(inputWidth).Render(ssidInput.View())
	if m.focused == 0 {
		ssidLabel = activeLabelStyle.Render("SSID:")
		ssidBox = activeBorderStyle.Width(inputWidth).Render(ssidInput.View())
	}

	// --- Identity input ---
	identityLabel := inactiveLabelStyle.Render("\nIdentity:")
	identityBox := inactiveBorderStyle.Render(identityInput.View())
	if m.focused == 3 {
		identityLabel = activeLabelStyle.Render("\nIdentity:")
		identityBox = activeBorderStyle.Render(identityInput.View())
	}

	// --- Password input ---
	pwHint := " [ctrl+p: show]"
	if m.showPassword {
		pwHint = " [ctrl+p: hide]"
	}
	passwordLabel := inactiveLabelStyle.Render("\nPassword:")
	passwordBox := inactiveBorderStyle.Render(passwordInput.View())
	if m.focused == 4 {
		passwordLabel = activeLabelStyle.Render("\nPassword:" + pwHint)
		passwordBox = activeBorderStyle.Render(passwordInput.View())
	}

	// --- CA Certificate input ---
	caCertLabel := inactiveLabelStyle.Render("\nCA Certificate:")
	caCertBox := inactiveBorderStyle.Render(caCertInput.View())
	if m.focused == 5 {
		caCertLabel = activeLabelStyle.Render("\nCA Certificate:")
		caCertBox = activeBorderStyle.Render(caCertInput.View())
	}

	// --- Toggle boxes ---
	checkmark := func(on bool) string {
		if on {
			return "✓ Enabled "
		}
		return "✗ Disabled"
	}
	autoConnectBox := inactiveBorderStyle.Width(26).Render("AutoConnect: " + checkmark(m.AutoConnect))
	hiddenBox := inactiveBorderStyle.Width(26).Render("Hidden:      " + checkmark(m.Hidden))
	if m.focused == 6 {
		autoConnectBox = activeBorderStyle.
			Width(26).
			Bold(true).
			BorderForeground(lipgloss.Color(m.Colors.Primary)).
			Render("AutoConnect: " + checkmark(m.AutoConnect))
	}
	if m.focused == 7 {
		hiddenBox = activeBorderStyle.
			Width(26).
			Bold(true).
			BorderForeground(lipgloss.Color(m.Colors.Primary)).
			Render("Hidden:      " + checkmark(m.Hidden))
	}
	toggleRow := lipgloss.JoinHorizontal(lipgloss.Top, autoConnectBox, hiddenBox)

	// --- Cancel / Save buttons ---
	cancelButton := inactiveBorderStyle.Width(26).Align(lipgloss.Center).Render("Cancel")
	saveButton := inactiveBorderStyle.Width(26).Align(lipgloss.Center).Render("Save")
	if m.focused == 8 {
		cancelButton = activeBorderStyle.
			Width(26).
			Bold(true).
			Align(lipgloss.Center).
			BorderForeground(lipgloss.Color(m.Colors.ActiveText)).
			Render("Cancel")
	}
	if m.focused == 9 {
		saveButton = activeBorderStyle.
			Width(26).
			Bold(true).
			Align(lipgloss.Center).
			BorderForeground(lipgloss.Color(m.Colors.ActiveText)).
			Render("Save")
	}
	buttonRow := lipgloss.JoinHorizontal(lipgloss.Top, cancelButton, saveButton)

	content := lipgloss.JoinVertical(lipgloss.Left,
		hdrTitleStyle.Render(ssid),
		headerInfoRow,
		headerDivider,
		"",
		ssidLabel,
		ssidBox,
		"",
		selectorsRow,
		identityLabel,
		identityBox,
		passwordLabel,
		passwordBox,
		caCertLabel,
		caCertBox,
		"",
		toggleRow,
		"",
		buttonRow,
	)
	return formStyle.Render(content)
}
