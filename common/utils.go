package common

import (
	"fmt"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

func WindowDimensions() struct{ Width, Height int } {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return struct{ Width, Height int }{80, 80}
	}
	return struct{ Width, Height int }{width, height}
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

func padHeaders(headers []string, headerLengths []int) []string {
	if len(headers) == 0 {
		return headers
	}

	// Fallback: if no lengths provided, auto-fill with -1 (flex)
	if headerLengths == nil || len(headerLengths) != len(headers) {
		headerLengths = make([]int, len(headers))
		for i := range headerLengths {
			headerLengths[i] = -1
		}
	}

	availableWidth := max(WindowDimensions().Width-2, 1)

	// Calculate fixed width and identify flexible columns
	fixedWidth := 0
	flexColumns := []int{} // indices of flexible columns
	for i, w := range headerLengths {
		if w == -1 {
			flexColumns = append(flexColumns, i)
		} else {
			fixedWidth += w
		}
	}

	remaining := max(availableWidth-fixedWidth, 0)

	// Calculate base width and remainder for flexible columns
	flexCount := len(flexColumns)
	baseWidth := 0
	remainder := 0

	if flexCount > 0 {
		baseWidth = remaining / flexCount
		remainder = remaining % flexCount
	}

	// Distribute widths to flexible columns, handling remainder
	flexWidths := make([]int, flexCount)
	for i := range flexWidths {
		flexWidths[i] = baseWidth
		if i < remainder {
			flexWidths[i]++
		}
	}

	// Assign the calculated widths back to headerLengths
	for i, flexIndex := range flexColumns {
		headerLengths[flexIndex] = flexWidths[i]
	}

	// Render headers with their respective widths
	finalHeaders := make([]string, len(headers))
	for i, h := range headers {
		width := max(headerLengths[i], 1)
		finalHeaders[i] = lipgloss.NewStyle().
			Width(width).
			Align(lipgloss.Center).
			Render(h)
	}

	return finalHeaders
}

func CalcTitle(title string, selected bool) string {
	color := "#a7abca"
	bold := false
	if selected {
		color = "#9cca69"
		bold = true
	}
	width := WindowDimensions().Width
	repeatCount := max(width-4-len(title), 0)
	return lipgloss.NewStyle().
		Bold(bold).
		Foreground(lipgloss.Color(color)).
		Align(lipgloss.Center).
		Render(fmt.Sprintf("┌ %s %s┐", title, strings.Repeat("─", repeatCount)))
}

var BoxBorder = lipgloss.Border{
	Bottom: "─", Left: "│", Right: "│",
	BottomLeft: "└", BottomRight: "┘",
}
var ActiveBorderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#9cca69"))
var InactiveBorderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#a7abca"))

func BoxStyle(selectedRow int, selectedBox bool, height int) func(row, col int) lipgloss.Style {
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
		case row == min(selectedRow+2, height+1) && selectedBox:
			return lipgloss.NewStyle().
				Background(lipgloss.Color("#a7abca")).
				Foreground(lipgloss.Color("#444a66")).
				AlignHorizontal(lipgloss.Center)
		default:
			return lipgloss.NewStyle().Foreground(lipgloss.Color("#a7abca")).AlignHorizontal(lipgloss.Center)
		}
	}
}

func FormatDeviceData(devices []Device) [][]string {
	data := [][]string{
		padHeaders([]string{"Name", "Mode", "Powered", "Status"}, []int{-1, -1, -1, -1}), {""},
	}
	for _, d := range devices {
		powered := "Off"
		if d.Powered {
			powered = "On"
		}

		row := []string{d.Name, d.Mode, powered, d.Address}
		for i := range row {
			if lipgloss.Width(row[i]) > lipgloss.Width(data[0][i]) {
				row[i] = row[i][:max(0, lipgloss.Width(data[0][i])-3)] + "..."
			}
		}

		data = append(data, row)
	}
	return data
}

