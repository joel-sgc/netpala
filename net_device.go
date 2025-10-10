package main

import (
	"fmt"
	"strings"

	"github.com/godbus/dbus/v5"
)

const (
	nmDest        = "org.freedesktop.NetworkManager"
	nmPath        = "/org/freedesktop/NetworkManager"
	propsIF       = "org.freedesktop.DBus.Properties"
	devIF         = "org.freedesktop.NetworkManager.Device"
	wifiIF        = "org.freedesktop.NetworkManager.Device.Wireless"
	accessPointIF = "org.freedesktop.NetworkManager.AccessPoint"
)

func get_devices_data(c *dbus.Conn) []device {
	var devicesList []device
	nm := c.Object(nmDest, dbus.ObjectPath(nmPath))
	p := get_props(nm, nmDest)

	settingsObj := c.Object(nmDest, "/org/freedesktop/NetworkManager/Settings")
	var connPaths []dbus.ObjectPath
	_ = settingsObj.Call("org.freedesktop.NetworkManager.Settings.ListConnections", 0).Store(&connPaths)

	inferSecurity := func(sec map[string]dbus.Variant) string {
		if sec == nil {
			return "-"
		}
		if v, ok := sec["key-mgmt"]; ok {
			km := strings.ToLower(v.Value().(string))
			switch {
			case strings.Contains(km, "sae"):
				return "wpa-sae"
			case strings.Contains(km, "wpa-psk"):
				return "wpa-psk"
			case strings.Contains(km, "wpa-eap"):
				return "wpa-eap"
			case strings.Contains(km, "none"):
				return "open"
			}
		}
		if _, ok := sec["psk"]; ok {
			return "wpa-psk"
		}
		return "encrypted"
	}

	var devs []dbus.ObjectPath
	nm.Call(nmDest+".GetDevices", 0).Store(&devs)

	for _, d := range devs {
		obj := c.Object(nmDest, d)
		dp := get_props(obj, devIF)
		if dp["DeviceType"].Value().(uint32) != 2 {
			continue
		}

		var deviceState int = -1 // Default to disconnected
		if stateVar, ok := dp["State"]; ok {
			state, _ := stateVar.Value().(uint32)
			switch state {
			case 100: // ACTIVATED
				deviceState = 1
			case 30: // DISCONNECTED
				deviceState = -1
			default: // PREPARE, CONFIG, NEED_AUTH, IP_CONFIG, etc.
				deviceState = 0
			}
		}

		iface := dp["Interface"].Value().(string)
		mac := strings.ToLower(dp["HwAddress"].Value().(string))
		wp := get_props(obj, wifiIF)
		mode := wp["Mode"].Value().(uint32)
		ap := wp["ActiveAccessPoint"].Value().(dbus.ObjectPath)

		var isScanning bool
		if scanningVar, ok := wp["Scanning"]; ok {
			isScanning, _ = scanningVar.Value().(bool)
		}

		bssid, frequency, security := "-", 0, "-"
		if ap != "/" {
			apObj := c.Object(nmDest, ap)
			if bssidVar, err := apObj.GetProperty("org.freedesktop.NetworkManager.AccessPoint.HwAddress"); err == nil {
				bssid = bssidVar.Value().(string)
			}
			if freqVar, err := apObj.GetProperty("org.freedesktop.NetworkManager.AccessPoint.Frequency"); err == nil {
				frequency = int(freqVar.Value().(uint32))
			}
			ssidVar, _ := apObj.GetProperty("org.freedesktop.NetworkManager.AccessPoint.Ssid")
			activeSSID := ""
			if b, ok := ssidVar.Value().([]byte); ok {
				activeSSID = strings.TrimRight(string(b), "\x00")
			}
			for _, cpath := range connPaths {
				cobj := c.Object(nmDest, cpath)
				var settings map[string]map[string]dbus.Variant
				if cobj.Call("org.freedesktop.NetworkManager.Settings.Connection.GetSettings", 0).Store(&settings) != nil {
					continue
				}
				if wcfg, ok := settings["802-11-wireless"]; ok {
					if ssidV, ok := wcfg["ssid"]; ok {
						if b, ok := ssidV.Value().([]byte); ok && strings.TrimRight(string(b), "\x00") == activeSSID {
							security = inferSecurity(settings["802-11-wireless-security"])
							break
						}
					}
				}
			}
		}
		modeStr := map[uint32]string{1: "ad-hoc", 2: "station", 3: "ap", 4: "mesh"}[mode]
		if modeStr == "" {
			modeStr = fmt.Sprintf("%d", mode)
		}

		devicesList = append(devicesList, device{
			path: d,
			name: iface, mode: modeStr,
			powered:      p["WirelessEnabled"].Value().(bool) && p["WirelessHardwareEnabled"].Value().(bool),
			address:      mac,
			state:        deviceState, // **FIX:** Use the accurate per-device state.
			currentbssid: bssid,
			scanning:     isScanning,
			frequency:    frequency, security: security,
		})
	}
	return devicesList
}
