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
		s := <-sig // Block for next signal

		switch s.Name {
		case "org.freedesktop.DBus.Properties.PropertiesChanged":
			// Body[0] is the interface name whose properties changed.
			if len(s.Body) > 0 {
				if iface, ok := s.Body[0].(string); ok {
					// --- THIS IS THE FIX ---
					// Check if properties changed on the main NM object OR a Device object.
					// This ensures we catch WirelessEnabled changes AND device state/scanning changes.
					if iface == network.NMDest || iface == network.DevIF || iface == network.WifiIF {
						// Refresh devices and potentially VPNs (as device state affects VPN)
						return tea.BatchMsg{
							func() tea.Msg { return common.DeviceUpdateMsg(network.GetDevicesData(conn)) },
							func() tea.Msg { return common.VpnUpdateMsg(network.GetVpnData(conn)) }, // VPN status might depend on device state
						}
					}
					// --- END FIX ---
				}
			}
			// If it wasn't a property change we care about, listen again.

		case "org.freedesktop.NetworkManager.DeviceAdded",
			"org.freedesktop.NetworkManager.DeviceRemoved",
			"org.freedesktop.NetworkManager.Device.StateChanged":
			// Device state changes definitely affect connectivity. Refresh relevant lists.
			return tea.BatchMsg{
				func() tea.Msg { return common.DeviceUpdateMsg(network.GetDevicesData(conn)) },
				func() tea.Msg { return common.KnownNetworksUpdateMsg(network.GetKnownNetworks(conn)) },
				func() tea.Msg { return common.VpnUpdateMsg(network.GetVpnData(conn)) }, // VPN status might depend on device state
			}

		case "org.freedesktop.NetworkManager.Settings.NewConnection",
			"org.freedesktop.NetworkManager.Settings.ConnectionRemoved":
			// Adding/Removing connections affects Known Networks and VPN lists.
			return tea.BatchMsg{
				func() tea.Msg { return common.KnownNetworksUpdateMsg(network.GetKnownNetworks(conn)) },
				func() tea.Msg { return common.VpnUpdateMsg(network.GetVpnData(conn)) },
			}

		case "org.freedesktop.NetworkManager.Device.Wireless.AccessPointAdded",
			"org.freedesktop.NetworkManager.Device.Wireless.AccessPointRemoved":
			// Signals that scan results *might* have changed. Trigger debounce.
			return common.ScannedNetworksUpdateMsg(nil)
		}

		// If we fall through, it was a signal we don't handle. Listen again.
		return WaitForDBusSignal(conn, sig)()
	}
}

// Command to periodically trigger a full data refresh.
func RefreshTicker() tea.Cmd {
	return tea.Tick(15*time.Second, func(t time.Time) tea.Msg {
		return common.PeriodicRefreshMsg{}
	})
}

func RequestScan(conn *dbus.Conn) tea.Cmd {
	return func() tea.Msg {
		nm := conn.Object(network.NMDest, dbus.ObjectPath(network.NMPath))
		var devPaths []dbus.ObjectPath
		// Perform the D-Bus call, return error if it fails
		if err := nm.Call(network.NMDest+".GetDevices", 0).Store(&devPaths); err != nil {
			return common.ErrMsg{Err: err}
		}

		// Trigger scan on WiFi devices
		scanRequested := false
		for _, devPath := range devPaths {
			devObj := conn.Object(network.NMDest, devPath)
			devProps := network.GetProps(devObj, network.DevIF) // Use GetProps from network package
			if devProps != nil {
				if deviceType, ok := devProps["DeviceType"].Value().(uint32); ok && deviceType == 2 {
					_ = devObj.Call(network.WifiIF+".RequestScan", 0, map[string]dbus.Variant{})
					scanRequested = true // Mark that we asked at least one device
				}
			}
		}

		if !scanRequested {
			// Optional: Inform user if no WiFi device was found to scan
			// return common.ErrMsg{Err: fmt.Errorf("no wifi device found to scan")}
		}

		// Success: Scan requested, return nil. The signal listener will handle updates.
		return nil
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
