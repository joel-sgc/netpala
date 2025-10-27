package main

import (
	"fmt"
	"netpala/common"
	"netpala/dbus"
	"netpala/models"
	"netpala/network"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	godbus "github.com/godbus/dbus/v5"
	overlay "github.com/rmhubbert/bubbletea-overlay"
)

type NetpalaData struct {
	Width, Height int
	selectedBox   int
	SelectedEntry int

	DeviceData      []common.Device
	VpnData         []common.VpnConnection
	KnownNetworks   []common.KnownNetwork
	ScannedNetworks []common.ScannedNetwork
	
	Tables          models.TablesModel
	StatusBar       models.StatusBarData
	
	Form           	models.WpaEapForm
	Overlay        	overlay.Model
	Confirmation   	models.Confirmation

	SelectedNetwork	common.ScannedNetwork
	IsTyping       	bool
	PopupState     	int	// -1: no popup, 0: form, 1: confirm

	InitialLoadComplete bool
	Conn                *godbus.Conn
	Err                 error
	DBusSignals         chan *godbus.Signal
}

// The initial command to load all data at startup.
func loadInitialData(Conn *godbus.Conn) tea.Cmd {
	return func() tea.Msg {
		// Step 1: Fetch all data first to ensure we have both lists.
		devices := network.GetDevicesData(Conn)
		vpns := network.GetVpnData(Conn)
		known := network.GetKnownNetworks(Conn)
		scanned := network.GetScannedNetworks(Conn)

		// Step 2: Perform the filtering logic on the initial data.
		knownSSIDs := make(map[string]struct{})
		for _, k := range known {
			knownSSIDs[k.SSID] = struct{}{}
		}

		var filteredScanned []common.ScannedNetwork
		for _, s := range scanned {
			if _, exists := knownSSIDs[s.SSID]; !exists {
				filteredScanned = append(filteredScanned, s)
			}
		}

		// Step 3: Send the final, filtered data to the UI in a single batch.
		return tea.BatchMsg{
			func() tea.Msg { return common.DeviceUpdateMsg(devices) },
			func() tea.Msg { return common.KnownNetworksUpdateMsg(known) },
			func() tea.Msg { return common.ScannedNetworksUpdateMsg(filteredScanned) },
			func() tea.Msg { return common.VpnUpdateMsg(vpns) },
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
	var filteredScanned []common.ScannedNetwork
	for _, scanned := range m.ScannedNetworks {
		if _, exists := knownSSIDs[scanned.SSID]; !exists {
			filteredScanned = append(filteredScanned, scanned)
		}
	}
	m.ScannedNetworks = filteredScanned
}

func NetpalaModel() NetpalaData {
	Conn, err := godbus.SystemBus()
	if err != nil {
		return NetpalaData{Err: fmt.Errorf("failed to Connect to D-Bus: %w", err)}
	}

	sigChan := make(chan *godbus.Signal, 10)
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

		DeviceData:      []common.Device{},
		VpnData:         []common.VpnConnection{},
		KnownNetworks:   []common.KnownNetwork{},
		ScannedNetworks: []common.ScannedNetwork{},
		
		Tables:          models.TablesModel{},
		StatusBar:       models.ModelStatusBar(),
		
		Form:            models.ModelWpaEapForm(),
		Overlay:         overlay.Model{
			XPosition: overlay.Left,
			YPosition: overlay.Center,
			XOffset:   0,
			YOffset:   0,
		},

		IsTyping:            false,
		PopupState:          -1,
		InitialLoadComplete: false,
	}
}

func (m NetpalaData) Init() tea.Cmd {
	if m.Err != nil {
		return nil
	}

	return tea.Batch(
		loadInitialData(m.Conn),
		dbus.RefreshTicker(),
		dbus.WaitForDBusSignal(m.Conn, m.DBusSignals),
	)
}

func (m NetpalaData) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.PopupState {
	case 0:
		// Handle the EAP form popup state
		switch msg := msg.(type) {
		case common.ExitFormMsg:
			m.PopupState = -1
			m.Form = models.ModelWpaEapForm()

			var formCmd tea.Cmd
			var newForm tea.Model
			newForm, formCmd = m.Form.Update(tea.WindowSizeMsg{Width: m.Width, Height: m.Height})
			m.Form = newForm.(models.WpaEapForm)
			return m, formCmd
		case common.SubmitEapFormMsg:
			m.SelectedEntry = 0
			m.PopupState = -1
			m.Form = models.ModelWpaEapForm()

			// Re-initialize the new form with the window size
			var formCmd tea.Cmd
			var newForm tea.Model
			newForm, formCmd = m.Form.Update(tea.WindowSizeMsg{Width: m.Width, Height: m.Height})
			m.Form = newForm.(models.WpaEapForm)

			// Get the Wi-Fi device to connect with
			if len(m.DeviceData) == 0 {
				return m, func() tea.Msg {
					return common.ErrMsg{Err: fmt.Errorf("no wifi device found to connect with")}
				}
			}
			wifiDevice := m.DeviceData[0]

			// Add the EAP connection config from the message
			// and combine it with the form's init command.
			eapCmd := dbus.AddAndConnectEAPCmd(m.Conn, msg.Config, wifiDevice.Path)
			return m, tea.Batch(formCmd, eapCmd)
		}	

		var newForm tea.Model
		newForm, cmd = m.Form.Update(msg)
		m.Form = newForm.(models.WpaEapForm)
		return m, cmd
	case 1:
		// Handle the confirmation popup state
		switch msg := msg.(type) {
		case common.SubmitConfirmationMsg:
			m.PopupState = -1 // Exit popup
			m.Confirmation = models.ModelConfirmation() // Reset

			if msg.Value { // User confirmed
				// Delete the known network
				// NOTE: Ensure m.SelectedNetwork holds the correct data before entering state 1
				deleteCmd := dbus.DeleteConnectionCmd(m.Conn, m.SelectedNetwork.Path)
				// Return delete command AND re-arm listener
				return m, tea.Batch(deleteCmd, dbus.WaitForDBusSignal(m.Conn, m.DBusSignals))
			} else { // User cancelled
				// Just return and re-arm listener
				return m, dbus.WaitForDBusSignal(m.Conn, m.DBusSignals)
			}

		default: // If it's not a SubmitConfirmationMsg...
			// Forward the *original* message down to the confirmation model
			var newConfirmation tea.Model
			newConfirmation, cmd = m.Confirmation.Update(msg)
			m.Confirmation = newConfirmation.(models.Confirmation)
			// Return the confirmation model and any command it produced
			return m, cmd
		}
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
						return common.ErrMsg{Err: fmt.Errorf("no wifi device found")}
					}
				}
				wifiDevice := m.DeviceData[0]

				// Use the stored network to Connect, not the current selection
				return m, dbus.AddAndConnectToNetworkCmd(m.Conn, m.SelectedNetwork, password, wifiDevice.Path)
			}
		}

		m.StatusBar.Input, cmd = m.StatusBar.Input.Update(msg)
		return m, cmd
	}

	switch msg := msg.(type) {
	case common.DeviceUpdateMsg:
		m.DeviceData = msg
		return m, dbus.WaitForDBusSignal(m.Conn, m.DBusSignals)

	case common.VpnUpdateMsg:
		m.VpnData = msg

	case common.KnownNetworksUpdateMsg:
		m.FilterKnownFromScanned()
		m.KnownNetworks = msg

		return m, dbus.WaitForDBusSignal(m.Conn, m.DBusSignals)

	case common.ScannedNetworksUpdateMsg:
		// The `nil` message is the trigger from the listener.
		if msg == nil {
			debounceCmd := tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
				return common.PerformScanRefreshMsg{}
			})
			// Re-arm the main listener right away, but start the debounce timer.
			return m, tea.Batch(dbus.WaitForDBusSignal(m.Conn, m.DBusSignals), debounceCmd)
		}
		// This is the actual data from a completed scan.
		m.ScannedNetworks = msg
		m.FilterKnownFromScanned()

		// No need to re-arm listener here, as it's handled by the debounce logic.
		return m, nil

	case common.PerformScanRefreshMsg:
		// The debounce timer fired, now perform the scan.
		return m, dbus.GetScanResults(m.Conn)

	case common.ErrMsg:
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

	case common.PeriodicRefreshMsg:
		return m, dbus.RefreshAllData(m.Conn)

	case tea.KeyMsg:
		switch msg.String() {
		// case "e":
		case "ctrl+c", "ctrl+q", "q", "ctrl+w":
			m.Conn.RemoveSignal(m.DBusSignals)
			m.Conn.Close()
			return m, tea.Quit

	case "r":
		var cmds []tea.Cmd
		cmds = append(cmds, dbus.RequestScan(m.Conn))
		cmds = append(cmds, func() tea.Msg {
			return common.KnownNetworksUpdateMsg(network.GetKnownNetworks(m.Conn))
		})

		return m, tea.Batch(cmds...)

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
				return m, dbus.ToggleWifiCmd(m.Conn, !m.DeviceData[0].Powered)
			} else if m.selectedBox == 2 && len(m.VpnData) > 0 && len(m.DeviceData) > 0 {
				// Toggle VPN
				selectedVpn := m.VpnData[m.SelectedEntry]
				return m, dbus.ToggleVpnCmd(m.Conn, selectedVpn.Path, selectedVpn.ActivePath, !selectedVpn.Connected)
			} else if m.selectedBox == 3 && len(m.KnownNetworks) > 0 && len(m.DeviceData) > 0 {
				// Connect to known network
				selectedNetwork := m.KnownNetworks[m.SelectedEntry]
				wifiDevice := m.DeviceData[0]
				return m, dbus.ConnectToNetworkCmd(m.Conn, selectedNetwork.Path, wifiDevice.Path)
			} else if m.selectedBox == 4 && len(m.ScannedNetworks) > 0 && len(m.DeviceData) > 0 {
				// Store the selected network before entering typing mode
				m.SelectedNetwork = m.ScannedNetworks[m.SelectedEntry]

				switch m.SelectedNetwork.Security {
				case "wpa2-eap":
					m.Form.SSIDSelected = m.SelectedNetwork.SSID
					m.PopupState = 0

					m.Overlay = updateOverlayModel(m, &m.Form)
					return m, nil
				case "open":
					// Open network, connect directly
					wifiDevice := m.DeviceData[0]
					return m, dbus.AddAndConnectToNetworkCmd(m.Conn, m.SelectedNetwork, "", wifiDevice.Path)
				default:
					// Most common case: prompt for password
					m.IsTyping = true
					m.StatusBar.Input.Placeholder = "Enter Wi-Fi Password..."
					m.StatusBar.Input.Focus()
				}
				return m, nil
			}
		case "delete":
			if !m.IsTyping && m.selectedBox == 3 && len(m.KnownNetworks) > 0 {
				// Delete known network
				m.SelectedNetwork = common.ScannedNetwork{
					Path: m.KnownNetworks[m.SelectedEntry].Path,
					SSID: m.KnownNetworks[m.SelectedEntry].SSID,
					BSSID: m.KnownNetworks[m.SelectedEntry].BSSID,
					Security: m.KnownNetworks[m.SelectedEntry].Security,
					Signal: m.KnownNetworks[m.SelectedEntry].Signal,
				}
				m.PopupState = 1
				m.Confirmation.Message = fmt.Sprintf("Are you sure you want to delete the known network '%s'?\n", m.SelectedNetwork.SSID)

				m.Overlay = updateOverlayModel(m, &m.Confirmation)
				return m, nil
			}
		}
	}
	return m, nil
}

func (m NetpalaData) View() string {
	if m.Err != nil {
		return fmt.Sprintf("An error occurred: %v\n\nPress 'q' to quit.", m.Err)
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

	switch m.PopupState {
	case 0:
		m.Overlay = updateOverlayModel(m, &m.Form)
		return m.Overlay.View() + m.StatusBar.View()
	case 1:
		m.Overlay = updateOverlayModel(m, &m.Confirmation)
		return m.Overlay.View() + m.StatusBar.View()
	default:
		return m.Tables.View() + m.StatusBar.View()
	}
}

func main() {
	p := tea.NewProgram(NetpalaModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		os.Exit(1)
		// tea.NewProgram(models.ModelError(err), tea.WithAltScreen()).Run()
	}
}

func updateOverlayModel(m NetpalaData, popup tea.Model) overlay.Model {
	newOverlay := overlay.Model{
		Background: &m.Tables,
		Foreground: popup,
		XPosition:  overlay.Left,
		YPosition:  overlay.Center,
		XOffset:    common.CalculatePadding(popup.View()),
		YOffset:    0,
	}

	newOverlay.Update(tea.WindowSizeMsg{Width: m.Width, Height: m.Height})
	return newOverlay
}