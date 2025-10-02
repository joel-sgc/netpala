from textual.worker import Worker, WorkerState
from textual.app import App, ComposeResult
from textual.containers import Vertical
import asyncio

from ui.box import Box
from net_data.adapter_data import get_adapter_data
from net_data.saved_networks import get_known_networks
from net_data.found_networks import get_scanned_networks


class FieldsetUI(App):
    CSS = """
        Screen { background: #1b1b26; }
        Box { height: auto; min-height: 1; margin-bottom: 1; }
        GridTable { overflow-y: auto; height: auto; }
        .grid4 { grid-size: 4; height: auto; grid-gutter: 0; }
        .grid3 { grid-size: 3; height: auto; grid-gutter: 0; }
        .grid5 { grid-size: 5; height: auto; grid-gutter: 0; }
        .header { text-align: center; padding: 0 1 1 1; color: #a7abca; }
        .cell { height: 1; text-align: center; color: #a7abca }
        .selected-header .header { color: #cda162; }
        .selected-header .cell.selected-row { background: #444a66; }
    """

    def __init__(self):
        super().__init__()
        self.selected_box = 0
        self.current_network = None

        self.known_networks = []
        self.found_networks = []
        self.is_connecting = False
        # Initialize boxes with placeholder data for a fast startup

        self.boxes = [
            Box(
                columns={
                    "Name": ["-"],
                    "Mode": ["-"],
                    "Powered": ["-"],
                    "Address": ["-"],
                },
                title="Device",
                is_active=True,
                id="device_box",
                return_key="Name",
            ),
            Box(
                columns={
                    "State": ["-"],
                    "Scanning": ["-"],
                    "Frequency": ["-"],
                    "Security": ["-"],
                },
                title="Station",
                id="station_box",
                return_key="State",
            ),
            Box(
                columns={
                    "": ["-"],
                    "Name": ["-"],
                    "Security": ["-"],
                    "Hidden": ["-"],
                    "Auto Connect": ["-"],
                },
                title="Known Networks",
                id="known_networks_box",
                return_key="Name",
            ),
            Box(
                columns={"Name": ["-"], "Security": ["-"], "Signal": ["-"]},
                title="New Networks",
                id="new_networks_box",
                return_key="Name",
            ),
        ]

    def compose(self) -> ComposeResult:
        with Vertical(id="container"):
            yield from self.boxes

    def fetch_adapters_data(self) -> list:
        """Synchronous adapter data fetch"""
        self.log("Starting adapters data fetch")
        try:
            result = get_adapter_data()
            self.log(f"Adapters data fetched: {len(result)} adapters")
            return result
        except Exception as e:
            self.log(f"Error fetching adapters: {e}")
            return []

    def fetch_known_networks_data(self) -> list:
        """Synchronous known networks data fetch"""
        self.log("Starting known networks data fetch")
        try:
            result = get_known_networks()
            self.log(f"Known networks fetched: {len(result)} networks")
            return result
        except Exception as e:
            self.log(f"Error fetching known networks: {e}")
            return []

    def fetch_found_networks_data(self) -> dict:
        """Synchronous found networks data fetch"""
        self.log("Starting found networks data fetch")
        try:
            result = get_scanned_networks()
            self.log(f"Found networks fetched: {len(result)} networks")
            return result
        except Exception as e:
            self.log(f"Error fetching found networks: {e}")
            return []

    async def on_mount(self) -> None:
        """
        Start background workers and set up periodic refresh with different intervals
        """
        self.log("Mounting app, starting workers...")

        # Initial data load
        await self.refresh_all_data()

        # Set up different refresh rates for different data types
        self.set_interval(2, self.refresh_adapters)
        self.set_interval(5, self.refresh_networks)
        self.set_interval(10, self.refresh_known_networks)

        self.log("All workers started and auto-refresh enabled")

    async def refresh_all_data(self) -> None:
        """Refresh all data sources"""
        self.log("Refreshing all data...")

        # Start all workers with thread=True since they're synchronous functions
        self.run_worker(
            self.fetch_adapters_data,
            name="adapters_loader",
            thread=True,
            exclusive=False,
        )
        self.run_worker(
            self.fetch_known_networks_data,
            name="known_networks_loader",
            thread=True,
            exclusive=False,
        )
        self.run_worker(
            self.fetch_found_networks_data,
            name="found_networks_loader",
            thread=True,
            exclusive=False,
        )

    async def refresh_adapters(self) -> None:
        """Refresh only adapter data"""
        self.run_worker(
            self.fetch_adapters_data,
            name="adapters_loader",
            thread=True,
            exclusive=False,
        )

    async def refresh_networks(self) -> None:
        """Refresh only found networks"""
        self.run_worker(
            self.fetch_found_networks_data,
            name="found_networks_loader",
            thread=True,
            exclusive=False,
        )

    async def refresh_known_networks(self) -> None:
        """Refresh only known networks"""
        self.run_worker(
            self.fetch_known_networks_data,
            name="known_networks_loader",
            thread=True,
            exclusive=False,
        )

    def on_worker_state_changed(self, event: Worker.StateChanged) -> None:
        """
        Update the UI with fresh data as it arrives
        """
        worker = event.worker
        self.log(f"Worker {worker.name} state: {worker.state}")

        if worker.state == WorkerState.SUCCESS:
            if worker.name == "adapters_loader":
                adapters_data = worker.result
                for item in adapters_data:
                    if item["state"] == "connected":
                        self.current_network = item["ssid"]

                self.log(f"Updating adapters UI with {len(adapters_data)} items")

                # Update device box
                device_box = self.query_one("#device_box")
                if device_box:
                    device_box.update_columns(
                        {
                            "Name": [item["name"] for item in adapters_data],
                            "Mode": [item["mode"] for item in adapters_data],
                            "Powered": [
                                "On" if item["powered"] else "Off"
                                for item in adapters_data
                            ],
                            "Address": [item["address"] for item in adapters_data],
                        }
                    )

                # Update station box
                station_box = self.query_one("#station_box")
                if station_box:
                    station_box.update_columns(
                        {
                            "State": [item["state"] for item in adapters_data],
                            "Scanning": [
                                str(item["scanning"]).lower() for item in adapters_data
                            ],
                            "Frequency": [item["frequency"] for item in adapters_data],
                            "Security": [item["security"] for item in adapters_data],
                        }
                    )
                self.log("Adapters UI updated")

            elif worker.name == "known_networks_loader":
                known_data = worker.result
                self.known_networks = known_data
                self.log(f"Updating known networks UI with {len(known_data)} items")

                known_box = self.query_one("#known_networks_box")
                if known_box:
                    known_box.update_columns(
                        {
                            "": [
                                "✓" if item["ssid"] == self.current_network else ""
                                for item in known_data
                            ],
                            "Name": [item["ssid"] for item in known_data],
                            "Security": [item["security"] for item in known_data],
                            "Hidden": [
                                str(item["hidden"]).lower() for item in known_data
                            ],
                            "Auto Connect": [
                                str(item["autoconnect"]).lower() for item in known_data
                            ],
                        }
                    )
                self.log("Known networks UI updated")

            elif worker.name == "found_networks_loader":
                found_data = worker.result
                self.found_networks = found_data
                self.log(f"Updating found networks UI with {len(found_data)} items")

                found_box = self.query_one("#new_networks_box")
                if found_box:
                    found_box.update_columns(
                        {
                            "Name": [item["ssid"] for item in found_data],
                            "Security": [item["security"] for item in found_data],
                            "Signal": [f"{item["signal"]}%" for item in found_data],
                        }
                    )
                    self.log("Found networks UI updated")

        elif worker.state == WorkerState.ERROR:
            self.log(f"Worker {worker.name} failed with error: {worker.error}")

    async def connect_to_network(self, network_name: str) -> None:
        """Connect to a network asynchronously without blocking the UI"""
        if self.is_connecting:
            self.log("Already connecting to a network, please wait...")
            return

        self.is_connecting = True
        known_box = self.query_one("#known_networks_box")
        known_box.add_class("connecting")
        known_box.border_title = f"Connecting to {network_name}..."

        try:
            # Run nmcli asynchronously
            process = await asyncio.create_subprocess_exec(
                "nmcli",
                "connection",
                "up",
                network_name,
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE,
            )

            stdout, stderr = await process.communicate()

            if process.returncode == 0:
                self.log(f"Successfully connected to {network_name}")
                known_box.border_title = f"Connected to {network_name} ✓"
                # Refresh adapter data to show new connection state
                await self.refresh_adapters()
            else:
                error_msg = stderr.decode().strip() if stderr else "Unknown error"
                self.log(f"Failed to connect to {network_name}: {error_msg}")
                known_box.border_title = f"Failed to connect to {network_name}"

        except Exception as e:
            self.log(f"Error running nmcli: {e}")
            known_box.border_title = f"Error: {str(e)}"
        finally:
            self.is_connecting = False
            known_box.remove_class("connecting")
            # Reset title after a delay
            self.set_timer(
                3, lambda: setattr(known_box, "border_title", "Known Networks")
            )

    def on_key(self, event) -> None:
        if event.key not in ("tab", "shift+tab", "up", "down", "space", "enter"):
            return

        num_boxes = len(self.boxes)
        if num_boxes < 2:
            return

        # Deactivate the current box
        self.boxes[self.selected_box].set_active(False)

        # Navigation between boxes
        if event.key == "shift+tab":
            self.selected_box = (self.selected_box - 1 + num_boxes) % num_boxes
        elif event.key == "tab":
            self.selected_box = (self.selected_box + 1) % num_boxes

        # Navigation within boxes
        if event.key == "up":
            self.boxes[self.selected_box].shift_row(-1)
        elif event.key == "down":
            self.boxes[self.selected_box].shift_row(1)

        # Connect to network on space or enter (for known networks box)
        if (
            event.key in ("space", "enter") and self.selected_box == 2
        ):  # Known Networks box
            if self.is_connecting:
                self.log("Please wait, already connecting to a network...")
                return

            value = self.boxes[self.selected_box].get_value()
            if value and value != "-":  # Only connect if there's a real network name
                self.log(f"Attempting to connect to: {value}")
                # Run the connection asynchronously
                self.run_worker(
                    self.connect_to_network(value),
                    name="network_connector",
                    exclusive=False,
                )

        # Activate the new box
        self.boxes[self.selected_box].set_active(True)


if __name__ == "__main__":
    FieldsetUI().run()
