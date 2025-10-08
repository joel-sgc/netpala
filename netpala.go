package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)



func NetpalaModel() netpala_data {
	return netpala_data{
        selected_box: 0,
        selected_entry: 0,
        device_data: get_devices_data(),
        known_networks: get_known_networks(),
        scanned_networks: get_scanned_networks(),
    }
}

func (m netpala_data) Init() tea.Cmd {
    // Just return `nil`, which means "no I/O right now, please."
    return nil
}

func (m netpala_data) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {

    // Is it a key press?
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
    case tea.KeyMsg:

        // Cool, what was the actual key pressed?
        switch msg.String() {

        // These keys should exit the program.
        case "ctrl+c", "ctrl+q", "q", "ctrl+w":
            return m, tea.Quit

        // "up" and "k" for upwards cursor nav, "down" and "j" for downwards cursor nav within our box
        case "up", "k":
            if m.selected_entry > 0 {
                m.selected_entry--
            }
        case "down", "j":
            boxes := []int{len(m.device_data), len(m.device_data), len(m.known_networks), len(m.scanned_networks)}

            if m.selected_entry < boxes[m.selected_box] - 1 {
                // Type application doesn't change anything here, just for linting
                m.selected_entry++
            }
        
        // "shift+tab" for upwards box nav, "tab" for downwards box nav
        case "shift+tab":
            if m.selected_box > 0 {
                m.selected_box--
                m.selected_entry = 0
            }
        case "tab":
            if m.selected_box < 3 {
                m.selected_box ++
                m.selected_entry = 0
            }

        // The "enter" key and the spacebar (a literal space) toggle
        // the selected state for the item that the cursor is pointing at.
        // case "enter", " ":
        //     _, ok := m.selected[m.cursor]
        //     if ok {
        //         delete(m.selected, m.cursor)
        //     } else {
        //         m.selected[m.cursor] = struct{}{}
        //     }
        }
    }

    // Return the updated model to the Bubble Tea runtime for processing.
    // Note that we're not returning a command.
    return m, nil
}

func (m netpala_data) View() string {
    border_style_device := inactive_border_style
    border_style_station := inactive_border_style
    border_style_known_networks := inactive_border_style
    border_style_scanned_networks := inactive_border_style

    switch m.selected_box {
        case 0:
            border_style_device = active_border_style
        case 1:
            border_style_station = active_border_style
        case 2:
            border_style_known_networks = active_border_style
        case 3:
            border_style_scanned_networks = active_border_style
    }

    device_table_data := format_device_data(m.device_data)
    device_table := table.New().
        Border(box_border).
        BorderColumn(false).
        BorderStyle(border_style_device).
        StyleFunc(box_style(m.selected_entry, m.selected_box == 0)).
        Rows( device_table_data... )

    station_table_data := format_station_data(m.device_data)
    station_table := table.New().
        Border(box_border).
        BorderColumn(false).
        BorderStyle(border_style_station).
        StyleFunc(box_style(m.selected_entry, m.selected_box == 1)).
        Rows( station_table_data... )

    known_networks_table_data := format_known_networks_data(m.known_networks, m.selected_entry)
    known_networks_table := table.New().
        Border(box_border).
        BorderColumn(false).
        BorderStyle(border_style_known_networks).
        StyleFunc(box_style(m.selected_entry, m.selected_box == 2)).
        Rows( known_networks_table_data... )

    scanned_networks_table_data := format_scanned_networks_data(m.scanned_networks, m.selected_entry)
    scanned_networks_table := table.New().
        Border(box_border).
        BorderColumn(false).
        BorderStyle(border_style_scanned_networks).
        StyleFunc(box_style(m.selected_entry, m.selected_box == 3)).
        Rows( scanned_networks_table_data... )

    return  (calc_title("Device", m.selected_box == 0) + device_table.Render()) + "\n" + 
            (calc_title("Station", m.selected_box == 1) + station_table.Render()) + "\n" + 
            (calc_title("Known Networks", m.selected_box == 2) + lipgloss.NewStyle().Render(known_networks_table.Render())) + "\n" + 
            (calc_title("New Networks", m.selected_box == 3) + lipgloss.NewStyle().Render(scanned_networks_table.Render()))
}

func main() {
    p := tea.NewProgram(NetpalaModel())
    if _, err := p.Run(); err != nil {
        fmt.Printf("Alas, there's been an error: %v", err)
        os.Exit(1)
    }
}