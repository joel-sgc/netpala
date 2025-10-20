package models

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type keyMap struct {
	Scan   key.Binding
	Select key.Binding
	Quit   key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Scan, k.Select, k.Quit}
}

// FullHelp returns keybindings for the expanded help view. It's part of the
// key.Map interface.
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Scan, k.Select, k.Quit}, // first column
	}
}

var keys = keyMap{
	Scan: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r:", "scan networks"),
	),
	Select: key.NewBinding(
		key.WithKeys("enter", "space"),
		key.WithHelp("↵/space:", "select row"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "esc", "ctrl+c"),
		key.WithHelp("ctrl+q/esc:", "quit"),
	),
}

type StatusBarData struct {
	Input textinput.Model
	Err   error
}

func ModelStatusBar() StatusBarData {
	ti := textinput.New()
	ti.CharLimit = 156
	ti.Width = 32

	return StatusBarData{
		Input: ti,
		Err:   nil,
	}
}

func (m StatusBarData) Init() tea.Cmd {
	return textinput.Blink
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
	case ErrMsg:
		return m, nil
	}

	m.Input, cmd = m.Input.Update(msg)
	return m, cmd
}

// I don't understand why these numbers work, I just know that they do. Periodt.
func (m StatusBarData) View() string {
	keyHelp := help.New().View(keys)
	inputLen := len(m.Input.View()) - 12
	if m.Input.Focused() {
		if len(m.Input.Value()) == 0 {
			inputLen = len(m.Input.Placeholder)
		} else {
			inputLen = 23
		}
	}

	ansi := regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)
	clean := ansi.ReplaceAllString(keyHelp, "")

	totalWidth := windowWidth()
	remainingWidth := totalWidth - (inputLen + len(clean)) - 6 // extra 6 to account for automatic padding

	return m.Input.View() + strings.Repeat(" ", max(remainingWidth, 0)) + keyHelp
}