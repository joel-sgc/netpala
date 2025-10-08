package main

type netpala_data struct {
	width, height int		// To denote width,length of the terminal window
	
	selected_box int        // To denote which box is selected
	selected_entry int      // To denote which option within our selected box is selected

	device_data []device
	known_networks []known_network
	scanned_networks []scanned_network
}

type device struct {
	name string
	mode string
	powered bool
	address string
	state int   // -1 for disconnected, 0 for connecting, 1 for connected
	currentbssid string     // "" if not connected, or ig we could just not check it if state != 1
	scanning bool
	frequency int
	security string
}

type known_network struct {
	bssid string
	ssid string
	security string
	hidden bool
	auto_connect bool
	signal int      // 1-100 or maybe just 1-5, not sure yet
	connected bool
}

type scanned_network struct {
	bssid string
	ssid string
	security string
	signal int      // Same as known_networks.signal
}