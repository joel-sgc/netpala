package network

import (
	"fmt"
	"netpala/common"
	"strings"

	"github.com/godbus/dbus/v5"
)

func GetVpnData(c *dbus.Conn) []common.VpnConnection {
	var vpnList []common.VpnConnection

	nm := c.Object(NMDest, NMPath)
	settingsObj := c.Object(NMDest, "/org/freedesktop/NetworkManager/Settings")

	// 1. Get all active connections and map the saved path to the active path.
	activeConnPathsVariant, err := nm.GetProperty(NMDest + ".ActiveConnections")
	if err != nil {
		fmt.Printf("Error getting active connections: %v\n", err)
		return nil
	}
	activeConnPaths, _ := activeConnPathsVariant.Value().([]dbus.ObjectPath)
	// Map a saved connection path to its active connection path
	activeConnections := make(map[dbus.ObjectPath]dbus.ObjectPath)
	for _, acPath := range activeConnPaths {
		acObj := c.Object(NMDest, acPath)
		connProp, err := acObj.GetProperty("org.freedesktop.NetworkManager.Connection.Active.Connection")
		if err == nil {
			if connPath, ok := connProp.Value().(dbus.ObjectPath); ok {
				activeConnections[connPath] = acPath // Map saved path -> active path
			}
		}
	}

	// 2. Get all saved connection profiles.
	var savedConnPaths []dbus.ObjectPath
	if err := settingsObj.Call("org.freedesktop.NetworkManager.Settings.ListConnections", 0).Store(&savedConnPaths); err != nil {
		fmt.Printf("Error listing connections: %v\n", err)
		return nil
	}

	// 3. Iterate through saved connections and find the VPNs.
	for _, path := range savedConnPaths {
		connObj := c.Object(NMDest, path)
		var settings map[string]map[string]dbus.Variant
		if err := connObj.Call("org.freedesktop.NetworkManager.Settings.Connection.GetSettings", 0).Store(&settings); err != nil {
			continue
		}

		connSettings, ok := settings["connection"]
		if !ok {
			continue
		}

		connTypeVar, ok := connSettings["type"]
		if !ok {
			continue
		}

		connType, _ := connTypeVar.Value().(string)

		if connType == "wireguard" || connType == "vpn" {
			name, _ := connSettings["id"].Value().(string)

			friendlyType := "VPN"
			if connType == "wireguard" {
				friendlyType = "WireGuard"
			} else if vpnSettings, vpnOk := settings["vpn"]; vpnOk {
				if vpnTypeVar, vpnTypeOk := vpnSettings["service-type"]; vpnTypeOk {
					vpnTypeValue, _ := vpnTypeVar.Value().(string)
					parts := strings.Split(vpnTypeValue, ".")
					friendlyType = strings.ToUpper(parts[len(parts)-1])
				}
			}

			// Check if this connection is active and get its active path.
			activePath, isConnected := activeConnections[path]

			vpnList = append(vpnList, common.VpnConnection{
				Path:       path,
				ActivePath: activePath, // Store the active path
				Name:       name,
				ConnType:   friendlyType,
				Connected:  isConnected,
			})
		}
	}

	return vpnList
}
