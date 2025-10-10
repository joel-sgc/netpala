package main

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/godbus/dbus/v5"
)

// The initial command to load all data at startup.
func loadInitialData(conn *dbus.Conn) tea.Cmd {
	return tea.Batch(
		func() tea.Msg { return deviceUpdateMsg(get_devices_data(conn)) },
		func() tea.Msg { return knownNetworksUpdateMsg(get_known_networks(conn)) },
		func() tea.Msg { return scannedNetworksUpdateMsg(get_scanned_networks(conn)) },
	)
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
		known_networks:   []known_network{},
		scanned_networks: []scanned_network{},
		status_bar: 			StatusBarModel(),
		is_typing: false,
	}
}

func (m netpala_data) Init() tea.Cmd {
	if m.err != nil {
		return nil
	}
	return tea.Batch(
		loadInitialData(m.conn),
		refreshTicker(),
		// Start the first listener.
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

	case knownNetworksUpdateMsg:
		m.known_networks = msg
		return m, waitForDBusSignal(m.conn, m.dbusSignals)

	case scannedNetworksUpdateMsg:
		if msg == nil {
			debounceCmd := tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
				return performScanRefreshMsg{}
			})
			return m, tea.Batch(waitForDBusSignal(m.conn, m.dbusSignals), debounceCmd)
		}

		// NEW: Filter out known networks from the scanned list
		knownSSIDs := make(map[string]struct{})
		for _, known := range m.known_networks {
			knownSSIDs[known.ssid] = struct{}{}
		}

		var filteredNetworks []scanned_network
		for _, scanned := range msg {
			if _, exists := knownSSIDs[scanned.ssid]; !exists {
				filteredNetworks = append(filteredNetworks, scanned)
			}
		}
		m.scanned_networks = filteredNetworks
		// END NEW

		return m, nil

	case performScanRefreshMsg:
		return m, requestScan(m.conn)

	case errMsg:
		m.err = msg.err
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case periodicRefreshMsg:
		return m, refreshAllData(m.conn)

	case tea.KeyMsg:
		switch msg.String() {
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
			boxes := []int{len(m.device_data), len(m.device_data), len(m.known_networks), len(m.scanned_networks)}
			if m.selected_box < len(boxes) && m.selected_entry < boxes[m.selected_box]-1 && !m.is_typing {
				m.selected_entry++
			}

		case "shift+tab":
			if m.selected_box > 0 {
				m.selected_box--
				m.selected_entry = 0
			}
		case "tab":
			if m.selected_box < 3 {
				m.selected_box++
				m.selected_entry = 0
			}
		case "enter", " ":
			if m.selected_box == 2 && len(m.known_networks) > 0 && len(m.device_data) > 0 {
				selectedNetwork := m.known_networks[m.selected_entry]
				wifiDevice := m.device_data[0]
				return m, connectToNetworkCmd(m.conn, selectedNetwork.path, wifiDevice.path)

			} else if m.selected_box == 3 && len(m.scanned_networks) > 0 && len(m.device_data) > 0 {
				// Store the selected network before entering typing mode
				m.network_to_connect = m.scanned_networks[m.selected_entry]
				m.is_typing = true
				m.status_bar.input.Placeholder = "Enter Password for " + m.network_to_connect.ssid
				m.status_bar.input.Focus()
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

	device_table := TableModel("Device", m.selected_box == 0, m.selected_entry, m.device_data, nil, nil, nil)
	station_table := TableModel("Station", m.selected_box == 1, m.selected_entry, nil, m.device_data, nil, nil)
	known_nets_table := TableModel("Known Networks", m.selected_box == 2, m.selected_entry, nil, nil, m.known_networks, nil)
	scanned_nets_table := TableModel("New Networks", m.selected_box == 3, m.selected_entry, nil, nil, nil, m.scanned_networks)
	
	return device_table.View() + station_table.View() + known_nets_table.View() + scanned_nets_table.View() + m.status_bar.View()
}

func main() {
	p := tea.NewProgram(NetpalaModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
