from pydbus import SystemBus


def get_known_networks():
    """
    Gets all known networks and their settings from NetworkManager via D-Bus.
    """
    all_connections = []
    bus = SystemBus()

    try:
        # Get the proxy object for the NetworkManager settings service
        settings_proxy = bus.get(
            "org.freedesktop.NetworkManager", "/org/freedesktop/NetworkManager/Settings"
        )

        # Ask NetworkManager to list all connection paths
        connection_paths = settings_proxy.ListConnections()

        for path in connection_paths:
            # For each connection path, get another proxy object
            con_proxy = bus.get("org.freedesktop.NetworkManager", path)

            # Ask the connection for its settings
            settings = con_proxy.GetSettings()

            # The settings are returned as a complex dictionary. We need to navigate it.
            connection_settings = settings.get("connection", {})
            wifi_settings = settings.get("802-11-wireless", {})
            security_settings = settings.get("802-11-wireless-security", {})

            # Only process wireless connections
            if connection_settings.get("type") == "802-11-wireless":
                # SSID is stored as a byte array, so we must decode it
                ssid_bytes = wifi_settings.get("ssid", [])
                ssid = bytes(ssid_bytes).decode("utf-8", errors="replace")

                all_connections.append(
                    {
                        "ssid": ssid,
                        "autoconnect": connection_settings.get("autoconnect", True),
                        "hidden": wifi_settings.get("hidden", False),
                        "security": security_settings.get("key-mgmt", "open"),
                    }
                )

        return all_connections

    except Exception as e:
        print(f"Error communicating with NetworkManager via D-Bus: {e}")
        print("Ensure NetworkManager service is running.")
        return None
