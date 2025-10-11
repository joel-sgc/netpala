package main

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/godbus/dbus/v5"
)

// This command waits for a single signal from the provided channel
// and translates it into a BubbleTea message.
func waitForDBusSignal(conn *dbus.Conn, sig chan *dbus.Signal) tea.Cmd {
	return func() tea.Msg {
		s := <-sig // Block and wait for the next signal

		switch s.Name {
		case "org.freedesktop.DBus.Properties.PropertiesChanged",
			"org.freedesktop.NetworkManager.DeviceAdded",
			"org.freedesktop.NetworkManager.DeviceRemoved",
			"org.freedesktop.NetworkManager.Device.StateChanged":
			// **FIX:** The values in a BatchMsg must be tea.Cmds (functions),
			// not tea.Msgs (data). We wrap our calls in functions.
			return tea.BatchMsg{
				func() tea.Msg { return deviceUpdateMsg(get_devices_data(conn)) },
				func() tea.Msg { return knownNetworksUpdateMsg(get_known_networks(conn)) },
			}

		case "org.freedesktop.NetworkManager.Settings.NewConnection",
			"org.freedesktop.NetworkManager.Settings.ConnectionRemoved":
			return knownNetworksUpdateMsg(get_known_networks(conn))

		case "org.freedesktop.NetworkManager.Device.Wireless.AccessPointAdded",
			"org.freedesktop.NetworkManager.Device.Wireless.AccessPointRemoved":
			return scannedNetworksUpdateMsg(nil)
		}
		// If we get a signal we don't care about, listen again immediately.
		return waitForDBusSignal(conn, sig)()
	}
}

// Command to periodically trigger a full data refresh.
func refreshTicker() tea.Cmd {
	return tea.Tick(15*time.Second, func(t time.Time) tea.Msg {
		return periodicRefreshMsg{}
	})
}

func requestScan(conn *dbus.Conn) tea.Cmd {
	return func() tea.Msg {
		nm := conn.Object(nmDest, dbus.ObjectPath(nmPath))
		var devPaths []dbus.ObjectPath
		if err := nm.Call(nmDest+".GetDevices", 0).Store(&devPaths); err != nil {
			return errMsg{err}
		}

		for _, devPath := range devPaths {
			devObj := conn.Object(nmDest, devPath)
			devProps := get_props(devObj, devIF)
			if devProps["DeviceType"].Value().(uint32) == 2 { // WiFi device
				_ = devObj.Call(wifiIF+".RequestScan", 0, map[string]dbus.Variant{})
			}
		}
		return scannedNetworksUpdateMsg(get_scanned_networks(conn))
	}
}

func refreshAllData(conn *dbus.Conn) tea.Cmd {
	return tea.Batch(
		requestScan(conn),
		func() tea.Msg { return deviceUpdateMsg(get_devices_data(conn)) },
		func() tea.Msg { return knownNetworksUpdateMsg(get_known_networks(conn)) },
	)
}