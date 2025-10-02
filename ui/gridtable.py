from textual.app import ComposeResult
from textual.containers import Grid
from textual.widgets import Static
from textual.widget import Widget
from textual.message import Message
from textual.errors import NoWidget
from rich.text import Text


class GridTableSelected(Message):
    """Posted when a row is selected in the GridTable."""

    def __init__(self, row_data: dict[str, str], row_index: int) -> None:
        self.row_data = row_data
        self.row_index = row_index
        super().__init__()


class GridTable(Widget):
    """
    A robust, reactive grid-based table that uses dynamic IDs for CSS styling,
    respecting Textual's widget lifecycle.
    """

    selected_row: int = 0

    def __init__(self, columns: dict[str, list[str]], *args, **kwargs):
        super().__init__(*args, **kwargs)
        self.columns = columns
        self.headers: list[Static] = []
        self.cells: list[Static] = []
        self.num_rows = max((len(vals) for vals in self.columns.values()), default=0)

    def compose(self) -> ComposeResult:
        """Compose the initial grid. This runs only once."""
        if not self.columns or not self.num_rows:
            yield Static("No data")
            return

        # Your original 'with' block was correct. It ensures the grid is mounted
        # before its children are yielded.
        with Grid(classes=f"grid{len(self.columns)}"):
            for title in self.columns.keys():
                header = Static(Text(title, style="bold"), classes="header")
                self.headers.append(header)
                yield header

            column_values = list(self.columns.values())
            for row_index in range(self.num_rows):
                cell_classes = "cell"
                if row_index == self.selected_row:
                    cell_classes += " selected-row"
                for col_vals in column_values:
                    value = col_vals[row_index] if row_index < len(col_vals) else ""
                    cell = Static(value, classes=cell_classes)
                    self.cells.append(cell)
                    yield cell

    def select_row(self, index: int):
        self.selected_row = index
        self.refresh_table()

    def update_data(self, new_columns: dict[str, list[str]]) -> None:
        """Update the table data, deciding whether to refresh or update."""
        new_num_rows = max((len(vals) for vals in new_columns.values()), default=0)

        if new_num_rows != self.num_rows or len(new_columns) != len(self.columns):
            self.columns = new_columns
            self.refresh_table()
            return

        # Fast path for same-structure updates
        self.columns = new_columns
        column_values = list(self.columns.values())
        cell_index = 0
        new_headers = list(new_columns.keys())
        for i, header in enumerate(self.headers):
            if i < len(new_headers):
                header.update(Text(new_headers[i], style="bold"))
        for row_index in range(new_num_rows):
            for col_vals in column_values:
                if cell_index < len(self.cells):
                    value = col_vals[row_index] if row_index < len(col_vals) else ""
                    self.cells[cell_index].update(value)
                    cell_index += 1

    def refresh_table(self) -> None:
        """Step 1: Safely remove the existing grid and schedule the rebuild."""
        try:
            # Find any existing Grid or placeholder Static and remove it.
            self.query("Grid, Static").first().remove()
        except NoWidget:
            pass  # Nothing to remove

        # Schedule Step 2 to run AFTER the removal has been processed.
        self.call_after_refresh(self._rebuild_grid)

    def _rebuild_grid(self) -> None:
        """Step 2: Build and mount the new grid. This runs on a clean slate."""
        self.headers.clear()
        self.cells.clear()

        if not self.columns:
            self.num_rows = 0
            self.mount(Static("No data to display."))
            return

        self.num_rows = max((len(v) for v in self.columns.values()), default=0)

        # 1. Create the new Grid container.
        new_grid = Grid(classes=f"grid{len(self.columns)}")

        # 2. Mount the container FIRST. This is the crucial fix.
        self.mount(new_grid)

        # 3. NOW that the grid is mounted, mount children INTO it.
        for title in self.columns.keys():
            header = Static(Text(title, style="bold"), classes="header")
            self.headers.append(header)
            new_grid.mount(header)

        column_values = list(self.columns.values())
        for row_index in range(self.num_rows):
            cell_classes = "cell"
            if row_index == self.selected_row:
                cell_classes += " selected-row"
            for col_vals in column_values:
                value = col_vals[row_index] if row_index < len(col_vals) else ""
                cell = Static(value, classes=cell_classes)
                self.cells.append(cell)
                new_grid.mount(cell)
