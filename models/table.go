package models

import (
	"netpala/common"
	"netpala/config"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss/table"
)

type TableData struct {
	title           string
	isTableSelected bool
	selectedRow     int
	height          int

	deviceData      []common.Device
	stationData     []common.Device
	vpnData         []common.VpnConnection
	knownNetworks   []common.KnownNetwork
	scannedNetworks []common.ScannedNetwork
	
	colors          config.Colors
}

func TableModel(
	title string,
	isTableSelected bool,
	selectedRow int,
	height int,

	knownNets []common.KnownNetwork,
	scannedNets []common.ScannedNetwork,
	vpnData []common.VpnConnection,
	devData []common.Device,
	
	colors config.Colors,
) TableData {
	return TableData{
		title:           title,
		isTableSelected: isTableSelected,
		selectedRow:     selectedRow,
		height:          height,

		deviceData:      devData,
		vpnData:         vpnData,
		knownNetworks:   knownNets,
		scannedNetworks: scannedNets,
		
		colors:          colors,
	}
}

func (m TableData) Init() tea.Cmd {
	return nil
}

func (m TableData) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m TableData) View() string {
	borderStyle := common.InactiveBorderStyle(m.colors.Primary)
	if m.isTableSelected {
		borderStyle = common.ActiveBorderStyle(m.colors.Active)
	}

	var tableData [][]string
	if m.deviceData != nil {
		tableData = common.FormatDeviceData(m.deviceData)
	} else if m.stationData != nil {
		tableData = common.FormatStationData(m.stationData)
	} else if m.vpnData != nil {
		tableData = common.FormatVpnData(m.vpnData)
	} else if m.knownNetworks != nil {
		tableData = common.FormatKnownNetworksData(m.knownNetworks, m.selectedRow, m.height)
	} else {
		tableData = common.FormatScannedNetworksData(m.scannedNetworks, m.selectedRow, m.height)
	}

	table := table.New().
		Border(common.BoxBorder).
		BorderColumn(false).
		BorderStyle(borderStyle).
		StyleFunc(common.BoxStyle(m.selectedRow, m.isTableSelected, m.height, m.colors.Primary, m.colors.ActiveText, m.colors.Inactive, m.colors.SelectionBg)).
		Rows(tableData...)

	return (common.CalcTitle(m.title, m.isTableSelected, m.colors.Primary, m.colors.Active) + table.Render()) + "\n"
}