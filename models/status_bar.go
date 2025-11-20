package models

import (
	"fmt"
	"netpala/common"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type keyMap struct {
	Up     key.Binding
	Down   key.Binding
	Toggle key.Binding
	Remove key.Binding
	Scan   key.Binding
	Nav    key.Binding
	Quit   key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Up, k.Down, k.Toggle, k.Remove,
		k.Scan, k.Nav, k.Quit,
	}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Toggle, k.Remove, k.Scan, k.Nav, k.Quit},
	}
}

var keys = keyMap{
	Nav: key.NewBinding(
		key.WithKeys("shift+tab/tab"),
		key.WithHelp("⇄", "Nav"),
	),
	Up: key.NewBinding(
		key.WithKeys("k", "up"),
		key.WithHelp("k/↑", "Up"),
	),
	Down: key.NewBinding(
		key.WithKeys("j", "down"),
		key.WithHelp("j/↓", "Down"),
	),
	Toggle: key.NewBinding(
		key.WithKeys("enter", "ctrl+d"),
		key.WithHelp("␣/⤶", "Dis/Connect"),
	),
	Scan: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "Scan"),
	),
	Remove: key.NewBinding(
		key.WithKeys("⌫"),
		key.WithHelp("⌫", "Remove"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q/ctrl+c", "Quit"),
	),
}

type StatusBarData struct {
	Err error
}

func ModelStatusBar() StatusBarData {
	return StatusBarData{
		Err: nil,
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
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("#a7abca"))

	keyHelp := help.New()
	keyHelp.Styles.ShortDesc = style
	keyHelp.Styles.ShortKey = style

	return renderShortHelp("|", style, style)
}

func renderShortHelp(sep string, keyStyle lipgloss.Style, descStyle lipgloss.Style) string {
	keybinds := keys.ShortHelp()
	helpObj := help.New()
	helpObj.ShortSeparator = ""
	allKeybindsWidth := lipgloss.Width(helpObj.ShortHelpView(keys.ShortHelp()))

	totalWidth := common.WindowDimensions().Width
	totalPaddingWidth := max(totalWidth-allKeybindsWidth, 0)
	columnWidth := totalPaddingWidth / (len(keybinds) * 3)

	finalStr := make([]string, len(keys.ShortHelp()))

	for i, key := range keybinds {
		finalStr[i] += strings.Repeat(" ", columnWidth)
		bind := keyStyle.Render(key.Help().Key)
		desc := descStyle.Bold(true).Render(key.Help().Desc)
		finalStr[i] += fmt.Sprintf("%s %s", bind, desc)
		finalStr[i] += strings.Repeat(" ", columnWidth)
	}

	return lipgloss.NewStyle().Width(totalWidth).Align(lipgloss.Center).Render(strings.Join(finalStr, sep))
}
