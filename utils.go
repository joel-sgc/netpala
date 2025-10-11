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
		return 80
	}
	return width
}

func freq_to_band(freq int) string {
	switch {
	case freq >= 2400 && freq < 2500:
		return "2.4 GHz"
	case freq >= 5000 && freq < 6000:
		return "5 GHz"
	case freq >= 5925 && freq < 7125:
		return "6 GHz"
	default:
		return fmt.Sprintf("%d MHz", freq)
	}
}

func pad_headers(headers []string, headers_lengths []int) []string {
	if len(headers) == 0 {
		return headers
	}
	total_width := max(window_width()-2, 1)
	num_headers := len(headers)
	fixed_total := 0
	var flexible_indices []int
	for i, length := range headers_lengths {
		if length > 0 {
			fixed_total += length + 4
		} else {
			flexible_indices = append(flexible_indices, i)
		}
	}
	remaining_width := total_width - fixed_total
	if remaining_width < 0 {
		remaining_width = 0
	}
	flex_col_width := 0
	if len(flexible_indices) > 0 {
		flex_col_width = remaining_width / len(flexible_indices)
	}
	for i := range headers {
		var left_padding_count, right_padding_count int
		var col_width int
		if headers_lengths[i] > 0 {
			left_padding_count = 2
			right_padding_count = 2
		} else {
			col_width = flex_col_width
			headerLen := len(headers[i])
			extra := col_width - headerLen
			if extra <= 0 {
				continue
			}
			left_padding_count = extra / 2
			right_padding_count = extra - left_padding_count
		}
		left_padding := strings.Repeat(" ", left_padding_count)
		right_padding := strings.Repeat(" ", right_padding_count)
		headers[i] = fmt.Sprintf("%s%s%s", left_padding, headers[i], right_padding)
	}
	current_total := 0
	for _, h := range headers {
		current_total += len(h)
	}
	diff := total_width - current_total
	for i := range diff {
		headers[i%num_headers] += " "
	}
	return headers
}

func calc_title(title string, selected bool) string {
	color := "#a7abca"
	bold := false
	if selected {
		color = "#9cca69"
		bold = true
	}
	width := window_width()
	repeatCount := max(width-4-len(title), 0)
	return lipgloss.NewStyle().
		Bold(bold).
		Foreground(lipgloss.Color(color)).
		Align(lipgloss.Center).
		Render(fmt.Sprintf("┌ %s %s┐", title, strings.Repeat("─", repeatCount)))
}

var box_border = lipgloss.Border{
	Bottom: "─", Left: "│", Right: "│",
	BottomLeft: "└", BottomRight: "┘",
}
var active_border_style = lipgloss.NewStyle().Foreground(lipgloss.Color("#9cca69"))
var inactive_border_style = lipgloss.NewStyle().Foreground(lipgloss.Color("#a7abca"))

func box_style(selectedRow int, selectedBox bool) func(row, col int) lipgloss.Style {
	return func(row int, col int) lipgloss.Style {
		switch {
		case row == 0:
			return lipgloss.NewStyle().
				Bold(true).
				Foreground(func() lipgloss.Color {
					if selectedBox {
						return lipgloss.Color("#cda162")
					}
					return lipgloss.Color("#a7abca")
				}()).
				AlignHorizontal(lipgloss.Center)
		case row == min(selectedRow+2, 11) && selectedBox:
			return lipgloss.NewStyle().
				Background(lipgloss.Color("#a7abca")).
				Foreground(lipgloss.Color("#444a66")).
				AlignHorizontal(lipgloss.Center)
		default:
			return lipgloss.NewStyle().Foreground(lipgloss.Color("#a7abca")).AlignHorizontal(lipgloss.Center)
		}
	}
}

func format_device_data(devices []device) [][]string {
	data := [][]string{
		pad_headers([]string{"Name", "Mode", "Powered", "Status"}, []int{-1, -1, -1, -1}), {""},
	}
	for _, d := range devices {
		powered := "Off"
		if d.powered {
			powered = "On"
		}
		row := []string{d.name, d.mode, powered, d.address}
		data = append(data, row)
	}
	return data
}

func format_station_data(devices []device) [][]string {
	data := [][]string{
		pad_headers([]string{"State", "Scanning", "Frequency", "Security"}, []int{-1, -1, -1, -1}), {""},
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
		row := []string{state, strconv.FormatBool(d.scanning), freq_to_band(d.frequency), d.security}
		data = append(data, row)
	}
	return data
}

func format_known_networks_data(networks []known_network, selected_row int) [][]string {
	base := [][]string{
		pad_headers([]string{"", "Name", "Security", "Hidden", "Auto Connect", "Signal"}, []int{5, -1, 23, 5, 5, 6}), {""},
	}
	window := format_arrays(networks, selected_row)
	for _, n := range window {
		connected := "     "
		if n.connected {
			connected = "  >  "
		}
		row := []string{connected, n.ssid, n.security, strconv.FormatBool(n.hidden), strconv.FormatBool(n.auto_connect), strconv.Itoa(n.signal) + "%"}
		base = append(base, row)
	}
	for i := 0; i < 10-len(networks); i++ {
		base = append(base, []string{""})
	}
	return base
}

func format_scanned_networks_data(networks []scanned_network, selected_row int) [][]string {
	data := [][]string{
		pad_headers([]string{"Name", "Security", "Signal"}, []int{-1, -1, -1}), {""},
	}
	window := format_arrays(networks, selected_row)
	for _, n := range window {
		row := []string{n.ssid, n.security, strconv.Itoa(n.signal) + "%"}
		data = append(data, row)
	}
	for i := 0; i < 10-len(networks); i++ {
		data = append(data, []string{""})
	}
	return data
}

func format_arrays[arrType known_network | scanned_network](arr []arrType, selected_index int) []arrType {
	windowSize := 10
	start := 0
	if selected_index >= windowSize {
		start = selected_index - windowSize + 1
	}
	end := start + windowSize
	if end > len(arr) {
		end = len(arr)
		start = max(end-windowSize, 0)
	}
	if start > end {
		start = end
	}
	return arr[start:end]
}

func (m *netpala_data) filterKnownFromScanned() {
	// Create a map of known SSIDs for fast lookups (more efficient than a nested loop)
	knownSSIDs := make(map[string]struct{})
	for _, known := range m.known_networks {
		knownSSIDs[known.ssid] = struct{}{}
	}

	// Build a new slice containing only the networks we want to keep.
	// This is more efficient than deleting elements from the slice in-place.
	var filteredScanned []scanned_network
	for _, scanned := range m.scanned_networks {
		if _, exists := knownSSIDs[scanned.ssid]; !exists {
			filteredScanned = append(filteredScanned, scanned)
		}
	}
	m.scanned_networks = filteredScanned
}
