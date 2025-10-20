package models

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss/table"
)

type TableData struct {
	title           string
	isTableSelected bool
	selectedRow     int
	height          int
	deviceData      []Device
	stationData     []Device
	vpnData         []VpnConnection
	knownNetworks   []KnownNetwork
	scannedNetworks []ScannedNetwork
}

func TableModel(
	title string,
	isTableSelected bool,
	selectedRow int,
	height int,
	devData []Device,
	stationData []Device,
	vpnData []VpnConnection,
	knownNets []KnownNetwork,
	scannedNets []ScannedNetwork,
) TableData {
	return TableData{
		title:           title,
		isTableSelected: isTableSelected,
		selectedRow:     selectedRow,
		height:          height,
		deviceData:      devData,
		stationData:     stationData,
		vpnData:         vpnData,
		knownNetworks:   knownNets,
		scannedNetworks: scannedNets,
	}
}

func (m TableData) Init() tea.Cmd {
	return nil
}

func (m TableData) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m TableData) View() string {
	borderStyle := inactiveBorderStyle
	if m.isTableSelected {
		borderStyle = activeBorderStyle
	}

	var tableData [][]string
	if m.deviceData != nil {
		tableData = formatDeviceData(m.deviceData)
	} else if m.stationData != nil {
		tableData = formatStationData(m.stationData)
	} else if m.vpnData != nil {
		tableData = formatVpnData(m.vpnData)
	} else if m.knownNetworks != nil {
		tableData = formatKnownNetworksData(m.knownNetworks, m.selectedRow, m.height)
	} else {
		tableData = formatScannedNetworksData(m.scannedNetworks, m.selectedRow, m.height)
	}

	table := table.New().
		Border(boxBorder).
		BorderColumn(false).
		BorderStyle(borderStyle).
		StyleFunc(boxStyle(m.selectedRow, m.isTableSelected)).
		Rows(tableData...)

	return (calcTitle(m.title, m.isTableSelected) + table.Render()) + "\n"
}