package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/godbus/dbus/v5"
	"github.com/google/uuid"
)

// connectToNetworkCmd tells NetworkManager to activate a connection on a specific device.
func connectToNetworkCmd(conn *dbus.Conn, connectionPath, devicePath dbus.ObjectPath) tea.Cmd {
	return func() tea.Msg {
		nm := conn.Object(nmDest, dbus.ObjectPath(nmPath))

		// The D-Bus method call to activate the connection.
		// The final argument is for a "specific object" (like a particular AP),
		// which we leave as "/" to let NetworkManager decide.
		call := nm.Call(
			"org.freedesktop.NetworkManager.ActivateConnection",
			0,
			connectionPath,
			devicePath,
			dbus.ObjectPath("/"),
		)

		if call.Err != nil {
			return errMsg{fmt.Errorf("failed to activate connection: %w", call.Err)}
		}

		// We don't need to return a success message. If the call succeeds,
		// our D-Bus signal listener will automatically pick up the state
		// change and refresh the UI for us.
		return nil
	}
}

func addAndConnectToNetworkCmd(conn *dbus.Conn, network scanned_network, password string, devicePath dbus.ObjectPath) tea.Cmd {
	return func() tea.Msg {
		// 1. Generate the connection settings map
		newUUID, err := uuid.NewRandom()
		if err != nil {
			return errMsg{fmt.Errorf("failed to generate uuid: %w", err)}
		}

		// Base connection settings
		settings := map[string]map[string]dbus.Variant{
			"connection": {
				"id":          dbus.MakeVariant(network.ssid),
				"uuid":        dbus.MakeVariant(newUUID.String()),
				"type":        dbus.MakeVariant("802-11-wireless"),
				"autoconnect": dbus.MakeVariant(true),
			},
			"802-11-wireless": {
				"ssid":     dbus.MakeVariant([]byte(network.ssid)),
				"mode":     dbus.MakeVariant("infrastructure"),
				"security": dbus.MakeVariant("802-11-wireless-security"),
			},
			"ipv4": {"method": dbus.MakeVariant("auto")},
			"ipv6": {"method": dbus.MakeVariant("auto")},
		}

		// Add security-specific settings
		securitySettings := make(map[string]dbus.Variant)
		// NOTE: This logic might need to be expanded for more complex security types
		// like WPA-EAP, but it covers the common WPA2/WPA3 cases.
		switch network.security {
		case "wpa3-sae":
			securitySettings["key-mgmt"] = dbus.MakeVariant("sae")
			securitySettings["psk"] = dbus.MakeVariant(password)
		case "wpa2-psk":
			securitySettings["key-mgmt"] = dbus.MakeVariant("wpa-psk")
			securitySettings["psk"] = dbus.MakeVariant(password)
		default: // Assuming WPA2/WPA3 for anything encrypted that isn't SAE
			if network.security != "open" {
				securitySettings["key-mgmt"] = dbus.MakeVariant("wpa-psk")
				securitySettings["psk"] = dbus.MakeVariant(password)
			}
		}
		if len(securitySettings) > 0 {
			settings["802-11-wireless-security"] = securitySettings
		}

		// 2. Add the connection via D-Bus
		settingsObj := conn.Object(nmDest, "/org/freedesktop/NetworkManager/Settings")
		call := settingsObj.Call("org.freedesktop.NetworkManager.Settings.AddConnection", 0, settings)
		if call.Err != nil {
			return errMsg{fmt.Errorf("failed to add connection: %w", call.Err)}
		}

		// 3. Get the path of the newly created connection
		var newConnectionPath dbus.ObjectPath
		if err := call.Store(&newConnectionPath); err != nil {
			return errMsg{fmt.Errorf("could not read new connection path: %w", err)}
		}

		// 4. Activate the new connection
		// We can reuse the same logic as our other connect command.
		return connectToNetworkCmd(conn, newConnectionPath, devicePath)()
	}
}

func toggleWifiCmd(conn *dbus.Conn, enable bool) tea.Cmd {
	return func() tea.Msg {
		nm := conn.Object(nmDest, dbus.ObjectPath(nmPath))

		// Call the Set method on the standard D-Bus Properties interface
		call := nm.Call(
			"org.freedesktop.DBus.Properties.Set",
			0,
			nmDest,                   // The interface that owns the property
			"WirelessEnabled",        // The property to change
			dbus.MakeVariant(enable), // The new value (true for on, false for off)
		)

		if call.Err != nil {
			return errMsg{fmt.Errorf("failed to set WirelessEnabled property: %w", call.Err)}
		}

		// A successful call will automatically trigger a D-Bus signal.
		// Our existing signal listener will then refresh the device data for us.
		return nil
	}
}
