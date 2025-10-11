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

func get_scanned_networks(c *dbus.Conn) []scanned_network {
	nm := c.Object(nmDest, dbus.ObjectPath(nmPath))
	var devPaths []dbus.ObjectPath
	if err := nm.Call(nmDest+".GetDevices", 0).Store(&devPaths); err != nil {
		fmt.Printf("failed to get devices: %v", err)
	}

	var allNetworks []scanned_network
	for _, devPath := range devPaths {
		devObj := c.Object(nmDest, devPath)
		devProps := get_props(devObj, devIF)
		if devProps["DeviceType"].Value().(uint32) != 2 {
			continue
		}

		var apPaths []dbus.ObjectPath
		if err := devObj.Call(wifiIF+".GetAllAccessPoints", 0).Store(&apPaths); err != nil {
			fmt.Printf("failed to get access points: %v", err)
		}

		for _, apPath := range apPaths {
			apObj := c.Object(nmDest, apPath)
			apProps := get_props(apObj, accessPointIF)

			var ssid string
			if ssidVal, ok := apProps["Ssid"]; ok {
				if ssidBytes, ok := ssidVal.Value().([]byte); ok {
					ssid = strings.TrimRight(string(ssidBytes), "\x00")
				}
			}
			if ssid == "" {
				continue
			}

			bssid := ""
			if bssidVal, ok := apProps["HwAddress"]; ok {
				bssid = bssidVal.Value().(string)
			}
			var signal int
			if strengthVal, ok := apProps["Strength"]; ok {
				signal = int(strengthVal.Value().(byte))
			}

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

			allNetworks = append(allNetworks, scanned_network{
				ssid:     ssid,
				bssid:    bssid,
				security: get_security_type(wpaFlags, rsnFlags),
				signal:   signal,
			})
		}
	}
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
	if rsnFlags&NM_802_11_AP_SEC_KEY_MGMT_SAE != 0 {
		security = append(security, "wpa3-sae")
	}
	if rsnFlags&NM_802_11_AP_SEC_KEY_MGMT_EAP_SUITE_B_192 != 0 {
		security = append(security, "wpa3-eap-192")
	}
	if (rsnFlags&NM_802_11_AP_SEC_KEY_MGMT_802_1X != 0) || (wpaFlags&NM_802_11_AP_SEC_KEY_MGMT_802_1X != 0) {
		security = append(security, "wpa2-eap")
	}
	if (rsnFlags&NM_802_11_AP_SEC_KEY_MGMT_PSK != 0) || (wpaFlags&NM_802_11_AP_SEC_KEY_MGMT_PSK != 0) {
		security = append(security, "wpa2-psk")
	}
	if rsnFlags&(NM_802_11_AP_SEC_KEY_MGMT_OWE|NM_802_11_AP_SEC_KEY_MGMT_OWE_TM) != 0 {
		security = append(security, "wpa-owe")
	}
	if len(security) == 0 {
		return "open"
	}
	return strings.Join(security, " / ")
}

func remove_duplicates(networks []scanned_network) []scanned_network {
	networkMap := make(map[string]scanned_network)
	for _, network := range networks {
		existing, exists := networkMap[network.ssid]
		if !exists || network.signal > existing.signal {
			networkMap[network.ssid] = network
		}
	}

	var result []scanned_network
	for _, network := range networkMap {
		result = append(result, network)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].signal > result[j].signal })
	return result
}
