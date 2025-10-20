package network

import (
	"fmt"
	"netpala/common"
	"strings"

	"github.com/godbus/dbus/v5"
)

const (
	NMDest        = "org.freedesktop.NetworkManager"
	NMPath        = "/org/freedesktop/NetworkManager"
	PropsIF       = "org.freedesktop.DBus.Properties"
	DevIF         = "org.freedesktop.NetworkManager.Device"
	WifiIF        = "org.freedesktop.NetworkManager.Device.Wireless"
	AccessPointIF = "org.freedesktop.NetworkManager.AccessPoint"
)

func GetDevicesData(c *dbus.Conn) []common.Device {
	var devicesList []common.Device
	nm := c.Object(NMDest, dbus.ObjectPath(NMPath))
	p := GetProps(nm, NMDest)

	settingsObj := c.Object(NMDest, "/org/freedesktop/NetworkManager/Settings")
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
				return "wpa3-sae"
			case strings.Contains(km, "wpa-psk"):
				return "wpa2-psk"
			case strings.Contains(km, "wpa-eap"):
				return "wpa2-eap"
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
	nm.Call(NMDest+".GetDevices", 0).Store(&devs)

	for _, d := range devs {
		obj := c.Object(NMDest, d)
		dp := GetProps(obj, DevIF)
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
		wp := GetProps(obj, WifiIF)
		mode := wp["Mode"].Value().(uint32)
		ap := wp["ActiveAccessPoint"].Value().(dbus.ObjectPath)

		var isScanning bool
		if scanningVar, ok := wp["Scanning"]; ok {
			isScanning, _ = scanningVar.Value().(bool)
		}

		bssid, frequency, security := "-", 0, "-"
		if ap != "/" {
			apObj := c.Object(NMDest, ap)
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
				cobj := c.Object(NMDest, cpath)
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

		devicesList = append(devicesList, common.Device{
			Path: d,
			Name: iface, Mode: modeStr,
			Powered:      p["WirelessEnabled"].Value().(bool) && p["WirelessHardwareEnabled"].Value().(bool),
			Address:      mac,
			State:        deviceState, // **FIX:** Use the accurate per-device state.
			CurrentBSSID: bssid,
			Scanning:     isScanning,
			Frequency:    frequency, Security: security,
		})
	}
	return devicesList
}
