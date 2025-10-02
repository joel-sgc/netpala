from textual.app import ComposeResult
from textual.containers import Grid
from textual.widgets import Static
from textual.widget import Widget
from rich.text import Text


class GridTable(Widget):
    """
    A robust, reactive grid-based table with optimized updates to prevent flashing.
    """

    selected_row: int = 0

    def __init__(self, columns: dict[str, list[str]], *args, **kwargs):
        super().__init__(*args, **kwargs)
        self.columns = columns
        self.headers: list[Static] = []
        self.cells: list[Static] = []
        self.num_rows = max((len(vals) for vals in self.columns.values()), default=0)
        self._grid_container = None

    def compose(self) -> ComposeResult:
        """Compose the initial grid. This runs only once."""
        self._grid_container = Grid(classes=f"grid{len(self.columns)}")
        yield self._grid_container

    def on_mount(self) -> None:
        """Build the grid after mounting to prevent flashing."""
        self._build_grid()

    def _build_grid(self) -> None:
        """Build or rebuild the grid content without removing the container."""
        if not self._grid_container:
            return

        # Clear existing content
        self.headers.clear()
        self.cells.clear()
        self._grid_container.remove_children()

        if not self.columns or not self.num_rows:
            self._grid_container.mount(Static("No data"))
            return

        # Build headers
        for title in self.columns.keys():
            header = Static(Text(title, style="bold"), classes="header")
            self.headers.append(header)
            self._grid_container.mount(header)

        # Build cells
        column_values = list(self.columns.values())
        for row_index in range(self.num_rows):
            cell_classes = "cell"
            if row_index == self.selected_row:
                cell_classes += " selected-row"
            for col_vals in column_values:
                value = col_vals[row_index] if row_index < len(col_vals) else ""
                cell = Static(value, classes=cell_classes)
                self.cells.append(cell)
                self._grid_container.mount(cell)

    def select_row(self, index: int):
        """Update row selection without full rebuild."""
        if index < 0 or index >= self.num_rows:
            return

        self.selected_row = index

        # Update selection styling without rebuilding entire table
        cell_index = 0
        column_count = len(self.columns)

        for row_index in range(self.num_rows):
            for col_index in range(column_count):
                if cell_index < len(self.cells):
                    cell = self.cells[cell_index]
                    cell_classes = "cell"
                    if row_index == self.selected_row:
                        cell_classes += " selected-row"
                    cell.set_classes(cell_classes)
                    cell_index += 1

    def update_data(self, new_columns: dict[str, list[str]]) -> None:
        """Update table data with minimal DOM changes."""
        new_num_rows = max((len(vals) for vals in new_columns.values()), default=0)
        new_column_count = len(new_columns.keys())
        current_column_count = len(self.columns.keys())

        # Only rebuild if structure changed significantly
        if (
            new_num_rows != self.num_rows
            or new_column_count != current_column_count
            or set(new_columns.keys()) != set(self.columns.keys())
        ):

            self.columns = new_columns
            self.num_rows = new_num_rows
            self._build_grid()
            return

        # Fast path: update existing cells
        self.columns = new_columns
        column_values = list(new_columns.values())

        # Update headers if needed
        new_headers = list(new_columns.keys())
        for i, header in enumerate(self.headers):
            if i < len(new_headers):
                header.update(Text(new_headers[i], style="bold"))

        # Update cell content
        cell_index = 0
        for row_index in range(new_num_rows):
            for col_vals in column_values:
                if cell_index < len(self.cells):
                    value = col_vals[row_index] if row_index < len(col_vals) else ""
                    self.cells[cell_index].update(value)

                    # Update selection styling
                    cell_classes = "cell"
                    if row_index == self.selected_row:
                        cell_classes += " selected-row"
                    self.cells[cell_index].set_classes(cell_classes)

                    cell_index += 1
