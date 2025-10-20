package main

import (
	"fmt"
	"netpala/models"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/godbus/dbus/v5"
	overlay "github.com/rmhubbert/bubbletea-overlay"
)

type NetpalaData struct {
	Width, Height int
	selectedBox  	int
	SelectedEntry	int

	DeviceData      []models.Device
	VpnData         []models.VpnConnection
	KnownNetworks   []models.KnownNetwork
	ScannedNetworks []models.ScannedNetwork
	Tables          models.TablesModel
	StatusBar       models.StatusBarData
	Form            models.WpaEapForm

	NetworkToConnect models.ScannedNetwork
	IsTyping         bool
	IsInForm         bool

	InitialLoadComplete bool
	Conn                *dbus.Conn
	Err                 error
	DBusSignals         chan *dbus.Signal
}

// The initial command to load all data at startup.
func loadInitialData(Conn *dbus.Conn) tea.Cmd {
	return func() tea.Msg {
		// Step 1: Fetch all data first to ensure we have both lists.
		devices := getDevicesData(Conn)
		vpns := getVpnData(Conn)
		known := getKnownNetworks(Conn)
		scanned := getScannedNetworks(Conn)

		// Step 2: Perform the filtering logic on the initial data.
		knownSSIDs := make(map[string]struct{})
		for _, k := range known {
			knownSSIDs[k.SSID] = struct{}{}
		}

		var filteredScanned []models.ScannedNetwork
		for _, s := range scanned {
			if _, exists := knownSSIDs[s.SSID]; !exists {
				filteredScanned = append(filteredScanned, s)
			}
		}

		// Step 3: Send the final, filtered data to the UI in a single batch.
		return tea.BatchMsg{
			func() tea.Msg { return models.DeviceUpdateMsg(devices) },
			func() tea.Msg { return models.KnownNetworksUpdateMsg(known) },
			func() tea.Msg { return models.ScannedNetworksUpdateMsg(filteredScanned) },
			func() tea.Msg { return models.VpnUpdateMsg(vpns) },
		}
	}
}

func (m *NetpalaData) FilterKnownFromScanned() {
	// Create a map of known SSIDs for fast lookups (more efficient than a nested loop)
	knownSSIDs := make(map[string]struct{})
	for _, known := range m.KnownNetworks {
		knownSSIDs[known.SSID] = struct{}{}
	}

	// Build a new slice containing only the networks we want to keep.
	// This is more efficient than deleting elements from the slice in-place.
	var filteredScanned []models.ScannedNetwork
	for _, scanned := range m.ScannedNetworks {
		if _, exists := knownSSIDs[scanned.SSID]; !exists {
			filteredScanned = append(filteredScanned, scanned)
		}
	}
	m.ScannedNetworks = filteredScanned
}

func NetpalaModel() NetpalaData {
	Conn, err := dbus.SystemBus()
	if err != nil {
		return NetpalaData{Err: fmt.Errorf("failed to Connect to D-Bus: %w", err)}
	}

	sigChan := make(chan *dbus.Signal, 10)
	Conn.Signal(sigChan)

	rules := []string{
		"type='signal',interface='org.freedesktop.DBus.Properties',member='PropertiesChanged'",
		"type='signal',interface='org.freedesktop.NetworkManager.Device',member='StateChanged'",
		"type='signal',interface='org.freedesktop.NetworkManager',member='DeviceAdded'",
		"type='signal',interface='org.freedesktop.NetworkManager',member='DeviceRemoved'",
		"type='signal',interface='org.freedesktop.NetworkManager.Settings',member='NewConnection'",
		"type='signal',interface='org.freedesktop.NetworkManager.Settings',member='ConnectionRemoved'",
		"type='signal',interface='org.freedesktop.NetworkManager.Device.Wireless',member='AccessPointAdded'",
		"type='signal',interface='org.freedesktop.NetworkManager.Device.Wireless',member='AccessPointRemoved'",
	}
	busObject := Conn.BusObject()
	for _, rule := range rules {
		call := busObject.Call("org.freedesktop.DBus.AddMatch", 0, rule)
		if call.Err != nil {
			err = fmt.Errorf("could not add match rule '%s': %w", rule, call.Err)
			break
		}
	}

	return NetpalaData{
		Conn:        Conn,
		Err:         err,
		DBusSignals: sigChan,

		DeviceData:      []models.Device{},
		VpnData:         []models.VpnConnection{},
		KnownNetworks:   []models.KnownNetwork{},
		ScannedNetworks: []models.ScannedNetwork{},
		StatusBar:       models.ModelStatusBar(),
		Form:            models.ModelWpaEapForm(),
		Tables:          models.TablesModel{},

		IsTyping:             false,
		IsInForm:             false,
		InitialLoadComplete: false,
	}
}

