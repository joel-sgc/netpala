package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss/table"
)

type table_data struct {
	title             string
	is_table_selected bool
	selected_row      int
	device_data       []device
	station_data      []device
	known_networks    []known_network
	scanned_networks  []scanned_network
}

func TableModel(
	title string,
	is_table_selected bool,
	selected_row int,
	dev_data []device,
	station_data []device,
	known_nets []known_network,
	scanned_nets []scanned_network,
) table_data {
	return table_data{
		title:             title,
		is_table_selected: is_table_selected,
		selected_row:      selected_row,
		device_data:       dev_data,
		station_data:      station_data,
		known_networks:    known_nets,
		scanned_networks:  scanned_nets,
	}
}

func (m table_data) Init() tea.Cmd {
	return nil
}

func (m table_data) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m table_data) View() string {
	border_style := inactive_border_style
	if m.is_table_selected {
		border_style = active_border_style
	}

	var table_data [][]string
	if m.device_data != nil {
		table_data = format_device_data(m.device_data)
	} else if m.station_data != nil {
		table_data = format_station_data(m.station_data)
	} else if m.known_networks != nil {
		table_data = format_known_networks_data(m.known_networks, m.selected_row)
	} else {
		table_data = format_scanned_networks_data(m.scanned_networks, m.selected_row)
	}

	table := table.New().
		Border(box_border).
		BorderColumn(false).
		BorderStyle(border_style).
		StyleFunc(box_style(m.selected_row, m.is_table_selected)).
		Rows(table_data...)

	return (calc_title(m.title, m.is_table_selected) + table.Render()) + "\n"
}
