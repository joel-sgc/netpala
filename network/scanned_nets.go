package network

import (
	"fmt"
	"netpala/common"
	"sort"
	"strings"

	"github.com/godbus/dbus/v5"
)

func GetProps(o dbus.BusObject, iface string) map[string]dbus.Variant {
	var m map[string]dbus.Variant
	o.Call(PropsIF+".GetAll", 0, iface).Store(&m)
	return m
}

func GetScannedNetworks(c *dbus.Conn) []common.ScannedNetwork {
	nm := c.Object(NMDest, dbus.ObjectPath(NMPath))
	var devPaths []dbus.ObjectPath
	if err := nm.Call(NMDest+".GetDevices", 0).Store(&devPaths); err != nil {
		fmt.Printf("failed to get devices: %v", err)
	}

	var allNetworks []common.ScannedNetwork
	for _, devPath := range devPaths {
		devObj := c.Object(NMDest, devPath)
		devProps := GetProps(devObj, DevIF)
		if devProps["DeviceType"].Value().(uint32) != 2 {
			continue
		}

		var apPaths []dbus.ObjectPath
		if err := devObj.Call(WifiIF+".GetAllAccessPoints", 0).Store(&apPaths); err != nil {
			fmt.Printf("failed to get access points: %v", err)
		}

		for _, apPath := range apPaths {
			apObj := c.Object(NMDest, apPath)
			apProps := GetProps(apObj, AccessPointIF)

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

			ap := c.Object(NMDest, apPath)
			wpaFlagsVar, _ := ap.GetProperty("org.freedesktop.NetworkManager.AccessPoint.WpaFlags")
			rsnFlagsVar, _ := ap.GetProperty("org.freedesktop.NetworkManager.AccessPoint.RsnFlags")
			var wpaFlags, rsnFlags uint32
			if val, ok := wpaFlagsVar.Value().(uint32); ok {
				wpaFlags = val
			}
			if val, ok := rsnFlagsVar.Value().(uint32); ok {
				rsnFlags = val
			}

			allNetworks = append(allNetworks, common.ScannedNetwork{
				SSID:     ssid,
				BSSID:    bssid,
				Security: getSecurityType(wpaFlags, rsnFlags),
				Signal:   signal,
			})
		}
	}
	return removeDuplicates(allNetworks)
}

func getSecurityType(wpaFlags, rsnFlags uint32) string {
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

func removeDuplicates(networks []common.ScannedNetwork) []common.ScannedNetwork {
	networkMap := make(map[string]common.ScannedNetwork)
	for _, network := range networks {
		existing, exists := networkMap[network.SSID]
		if !exists || network.Signal > existing.Signal {
			networkMap[network.SSID] = network
		}
	}

	var result []common.ScannedNetwork
	for _, network := range networkMap {
		result = append(result, network)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Signal > result[j].Signal })
	return result
}