func (m NetpalaData) Init() tea.Cmd {
	if m.Err != nil {
		return nil
	}

	return tea.Batch(
		loadInitialData(m.Conn),
		refreshTicker(),
		waitForDBusSignal(m.Conn, m.DBusSignals),
	)
}

func (m NetpalaData) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if m.Err != nil {
		if key, ok := msg.(tea.KeyMsg); ok {
			switch key.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			}
		}
		return m, nil
	}

	if m.IsInForm {
		var newForm tea.Model
		newForm, cmd = m.Form.Update(msg)
		m.Form = newForm.(models.WpaEapForm)
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.IsTyping = false
			m.StatusBar.Input.Placeholder = ""
			m.StatusBar.Input.Blur()
			m.StatusBar.Input.SetValue("")
			return m, tea.Quit
		}
	}

	if m.IsTyping {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+c", "esc":
				m.IsTyping = false
				m.StatusBar.Input.Placeholder = ""
				m.StatusBar.Input.Blur()
				m.StatusBar.Input.SetValue("")
				return m, nil

			case "enter":
				password := m.StatusBar.Input.Value()
				m.IsTyping = false
				m.StatusBar.Input.Placeholder = ""
				m.StatusBar.Input.Blur()
				m.StatusBar.Input.SetValue("")

				if len(m.DeviceData) == 0 {
					return m, func() tea.Msg {
						return models.ErrMsg{Err: fmt.Errorf("no wifi device found")}
					}
				}
				wifiDevice := m.DeviceData[0]

				// Use the stored network to Connect, not the current selection
				return m, AddAndConnectToNetworkCmd(m.Conn, m.NetworkToConnect, password, wifiDevice.Path)
			}
		}

		m.StatusBar.Input, cmd = m.StatusBar.Input.Update(msg)
		return m, cmd
	}

	switch msg := msg.(type) {
	case models.DeviceUpdateMsg:
		m.DeviceData = msg
		return m, waitForDBusSignal(m.Conn, m.DBusSignals)

	case models.VpnUpdateMsg:
		m.VpnData = msg
		return m, waitForDBusSignal(m.Conn, m.DBusSignals)

	case models.KnownNetworksUpdateMsg:
		m.FilterKnownFromScanned()
		m.KnownNetworks = msg
		return m, waitForDBusSignal(m.Conn, m.DBusSignals)

	case models.ScannedNetworksUpdateMsg:
		// The `nil` message is the trigger from the listener.
		if msg == nil {
			debounceCmd := tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
				return models.PerformScanRefreshMsg{}
			})
			// Re-arm the main listener right away, but start the debounce timer.
			return m, tea.Batch(waitForDBusSignal(m.Conn, m.DBusSignals), debounceCmd)
		}
		// This is the actual data from a completed scan.
		m.ScannedNetworks = msg
		m.FilterKnownFromScanned()
		// No need to re-arm listener here, as it's handled by the debounce logic.
		return m, nil

	case models.PerformScanRefreshMsg:
		// The debounce timer fired, now perform the scan.
		return m, requestScan(m.Conn)

	case models.ErrMsg: 
		m.Err = msg.Err
		return m, nil

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height

		var formCmd tea.Cmd
		var newForm tea.Model
		newForm, formCmd = m.Form.Update(msg)
		m.Form = newForm.(models.WpaEapForm)
		return m, formCmd

	case models.PeriodicRefreshMsg:
		return m, refreshAllData(m.Conn)

	case tea.KeyMsg:
		switch msg.String() {
		// case "e":
		//  return m, openEditorCmd()
		case "ctrl+c", "ctrl+q", "q", "ctrl+w":
			m.Conn.RemoveSignal(m.DBusSignals)
			m.Conn.Close()
			return m, tea.Quit

		case "r":
			return m, requestScan(m.Conn)

		case "up", "k":
			if m.SelectedEntry > 0 && !m.IsTyping {
				m.SelectedEntry--
			}
		case "down", "j":
			boxes := []int{len(m.DeviceData), len(m.DeviceData), len(m.VpnData), len(m.KnownNetworks), len(m.ScannedNetworks)}
			if m.selectedBox < len(boxes) && m.SelectedEntry < boxes[m.selectedBox]-1 && !m.IsTyping {
				m.SelectedEntry++
			}

		case "shift+tab":
			if m.selectedBox > 0 {
				m.selectedBox--
				m.SelectedEntry = 0
			}

			if len(m.VpnData) == 0 && m.selectedBox == 2 {
				m.selectedBox--
			}
		case "tab":
			if m.selectedBox < 4 {
				m.selectedBox++
				m.SelectedEntry = 0
			}

			if len(m.VpnData) == 0 && m.selectedBox == 2 {
				m.selectedBox++
			}
		case "enter", " ":
			if m.selectedBox == 0 && len(m.DeviceData) > 0 {
				// Enable/Disable Wifi Card
				return m, ToggleWifiCmd(m.Conn, !m.DeviceData[0].Powered)
			} else if m.selectedBox == 2 && len(m.VpnData) > 0 && len(m.DeviceData) > 0 {
				// Connect to VPN

			} else if m.selectedBox == 3 && len(m.KnownNetworks) > 0 && len(m.DeviceData) > 0 {
				// Connect to known network
				selectedNetwork := m.KnownNetworks[m.SelectedEntry]
				wifiDevice := m.DeviceData[0]
				return m, ConnectToNetworkCmd(m.Conn, selectedNetwork.Path, wifiDevice.Path)
			} else if m.selectedBox == 4 && len(m.ScannedNetworks) > 0 && len(m.DeviceData) > 0 {
				// Store the selected network before entering typing mode
				m.NetworkToConnect = m.ScannedNetworks[m.SelectedEntry]

				if m.NetworkToConnect.Security == "wpa2-eap" {
					m.IsInForm = true
					return m, nil
				} else {
					m.IsTyping = true
					m.StatusBar.Input.Placeholder = "Enter Wi-Fi Password..."
					m.StatusBar.Input.Focus()
				}
				return m, nil
			}
		}
	}
	return m, nil
}

func (m NetpalaData) View() string {
	if m.Err != nil {
		return fmt.Sprintf("\nAn error occurred: %v\n\nPress 'q' to quit.", m.Err)
	}

	var netsHeight int = 10
	if len(m.VpnData) > 0 {
		netsHeight = 8
	}

	m.Tables.SelectedBox = m.selectedBox
	m.Tables.SelectedEntry = m.SelectedEntry
	m.Tables.NetsHeight = netsHeight
	m.Tables.DeviceData = m.DeviceData
	m.Tables.VpnData = m.VpnData
	m.Tables.KnownNetworks = m.KnownNetworks
	m.Tables.ScannedNetworks = m.ScannedNetworks

	if m.IsInForm {
		bgModel := &m.Tables
		fgModel := &m.Form
		xPosition := overlay.Left
		yPosition := overlay.Center
		xOffset := models.CalculatePadding(m.Form.View())
		yOffset := 0

		overlayModel := overlay.New(fgModel, bgModel, xPosition, yPosition, xOffset, yOffset)

		return overlayModel.View()
	} else {
		return m.Tables.View() + m.StatusBar.View()
	}
}

func main() {
	p := tea.NewProgram(NetpalaModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}