package main

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type status_bar_data struct {
	input textinput.Model
	err       error
}

func StatusBarModel() status_bar_data {
	ti := textinput.New()
	ti.CharLimit = 156
	ti.Width = 32

	return status_bar_data{
		input: ti,
		err:       nil,
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

func (m status_bar_data) View() string {
	return m.input.View()
}