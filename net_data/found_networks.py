from pydbus import SystemBus


def get_wifi_security_type(flags, wpa_flags, rsn_flags):
    """
    Determines the security type of a Wi-Fi network based on its flags.
    Uses corrected bitmask values based on actual NetworkManager flags.
    """
    # CORRECTED key management bitmask values
    KEY_MGMT_PSK = 0x100  # WPA-Personal / Pre-shared Key (from 0x188 in debug)
    KEY_MGMT_EAP = 0x200  # WPA-Enterprise / 802.1X Authentication (from 0x288 in debug)

    # Check for WPA-PSK (Personal)
    if (wpa_flags & KEY_MGMT_PSK) or (rsn_flags & KEY_MGMT_PSK):
        return "wpa-psk"

    # Check for WPA-EAP (Enterprise)
    if (wpa_flags & KEY_MGMT_EAP) or (rsn_flags & KEY_MGMT_EAP):
        return "wpa-eap"

    # Check for WEP (Privacy flag set but no WPA flags)
    if flags & 0x1:  # NM_802_11_AP_FLAGS_PRIVACY
        return "wep"

    # No security flags found
    return "open"


def get_scanned_networks(interface="wlan0"):
    """
    Gets scanned Wi-Fi networks via D-Bus, filtering duplicates for the strongest signal.
    """
    bus = SystemBus()

    try:
        nm_proxy = bus.get(
            "org.freedesktop.NetworkManager", "/org/freedesktop/NetworkManager"
        )
        device_path = nm_proxy.GetDeviceByIpIface(interface)
        dev_proxy = bus.get("org.freedesktop.NetworkManager", device_path)

        if dev_proxy.DeviceType != 2:  # NM_DEVICE_TYPE_WIFI
            print(f"Error: Device '{interface}' is not a Wi-Fi device.")
            return None

        ap_paths = dev_proxy.GetAllAccessPoints()

        # Use a dictionary to store the best access point for each SSID
        best_aps = {}

        for path in ap_paths:
            ap_proxy = bus.get("org.freedesktop.NetworkManager", path)

            ssid_bytes = bytes(ap_proxy.Ssid).decode("utf-8", errors="replace")
            signal_strength = ap_proxy.Strength
            flags = ap_proxy.Flags
            wpa_flags = ap_proxy.WpaFlags
            rsn_flags = ap_proxy.RsnFlags

            # Skip hidden networks (empty SSID)
            if not ssid_bytes:
                continue

            security_type = get_wifi_security_type(flags, wpa_flags, rsn_flags)
            print(
                f"Debug: SSID='{ssid_bytes}', Flags={flags}, WpaFlags=0x{wpa_flags:x}, RsnFlags=0x{rsn_flags:x}"
            )

            # If we haven't seen this SSID, or if the new signal is stronger, update it
            if (
                ssid_bytes not in best_aps
                or signal_strength > best_aps[ssid_bytes]["signal"]
            ):
                best_aps[ssid_bytes] = {
                    "ssid": ssid_bytes,
                    "signal": signal_strength,
                    "security": security_type,
                }

        scanned_networks = sorted(
            best_aps.values(), key=lambda x: x["signal"], reverse=True
        )
        return scanned_networks

    except Exception as e:
        print(f"Error communicating with NetworkManager via D-Bus: {e}")
        return None
