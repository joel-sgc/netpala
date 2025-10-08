package main

import (
	"fmt"
	"strings"

	"github.com/godbus/dbus/v5"
)

const (
	nmDest    = "org.freedesktop.NetworkManager"
	nmPath    = "/org/freedesktop/NetworkManager"
	propsIF   = "org.freedesktop.DBus.Properties"
	devIF     = "org.freedesktop.NetworkManager.Device"
	wifiIF    = "org.freedesktop.NetworkManager.Device.Wireless"
	accessPointIF       = "org.freedesktop.NetworkManager.AccessPoint"
)

func get_devices_data() []device {
	var devicesList []device

	c, _ := dbus.SystemBus()
	nm := c.Object(nmDest, dbus.ObjectPath(nmPath))
	p := get_props(nm, nmDest)

	// Get overall network state
	stateVar, _ := nm.GetProperty("org.freedesktop.NetworkManager.State")
	state := stateVar.Value().(uint32)
	status := map[uint32]int{
		20: -1, // Disconnected
		40: 0,  // Connecting
		50: 1,  // Connected local
		60: 1,  // Connected site
		70: 1,  // Connected global
	}[state]

	// Get all saved connections (to look up security)
	settingsObj := c.Object(nmDest, "/org/freedesktop/NetworkManager/Settings")
	var connPaths []dbus.ObjectPath
	_ = settingsObj.Call("org.freedesktop.NetworkManager.Settings.ListConnections", 0).Store(&connPaths)

	// helper to infer security from saved connection settings
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

	// Get all devices
	var devs []dbus.ObjectPath
	nm.Call(nmDest+".GetDevices", 0).Store(&devs)

	for _, d := range devs {
		obj := c.Object(nmDest, d)
		dp := get_props(obj, devIF)
		if dp["DeviceType"].Value().(uint32) != 2 { // Wi-Fi only
			continue
		}

		iface := dp["Interface"].Value().(string)
		mac := strings.ToLower(dp["HwAddress"].Value().(string))
		wp := get_props(obj, wifiIF)
		mode := wp["Mode"].Value().(uint32)
		ap := wp["ActiveAccessPoint"].Value().(dbus.ObjectPath)

		bssid := "-"
		frequency := 0
		security := "-"

		if ap != "/" {
			apObj := c.Object(nmDest, ap)
			if bssidVar, err := apObj.GetProperty("org.freedesktop.NetworkManager.AccessPoint.HwAddress"); err == nil {
				bssid = bssidVar.Value().(string)
			}
			if freqVar, err := apObj.GetProperty("org.freedesktop.NetworkManager.AccessPoint.Frequency"); err == nil {
				frequency = int(freqVar.Value().(uint32))
			}
			// Active SSID
			ssidVar, _ := apObj.GetProperty("org.freedesktop.NetworkManager.AccessPoint.Ssid")
			activeSSID := ""
			if b, ok := ssidVar.Value().([]byte); ok {
				activeSSID = strings.TrimRight(string(b), "\x00")
			}

			// Lookup saved connections to infer security
			for _, cpath := range connPaths {
				cobj := c.Object(nmDest, cpath)
				var settings map[string]map[string]dbus.Variant
				if err := cobj.Call("org.freedesktop.NetworkManager.Settings.Connection.GetSettings", 0).Store(&settings); err != nil {
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
			name:         iface,
			mode:         modeStr,
			powered:      p["WirelessEnabled"].Value().(bool) && p["WirelessHardwareEnabled"].Value().(bool),
			address:      mac,
			state:        status,
			currentbssid: bssid,
			scanning:     false,
			frequency:    frequency,
			security:     security,
		})
	}

	return devicesList
}