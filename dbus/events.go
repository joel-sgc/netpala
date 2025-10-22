package dbus

import (
	"netpala/common"
	"netpala/network"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/godbus/dbus/v5"
)

// This command waits for a single signal from the provided channel
// and translates it into a BubbleTea message.
func WaitForDBusSignal(conn *dbus.Conn, sig chan *dbus.Signal) tea.Cmd {
	return func() tea.Msg {
		s := <-sig

		switch s.Name {
		case "org.freedesktop.DBus.Properties.PropertiesChanged":
			if s.Path == network.NMPath {
				// Handle property changes on the main NetworkManager object
				// e.g., WirelessEnabled, VpnEnabled
				return tea.BatchMsg{
					func() tea.Msg { return common.DeviceUpdateMsg(network.GetDevicesData(conn)) },
					func() tea.Msg { return common.VpnUpdateMsg(network.GetVpnData(conn)) },
				}
			}
			// Check if it's a device property change, which can indicate VPN state changes.
			if len(s.Body) > 0 {
				if iface, ok := s.Body[0].(string); ok && iface == network.DevIF {
					return tea.BatchMsg{
						func() tea.Msg { return common.DeviceUpdateMsg(network.GetDevicesData(conn)) },
						func() tea.Msg { return common.VpnUpdateMsg(network.GetVpnData(conn)) },
					}
				}
			}
			// Fallthrough to wait for the next relevant signal if this one wasn't for us.

		case "org.freedesktop.NetworkManager.DeviceAdded",
			"org.freedesktop.NetworkManager.DeviceRemoved",
			"org.freedesktop.NetworkManager.Device.StateChanged":
			// Device state changes can affect connectivity.
			return tea.BatchMsg{
				func() tea.Msg { return common.DeviceUpdateMsg(network.GetDevicesData(conn)) },
				func() tea.Msg { return common.KnownNetworksUpdateMsg(network.GetKnownNetworks(conn)) },
			}

		case "org.freedesktop.NetworkManager.Settings.NewConnection",
			"org.freedesktop.NetworkManager.Settings.ConnectionRemoved":
			// A new VPN profile is a new connection, so we refresh here too.
			return tea.BatchMsg{
				func() tea.Msg { return common.KnownNetworksUpdateMsg(network.GetKnownNetworks(conn)) },
				func() tea.Msg { return common.VpnUpdateMsg(network.GetVpnData(conn)) },
			}

		case "org.freedesktop.NetworkManager.Device.Wireless.AccessPointAdded",
			"org.freedesktop.NetworkManager.Device.Wireless.AccessPointRemoved":
			return common.ScannedNetworksUpdateMsg(nil)
		}
		return WaitForDBusSignal(conn, sig)()
	}
}

// Command to periodically trigger a full data refresh.
func RefreshTicker() tea.Cmd {
	return tea.Tick(15*time.Second, func(t time.Time) tea.Msg {
		return common.PeriodicRefreshMsg{}
	})
}

func RefreshDevices(conn *dbus.Conn) tea.Cmd {
	return func() tea.Msg {
		return common.DeviceUpdateMsg(network.GetDevicesData(conn))
	}
}

func RequestScan(conn *dbus.Conn) tea.Cmd {
	return func() tea.Msg {
		nm := conn.Object(network.NMDest, dbus.ObjectPath(network.NMPath))
		var devPaths []dbus.ObjectPath
		if err := nm.Call(network.NMDest+".GetDevices", 0).Store(&devPaths); err != nil {
			return common.ErrMsg{Err: err}
		}

		for _, devPath := range devPaths {
			devObj := conn.Object(network.NMDest, devPath)
			devProps := network.GetProps(devObj, network.DevIF)
			if devProps != nil {
				if deviceType, ok := devProps["DeviceType"].Value().(uint32); ok && deviceType == 2 { // WiFi device
					_ = devObj.Call(network.WifiIF+".RequestScan", 0, map[string]dbus.Variant{})
				}
			}
		}

		// Instead of returning data, return a non-blocking timer (Tick)
		// that will send a message to refresh the device status
		// after 250ms, giving NM time to update.
		return tea.Tick(250*time.Millisecond, func(t time.Time) tea.Msg {
			return common.RefreshDeviceStatusMsg{}
		})()
	}
}

func GetScanResults(conn *dbus.Conn) tea.Cmd {
	return func() tea.Msg {
		nm := conn.Object(network.NMDest, dbus.ObjectPath(network.NMPath))
		var devPaths []dbus.ObjectPath
		if err := nm.Call(network.NMDest+".GetDevices", 0).Store(&devPaths); err != nil {
			return common.ErrMsg{Err: err}
		}

		for _, devPath := range devPaths {
			devObj := conn.Object(network.NMDest, devPath)
			devProps := network.GetProps(devObj, network.DevIF)
			if devProps["DeviceType"].Value().(uint32) == 2 { // WiFi device
				_ = devObj.Call(network.WifiIF+".RequestScan", 0, map[string]dbus.Variant{})
			}
		}
		
		return common.ScannedNetworksUpdateMsg(network.GetScannedNetworks(conn))
	}
}

func RefreshAllData(conn *dbus.Conn) tea.Cmd {
	return tea.Batch(
		GetScanResults(conn),
		func() tea.Msg { return common.DeviceUpdateMsg(network.GetDevicesData(conn)) },
		func() tea.Msg { return common.KnownNetworksUpdateMsg(network.GetKnownNetworks(conn)) },
		func() tea.Msg { return common.VpnUpdateMsg(network.GetVpnData(conn)) },
	)
}
