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
			"password-flags": dbus.MakeVariant(uint32(0)),
		}
		if phase2, ok := config["phase2-auth"]; ok && phase2 != "" && phase2 != "NONE" {
			eapSettings["phase2-auth"] = dbus.MakeVariant(strings.ToLower(phase2))
		}
		// Intentionally omit ca-cert when none is provided: per the NM 802-1x
		// spec, an absent ca-cert tells wpa_supplicant to skip server certificate
		// validation entirely. Do NOT set it to an empty string here or cert
		// validation will behave unpredictably across NM versions.
		if certPath := strings.TrimSpace(config["ca_cert"]); certPath != "" {
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

		// 5. Add and activate the connection atomically via D-Bus.
		// Using AddAndActivateConnection instead of separate AddConnection +
		// ActivateConnection ensures secrets remain available throughout the
		// activation handshake, avoiding intermittent "secrets required" errors.
		nm := conn.Object(network.NMDest, dbus.ObjectPath(network.NMPath))
		call := nm.Call("org.freedesktop.NetworkManager.AddAndActivateConnection", 0, settings, devicePath, dbus.ObjectPath("/"))
		if call.Err != nil {
			return common.ErrMsg{Err: fmt.Errorf("failed to add and activate EAP connection: %w", call.Err)}
		}

		// 6. Create the delayed refresh command
		refreshCmd := tea.Tick(150*time.Millisecond, func(t time.Time) tea.Msg {
			return common.RefreshKnownNetworksMsg{}
		})

		// 7. Create the optimistic update message
		optimisticMsg := common.OptimisticAddMsg{
			SSID:     ssid,
			Security: "wpa2-eap",
		}

		return tea.Batch(
			func() tea.Msg { return optimisticMsg },
			refreshCmd,
		)
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

// ToggleAutoconnectCmd flips the autoconnect setting on a saved connection profile.
func ToggleAutoconnectCmd(conn *dbus.Conn, connectionPath dbus.ObjectPath, currentAutoConnect bool) tea.Cmd {
	return func() tea.Msg {
		connObj := conn.Object(network.NMDest, connectionPath)

		var settings map[string]map[string]dbus.Variant
		if err := connObj.Call("org.freedesktop.NetworkManager.Settings.Connection.GetSettings", 0).Store(&settings); err != nil {
			return common.ErrMsg{Err: fmt.Errorf("failed to get connection settings: %w", err)}
		}

		if settings["connection"] == nil {
			settings["connection"] = map[string]dbus.Variant{}
		}
		settings["connection"]["autoconnect"] = dbus.MakeVariant(!currentAutoConnect)

		// Strip legacy fields that don't survive a GetSettings→Update round-trip.
		// NM returns addresses/routes as a(ayuay) but godbus re-serialises them as
		// aav, causing a type-mismatch error. NM rebuilds these from address-data /
		// route-data, so removing them is safe.
		for _, section := range []string{"ipv4", "ipv6"} {
			if sec, ok := settings[section]; ok {
				delete(sec, "addresses")
				delete(sec, "routes")
			}
		}

		call := connObj.Call("org.freedesktop.NetworkManager.Settings.Connection.Update", 0, settings)
		if call.Err != nil {
			return common.ErrMsg{Err: fmt.Errorf("failed to update connection: %w", call.Err)}
		}

		return common.KnownNetworksUpdateMsg(network.GetKnownNetworks(conn))
	}
}

// ToggleHiddenCmd flips the hidden (SSID broadcast) setting on a saved connection profile.
func ToggleHiddenCmd(conn *dbus.Conn, connectionPath dbus.ObjectPath, currentHidden bool) tea.Cmd {
	return func() tea.Msg {
		connObj := conn.Object(network.NMDest, connectionPath)

		var settings map[string]map[string]dbus.Variant
		if err := connObj.Call("org.freedesktop.NetworkManager.Settings.Connection.GetSettings", 0).Store(&settings); err != nil {
			return common.ErrMsg{Err: fmt.Errorf("failed to get connection settings: %w", err)}
		}

		if settings["802-11-wireless"] == nil {
			settings["802-11-wireless"] = map[string]dbus.Variant{}
		}
		settings["802-11-wireless"]["hidden"] = dbus.MakeVariant(!currentHidden)

		// Strip legacy fields that don't survive a GetSettings→Update round-trip.
		for _, section := range []string{"ipv4", "ipv6"} {
			if sec, ok := settings[section]; ok {
				delete(sec, "addresses")
				delete(sec, "routes")
			}
		}

		call := connObj.Call("org.freedesktop.NetworkManager.Settings.Connection.Update", 0, settings)
		if call.Err != nil {
			return common.ErrMsg{Err: fmt.Errorf("failed to update connection: %w", call.Err)}
		}

		return common.KnownNetworksUpdateMsg(network.GetKnownNetworks(conn))
	}
}

// UpdateConnectionCmd reads a saved WPA-PSK/open connection, applies the user's
// edits (SSID, password, hidden, autoconnect), and writes them back via D-Bus.
func UpdateConnectionCmd(conn *dbus.Conn, connectionPath dbus.ObjectPath, config map[string]string) tea.Cmd {
	return func() tea.Msg {
		connObj := conn.Object(network.NMDest, connectionPath)

		var settings map[string]map[string]dbus.Variant
		if err := connObj.Call("org.freedesktop.NetworkManager.Settings.Connection.GetSettings", 0).Store(&settings); err != nil {
			return common.ErrMsg{Err: fmt.Errorf("failed to get connection settings: %w", err)}
		}

		// -- connection section --
		if settings["connection"] == nil {
			settings["connection"] = map[string]dbus.Variant{}
		}
		if ssid, ok := config["ssid"]; ok && ssid != "" {
			settings["connection"]["id"] = dbus.MakeVariant(ssid)
		}
		settings["connection"]["autoconnect"] = dbus.MakeVariant(config["autoconnect"] == "true")

		// -- 802-11-wireless section --
		if settings["802-11-wireless"] == nil {
			settings["802-11-wireless"] = map[string]dbus.Variant{}
		}
		if ssid, ok := config["ssid"]; ok && ssid != "" {
			settings["802-11-wireless"]["ssid"] = dbus.MakeVariant([]byte(ssid))
		}
		settings["802-11-wireless"]["hidden"] = dbus.MakeVariant(config["hidden"] == "true")

		// -- password (PSK) -- only update if a new password was supplied
		if password, ok := config["password"]; ok && password != "" {
			if settings["802-11-wireless-security"] == nil {
				settings["802-11-wireless-security"] = map[string]dbus.Variant{}
			}
			settings["802-11-wireless-security"]["psk"] = dbus.MakeVariant(password)
		}

		// Strip legacy fields that don't survive a GetSettings→Update round-trip.
		for _, section := range []string{"ipv4", "ipv6"} {
			if sec, ok := settings[section]; ok {
				delete(sec, "addresses")
				delete(sec, "routes")
			}
		}

		call := connObj.Call("org.freedesktop.NetworkManager.Settings.Connection.Update", 0, settings)
		if call.Err != nil {
			return common.ErrMsg{Err: fmt.Errorf("failed to update connection: %w", call.Err)}
		}

		return common.KnownNetworksUpdateMsg(network.GetKnownNetworks(conn))
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

// LoadEapEditSettingsCmd reads the current EAP connection settings and secrets
// from NetworkManager so the edit form can be pre-populated with real values.
func LoadEapEditSettingsCmd(conn *dbus.Conn, connectionPath dbus.ObjectPath, net common.KnownNetwork) tea.Cmd {
	return func() tea.Msg {
		connObj := conn.Object(network.NMDest, connectionPath)

		var settings map[string]map[string]dbus.Variant
		if err := connObj.Call("org.freedesktop.NetworkManager.Settings.Connection.GetSettings", 0).Store(&settings); err != nil {
			return common.ErrMsg{Err: fmt.Errorf("failed to load EAP settings: %w", err)}
		}

		msg := common.EapEditSettingsMsg{
			ConnectionPath: connectionPath,
			SSID:           net.SSID,
			AutoConnect:    net.AutoConnect,
			Hidden:         net.Hidden,
		}

		// Parse 802-11-wireless SSID (overrides KnownNetwork value if present)
		if wlan, ok := settings["802-11-wireless"]; ok {
			if v, ok := wlan["ssid"]; ok {
				if b, ok := v.Value().([]byte); ok {
					msg.SSID = string(b)
				}
			}
		}

		// Parse 802-1x section
		if eap, ok := settings["802-1x"]; ok {
			// EAP method: stored as []string, e.g. ["peap"]
			if v, ok := eap["eap"]; ok {
				if methods, ok := v.Value().([]string); ok && len(methods) > 0 {
					msg.EapMethod = strings.ToUpper(methods[0])
				}
			}
			if v, ok := eap["identity"]; ok {
				if s, ok := v.Value().(string); ok {
					msg.Identity = s
				}
			}
			// Phase2 auth: lowercase string, e.g. "mschapv2"
			if v, ok := eap["phase2-auth"]; ok {
				if s, ok := v.Value().(string); ok {
					msg.Phase2Auth = strings.ToUpper(s)
				}
			}
			// CA cert: stored as []byte "file:///path..." with NM null terminator
			if v, ok := eap["ca-cert"]; ok {
				if b, ok := v.Value().([]byte); ok {
					certStr := strings.TrimPrefix(string(b), "file://")
					certStr = strings.TrimRight(certStr, "\x00")
					msg.CaCert = certStr
				}
			}
		}

		// Try to retrieve the password from secrets (graceful degradation on failure)
		var secrets map[string]map[string]dbus.Variant
		if err := connObj.Call("org.freedesktop.NetworkManager.Settings.Connection.GetSecrets", 0, "802-1x").Store(&secrets); err == nil {
			if sec, ok := secrets["802-1x"]; ok {
				if v, ok := sec["password"]; ok {
					if pw, ok := v.Value().(string); ok {
						msg.Password = pw
					}
				}
			}
		}

		return msg
	}
}

// UpdateEapConnectionCmd writes edited WPA-EAP connection settings back to
// NetworkManager. If the submitted password is empty the existing secret is
// preserved via a GetSecrets call before the update.
func UpdateEapConnectionCmd(conn *dbus.Conn, connectionPath dbus.ObjectPath, config map[string]string) tea.Cmd {
	return func() tea.Msg {
		connObj := conn.Object(network.NMDest, connectionPath)

		var settings map[string]map[string]dbus.Variant
		if err := connObj.Call("org.freedesktop.NetworkManager.Settings.Connection.GetSettings", 0).Store(&settings); err != nil {
			return common.ErrMsg{Err: fmt.Errorf("failed to get EAP connection settings: %w", err)}
		}

		// -- connection section --
		if settings["connection"] == nil {
			settings["connection"] = map[string]dbus.Variant{}
		}
		if ssid := config["ssid"]; ssid != "" {
			settings["connection"]["id"] = dbus.MakeVariant(ssid)
		}
		settings["connection"]["autoconnect"] = dbus.MakeVariant(config["autoconnect"] == "true")

		// -- 802-11-wireless section --
		if settings["802-11-wireless"] == nil {
			settings["802-11-wireless"] = map[string]dbus.Variant{}
		}
		if ssid := config["ssid"]; ssid != "" {
			settings["802-11-wireless"]["ssid"] = dbus.MakeVariant([]byte(ssid))
		}
		settings["802-11-wireless"]["hidden"] = dbus.MakeVariant(config["hidden"] == "true")

		// -- 802-1x section --
		if settings["802-1x"] == nil {
			settings["802-1x"] = map[string]dbus.Variant{}
		}
		eap1x := settings["802-1x"]

		if eapMethod := config["eap"]; eapMethod != "" {
			eap1x["eap"] = dbus.MakeVariant([]string{strings.ToLower(eapMethod)})
		}
		if identity := config["identity"]; identity != "" {
			eap1x["identity"] = dbus.MakeVariant(identity)
		}
		if phase2 := config["phase2-auth"]; phase2 != "" && strings.ToUpper(phase2) != "NONE" {
			eap1x["phase2-auth"] = dbus.MakeVariant(strings.ToLower(phase2))
		} else {
			delete(eap1x, "phase2-auth")
		}
		// CA cert: absent = no validation; present = "file://" + path + NM null terminator
		if certPath := strings.TrimSpace(config["ca_cert"]); certPath != "" {
			eap1x["ca-cert"] = dbus.MakeVariant([]byte("file://" + certPath + "\x00"))
		} else {
			delete(eap1x, "ca-cert")
		}

		// Password: if empty, try to preserve the existing secret
		password := config["password"]
		if password == "" {
			var secrets map[string]map[string]dbus.Variant
			if err := connObj.Call("org.freedesktop.NetworkManager.Settings.Connection.GetSecrets", 0, "802-1x").Store(&secrets); err == nil {
				if sec, ok := secrets["802-1x"]; ok {
					if v, ok := sec["password"]; ok {
						if pw, ok := v.Value().(string); ok {
							password = pw
						}
					}
				}
			}
		}
		if password != "" {
			eap1x["password"] = dbus.MakeVariant(password)
			eap1x["password-flags"] = dbus.MakeVariant(uint32(0))
		}

		// Strip legacy fields that don't survive a GetSettings→Update round-trip.
		for _, section := range []string{"ipv4", "ipv6"} {
			if sec, ok := settings[section]; ok {
				delete(sec, "addresses")
				delete(sec, "routes")
			}
		}

		call := connObj.Call("org.freedesktop.NetworkManager.Settings.Connection.Update", 0, settings)
		if call.Err != nil {
			return common.ErrMsg{Err: fmt.Errorf("failed to update EAP connection: %w", call.Err)}
		}

		return common.KnownNetworksUpdateMsg(network.GetKnownNetworks(conn))
	}
}