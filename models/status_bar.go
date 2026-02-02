package models

import (
	"fmt"
	"netpala/common"
	"netpala/config"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type StatusBarData struct {
	Err    error
	KeyMap config.AppKeyMap
	Colors config.Colors
}

func ModelStatusBar(keyMap config.AppKeyMap, colors config.Colors) StatusBarData {
	return StatusBarData{
		Err:    nil,
		KeyMap: keyMap,
		Colors: colors,
	}
}

func (m StatusBarData) Init() tea.Cmd {
	return nil
}

func (m StatusBarData) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter, tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		}

	// We handle errors just like any other message
	case common.ErrMsg:
		return m, nil
	}

	return m, cmd
}

// I don't understand why these numbers work, I just know that they do. Periodt.
func (m StatusBarData) View() string {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color(m.Colors.HelpText))

	keyHelp := help.New()
	keyHelp.Styles.ShortDesc = style
	keyHelp.Styles.ShortKey = style

	return renderShortHelp("|", style, style, m.KeyMap)
}

func renderShortHelp(sep string, keyStyle lipgloss.Style, descStyle lipgloss.Style, keyMap config.AppKeyMap) string {
	keybinds := keyMap.ShortHelp()
	helpObj := help.New()
	helpObj.ShortSeparator = ""
	allKeybindsWidth := lipgloss.Width(helpObj.ShortHelpView(keybinds))

	totalWidth := common.WindowDimensions().Width
	totalPaddingWidth := max(totalWidth-allKeybindsWidth, 0)
	columnWidth := totalPaddingWidth / (len(keybinds) * 3)

	finalStr := make([]string, len(keybinds))

	for i, k := range keybinds {
		finalStr[i] += strings.Repeat(" ", columnWidth)
		bind := keyStyle.Render(k.Help().Key)
		desc := descStyle.Bold(true).Render(k.Help().Desc)
		finalStr[i] += fmt.Sprintf("%s %s", bind, desc)
		finalStr[i] += strings.Repeat(" ", columnWidth)
	}

	return lipgloss.NewStyle().Width(totalWidth).Align(lipgloss.Center).Render(strings.Join(finalStr, sep))
}

// ShortHelp implements help.KeyMap for backwards compatibility
func (m StatusBarData) ShortHelp() []key.Binding {
	return m.KeyMap.ShortHelp()
}

// FullHelp implements help.KeyMap for backwards compatibility
func (m StatusBarData) FullHelp() [][]key.Binding {
	return m.KeyMap.FullHelp()
}