package main

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss/table"
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
	if m.err != nil {
		if key, ok := msg.(tea.KeyMsg); ok {
			switch key.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			}
		}
		return m, nil
	}

	switch msg := msg.(type) {
	case deviceUpdateMsg:
		m.device_data = msg
		// After the model is updated, we can listen for the next signal.
		return m, waitForDBusSignal(m.conn, m.dbusSignals)

	case knownNetworksUpdateMsg:
		m.known_networks = msg
		// After the model is updated, we can listen for the next signal.
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
		// No need to re-arm listener here, as it's handled by the debounce logic.
		return m, nil

	case performScanRefreshMsg:
		// The debounce timer fired, now perform the scan.
		// The listener is already running from the previous step.
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

		case "up", "k":
			if m.selected_entry > 0 {
				m.selected_entry--
			}
		case "down", "j":
			boxes := []int{len(m.device_data), len(m.device_data), len(m.known_networks), len(m.scanned_networks)}
			if m.selected_box < len(boxes) && m.selected_entry < boxes[m.selected_box]-1 {
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
			// Check that we're in the "Known Networks" box and have networks to select from.
			if m.selected_box == 2 && len(m.known_networks) > 0 && len(m.device_data) > 0 {
				selectedNetwork := m.known_networks[m.selected_entry]
				
				// For simplicity, we'll use the first available Wi-Fi device.
				wifiDevice := m.device_data[0]

				// Return the command to connect!
				return m, connectToNetworkCmd(m.conn, selectedNetwork.path, wifiDevice.path)
			}
		}		
	}
	return m, nil
}

func (m netpala_data) View() string {
	if m.err != nil {
		return fmt.Sprintf("\nAn error occurred: %v\n\nPress 'q' to quit.", m.err)
	}

	border_style_device := inactive_border_style
	border_style_station := inactive_border_style
	border_style_known_networks := inactive_border_style
	border_style_scanned_networks := inactive_border_style

	switch m.selected_box {
	case 0:
		border_style_device = active_border_style
	case 1:
		border_style_station = active_border_style
	case 2:
		border_style_known_networks = active_border_style
	case 3:
		border_style_scanned_networks = active_border_style
	}

	device_table_data := format_device_data(m.device_data)
	device_table := table.New().
		Border(box_border).
		BorderColumn(false).
		BorderStyle(border_style_device).
		StyleFunc(box_style(m.selected_entry, m.selected_box == 0)).
		Rows(device_table_data...)

	station_table_data := format_station_data(m.device_data)
	station_table := table.New().
		Border(box_border).
		BorderColumn(false).
		BorderStyle(border_style_station).
		StyleFunc(box_style(m.selected_entry, m.selected_box == 1)).
		Rows(station_table_data...)

	known_networks_table_data := format_known_networks_data(m.known_networks, m.selected_entry)
	known_networks_table := table.New().
		Border(box_border).
		BorderColumn(false).
		BorderStyle(border_style_known_networks).
		StyleFunc(box_style(m.selected_entry, m.selected_box == 2)).
		Rows(known_networks_table_data...)

	scanned_networks_table_data := format_scanned_networks_data(m.scanned_networks, m.selected_entry)
	scanned_networks_table := table.New().
		Border(box_border).
		BorderColumn(false).
		BorderStyle(border_style_scanned_networks).
		StyleFunc(box_style(m.selected_entry, m.selected_box == 3)).
		Rows(scanned_networks_table_data...)

	return (calc_title("Device", m.selected_box == 0) + device_table.Render()) + "\n" +
		(calc_title("Station", m.selected_box == 1) + station_table.Render()) + "\n" +
		(calc_title("Known Networks", m.selected_box == 2) + known_networks_table.Render()) + "\n" +
		(calc_title("New Networks", m.selected_box == 3) + scanned_networks_table.Render())
}

func main() {
	p := tea.NewProgram(NetpalaModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}