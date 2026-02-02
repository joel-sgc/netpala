package models

import (
	"netpala/common"
	"netpala/config"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// TablesModel is a container model that holds all the main tables.
type TablesModel struct {
	SelectedBox   int
	SelectedEntry int
	KnownHeight   int
	ScannedHeight int
	VPNHeight     int
	DeviceHeight  int

	DeviceData      []common.Device
	VpnData         []common.VpnConnection
	KnownNetworks   []common.KnownNetwork
	ScannedNetworks []common.ScannedNetwork
	
	Colors config.Colors
}

func (m TablesModel) Init() tea.Cmd {
	return nil
}

func (m TablesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// This model is for viewing only; all updates are handled by NetpalaData
	return m, nil
}

// View renders all tables in order.
func (m TablesModel) View() string {
	knownNetsTable := TableModel("Known Networks", m.SelectedBox == 0, m.SelectedEntry, m.KnownHeight, m.KnownNetworks, nil, nil, nil, m.Colors)
	scannedNetsTable := TableModel("New Networks", m.SelectedBox == 1, m.SelectedEntry, m.ScannedHeight, nil, m.ScannedNetworks, nil, nil, m.Colors)
	vpnTableModel := TableModel("Virtual Private Networks", m.SelectedBox == 2, m.SelectedEntry, m.VPNHeight, nil, nil, m.VpnData, nil, m.Colors)
	deviceTable := TableModel("Device", m.SelectedBox == 3, m.SelectedEntry, m.DeviceHeight, nil, nil, nil, m.DeviceData, m.Colors)

	vpnView := vpnTableModel.View()
	if len(m.VpnData) == 0 {
		vpnView = ""
	}

	return strings.Join([]string{
		knownNetsTable.View(),
		scannedNetsTable.View(),
		vpnView,
		deviceTable.View(),
	}, "")
}
