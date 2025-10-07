package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)


func window_width() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		// fallback if we can’t detect terminal size
		return 80
	}
	return width
}

func pad_headers(headers []string) []string {
	total_width := window_width() - 2
	col_width := total_width / len(headers)
	
	for i := range headers {
		left_padding := strings.Repeat(" ", (col_width - len(headers[i])) / 2)
		right_padding:= strings.Repeat(" ", (col_width - len(headers[i])) - len(left_padding))

		headers[i] = fmt.Sprintf("%s%s%s", left_padding, headers[i], right_padding)
	}

	remaining := total_width % len(headers)
	for i := range remaining {
		headers[i]+= " "
	}

	return headers
}

func calc_title(title string, selected bool) string {
	color := "#a7abca"
	bold := false
	if (selected) {
		color = "#9cca69"
		bold = true
	}

	return lipgloss.NewStyle().
		Bold(bold).
		Foreground(lipgloss.Color(color)).
		Align(lipgloss.Center).
		Render(fmt.Sprintf("┌ %s %s┐", title, strings.Repeat("─", window_width() - 4 - len(title))))
}

var box_border = lipgloss.Border{
	Bottom:      "─",
	Left:        "│",
	Right:       "│",
	BottomLeft:  "└",
	BottomRight: "┘",
}

var active_border_style = lipgloss.NewStyle().Foreground(lipgloss.Color("#9cca69"))
var inactive_border_style = lipgloss.NewStyle().Foreground(lipgloss.Color("#a7abca"))

func box_style(selectedRow int, selectedBox bool) func(row, col int) lipgloss.Style {
	return func(row int, col int) lipgloss.Style {
		switch {
			case row == 0:
				// Header style
				return lipgloss.NewStyle().
					Bold(true).
					Foreground(func() (lipgloss.Color){
						if selectedBox {
							return lipgloss.Color("#cda162")
						}
						return lipgloss.Color("#a7abca")
					}()). // gold
					AlignHorizontal(lipgloss.Center)
			case row == selectedRow && selectedBox:
				// Signal column
				return lipgloss.NewStyle().
					Background(lipgloss.Color("#a7abca")).
					Foreground(lipgloss.Color("#444a66")).
					AlignHorizontal(lipgloss.Center)
			default:
				// Default cell style
				return lipgloss.NewStyle().Foreground(lipgloss.Color("#a7abca")).AlignHorizontal(lipgloss.Center)
		}
	}
}

func format_device_data(devices []device) [][]string {
	data := [][]string{
		pad_headers([]string{"Name", "Mode", "Powered", "Status"}),
		{""},
	}
	for _, d := range devices {
		powered := "Off"
		if (d.powered) {
			powered = "On"
		}

		row := []string{
			d.name,
			d.mode,
			powered,
			d.address,
		}
		data = append(data, row)
	}

	return data
}

func format_station_data(devices []device) [][]string {
	data := [][]string{
		pad_headers([]string{"State", "Scanning", "Frequency", "Security"}),
		{""},
	}
	for _, d := range devices {
		var state string
		switch d.state {
			case -1:
				state = "disconnected"
			case 0:
				state = "connecting"
			case 1:
				state = "connected"
		}
		

		row := []string{
			state,
			strconv.FormatBool(d.scanning),
			strconv.Itoa(d.frequency),
			d.security,
		}
		data = append(data, row)
	}

	return data
}

func format_known_networks_data(networks []known_network) [][]string {
	data := [][]string{
		pad_headers([]string{" ", "Name", "Security", "Hidden", "Auto Connect", "Signal"}),
		{""},
	}
	for _, n := range networks {
		
		row := []string{
			"",
			n.ssid,
			n.security,
			strconv.FormatBool(n.hidden),
			strconv.FormatBool(n.auto_connect),
			strconv.Itoa(n.signal) + "%",
		}
		data = append(data, row)
	}

	return data
}

func format_scanned_networks_data(networks []scanned_network) [][]string {
	data := [][]string{
		pad_headers([]string{"Name", "Security", "Signal"}),
		{""},
	}
	for _, n := range networks {
		
		row := []string{
			n.ssid,
			n.security,
			strconv.Itoa(n.signal) + "%",
		}
		data = append(data, row)
	}

	return data
}

// Gray: a7abca
// Gold: cda162

var default_netpala_data = netpala_data{
	selected_box: 0,
	selected_entry: 2,
	device_data: []device{
		{
			name: "wlan0",
			mode: "station",
			powered: true,
			address: "123",
			state: 1,
			currentbssid: "test01",
			scanning: false,
			frequency: 5000,
			security: "wpa-psk",
		},
	},
	known_networks: []known_network{
		{
			bssid: "living-la-buena-vida",
			ssid: "Living La Buena Vida",
			security: "wpa-psk",
			hidden: true,
			auto_connect: true,
			signal: 100,
		},
		{
			bssid: "luc",
			ssid: "LUC",
			security: "wpa-eap",
			hidden: false,
			auto_connect: true,
			signal: 0,
		},
	},
	scanned_networks: []scanned_network{
		{
			bssid: "luc",
			ssid: "LUC",
			security: "wpa-eap",
			signal: 99,
		},
	},
}