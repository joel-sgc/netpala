import subprocess
import psutil
import socket
import re
from typing import List, Dict, Optional, Tuple
from dataclasses import dataclass


@dataclass
class Network:
    ssid: str
    signal: str = "N/A"
    security: str = "Open"
    hidden: bool = False
    auto_connect: bool = False
    connected: bool = False


@dataclass
class Adapter:
    name: str
    mode: str = "Unknown"
    powered: bool = False
    connected: bool = False
    ipv4: Optional[str] = None
    ipv6: Optional[str] = None
    mac: Optional[str] = None
    ssid: str = "N/A"
    frequency: str = "N/A"
    security: str = "N/A"
    scanning: str = "N/A"


class NetworkScanner:
    """Unified network scanning and management."""

    WIFI_PREFIXES = ("wlan", "wlp")
    JUNK_PREFIXES = ("lo", "docker", "veth", "br-", "tailscale", "vpn", "tun")

    SECURITY_MAP = {
        "personal": "PSK",
        "psk": "PSK",
        "enterprise": "802.1x",
        "802.1x": "802.1x",
        "eap": "802.1x",
        "wep": "WEP",
        "open": "Open",
        "wpa": "PSK",
    }

    FREQUENCY_MAP = {"2": "2.4 GHz", "5": "5 GHz", "6": "6 GHz"}

    @staticmethod
    def map_security_term(term: Optional[str]) -> str:
        """Maps detailed security strings to simpler terms."""
        if not term:
            return "Open"

        term_lower = term.lower()
        for key, value in NetworkScanner.SECURITY_MAP.items():
            if key in term_lower:
                return value
        return term.strip()

    @staticmethod
    def has_wifi_adapter() -> bool:
        """Checks if any active Wi-Fi adapter exists."""
        try:
            stats = psutil.net_if_stats()
            return any(
                stat.isup and name.startswith(NetworkScanner.WIFI_PREFIXES)
                for name, stat in stats.items()
            )
        except Exception:
            return False

    @staticmethod
    def run_command(cmd: List[str], check: bool = False) -> Optional[str]:
        """Runs shell command with error handling."""
        try:
            if check:
                subprocess.run(
                    cmd,
                    check=True,
                    text=True,
                    stdout=subprocess.DEVNULL,
                    stderr=subprocess.DEVNULL,
                )
                return None
            result = subprocess.run(cmd, capture_output=True, text=True, check=False)
            return result.stdout if result.returncode == 0 else None
        except (subprocess.CalledProcessError, FileNotFoundError):
            return None

    def scan_networks(self, rescan: bool = False) -> List[Network]:
        """Scans for available Wi-Fi networks."""
        if not self.has_wifi_adapter():
            return []

        if rescan:
            self.run_command(["nmcli", "dev", "wifi", "rescan"], check=True)

        output = self.run_command(
            ["nmcli", "-t", "-f", "SSID,SIGNAL,SECURITY", "dev", "wifi"]
        )

        return self._parse_network_scan(output) if output else []

    def _parse_network_scan(self, output: str) -> List[Network]:
        """Parses nmcli network scan output."""
        networks = {}

        for line in output.strip().split("\n"):
            if not line:
                continue

            parts = line.split(":")
            if len(parts) >= 3:
                ssid, signal, security = (
                    parts[0],
                    parts[1],
                    self.map_security_term(parts[2]),
                )

                if not ssid:
                    continue

                key = (ssid, security)
                current_signal = int(signal)

                # Keep the strongest signal for each network
                if key not in networks or current_signal > int(
                    networks[key].signal.strip("%")
                ):
                    networks[key] = Network(
                        ssid=ssid, signal=f"{signal}%", security=security
                    )

        return list(networks.values())

    def get_known_networks(self) -> List[Network]:
        """Gets saved Wi-Fi networks with connection details."""
        if not self.has_wifi_adapter():
            return []

        active_ssid, active_signal = self._get_active_connection()
        connection_names = self._get_wifi_connections()

        return [
            self._build_network_info(name, active_ssid, active_signal)
            for name in connection_names
        ]

    def _get_active_connection(self) -> Tuple[Optional[str], str]:
        """Gets currently active Wi-Fi connection."""
        output = self.run_command(
            ["nmcli", "-t", "-f", "IN-USE,SSID,SIGNAL", "dev", "wifi"]
        )

        if output:
            for line in output.strip().split("\n"):
                parts = line.split(":")
                if len(parts) >= 3 and parts[0] == "*":
                    return parts[1], f"{parts[2]}%"
        return None, "N/A"

    def _get_wifi_connections(self) -> List[str]:
        """Gets list of saved Wi-Fi connection names."""
        output = self.run_command(
            ["nmcli", "-t", "-f", "NAME,TYPE", "connection", "show"]
        )

        if not output:
            return []

        return [
            name
            for name, conn_type in (
                line.split(":", 1) for line in output.strip().split("\n")
            )
            if conn_type == "802-11-wireless"
        ]

    def _build_network_info(
        self, name: str, active_ssid: str, active_signal: str
    ) -> Network:
        """Builds network information from connection details."""
        output = self.run_command(["nmcli", "connection", "show", "id", name])

        if not output:
            return Network(ssid=name)

        is_connected = name == active_ssid

        return Network(
            ssid=name,
            signal=active_signal if is_connected else "N/A",
            security=self._parse_security(output),
            hidden=self._parse_setting(output, "802-11-wireless.hidden"),
            auto_connect=self._parse_setting(output, "connection.autoconnect"),
            connected=is_connected,
        )

    def _parse_security(self, output: str) -> str:
        """Parses security setting from connection details."""
        match = re.search(
            r"^802-11-wireless-security\.key-mgmt:\s*([\w-]+)", output, re.MULTILINE
        )
        return self.map_security_term(match.group(1)) if match else "Open"

    def _parse_setting(self, output: str, setting: str) -> bool:
        """Parses boolean settings from connection details."""
        return bool(re.search(f"^{setting}:\\s*yes", output, re.MULTILINE))

    def get_adapters(self) -> List[Adapter]:
        """Gets detailed information for all network adapters."""
        adapters = []
        all_addrs = psutil.net_if_addrs()
        all_stats = psutil.net_if_stats()

        for name, addrs in all_addrs.items():
            if self._is_junk_adapter(name) or name not in all_stats:
                continue

            stats = all_stats[name]
            adapter = self._build_adapter_info(name, stats, addrs)

            # Only include adapters with IPv4 or Wi-Fi
            if adapter.ipv4 or self._is_wifi_adapter(name):
                adapters.append(adapter)

        return adapters

    def _is_junk_adapter(self, name: str) -> bool:
        """Checks if adapter should be ignored."""
        return any(name.startswith(prefix) for prefix in self.JUNK_PREFIXES)

    def _is_wifi_adapter(self, name: str) -> bool:
        """Checks if adapter is a Wi-Fi interface."""
        return any(name.startswith(prefix) for prefix in self.WIFI_PREFIXES)

    def _build_adapter_info(self, name: str, stats, addrs) -> Adapter:
        """Builds adapter information from system data."""
        is_wifi = self._is_wifi_adapter(name)
        is_ethernet = name.startswith(("en", "eth")) or "ethernet" in name.lower()

        # Determine adapter type and basic status
        if is_wifi:
            mode = "Wi-Fi"
            connected = False  # Will be updated by Wi-Fi details
        elif is_ethernet:
            mode = "Ethernet"
            connected = stats.isup
        else:
            mode = "Open"
            connected = stats.isup

        # Build base adapter
        adapter = Adapter(
            name=name,
            mode=mode,
            powered=stats.isup,
            connected=connected,
            **self._parse_addresses(addrs),
        )

        # Add Wi-Fi specific details
        if is_wifi:
            wifi_details = self._get_wifi_details(name)
            adapter.connected = wifi_details["connected"]
            adapter.ssid = wifi_details["ssid"] or "N/A"
            adapter.frequency = wifi_details["frequency"] or "N/A"
            adapter.security = wifi_details["security"] or "N/A"
            adapter.scanning = "Yes" if not adapter.connected else "No"

        return adapter

    def _parse_addresses(self, addrs) -> Dict[str, Optional[str]]:
        """Extracts IP and MAC addresses from adapter addresses."""
        info = {"ipv4": None, "ipv6": None, "mac": None}

        for addr in addrs:
            if addr.family == socket.AF_INET:
                info["ipv4"] = addr.address
            elif addr.family == socket.AF_INET6:
                info["ipv6"] = addr.address
            elif addr.family == psutil.AF_LINK:
                info["mac"] = addr.address

        return info

    def _get_wifi_details(self, iface_name: str) -> Dict[str, Optional[str]]:
        """Gets detailed Wi-Fi information for an interface."""
        details = {
            "connected": False,
            "frequency": None,
            "ssid": None,
            "mode": None,
            "security": None,
        }

        # Get interface mode
        info_output = self.run_command(["iw", "dev", iface_name, "info"])
        if info_output and (match := re.search(r"\s*type (\w+)", info_output)):
            details["mode"] = match.group(1)

        # Get active connection info
        active_output = self.run_command(
            ["nmcli", "-t", "-f", "ACTIVE,SSID,SECURITY,FREQ", "dev", "wifi"]
        )

        if active_output:
            for line in active_output.strip().split("\n"):
                parts = line.split(":")
                if len(parts) >= 4 and parts[0] == "yes":
                    details.update(
                        {
                            "connected": True,
                            "ssid": parts[1],
                            "security": self.map_security_term(parts[2]),
                            "frequency": self.FREQUENCY_MAP.get(parts[3][0], "N/A"),
                        }
                    )
                    break

        return details


