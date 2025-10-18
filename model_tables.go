package main

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// tables_model is a container model that holds all the main tables.
type tables_model struct {
	// We'll populate these fields from the main model just before rendering.
	selected_box     int
	selected_entry   int
	nets_height      int
	device_data      []device
	vpn_data         []vpn_connection
	known_networks   []known_network
	scanned_networks []scanned_network
}

func (m tables_model) Init() tea.Cmd {
	return nil
}

func (m tables_model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// This model is for viewing only; all updates are handled by netpala_data
	return m, nil
}

// View renders all tables in order.
func (m tables_model) View() string {
	device_table := TableModel("Device", m.selected_box == 0, m.selected_entry, -1, m.device_data, nil, nil, nil, nil)
	station_table := TableModel("Station", m.selected_box == 1, m.selected_entry, -1, nil, m.device_data, nil, nil, nil)
	vpn_table_model := TableModel("Virtual Private Networks", m.selected_box == 2, m.selected_entry, -1, nil, nil, m.vpn_data, nil, nil)
	known_nets_table := TableModel("Known Networks", m.selected_box == 3, m.selected_entry, m.nets_height, nil, nil, nil, m.known_networks, nil)
	scanned_nets_table := TableModel("New Networks", m.selected_box == 4, m.selected_entry, m.nets_height, nil, nil, nil, nil, m.scanned_networks)

	vpn_view := vpn_table_model.View()
	if len(m.vpn_data) == 0 {
		vpn_view = ""
	}

	return strings.Join([]string{
		device_table.View(),
		station_table.View(),
		vpn_view,
		known_nets_table.View(),
		scanned_nets_table.View(),
	}, "")
}