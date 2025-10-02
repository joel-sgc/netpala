import subprocess
import os
import re


def list_all_wifi_interfaces():
    """Return list of all Wi-Fi interfaces on the system."""
    interfaces = os.listdir("/sys/class/net")
    wifi_interfaces = [
        iface
        for iface in interfaces
        if os.path.isdir(f"/sys/class/net/{iface}/wireless")
    ]
    return wifi_interfaces


def translate_frequency(freq_str: str) -> str:
    """Convert frequency in MHz to Wi-Fi band string."""
    try:
        freq = int(freq_str)
        if 2400 <= freq < 2500:
            return "2.4GHz"
        elif 4900 <= freq < 5900:
            return "5GHz"
        elif 5925 <= freq < 7125:
            return "6GHz"
        else:
            return f"{freq}MHz"
    except (ValueError, TypeError):
        return "-"


def get_adapter_data():
    """
    Try to get status from wpa_cli.
    If wpa_cli fails (radio off or not managed), still return interfaces as powered=False.
    """
    wifi_interfaces = list_all_wifi_interfaces()

    try:
        result = subprocess.run(
            ["sudo", "wpa_cli", "status"], text=True, capture_output=True
        )

        # If wifi is off
        if result.returncode != 0 or "Failed to connect" in result.stderr:
            return [
                {
                    "name": iface,
                    "mode": "-",
                    "powered": False,
                    "address": "-",
                    "state": "disconnected",
                    "scanning": False,
                    "frequency": "-",
                    "security": "-",
                }
                for iface in wifi_interfaces
            ]

        # Otherwise parse output
        raw_status = {}
        iface_name = None
        for line in result.stdout.strip().split("\n"):
            if line.startswith("Selected interface '"):
                match = re.search(r"'([^']*)'", line)
                if match:
                    iface_name = match.group(1)
            elif "=" in line:
                key, value = line.split("=", 1)
                raw_status[key] = value

        freq_str = raw_status.get("freq", "-")
        band = translate_frequency(freq_str)

        # Normalize into same schema
        return [
            {
                "name": iface_name or "-",
                "mode": raw_status.get("mode", "-"),
                "powered": True,
                "address": raw_status.get("address", "-"),
                "state": (
                    "connected"
                    if raw_status.get("wpa_state") == "COMPLETED"
                    else "disconnected"
                ),
                "scanning": raw_status.get("scanning", "0") == "1",
                "frequency": band,
                "security": raw_status.get("key_mgmt", "-"),
                "ssid": raw_status.get("ssid", "-"),
            }
        ]

    except Exception as e:
        print("Unexpected error:", e)
        return [
            {
                "name": iface,
                "mode": "-",
                "powered": False,
                "address": "-",
                "state": "disconnected",
                "scanning": False,
                "frequency": "-",
                "security": "-",
            }
            for iface in wifi_interfaces
        ]
