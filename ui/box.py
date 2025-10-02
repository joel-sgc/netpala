from textual.containers import Vertical
from ui.gridtable import GridTable

green = "#9cca69"
gold = "#cda162"
gray = "#a7abca"


class Box(Vertical):
    """A container widget with built-in GridTable support."""

    is_active: bool = False
    selected_row: int = 0

    def __init__(
        self,
        is_active: bool = False,
        columns: dict[str, list[str]] = None,
        title: str = "",
        return_key: str = "",
        **kwargs,
    ):
        super().__init__(**kwargs)
        self.is_active = is_active
        self.columns = columns or {}
        self.title = title
        self.return_key = return_key
        self.grid_table = GridTable(columns=self.columns)

    def compose(self):
        """Yield the GridTable."""
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
            self.grid_table.select_row(self.selected_row)
        else:
            self.styles.border = ("solid", gray)
            self.remove_class("selected-header")

    def shift_row(self, offset: int):
        """Shift the selected row with bounds checking."""
        if self.grid_table.num_rows == 0:
            return

        self.selected_row = max(
            0, min(self.selected_row + offset, self.grid_table.num_rows - 1)
        )
        self.grid_table.select_row(self.selected_row)

    def update_columns(self, new_columns: dict[str, list[str]]):
        """Only update if data actually changed"""
        if self._has_data_changed(new_columns):
            self.columns = new_columns
            self.grid_table.update_data(new_columns)
            # Reset selection if we have new data
            if (
                self.grid_table.num_rows > 0
                and self.selected_row >= self.grid_table.num_rows
            ):
                self.selected_row = 0
                self.grid_table.select_row(self.selected_row)

    def _has_data_changed(self, new_columns: dict[str, list[str]]) -> bool:
        """Check if the new data is different from current data"""
        if set(self.columns.keys()) != set(new_columns.keys()):
            return True

        for key in self.columns:
            if self.columns[key] != new_columns[key]:
                return True

        return False

    def get_value(self) -> str:
        return self.columns[self.return_key][self.selected_row]
