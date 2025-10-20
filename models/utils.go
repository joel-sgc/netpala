package models

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

func windowWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 80
	}
	return width
}

func freqToBand(freq int) string {
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

func padHeaders(headers []string, headersLengths []int) []string {
	if len(headers) == 0 {
		return headers
	}
	totalWidth := max(windowWidth()-2, 1)
	numHeaders := len(headers)
	fixedTotal := 0
	var flexibleIndices []int
	for i, length := range headersLengths {
		if length > 0 {
			fixedTotal += length + 4
		} else {
			flexibleIndices = append(flexibleIndices, i)
		}
	}
	remainingWidth := totalWidth - fixedTotal
	if remainingWidth < 0 {
		remainingWidth = 0
	}
	flexColWidth := 0
	if len(flexibleIndices) > 0 {
		flexColWidth = remainingWidth / len(flexibleIndices)
	}
	for i := range headers {
		var leftPaddingCount, rightPaddingCount int
		var colWidth int
		if headersLengths[i] > 0 {
			leftPaddingCount = 2
			rightPaddingCount = 2
		} else {
			colWidth = flexColWidth
			headerLen := len(headers[i])
			extra := colWidth - headerLen
			if extra <= 0 {
				continue
			}
			leftPaddingCount = extra / 2
			rightPaddingCount = extra - leftPaddingCount
		}
		leftPadding := strings.Repeat(" ", leftPaddingCount)
		rightPadding := strings.Repeat(" ", rightPaddingCount)
		headers[i] = fmt.Sprintf("%s%s%s", leftPadding, headers[i], rightPadding)
	}
	currentTotal := 0
	for _, h := range headers {
		currentTotal += len(h)
	}
	diff := totalWidth - currentTotal
	for i := range diff {
		headers[i%numHeaders] += " "
	}
	return headers
}

func calcTitle(title string, selected bool) string {
	color := "#a7abca"
	bold := false
	if selected {
		color = "#9cca69"
		bold = true
	}
	width := windowWidth()
	repeatCount := max(width-4-len(title), 0)
	return lipgloss.NewStyle().
		Bold(bold).
		Foreground(lipgloss.Color(color)).
		Align(lipgloss.Center).
		Render(fmt.Sprintf("┌ %s %s┐", title, strings.Repeat("─", repeatCount)))
}

var boxBorder = lipgloss.Border{
	Bottom: "─", Left: "│", Right: "│",
	BottomLeft: "└", BottomRight: "┘",
}
var activeBorderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#9cca69"))
var inactiveBorderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#a7abca"))

func boxStyle(selectedRow int, selectedBox bool) func(row, col int) lipgloss.Style {
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

func formatDeviceData(devices []Device) [][]string {
	data := [][]string{
		padHeaders([]string{"Name", "Mode", "Powered", "Status"}, []int{-1, -1, -1, -1}), {""},
	}
	for _, d := range devices {
		powered := "Off"
		if d.Powered {
			powered = "On"
		}
		row := []string{d.Name, d.Mode, powered, d.Address}
		data = append(data, row)
	}
	return data
}

func formatStationData(devices []Device) [][]string {
	data := [][]string{
		padHeaders([]string{"State", "Scanning", "Frequency", "Security"}, []int{-1, -1, -1, -1}), {""},
	}
	for _, d := range devices {
		var state string
		switch d.State {
		case -1:
			state = "disconnected"
		case 0:
			state = "connecting"
		case 1:
			state = "connected"
		}
		row := []string{state, strconv.FormatBool(d.Scanning), freqToBand(d.Frequency), d.Security}
		data = append(data, row)
	}
	return data
}

func formatVpnData(vpns []VpnConnection) [][]string {
	data := [][]string{
		padHeaders([]string{"", "Name", "Type"}, []int{5, -1, -1}), {""},
	}
	for _, vpn := range vpns {
		state := "     "
		if vpn.Connected {
			state = "  >  "
		}

		row := []string{state, vpn.Name, vpn.ConnType}
		data = append(data, row)
	}
	return data
}

func formatKnownNetworksData(networks []KnownNetwork, selectedRow int, height int) [][]string {
	base := [][]string{
		padHeaders([]string{"", "Name", "Security", "Hidden", "Auto Connect", "Signal"}, []int{5, -1, 23, 5, 5, 6}), {""},
	}
	window := formatArrays(networks, selectedRow, height)
	for _, n := range window {
		connected := "     "
		if n.Connected {
			connected = "  >  "
		}
		row := []string{connected, n.SSID, n.Security, strconv.FormatBool(n.Hidden), strconv.FormatBool(n.AutoConnect), strconv.Itoa(n.Signal) + "%"}
		base = append(base, row)
	}

	if height < 10 {
		height--
	}
	for i := 0; i < height-len(networks); i++ {
		base = append(base, []string{""})
	}
	return base
}

func formatScannedNetworksData(networks []ScannedNetwork, selectedRow int, height int) [][]string {
	data := [][]string{
		padHeaders([]string{"Name", "Security", "Signal"}, []int{-1, -1, -1}), {""},
	}
	window := formatArrays(networks, selectedRow, height)
	for _, n := range window {
		row := []string{n.SSID, n.Security, strconv.Itoa(n.Signal) + "%"}
		data = append(data, row)
	}
	for i := 0; i < height-len(networks); i++ {
		data = append(data, []string{""})
	}
	return data
}

func formatArrays[ArrType KnownNetwork | ScannedNetwork](arr []ArrType, selectedIndex int, windowSize int) []ArrType {
	start := 0
	if selectedIndex >= windowSize {
		start = selectedIndex - windowSize + 1
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

func CalculatePadding(s string) int {
	totalWidth := windowWidth()
	line := strings.Split(s, "\n")[0]

	// Use lipgloss.Width to correctly calculate visible width, ignoring ANSI codes
	textWidth := lipgloss.Width(line)

	// Calculate padding and ensure it's not negative
	return max(0, (totalWidth-textWidth)/2)
}