class NetworkFormatter:
    """Formats network data for display."""

    @staticmethod
    def format_networks(networks: List[Network]) -> Dict[str, List]:
        """Formats networks into column-based structure."""
        return {
            "ssid": [n.ssid for n in networks],
            "signal": [n.signal for n in networks],
            "security": [n.security for n in networks],
        }

    @staticmethod
    def format_known_networks(networks: List[Network]) -> Dict[str, List]:
        """Formats known networks into column-based structure."""
        return {
            "ssid": [n.ssid for n in networks],
            "security": [n.security for n in networks],
            "signal": [n.signal for n in networks],
            "hidden": ["yes" if n.hidden else "no" for n in networks],
            "auto_connect": ["on" if n.auto_connect else "off" for n in networks],
            "connected": ["✓" if n.connected else "" for n in networks],
        }

    @staticmethod
    def format_adapters(adapters: List[Adapter]) -> Dict[str, List]:
        """Formats adapters into column-based structure."""
        return {
            "name": [a.name for a in adapters],
            "mode": [a.mode for a in adapters],
            "powered": ["on" if a.powered else "off" for a in adapters],
            "connected": [
                "connected" if a.connected else "disconnected" for a in adapters
            ],
            "ipv4": [a.ipv4 or "n/a" for a in adapters],
            "ipv6": [a.ipv6 or "n/a" for a in adapters],
            "mac": [a.mac or "n/a" for a in adapters],
            "ssid": [a.ssid for a in adapters],
            "frequency": [a.frequency for a in adapters],
            "security": [a.security for a in adapters],
            "scanning": [a.scanning for a in adapters],
        }


# Global instances for easy access
scanner = NetworkScanner()
formatter = NetworkFormatter()


# Simplified public API - KEEP THE ORIGINAL SYNCHRONOUS API
def get_found_networks():
    """Network scanning - synchronous version for compatibility."""
    return scanner.scan_networks(True)


def get_cached_found_networks():
    """Cached network scanning."""
    return scanner.scan_networks(False)


def get_known_networks():
    """Get saved networks."""
    return scanner.get_known_networks()


def get_network_adapters():
    """Get adapter information."""
    return formatter.format_adapters(scanner.get_adapters())


def format_networks_by_column(networks):
    """Format networks for display."""
    return formatter.format_networks(networks)


def format_known_networks_by_column(networks):
    """Format known networks for display."""
    return formatter.format_known_networks(networks)


# Backward compatibility
map_security_term = NetworkScanner.map_security_term
_is_wifi_adapter_present_and_up = NetworkScanner.has_wifi_adapter
_sync_get_found_networks = lambda: scanner.scan_networks(True)
parse_nmcli_linux = lambda iface_name: scanner._get_wifi_details(iface_name)
async_format_networks_by_column = format_networks_by_column
