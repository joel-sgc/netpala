import asyncio
from textual.app import App, ComposeResult
from textual.widgets import Header, Footer, Static
from textual.worker import Worker, WorkerState


class AsyncApp(App):
    """An app demonstrating how to update the UI from a background task."""

    CSS = """
    Screen {
        align: center middle;
    }
    #content {
        width: 80%;
        height: 10;
    }
    """

    def compose(self) -> ComposeResult:
        yield Header()
        yield Static("Loading...", id="content")
        yield Footer()

    async def fetch_new_data(self) -> str:
        """A simulated long-running async function (e.g., an API call)."""
        await asyncio.sleep(3)  # Simulate a 3-second network delay
        return "✨ Fresh data from the network! ✨"

    async def on_mount(self) -> None:
        """Called when the app is first mounted."""
        # 1. Display the initial cached data immediately
        self.query_one("#content").update("Displaying cached data...")

        # 2. Start the async function in the background
        self.run_worker(self.fetch_new_data, name="data_loader")

    def on_worker_state_changed(self, event: Worker.StateChanged) -> None:
        """
        Called when the background worker's state changes.
        This is where you get the result and update the UI.
        """
        # 3. Check if the worker that finished is our data loader
        if event.worker.name == "data_loader":
            # 4. Check if the worker finished successfully
            if event.worker.state == WorkerState.SUCCESS:
                # Get the result from the worker
                new_data = event.worker.result

                # Update the Static widget with the new data
                self.query_one("#content").update(new_data)
            # You could also handle the ERROR state here
            elif event.worker.state == WorkerState.ERROR:
                self.query_one("#content").update("⚠️ Error fetching data!")


if __name__ == "__main__":
    AsyncApp().run()
