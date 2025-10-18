package main

import (
	"github.com/godbus/dbus/v5"
)

// // Message sent when device data should be updated.
// type deviceUpdateMsg []device
// // Message sent when known networks should be updated.
// type knownNetworksUpdateMsg []known_network
// // Message sent when scanned networks should be updated.
// type scannedNetworksUpdateMsg []scanned_network

// // Message to signal an error from a goroutine.
// type errMsg struct{ err error }

// // Message sent from our periodic timer to trigger a full refresh.
// type periodicRefreshMsg struct{}

// // Message sent from our debounce timer to perform a scan.
// type performScanRefreshMsg struct{}

type vpnUpdateMsg []vpn_connection
type deviceUpdateMsg []device
type knownNetworksUpdateMsg []known_network
type scannedNetworksUpdateMsg []scanned_network
type errMsg struct{ err error }
type periodicRefreshMsg struct{}
type performScanRefreshMsg struct{}

type netpala_data struct {
	width, height 		int
	selected_box  		int
	selected_entry		int
	
	device_data      	[]device
	vpn_data         	[]vpn_connection
	known_networks   	[]known_network
	scanned_networks 	[]scanned_network
	tables            tables_model
	status_bar 		 	 	status_bar_data
	form 						 	wpa_eap_form

	network_to_connect 	scanned_network
	is_typing				 		bool
	is_in_form					bool

	initial_load_complete	bool
	conn        					*dbus.Conn
	err         					error
	dbusSignals 					chan *dbus.Signal
}

type device struct {
	path         dbus.ObjectPath
	name         string
	mode         string
	powered      bool
	address      string
	state        int
	currentbssid string
	scanning     bool
	frequency    int
	security     string
}

type known_network struct {
	path         dbus.ObjectPath
	bssid        string
	ssid         string
	security     string
	hidden       bool
	auto_connect bool
	signal       int
	connected    bool
}

type scanned_network struct {
	// path     dbus.ObjectPath
	bssid    string
	ssid     string
	security string
	signal   int
}

type vpn_connection struct {
	path       dbus.ObjectPath
	activePath dbus.ObjectPath
	name       string
	ctype   	 string
	connected  bool
}