package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/godbus/dbus/v5"
)

// connectToNetworkCmd tells NetworkManager to activate a connection on a specific device.
func connectToNetworkCmd(conn *dbus.Conn, connectionPath, devicePath dbus.ObjectPath) tea.Cmd {
	return func() tea.Msg {
		nm := conn.Object(nmDest, dbus.ObjectPath(nmPath))

		// The D-Bus method call to activate the connection.
		// The final argument is for a "specific object" (like a particular AP),
		// which we leave as "/" to let NetworkManager decide.
		call := nm.Call(
			"org.freedesktop.NetworkManager.ActivateConnection",
			0,
			connectionPath,
			devicePath,
			dbus.ObjectPath("/"),
		)

		if call.Err != nil {
			return errMsg{fmt.Errorf("failed to activate connection: %w", call.Err)}
		}

		// We don't need to return a success message. If the call succeeds,
		// our D-Bus signal listener will automatically pick up the state
		// change and refresh the UI for us.
		return nil
	}
}