func FormatStationData(devices []Device) [][]string {
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
		for i := range row {
			if lipgloss.Width(row[i]) > lipgloss.Width(data[0][i]) {
				row[i] = row[i][:max(0, lipgloss.Width(data[0][i])-3)] + "..."
			}
		}

		data = append(data, row)
	}
	return data
}

func FormatVpnData(vpns []VpnConnection) [][]string {
	data := [][]string{
		padHeaders([]string{"", "Name", "Type"}, []int{5, -1, -1}), {""},
	}
	for _, vpn := range vpns {
		state := "     "
		if vpn.Connected {
			state = "  >  "
		}

		row := []string{state, vpn.Name, vpn.ConnType}
		for i := range row {
			if lipgloss.Width(row[i]) > lipgloss.Width(data[0][i]) {
				row[i] = row[i][:max(0, lipgloss.Width(data[0][i])-3)] + "..."
			}
		}

		data = append(data, row)
	}
	return data
}

func FormatKnownNetworksData(networks []KnownNetwork, selectedRow int, height int) [][]string {
	base := [][]string{
		padHeaders([]string{"", "Name", "Security", "Hidden", "Auto Connect", "Signal"}, []int{5, -1, 12, 10, 16, 10}), {""},
	}
	window := FormatArrays(networks, selectedRow, height)
	for _, n := range window {
		connected := "     "
		if n.Connected {
			connected = "  >  "
		}
		row := []string{connected, strings.TrimSpace(n.SSID), n.Security, strconv.FormatBool(n.Hidden), strconv.FormatBool(n.AutoConnect), strconv.Itoa(n.Signal) + "%"}
		for i := range row {
			if len(row[i]) > lipgloss.Width(base[0][i]) {
				row[i] = row[i][:max(0, lipgloss.Width(base[0][i])-3)] + "..."
			}
		}

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

func FormatScannedNetworksData(networks []ScannedNetwork, selectedRow int, height int) [][]string {
	totalWidth := WindowDimensions().Width - 2

	signalWidth := totalWidth / 4
	securityWidth := (totalWidth / 4) + totalWidth%4

	data := [][]string{
		padHeaders([]string{"Name", "Security", "Signal"}, []int{-1, securityWidth, signalWidth}), {""},
	}
	window := FormatArrays(networks, selectedRow, height)
	for _, n := range window {
		row := []string{n.SSID, n.Security, strconv.Itoa(n.Signal) + "%"}
		for i := range row {
			if lipgloss.Width(row[i]) > lipgloss.Width(data[0][i]) {
				row[i] = row[i][:max(0, lipgloss.Width(data[0][i])-3)] + "..."
			}
		}

		data = append(data, row)
	}
	for i := 0; i < height-len(networks); i++ {
		data = append(data, []string{""})
	}
	return data
}

func FormatArrays[ArrType KnownNetwork | ScannedNetwork](arr []ArrType, selectedIndex int, windowSize int) []ArrType {
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
	totalWidth := WindowDimensions().Width
	line := strings.Split(s, "\n")[0]

	// Use lipgloss.Width to correctly calculate visible width, ignoring ANSI codes
	textWidth := lipgloss.Width(line)

	// Calculate padding and ensure it's not negative
	return max(0, (totalWidth-textWidth)/2)
}

func SanitizeSSID(s, replacement string) string {
	// Unicode regex range for emojis — covers most common sets (Emoticons, Misc Symbols, Transport, etc.)
	re := regexp.MustCompile(`[\p{So}\p{Sk}\p{Cs}\x{1F000}-\x{1FAFF}\x{2600}-\x{27BF}\x{1F300}-\x{1F6FF}]+`)
	return re.ReplaceAllString(s, replacement)
}

func SortDevicesBySignal(devices []ScannedNetwork) {
	slices.SortFunc(devices, func(a, b ScannedNetwork) int {
		// Primary sort: Signal descending (higher is better)
		if a.Signal > b.Signal {
			return -1
		}
		if a.Signal < b.Signal {
			return 1
		}
		// Secondary sort: SSID ascending (case-insensitive)
		return strings.Compare(strings.ToLower(a.SSID), strings.ToLower(b.SSID))
	})
}
