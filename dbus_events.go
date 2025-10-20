package main

import (
	"netpala/models"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/godbus/dbus/v5"
)

// This command waits for a single signal from the provided channel
// and translates it into a BubbleTea message.
func waitForDBusSignal(conn *dbus.Conn, sig chan *dbus.Signal) tea.Cmd {
	return func() tea.Msg {
		s := <-sig

		switch s.Name {
		case "org.freedesktop.DBus.Properties.PropertiesChanged",
			"org.freedesktop.NetworkManager.DeviceAdded",
			"org.freedesktop.NetworkManager.DeviceRemoved",
			"org.freedesktop.NetworkManager.Device.StateChanged":
			// VPN status changes are broadcast as property changes, so we refresh here.
			return tea.BatchMsg{
				func() tea.Msg { return models.DeviceUpdateMsg(getDevicesData(conn)) },
				func() tea.Msg { return models.KnownNetworksUpdateMsg(getKnownNetworks(conn)) },
				func() tea.Msg { return models.VpnUpdateMsg(getVpnData(conn)) }, // <-- ADDED
			}

		case "org.freedesktop.NetworkManager.Settings.NewConnection",
			"org.freedesktop.NetworkManager.Settings.ConnectionRemoved":
			// A new VPN profile is a new connection, so we refresh here too.
			return tea.BatchMsg{
				func() tea.Msg { return models.KnownNetworksUpdateMsg(getKnownNetworks(conn)) },
				func() tea.Msg { return models.VpnUpdateMsg(getVpnData(conn)) }, // <-- ADDED
			}

		case "org.freedesktop.NetworkManager.Device.Wireless.AccessPointAdded",
			"org.freedesktop.NetworkManager.Device.Wireless.AccessPointRemoved":
			return models.ScannedNetworksUpdateMsg(nil)
		}
		return waitForDBusSignal(conn, sig)()
	}
}

// Command to periodically trigger a full data refresh.
func refreshTicker() tea.Cmd {
	return tea.Tick(15*time.Second, func(t time.Time) tea.Msg {
		return models.PeriodicRefreshMsg{}
	})
}

func requestScan(conn *dbus.Conn) tea.Cmd {
	return func() tea.Msg {
		nm := conn.Object(nmDest, dbus.ObjectPath(nmPath))
		var devPaths []dbus.ObjectPath
		if err := nm.Call(nmDest+".GetDevices", 0).Store(&devPaths); err != nil {
			return models.ErrMsg{Err: err}
		}

		for _, devPath := range devPaths {
			devObj := conn.Object(nmDest, devPath)
			devProps := getProps(devObj, devIF)
			if devProps["DeviceType"].Value().(uint32) == 2 { // WiFi device
				_ = devObj.Call(wifiIF+".RequestScan", 0, map[string]dbus.Variant{})
			}
		}
		return models.ScannedNetworksUpdateMsg(getScannedNetworks(conn))
	}
}

func refreshAllData(conn *dbus.Conn) tea.Cmd {
	return tea.Batch(
		requestScan(conn),
		func() tea.Msg { return models.DeviceUpdateMsg(getDevicesData(conn)) },
		func() tea.Msg { return models.KnownNetworksUpdateMsg(getKnownNetworks(conn)) },
		func() tea.Msg { return models.VpnUpdateMsg(getVpnData(conn)) }, // <-- ADDED
	)
}