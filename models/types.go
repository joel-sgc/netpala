package models

import (
	"github.com/godbus/dbus/v5"
)

// // Message sent when device data should be updated.
// type deviceUpdateMsg []Device
// // Message sent when known networks should be updated.
// type knownNetworksUpdateMsg []KnownNetwork
// // Message sent when scanned networks should be updated.
// type scannedNetworksUpdateMsg []ScannedNetwork

// // Message to signal an error from a goroutine.
// type errMsg struct{ err error }

// // Message sent from our periodic timer to trigger a full refresh.
// type periodicRefreshMsg struct{}

// // Message sent from our debounce timer to perform a scan.
// type performScanRefreshMsg struct{}

type VpnUpdateMsg []VpnConnection
type DeviceUpdateMsg []Device
type KnownNetworksUpdateMsg []KnownNetwork
type ScannedNetworksUpdateMsg []ScannedNetwork
type ErrMsg struct{ Err error }
type PeriodicRefreshMsg struct{}
type PerformScanRefreshMsg struct{}

type Device struct {
	Path         dbus.ObjectPath
	Name         string
	Mode         string
	Powered      bool
	Address      string
	State        int
	CurrentBSSID string
	Scanning     bool
	Frequency    int
	Security     string
}

type KnownNetwork struct {
	Path         dbus.ObjectPath
	BSSID        string
	SSID         string
	Security     string
	Hidden       bool
	AutoConnect bool
	Signal       int
	Connected    bool
}

type ScannedNetwork struct {
	// path     dbus.ObjectPath
	BSSID    string
	SSID     string
	Security string
	Signal   int
}

type VpnConnection struct {
	Path       dbus.ObjectPath
	ActivePath dbus.ObjectPath
	Name       string
	ConnType   string
	Connected  bool
}