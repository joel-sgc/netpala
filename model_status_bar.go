package main

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

type status_bar_data struct {
	input textinput.Model
	err   error
}

func StatusBarModel() status_bar_data {
	ti := textinput.New()
	ti.CharLimit = 156
	ti.Width = 32

	return status_bar_data{
		input: ti,
		err:   nil,
	}
}

func (m status_bar_data) Init() tea.Cmd {
	return textinput.Blink
}

func (m status_bar_data) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter, tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		}

	// We handle errors just like any other message
	case errMsg:
		return m, nil
	}

	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// I don't understand why these numbers work, I just know that they do. Periodt.
func (m status_bar_data) View() string {
	key_help := help.New().View(keys)
	input_len := len(m.input.View()) - 12
	if m.input.Focused() {
		if len(m.input.Value()) == 0 {
			input_len = len(m.input.Placeholder)
		} else {
			input_len = 23
		}
	}

	ansi := regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)
	clean := ansi.ReplaceAllString(key_help, "")

	total_width := window_width()
	remaining_width := total_width - (input_len + len(clean)) - 6 // extra 6 to account for automatic padding

	return m.input.View() + strings.Repeat(" ", max(remaining_width, 0)) + key_help
}
