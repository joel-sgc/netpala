from textual.containers import Vertical
from ui.gridtable import GridTable

green = "#9cca69"
gold = "#cda162"
gray = "#a7abca"


class Box(Vertical):
    """A container widget with built-in GridTable support."""

    is_active: bool = False

    def __init__(
        self,
        is_active: bool = False,
        columns: dict[str, list[str]] = None,
        title: str = "",
        **kwargs,
    ):
        super().__init__(**kwargs)

        self.is_active = is_active
        self.columns = columns or {}
        self.title = title

        # FIX 1: Always create a GridTable instance. This ensures
        # self.grid_table is never None.
        self.grid_table = GridTable(columns=self.columns)

    def compose(self):
        """Yield the GridTable."""
        # This will now always yield a valid GridTable widget.
        yield self.grid_table

    def on_mount(self):
        """Set up border and title after mounting."""
        self.set_active(self.is_active)
        if self.title:
            self.border_title = self.title

    def set_active(self, active: bool = False):
        """Sets the active state for the box."""
        self.is_active = active
        if active:
            self.styles.border = ("solid", green)
            self.add_class("selected-header")
        else:
            self.styles.border = ("solid", gray)
            self.remove_class("selected-header")

    def update_columns(self, new_columns: dict[str, list[str]]):
        """Only update if data actually changed"""
        # if self._has_data_changed(new_columns):
        self.columns = new_columns
        self.grid_table.update_data(new_columns)

    def _has_data_changed(self, new_columns: dict[str, list[str]]) -> bool:
        """Check if the new data is different from current data"""
        if set(self.columns.keys()) != set(new_columns.keys()):
            return True

        for key in self.columns:
            if self.columns[key] != new_columns[key]:
                return True

        return False
