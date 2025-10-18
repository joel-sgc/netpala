package main

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/godbus/dbus/v5"
	overlay "github.com/rmhubbert/bubbletea-overlay"
)

// The initial command to load all data at startup.
func loadInitialData(conn *dbus.Conn) tea.Cmd {
	return func() tea.Msg {
		// Step 1: Fetch all data first to ensure we have both lists.
		devices := get_devices_data(conn)
		vpns := get_vpn_data(conn)
		known := get_known_networks(conn)
		scanned := get_scanned_networks(conn)

		// Step 2: Perform the filtering logic on the initial data.
		knownSSIDs := make(map[string]struct{})
		for _, k := range known {
			knownSSIDs[k.ssid] = struct{}{}
		}

		var filteredScanned []scanned_network
		for _, s := range scanned {
			if _, exists := knownSSIDs[s.ssid]; !exists {
				filteredScanned = append(filteredScanned, s)
			}
		}

		// Step 3: Send the final, filtered data to the UI in a single batch.
		return tea.BatchMsg{
			func() tea.Msg { return deviceUpdateMsg(devices) },
			func() tea.Msg { return knownNetworksUpdateMsg(known) },
			func() tea.Msg { return scannedNetworksUpdateMsg(filteredScanned) },
			func() tea.Msg { return vpnUpdateMsg(vpns) },
		}
	}
}

func NetpalaModel() netpala_data {
	conn, err := dbus.SystemBus()
	if err != nil {
		return netpala_data{err: fmt.Errorf("failed to connect to D-Bus: %w", err)}
	}

	sigChan := make(chan *dbus.Signal, 10)
	conn.Signal(sigChan)

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
	busObject := conn.BusObject()
	for _, rule := range rules {
		call := busObject.Call("org.freedesktop.DBus.AddMatch", 0, rule)
		if call.Err != nil {
			err = fmt.Errorf("could not add match rule '%s': %w", rule, call.Err)
			break
		}
	}

	return netpala_data{
		conn:             conn,
		err:              err,
		dbusSignals:      sigChan,
		
		device_data:      []device{},
		vpn_data: 				[]vpn_connection{},
		known_networks:   []known_network{},
		scanned_networks: []scanned_network{},
		status_bar: 			StatusBarModel(),
		form:  						WpaEapForm(),
		tables:           tables_model{},

		is_typing: false,
		is_in_form: false,
		initial_load_complete: false,
	}
}

func (m netpala_data) Init() tea.Cmd {
	if m.err != nil {
		return nil
	}
	
	return tea.Batch(
		loadInitialData(m.conn),
		refreshTicker(),
		waitForDBusSignal(m.conn, m.dbusSignals),
	)
}

