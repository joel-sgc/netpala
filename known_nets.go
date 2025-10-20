package main

import (
	"netpala/models"
	"sort"
	"strings"

	"github.com/godbus/dbus/v5"
)

func getKnownNetworks(conn *dbus.Conn) []models.KnownNetwork {
	nm := conn.Object(nmDest, "/org/freedesktop/NetworkManager")

	ssidStr := func(v dbus.Variant) string {
		if b, ok := v.Value().([]byte); ok {
			return strings.TrimRight(string(b), "\x00")
		}
		return ""
	}

	aps := map[string]models.KnownNetwork{}
	var devs []dbus.ObjectPath
	_ = nm.Call(nmDest+".GetDevices", 0).Store(&devs)
	for _, d := range devs {
		obj := conn.Object(nmDest, d)
		if t, _ := obj.GetProperty("org.freedesktop.NetworkManager.Device.DeviceType"); t.Value().(uint32) != 2 {
			continue
		}
		if apsVar, err := obj.GetProperty("org.freedesktop.NetworkManager.Device.Wireless.AccessPoints"); err == nil {
			if paths, ok := apsVar.Value().([]dbus.ObjectPath); ok {
				for _, ap := range paths {
					apObj := conn.Object(nmDest, ap)
					ssidVar, _ := apObj.GetProperty("org.freedesktop.NetworkManager.AccessPoint.Ssid")
					ss := ssidStr(ssidVar)
					if ss == "" {
						continue
					}
					str := 0
					if s, err := apObj.GetProperty("org.freedesktop.NetworkManager.AccessPoint.Strength"); err == nil {
						str = int(s.Value().(uint8))
					}
					bssid := "-"
					if hw, err := apObj.GetProperty("org.freedesktop.NetworkManager.AccessPoint.HwAddress"); err == nil {
						bssid = hw.Value().(string)
					}
					if old, ok := aps[ss]; !ok || str > old.Signal {
						aps[ss] = models.KnownNetwork{SSID: ss, Signal: str, BSSID: bssid}
					}
				}
			}
		}
	}

	for _, d := range devs {
		obj := conn.Object(nmDest, d)
		if t, _ := obj.GetProperty("org.freedesktop.NetworkManager.Device.DeviceType"); t.Value().(uint32) != 2 {
			continue
		}
		if apVar, err := obj.GetProperty("org.freedesktop.NetworkManager.Device.Wireless.ActiveAccessPoint"); err == nil {
			if apPath, ok := apVar.Value().(dbus.ObjectPath); ok && apPath != "/" {
				apObj := conn.Object(nmDest, apPath)
				ssidVar, _ := apObj.GetProperty("org.freedesktop.NetworkManager.AccessPoint.Ssid")
				ss := ssidStr(ssidVar)
				if ss != "" {
					k := aps[ss]
					k.Connected = true
					aps[ss] = k
				}
			}
		}
	}

	setObj := conn.Object(nmDest, "/org/freedesktop/NetworkManager/Settings")
	var conns []dbus.ObjectPath
	_ = setObj.Call("org.freedesktop.NetworkManager.Settings.ListConnections", 0).Store(&conns)

	var known []models.KnownNetwork
	for _, c := range conns {
		cobj := conn.Object(nmDest, c)
		var s map[string]map[string]dbus.Variant
		if cobj.Call("org.freedesktop.NetworkManager.Settings.Connection.GetSettings", 0).Store(&s) != nil {
			continue
		}
		wcfg, ok := s["802-11-wireless"]
		if !ok {
			continue
		}
		ssidV, ok := wcfg["ssid"]
		if !ok {
			continue
		}
		ss := ssidStr(ssidV)
		hidden := false
		if hv, ok := wcfg["hidden"]; ok {
			hidden = hv.Value().(bool)
		}
		auto := false
		if cSec, ok := s["connection"]; ok {
			if av, ok := cSec["autoconnect"]; ok {
				auto = av.Value().(bool)
			}
		}
		sec := "open"
		if wsec, ok := s["802-11-wireless-security"]; ok {
			if km, ok := wsec["key-mgmt"]; ok {
				kmStr := strings.ToLower(km.Value().(string))
				switch {
				case strings.Contains(kmStr, "sae"):
					sec = "wpa3-sae"
				case strings.Contains(kmStr, "wpa-psk"):
					sec = "wpa2-psk"
				case strings.Contains(kmStr, "wpa-eap"):
					sec = "wpa2-eap"
				case strings.Contains(kmStr, "none"):
					sec = "wep"
				default:
					sec = "encrypted"
				}
			}
		}
		apInfo := aps[ss]
		known = append(known, models.KnownNetwork{

			Path: c, SSID: ss, Security: sec, Connected: apInfo.Connected, Hidden: hidden,
			AutoConnect: auto, Signal: apInfo.Signal, BSSID: apInfo.BSSID,
		})
	}
	sort.SliceStable(known, func(i, j int) bool {
		if known[i].Connected != known[j].Connected {
			return known[i].Connected
		}
		return known[i].Signal > known[j].Signal
	})
	return known
}