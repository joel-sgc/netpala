package models

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// TablesModel is a container model that holds all the main tables.
type TablesModel struct {
	// We'll populate these fields from the main model just before rendering.
	SelectedBox     int
	SelectedEntry   int
	NetsHeight      int
	DeviceData      []Device
	VpnData         []VpnConnection
	KnownNetworks   []KnownNetwork
	ScannedNetworks []ScannedNetwork
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
	deviceTable := TableModel("Device", m.SelectedBox == 0, m.SelectedEntry, -1, m.DeviceData, nil, nil, nil, nil)
	stationTable := TableModel("Station", m.SelectedBox == 1, m.SelectedEntry, -1, nil, m.DeviceData, nil, nil, nil)
	vpnTableModel := TableModel("Virtual Private Networks", m.SelectedBox == 2, m.SelectedEntry, -1, nil, nil, m.VpnData, nil, nil)
	knownNetsTable := TableModel("Known Networks", m.SelectedBox == 3, m.SelectedEntry, m.NetsHeight, nil, nil, nil, m.KnownNetworks, nil)
	scannedNetsTable := TableModel("New Networks", m.SelectedBox == 4, m.SelectedEntry, m.NetsHeight, nil, nil, nil, nil, m.ScannedNetworks)

	vpnView := vpnTableModel.View()
	if len(m.VpnData) == 0 {
		vpnView = ""
	}

	return strings.Join([]string{
		deviceTable.View(),
		stationTable.View(),
		vpnView,
		knownNetsTable.View(),
		scannedNetsTable.View(),
	}, "")
}