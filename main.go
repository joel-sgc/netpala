package main

import (
	"fmt"
	"os"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss/table"
)



func NetpalaModel() netpala_data {
	return default_netpala_data
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
            if m.selected_entry > 2 {
                m.selected_entry--
            }
        case "down", "j":
            boxes := []any{m.device_data, m.known_networks, m.scanned_networks}

            if m.selected_entry < len(boxes[m.selected_box].([]device)) + 2 {
                // Type application doesn't change anything here, just for linting
                m.selected_entry++
            }
        
        // "shift+tab" for upwards box nav, "tab" for downwards box nav
        case "shift+tab":
            if m.selected_box > 0 {
                m.selected_box--
            }
        case "tab":
            if m.selected_box < 2 {
                m.selected_box ++
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

    tbl := table.New().
        Border(box_border).
        BorderColumn(false).
        BorderStyle(active_border_style).
        StyleFunc(box_style(m.selected_entry)).
        Rows(
            pad_headers([]string{"Name", "Mode", "Powered", "Status"}),
            []string{""},
            []string{default_netpala_data.device_data[0].name, default_netpala_data.device_data[0].mode, strconv.FormatBool(default_netpala_data.device_data[0].powered), fmt.Sprintf("%d",default_netpala_data.device_data[0].state)},
        )

    return calc_title("Device", true) + tbl.Render() + "\n" + calc_title("Network", true) + tbl.Render()
}

func main() {
    p := tea.NewProgram(NetpalaModel())
    if _, err := p.Run(); err != nil {
        fmt.Printf("Alas, there's been an error: %v", err)
        os.Exit(1)
    }
}