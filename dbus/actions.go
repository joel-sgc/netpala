package dbus

import (
	"fmt"
	"strings"
	"time" // Added time import

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
		// Success is handled by signal listener
		return nil
	}
}

// AddAndConnectToNetworkCmd adds a standard network and attempts connection.
func AddAndConnectToNetworkCmd(conn *dbus.Conn, net common.ScannedNetwork, password string, devicePath dbus.ObjectPath) tea.Cmd {
	return func() tea.Msg {
		// 1. Generate UUID
		newUUID, err := uuid.NewRandom()
		if err != nil {
			return common.ErrMsg{Err: fmt.Errorf("failed to generate uuid: %w", err)}
		}

		// 2. Build settings map
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
		securitySettings := make(map[string]dbus.Variant)
		switch net.Security {
		case "wpa3-sae":
			securitySettings["key-mgmt"] = dbus.MakeVariant("sae")
			securitySettings["psk"] = dbus.MakeVariant(password)
		case "wpa2-psk":
			securitySettings["key-mgmt"] = dbus.MakeVariant("wpa-psk")
			securitySettings["psk"] = dbus.MakeVariant(password)
		default:
			if net.Security != "open" {
				securitySettings["key-mgmt"] = dbus.MakeVariant("wpa-psk")
				securitySettings["psk"] = dbus.MakeVariant(password)
			}
		}
		if len(securitySettings) > 0 {
			settings["802-11-wireless-security"] = securitySettings
		}

		// 3. Add the connection via D-Bus
		settingsObj := conn.Object(network.NMDest, "/org/freedesktop/NetworkManager/Settings")
		call := settingsObj.Call("org.freedesktop.NetworkManager.Settings.AddConnection", 0, settings)
		if call.Err != nil {
			return common.ErrMsg{Err: fmt.Errorf("failed to add connection: %w", call.Err)}
		}

		// 4. Get the path (optional)
		var newConnectionPath dbus.ObjectPath
		err = call.Store(&newConnectionPath) // Store error

		// 5. Create the delayed refresh command
		refreshCmd := tea.Tick(150*time.Millisecond, func(t time.Time) tea.Msg {
			return common.RefreshKnownNetworksMsg{}
		})

		// 6. Create the optimistic update message
		optimisticMsg := common.OptimisticAddMsg{
			SSID:     net.SSID,
			Security: net.Security, // Use the security string from scanned network
		}

		// 7. Batch commands based on success
		var batchCmds []tea.Cmd
		batchCmds = append(batchCmds, func() tea.Msg { return optimisticMsg }) // Send optimistic update first
		batchCmds = append(batchCmds, refreshCmd)                             // Schedule real refresh

		if err == nil {
			// If we got the path, attempt connection
			batchCmds = append(batchCmds, ConnectToNetworkCmd(conn, newConnectionPath, devicePath))
		} else {
			// If we didn't get the path, report the error but still refresh
			batchCmds = append(batchCmds, func() tea.Msg { return common.ErrMsg{Err: fmt.Errorf("added connection but failed to read path: %w", err)} })
		}
		return tea.Batch(batchCmds...)
	}
}

