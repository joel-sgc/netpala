package dbus

import (
	"fmt"
	"strings"

	"netpala/common"
	"netpala/network"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/godbus/dbus/v5"
	"github.com/google/uuid"
)

// connectToNetworkCmd tells NetworkManager to activate a connection on a specific device.
func ConnectToNetworkCmd(conn *dbus.Conn, connectionPath, devicePath dbus.ObjectPath) tea.Cmd {
	return func() tea.Msg {
		nm := conn.Object(network.NMDest, dbus.ObjectPath(network.NMPath))

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
			return common.ErrMsg{Err: fmt.Errorf("failed to activate connection: %w", call.Err)}
		}

		// We don't need to return a success message. If the call succeeds,
		// our D-Bus signal listener will automatically pick up the state
		// change and refresh the UI for us.
		return nil
	}
}

func AddAndConnectToNetworkCmd(conn *dbus.Conn, net common.ScannedNetwork, password string, devicePath dbus.ObjectPath) tea.Cmd {
	return func() tea.Msg {
		// 1. Generate the connection settings map
		newUUID, err := uuid.NewRandom()
		if err != nil {
			return common.ErrMsg{Err: fmt.Errorf("failed to generate uuid: %w", err)}
		}

		// Base connection settings
		settings := map[string]map[string]dbus.Variant{
			"connection": {
				"id":          dbus.MakeVariant(net.SSID),
				"uuid":        dbus.MakeVariant(newUUID.String()),
				"type":        dbus.MakeVariant("802-11-wireless"),
				"autoconnect": dbus.MakeVariant(true),
			},
			"802-11-wireless": {
				"ssid":     dbus.MakeVariant([]byte(net.SSID)),
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
		switch net.Security {
		case "wpa3-sae":
			securitySettings["key-mgmt"] = dbus.MakeVariant("sae")
			securitySettings["psk"] = dbus.MakeVariant(password)
		case "wpa2-psk":
			securitySettings["key-mgmt"] = dbus.MakeVariant("wpa-psk")
			securitySettings["psk"] = dbus.MakeVariant(password)
		default: // Assuming WPA2/WPA3 for anything encrypted that isn't SAE
			if net.Security != "open" {
				securitySettings["key-mgmt"] = dbus.MakeVariant("wpa-psk")
				securitySettings["psk"] = dbus.MakeVariant(password)
			}
		}
		if len(securitySettings) > 0 {
			settings["802-11-wireless-security"] = securitySettings
		}

		// 2. Add the connection via D-Bus
		settingsObj := conn.Object(network.NMDest, "/org/freedesktop/NetworkManager/Settings")
		call := settingsObj.Call("org.freedesktop.NetworkManager.Settings.AddConnection", 0, settings)
		if call.Err != nil {
			return common.ErrMsg{Err: fmt.Errorf("failed to add connection: %w", call.Err)}
		}

		// 3. Get the path of the newly created connection
		var newConnectionPath dbus.ObjectPath
		if err := call.Store(&newConnectionPath); err != nil {
			return common.ErrMsg{Err: fmt.Errorf("could not read new connection path: %w", err)}
		}

		// 4. Activate the new connection
		// We can reuse the same logic as our other connect command.
		return ConnectToNetworkCmd(conn, newConnectionPath, devicePath)()
	}
}

func AddAndConnectEAPCmd(conn *dbus.Conn, config map[string]string, devicePath dbus.ObjectPath) tea.Cmd {
	return func() tea.Msg {
		// 1. Validate required fields
		ssid, ok := config["ssid"]
		if !ok || ssid == "" {
			return common.ErrMsg{Err: fmt.Errorf("EAP config is missing SSID")}
		}
		eapMethod, ok := config["eap"]
		if !ok || eapMethod == "" {
			return common.ErrMsg{Err: fmt.Errorf("EAP config is missing EAP method")}
		}
		identity, ok := config["identity"]
		if !ok || identity == "" {
			return common.ErrMsg{Err: fmt.Errorf("EAP config is missing identity")}
		}

		// 2. Generate a new UUID
		newUUID, err := uuid.NewRandom()
		if err != nil {
			return common.ErrMsg{Err: fmt.Errorf("failed to generate UUID for EAP: %w", err)}
		}

		// 3. Build the 802-1x (EAP) settings
		eapSettings := map[string]dbus.Variant{
			"eap":      dbus.MakeVariant([]string{strings.ToLower(eapMethod)}),
			"identity": dbus.MakeVariant(identity),
			"password": dbus.MakeVariant(config["password"]),
		}

		// Only add phase2-auth if it's not "NONE"
		if phase2, ok := config["phase2-auth"]; ok && phase2 != "" && phase2 != "NONE" {
			eapSettings["phase2-auth"] = dbus.MakeVariant(strings.ToLower(phase2))
		}

		// **Handle the certificate path**
		// If the user provided a path, add it.
		// If the path is empty, we add nothing, which NM interprets
		// as "do not use/validate a CA certificate."
		if certPath, ok := config["ca_cert"]; ok && certPath != "" {
			// Prepend "file://" to the path, which is required by NM
			eapSettings["ca-cert"] = dbus.MakeVariant("file://" + certPath)
		}

		// 4. Build the complete connection settings map
		settings := map[string]map[string]dbus.Variant{
			"connection": {
				"id":         dbus.MakeVariant(ssid),
				"uuid":       dbus.MakeVariant(newUUID.String()),
				"type":       dbus.MakeVariant("802-11-wireless"),
				"autoconnect": dbus.MakeVariant(true),
			},
			"802-11-wireless": {
				"ssid":     dbus.MakeVariant([]byte(ssid)),
				"mode":     dbus.MakeVariant("infrastructure"),
				"security": dbus.MakeVariant("802-11-wireless-security"),
			},
			"802-11-wireless-security": {
				"key-mgmt": dbus.MakeVariant("wpa-eap"),
			},
			"802-1x": eapSettings, // Add our EAP config
			"ipv4":   {"method": dbus.MakeVariant("auto")},
			"ipv6":   {"method": dbus.MakeVariant("auto")},
		}

		// 5. Add the connection via D-Bus
		settingsObj := conn.Object(network.NMDest, "/org/freedesktop/NetworkManager/Settings")
		call := settingsObj.Call("org.freedesktop.NetworkManager.Settings.AddConnection", 0, settings)
		if call.Err != nil {
			return common.ErrMsg{Err: fmt.Errorf("failed to add EAP connection: %w", call.Err)}
		}

		// 6. Get the path of the newly created connection
		var newConnectionPath dbus.ObjectPath
		if err := call.Store(&newConnectionPath); err != nil {
			return common.ErrMsg{Err: fmt.Errorf("could not read new EAP connection path: %w", err)}
		}

		// 7. Activate the new connection using our existing command
		return ConnectToNetworkCmd(conn, newConnectionPath, devicePath)()
	}
}

func ToggleVpnCmd(conn *dbus.Conn, vpnConnectionPath dbus.ObjectPath, active bool) tea.Cmd {
	return func() tea.Msg {
		nm := conn.Object(network.NMDest, dbus.ObjectPath(network.NMPath))

		var call *dbus.Call
		if active {
			// Deactivate the active VPN connection
			activeConnObj := conn.Object(network.NMDest, vpnConnectionPath)
			call = activeConnObj.Call("org.freedesktop.NetworkManager.Connection.Active.Deactivate", 0)
		} else {
			// Activate the selected VPN connection
			call = nm.Call(
				"org.freedesktop.NetworkManager.ActivateConnection",
				0,
				vpnConnectionPath,
				dbus.ObjectPath("/"), // device path is not needed for VPN
				dbus.ObjectPath("/"),
			)
		}

		if call.Err != nil {
			action := "activate"
			if active {
				action = "deactivate"
			}
			return common.ErrMsg{Err: fmt.Errorf("failed to %s vpn connection: %w", action, call.Err)}
		}

		return nil
	}
}

func ToggleWifiCmd(conn *dbus.Conn, enable bool) tea.Cmd {
	return func() tea.Msg {
		nm := conn.Object(network.NMDest, dbus.ObjectPath(network.NMPath))

		// Call the Set method on the standard D-Bus Properties interface
		call := nm.Call(
			"org.freedesktop.DBus.Properties.Set",
			0,
			network.NMDest,           // The interface that owns the property
			"WirelessEnabled",        // The property to change
			dbus.MakeVariant(enable), // The new value (true for on, false for off)
		)

		if call.Err != nil {
			return common.ErrMsg{Err: fmt.Errorf("failed to set WirelessEnabled property: %w", call.Err)}
		}

		// A successful call will automatically trigger a D-Bus signal.
		// Our existing signal listener will then refresh the device data for us.
		return nil
	}
}
