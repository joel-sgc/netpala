package main

import (
	"fmt"
	"os"
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
	padding := strings.Repeat(" ", (window_width() / (len(headers) * 2)) - 2)
	for i := range headers {
		headers[i] = fmt.Sprintf("%s%s%s", padding, headers[i], padding)
	}

	return headers
}
func calc_title(title string, selected bool) string {
	color := "#444a66"
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
	// Top:      	 "─",
	Bottom:      "─",
	Left:        "│",
	Right:       "│",
	// TopLeft:     "┌",
	// TopRight:    "┐",
	BottomLeft:  "└",
	BottomRight: "┘",
}

var active_border_style = lipgloss.NewStyle().Foreground(lipgloss.Color("#9cca69"))
func box_style(selectedRow int) func(row, col int) lipgloss.Style {
	return func(row int, col int) lipgloss.Style {
		switch {
			case row == 0:
				// Header style
				return lipgloss.NewStyle().
					Bold(true).
					Foreground(lipgloss.Color("#cda162")). // gold
					AlignHorizontal(lipgloss.Center)
			case row == selectedRow:
				// Signal column
				return lipgloss.NewStyle().
					Background(lipgloss.Color("#a7abca")).
					Foreground(lipgloss.Color("#444a66")).
					AlignHorizontal(lipgloss.Center)
			default:
				// Default cell style
				return lipgloss.NewStyle().Foreground(lipgloss.Color("#444a66")).AlignHorizontal(lipgloss.Center)
		}
	}
}
// var inactive_border_style = lipgloss.NewStyle().Foreground(lipgloss.Color("#a7abca"))

// Gray: a7abca
// Gold: cda162

var default_netpala_data = netpala_data{
	selected_box: 0,
	selected_entry: 3,
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