func (m netpala_data) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	
	if m.err != nil {
		if key, ok := msg.(tea.KeyMsg); ok {
			switch key.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			}
		}
		return m, nil
	}

	if (m.is_in_form) {
		var newForm tea.Model
		newForm, cmd = m.form.Update(msg)
		m.form = newForm.(wpa_eap_form)
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.is_typing = false
			m.status_bar.input.Placeholder = ""
			m.status_bar.input.Blur()
			m.status_bar.input.SetValue("")
			return m, tea.Quit
		}
	}
	
	if m.is_typing {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+c", "esc":
				m.is_typing = false
				m.status_bar.input.Placeholder = ""
				m.status_bar.input.Blur()
				m.status_bar.input.SetValue("")
				return m, nil

			case "enter":
				password := m.status_bar.input.Value()
				m.is_typing = false
				m.status_bar.input.Placeholder = ""
				m.status_bar.input.Blur()
				m.status_bar.input.SetValue("")

				if len(m.device_data) == 0 {
					return m, func() tea.Msg {
						return errMsg{fmt.Errorf("no wifi device found")}
					}
				}
				wifiDevice := m.device_data[0]

				// Use the stored network to connect, not the current selection
				return m, addAndConnectToNetworkCmd(m.conn, m.network_to_connect, password, wifiDevice.path)
			}
		}

		m.status_bar.input, cmd = m.status_bar.input.Update(msg)
		return m, cmd
	}

	switch msg := msg.(type) {
	case deviceUpdateMsg:
		m.device_data = msg
		return m, waitForDBusSignal(m.conn, m.dbusSignals)

	case vpnUpdateMsg:
		m.vpn_data = msg
		return m, waitForDBusSignal(m.conn, m.dbusSignals)

	case knownNetworksUpdateMsg:
		m.filterKnownFromScanned()
		m.known_networks = msg
		return m, waitForDBusSignal(m.conn, m.dbusSignals)

	case scannedNetworksUpdateMsg:
		// The `nil` message is the trigger from the listener.
		if msg == nil {
			debounceCmd := tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
				return performScanRefreshMsg{}
			})
			// Re-arm the main listener right away, but start the debounce timer.
			return m, tea.Batch(waitForDBusSignal(m.conn, m.dbusSignals), debounceCmd)
		}
		// This is the actual data from a completed scan.
		m.scanned_networks = msg
		m.filterKnownFromScanned()
		// No need to re-arm listener here, as it's handled by the debounce logic.
		return m, nil

	case performScanRefreshMsg:
		// The debounce timer fired, now perform the scan.
		return m, requestScan(m.conn)

	case errMsg:
		m.err = msg.err
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		var formCmd tea.Cmd
		var newForm tea.Model
		newForm, formCmd = m.form.Update(msg)
		m.form = newForm.(wpa_eap_form)
		return m, formCmd

	case periodicRefreshMsg:
		return m, refreshAllData(m.conn)

	case tea.KeyMsg:
		switch msg.String() {
		// case "e":
		// 	return m, openEditorCmd()
		case "ctrl+c", "ctrl+q", "q", "ctrl+w":
			m.conn.RemoveSignal(m.dbusSignals)
			m.conn.Close()
			return m, tea.Quit

		case "r":
			return m, requestScan(m.conn)

		case "up", "k":
			if m.selected_entry > 0 && !m.is_typing {
				m.selected_entry--
			}
		case "down", "j":
			boxes := []int{len(m.device_data), len(m.device_data), len(m.vpn_data), len(m.known_networks), len(m.scanned_networks)}
			if m.selected_box < len(boxes) && m.selected_entry < boxes[m.selected_box]-1 && !m.is_typing {
				m.selected_entry++
			}

		case "shift+tab":
			if m.selected_box > 0 {
				m.selected_box--
				m.selected_entry = 0
			}

			if len(m.vpn_data) == 0 && m.selected_box == 2 {
				m.selected_box--
			}
		case "tab":
			if m.selected_box < 4 {
				m.selected_box++
				m.selected_entry = 0
			}

			if len(m.vpn_data) == 0 && m.selected_box == 2 {
				m.selected_box++
			}
		case "enter", " ":
			if (m.selected_box == 0 && len(m.device_data) > 0) {
				// Enable/Disable Wifi Card
				return m, toggleWifiCmd(m.conn, !m.device_data[0].powered)
			} else if m.selected_box == 2 && len(m.vpn_data) > 0 && len(m.device_data) > 0 {
				// Connect to VPN
				
			} else if m.selected_box == 3 && len(m.known_networks) > 0 && len(m.device_data) > 0 {
				// Connect to known network
				selectedNetwork := m.known_networks[m.selected_entry]
				wifiDevice := m.device_data[0]
				return m, connectToNetworkCmd(m.conn, selectedNetwork.path, wifiDevice.path)				
			} else if m.selected_box == 4 && len(m.scanned_networks) > 0 && len(m.device_data) > 0 {
				// Store the selected network before entering typing mode
				m.network_to_connect = m.scanned_networks[m.selected_entry]

				if (m.network_to_connect.security == "wpa2-eap") {
					m.is_in_form = true
					return m, nil
				} else {
					m.is_typing = true
					m.status_bar.input.Placeholder = "Enter Wi-Fi Password..."
					m.status_bar.input.Focus()
				}
				return m, nil
			}
		}
	}
	return m, nil
}


func (m netpala_data) View() string {
	if m.err != nil {
		return fmt.Sprintf("\nAn error occurred: %v\n\nPress 'q' to quit.", m.err)
	}

	var nets_height int = 10
	if len(m.vpn_data) > 0 {
		nets_height = 8
	}

	m.tables.selected_box = m.selected_box
	m.tables.selected_entry = m.selected_entry
	m.tables.nets_height = nets_height
	m.tables.device_data = m.device_data
	m.tables.vpn_data = m.vpn_data
	m.tables.known_networks = m.known_networks
	m.tables.scanned_networks = m.scanned_networks

	if (m.is_in_form) {
		bgModel := &m.tables
		fgModel := &m.form
		xPosition := overlay.Left
		yPosition := overlay.Center
		xOffset := calculate_padding(m.form.View())
		yOffset := 0
		
		overlayModel := overlay.New(fgModel, bgModel, xPosition, yPosition, xOffset, yOffset)
		
		return overlayModel.View()
	} else {	
		return m.tables.View() + m.status_bar.View()
	}
}

func main() {
	p := tea.NewProgram(NetpalaModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}