// AddAndConnectEAPCmd adds a WPA-EAP network and attempts connection.
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

		// 2. Generate UUID
		newUUID, err := uuid.NewRandom()
		if err != nil {
			return common.ErrMsg{Err: fmt.Errorf("failed to generate UUID for EAP: %w", err)}
		}

		// 3. Build EAP settings
		eapSettings := map[string]dbus.Variant{
			"eap":      dbus.MakeVariant([]string{strings.ToLower(eapMethod)}),
			"identity": dbus.MakeVariant(identity),
			"password": dbus.MakeVariant(config["password"]),
		}
		if phase2, ok := config["phase2-auth"]; ok && phase2 != "" && phase2 != "NONE" {
			eapSettings["phase2-auth"] = dbus.MakeVariant(strings.ToLower(phase2))
		}
		if certPath, ok := config["ca_cert"]; ok && certPath != "" {
			eapSettings["ca-cert"] = dbus.MakeVariant("file://" + certPath)
		}

		// 4. Build complete settings map
		settings := map[string]map[string]dbus.Variant{
			"connection": {
				"id":          dbus.MakeVariant(ssid),
				"uuid":        dbus.MakeVariant(newUUID.String()),
				"type":        dbus.MakeVariant("802-11-wireless"),
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
			"802-1x": eapSettings,
			"ipv4":   {"method": dbus.MakeVariant("auto")},
			"ipv6":   {"method": dbus.MakeVariant("auto")},
		}

		// 5. Add the connection via D-Bus
		settingsObj := conn.Object(network.NMDest, "/org/freedesktop/NetworkManager/Settings")
		call := settingsObj.Call("org.freedesktop.NetworkManager.Settings.AddConnection", 0, settings)
		if call.Err != nil {
			return common.ErrMsg{Err: fmt.Errorf("failed to add EAP connection: %w", call.Err)}
		}

		// 6. Get the path (optional)
		var newConnectionPath dbus.ObjectPath
		err = call.Store(&newConnectionPath) // Store error

		// 7. Create the delayed refresh command
		refreshCmd := tea.Tick(150*time.Millisecond, func(t time.Time) tea.Msg {
			return common.RefreshKnownNetworksMsg{}
		})

		// 8. Create the optimistic update message
		optimisticMsg := common.OptimisticAddMsg{
			SSID:     ssid,
			Security: "wpa2-eap", // Generally correct assumption for EAP
		}

		// 9. Batch commands based on success
		var batchCmds []tea.Cmd
		batchCmds = append(batchCmds, func() tea.Msg { return optimisticMsg }) // Send optimistic update first
		batchCmds = append(batchCmds, refreshCmd)                             // Schedule real refresh

		if err == nil {
			// If we got the path, attempt connection
			batchCmds = append(batchCmds, ConnectToNetworkCmd(conn, newConnectionPath, devicePath))
		} else {
			// If we didn't get the path, report the error but still refresh
			batchCmds = append(batchCmds, func() tea.Msg { return common.ErrMsg{Err: fmt.Errorf("added EAP connection but failed to read path: %w", err)} })
		}
		return tea.Batch(batchCmds...)
	}
}

// ToggleVpnCmd activates or deactivates a VPN connection.
func ToggleVpnCmd(conn *dbus.Conn, vpnPath dbus.ObjectPath, activePath dbus.ObjectPath, active bool) tea.Cmd {
	return func() tea.Msg {
		nm := conn.Object(network.NMDest, dbus.ObjectPath(network.NMPath))
		var call *dbus.Call
		action := "activate" // For error message

		if active {
			// Deactivate using the *active* connection path
			action = "deactivate"
			if activePath == "/" { // Sanity check
				return common.ErrMsg{Err: fmt.Errorf("cannot deactivate VPN: no active connection path found")}
			}
			activeConnObj := conn.Object(network.NMDest, activePath)
			// Note: Deactivate is on the Active connection interface, not the main NM interface
			call = activeConnObj.Call("org.freedesktop.NetworkManager.Connection.Active.Deactivate", 0)
		} else {
			// Activate using the *saved* connection path
			call = nm.Call(
				"org.freedesktop.NetworkManager.ActivateConnection",
				0,
				vpnPath,                // Saved connection path
				dbus.ObjectPath("/"),   // device path is not needed for VPN
				dbus.ObjectPath("/"),   // specific object path
			)
		}

		if call.Err != nil {
			return common.ErrMsg{Err: fmt.Errorf("failed to %s vpn connection '%s': %w", action, vpnPath, call.Err)}
		}
		// Success handled by signal listener
		return nil
	}
}

// ToggleWifiCmd sets the master Wi-Fi radio state.
func ToggleWifiCmd(conn *dbus.Conn, enable bool) tea.Cmd {
	return func() tea.Msg {
		nm := conn.Object(network.NMDest, dbus.ObjectPath(network.NMPath))
		call := nm.Call(
			"org.freedesktop.DBus.Properties.Set",
			0,
			network.NMDest,
			"WirelessEnabled",
			dbus.MakeVariant(enable),
		)
		if call.Err != nil {
			return common.ErrMsg{Err: fmt.Errorf("failed to set WirelessEnabled property: %w", call.Err)}
		}
		// Success handled by signal listener
		return nil
	}
}

// DeleteConnectionCmd tells NetworkManager to delete a saved connection profile.
func DeleteConnectionCmd(conn *dbus.Conn, connectionPath dbus.ObjectPath) tea.Cmd {
	return func() tea.Msg {
		connObj := conn.Object(network.NMDest, connectionPath)
		call := connObj.Call(
			"org.freedesktop.NetworkManager.Settings.Connection.Delete",
			0,
		)
		if call.Err != nil {
			return common.ErrMsg{Err: fmt.Errorf("failed to delete connection %s: %w", connectionPath, call.Err)}
		}
		// Success handled by signal listener
		return nil
	}
}