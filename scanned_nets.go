package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/godbus/dbus/v5"
)

func get_props(o dbus.BusObject, iface string) map[string]dbus.Variant {
	var m map[string]dbus.Variant
	o.Call(propsIF+".GetAll", 0, iface).Store(&m)
	return m
}

// scanWifiNetworks scans for available WiFi networks and returns them with duplicates removed
func get_scanned_networks() []scanned_network {
	c, err := dbus.SystemBus()
	if err != nil {
		// fmt.Errorf("failed to connect to system bus: %v", err)
	}

	nm := c.Object(nmDest, dbus.ObjectPath(nmPath))

	// Get all devices
	var devPaths []dbus.ObjectPath
	if err := nm.Call(nmDest+".GetDevices", 0).Store(&devPaths); err != nil {
		// fmt.Errorf("failed to get devices: %v", err)
	}

	var allNetworks []scanned_network

	// Find WiFi devices and scan for networks
	for _, devPath := range devPaths {
		devObj := c.Object(nmDest, devPath)
		devProps := get_props(devObj, devIF)

		// Check if this is a WiFi device (DeviceType 2)
		if devProps["DeviceType"].Value().(uint32) != 2 {
			continue
		}

		// Request a new scan
		if err := devObj.Call(wifiIF+".RequestScan", 0, map[string]dbus.Variant{}).Err; err != nil {
			// If scan fails, we can still use cached results
			fmt.Printf("Note: Scan request failed (using cached results): %v\n", err)
		}

		// Get all access points
		var apPaths []dbus.ObjectPath
		if err := devObj.Call(wifiIF+".GetAllAccessPoints", 0).Store(&apPaths); err != nil {
			// fmt.Errorf("failed to get access points: %v", err)
		}

		// Process each access point
		for _, apPath := range apPaths {
			apObj := c.Object(nmDest, apPath)
			apProps := get_props(apObj, accessPointIF)

			// Get SSID
			var ssid string
			if ssidVal, ok := apProps["Ssid"]; ok {
				if ssidBytes, ok := ssidVal.Value().([]byte); ok {
					ssid = strings.TrimRight(string(ssidBytes), "\x00")
				}
			}

			// Skip if SSID is empty
			if ssid == "" {
				continue
			}

			// Get BSSID (MAC address)
			bssid := ""
			if bssidVal, ok := apProps["HwAddress"]; ok {
				bssid = bssidVal.Value().(string)
			}

			// Get signal strength (0–100)
			var signal int
			if strengthVal, ok := apProps["Strength"]; ok {
				signal = int(strengthVal.Value().(byte))
			}

			// Determine security type
			ap := c.Object(nmDest, apPath)
			wpaFlagsVar, _ := ap.GetProperty("org.freedesktop.NetworkManager.AccessPoint.WpaFlags")
			rsnFlagsVar, _ := ap.GetProperty("org.freedesktop.NetworkManager.AccessPoint.RsnFlags")

			var wpaFlags, rsnFlags uint32

			if val, ok := wpaFlagsVar.Value().(uint32); ok {
					wpaFlags = val
			}
			if val, ok := rsnFlagsVar.Value().(uint32); ok {
					rsnFlags = val
			}

			security := get_security_type(wpaFlags, rsnFlags)

			allNetworks = append(allNetworks, scanned_network{
				ssid:     ssid,   // SSID is the network name
				bssid:    bssid,  // BSSID is the MAC address
				security: security,
				signal:   signal,
			})
		}
	}

	// Remove duplicates and keep the strongest signal for each network
	return remove_duplicates(allNetworks)
}

func get_security_type(wpaFlags, rsnFlags uint32) string {
	const (
		NM_802_11_AP_SEC_KEY_MGMT_PSK             = 0x00000100
		NM_802_11_AP_SEC_KEY_MGMT_802_1X          = 0x00000200
		NM_802_11_AP_SEC_KEY_MGMT_SAE             = 0x00000400
		NM_802_11_AP_SEC_KEY_MGMT_OWE             = 0x00000800
		NM_802_11_AP_SEC_KEY_MGMT_OWE_TM          = 0x00001000
		NM_802_11_AP_SEC_KEY_MGMT_EAP_SUITE_B_192 = 0x00002000
	)

	var security []string

	// WPA3 Personal (SAE)
	if rsnFlags&NM_802_11_AP_SEC_KEY_MGMT_SAE != 0 {
		security = append(security, "wpa3-sae")
	}

	// WPA3 Enterprise Suite-B 192-bit
	if rsnFlags&NM_802_11_AP_SEC_KEY_MGMT_EAP_SUITE_B_192 != 0 {
		security = append(security, "wpa3-eap-192")
	}

	// WPA2/WPA Enterprise (802.1X)
	if (rsnFlags&NM_802_11_AP_SEC_KEY_MGMT_802_1X != 0) || (wpaFlags&NM_802_11_AP_SEC_KEY_MGMT_802_1X != 0) {
		security = append(security, "wpa2-eap")
	}

	// WPA2/WPA Personal (PSK)
	if (rsnFlags&NM_802_11_AP_SEC_KEY_MGMT_PSK != 0) || (wpaFlags&NM_802_11_AP_SEC_KEY_MGMT_PSK != 0) {
		security = append(security, "wpa2-psk")
	}

	// Opportunistic Wireless Encryption (OWE)
	if rsnFlags&(NM_802_11_AP_SEC_KEY_MGMT_OWE|NM_802_11_AP_SEC_KEY_MGMT_OWE_TM) != 0 {
		security = append(security, "wpa-owe")
	}

	if len(security) == 0 {
		return "open"
	}

	return strings.Join(security, " / ")
}

func remove_duplicates(networks []scanned_network) []scanned_network {
	// Group by SSID
	networkMap := make(map[string]scanned_network)
	
	for _, network := range networks {
		existing, exists := networkMap[network.ssid]
		if !exists || network.signal > existing.signal {
			networkMap[network.ssid] = network
		}
	}

	// Convert map back to slice
	var result []scanned_network
	for _, network := range networkMap {
		result = append(result, network)
	}

	// Sort by signal strength (descending)
	sort.Slice(result, func(i, j int) bool {
		return result[i].signal > result[j].signal
	})

	return result
}