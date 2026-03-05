package config

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/charmbracelet/bubbles/key"
)

// KeyBinding represents a single configurable key binding
type KeyBinding struct {
	Keys []string `toml:"keys"`
	Help string   `toml:"help"`
}

// KeyBindings holds all configurable keybindings for the application
type KeyBindings struct {
	// Navigation
	Up       KeyBinding `toml:"up"`
	Down     KeyBinding `toml:"down"`
	NextPane KeyBinding `toml:"next_pane"`
	PrevPane KeyBinding `toml:"prev_pane"`

	// Actions
	Select            KeyBinding `toml:"select"`
	Remove            KeyBinding `toml:"remove"`
	Scan              KeyBinding `toml:"scan"`
	ToggleAutoconnect KeyBinding `toml:"toggle_autoconnect"`
	ToggleHidden      KeyBinding `toml:"toggle_hidden"`
	Edit              KeyBinding `toml:"edit"`

	// Application
	Quit   KeyBinding `toml:"quit"`
	Cancel KeyBinding `toml:"cancel"`
}

// Colors holds all color configurations for the application
type Colors struct {
	// Primary colors
	Primary       string `toml:"primary"`         // Default text and UI elements
	Active        string `toml:"active"`          // Active/selected borders
	ActiveText    string `toml:"active_text"`    // Active/selected text
	SelectionBg   string `toml:"selection_bg"`    // Selection bar background
	Inactive      string `toml:"inactive"`        // Inactive/dimmed elements
	Error         string `toml:"error"`           // Error states
	ErrorText     string `toml:"error_text"`     // Error text
	HelpText      string `toml:"help_text"`      // Help text at bottom of window
}

// Config holds the entire application configuration
type Config struct {
	KeyBindings KeyBindings `toml:"keybindings"`
	Colors      Colors      `toml:"colors"`
}

// DefaultKeyBindings returns the default keybinding configuration
func DefaultKeyBindings() KeyBindings {
	return KeyBindings{
		Up: KeyBinding{
			Keys: []string{"k", "up"},
			Help: "Up",
		},
		Down: KeyBinding{
			Keys: []string{"j", "down"},
			Help: "Down",
		},
		NextPane: KeyBinding{
			Keys: []string{"tab"},
			Help: "Next",
		},
		PrevPane: KeyBinding{
			Keys: []string{"shift+tab"},
			Help: "Prev",
		},
		Select: KeyBinding{
			Keys: []string{"enter", " "},
			Help: "Dis/Connect",
		},
		Remove: KeyBinding{
			Keys: []string{"backspace", "delete"},
			Help: "Remove",
		},
		Scan: KeyBinding{
			Keys: []string{"s"},
			Help: "Scan",
		},
		ToggleAutoconnect: KeyBinding{
			Keys: []string{"a"},
			Help: "Auto",
		},
		ToggleHidden: KeyBinding{
			Keys: []string{"h"},
			Help: "Hidden",
		},
		Edit: KeyBinding{
			Keys: []string{"e"},
			Help: "Edit",
		},
		Quit: KeyBinding{
			Keys: []string{"q", "ctrl+c", "ctrl+q", "ctrl+w"},
			Help: "Quit",
		},
		Cancel: KeyBinding{
			Keys: []string{"esc"},
			Help: "Cancel",
		},
	}
}

// DefaultColors returns the default color configuration
func DefaultColors() Colors {
	return Colors{
		Primary:    "#a7abca",  // Light blue-gray
		Active:     "#9cca69",  // Green
		ActiveText: "#cda162",  // Orange
		SelectionBg: "#5a6988", // Darker blue-gray for better contrast
		Inactive:   "#444a66",  // Dark gray
		Error:      "#ff0000",  // Red
		ErrorText:  "#aa0000",  // Dark red
		HelpText:   "#a7abca",  // Help text at bottom (same as Primary by default)
	}
}

// DefaultConfig returns a new Config with default values
func DefaultConfig() Config {
	return Config{
		KeyBindings: DefaultKeyBindings(),
		Colors:      DefaultColors(),
	}
}

// GetConfigPath returns the path to the config file following XDG specification
func GetConfigPath() (string, error) {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configDir = filepath.Join(homeDir, ".config")
	}

	return filepath.Join(configDir, "netpala", "config.toml"), nil
}

// Load loads the configuration from the config file
// If the file doesn't exist, it creates one with default values
func Load() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		cfg := DefaultConfig()
		return &cfg, nil
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); errors.Is(err, os.ErrNotExist) {
		// Create default config
		cfg := DefaultConfig()
		if saveErr := Save(&cfg); saveErr != nil {
			// If we can't save, just use defaults in memory
			return &cfg, nil
		}
		return &cfg, nil
	}

	// Read existing config
	var cfg Config
	if _, err := toml.DecodeFile(configPath, &cfg); err != nil {
		// If parsing fails, return defaults
		cfg = DefaultConfig()
		return &cfg, nil
	}

	// Merge with defaults to ensure all keys are present
	cfg = mergeWithDefaults(cfg)

	return &cfg, nil
}

// Save saves the configuration to the config file
func Save(cfg *Config) error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	// Create directory if it doesn't exist
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	// Create/truncate file
	file, err := os.Create(configPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write header comment
	header := `# Netpala Configuration File
# Keybindings can be customized below.
# Available modifiers: ctrl, alt, shift
# Examples: "ctrl+c", "shift+tab", "alt+enter", "a", "up", "down", "left", "right"
# Special keys: enter, space (use " "), tab, backspace, delete, esc, up, down, left, right
# Multiple keys can be assigned to the same action.

`
	if _, err := file.WriteString(header); err != nil {
		return err
	}

	// Encode config
	encoder := toml.NewEncoder(file)
	encoder.Indent = "  "
	return encoder.Encode(cfg)
}

// mergeWithDefaults ensures all keybindings have values, using defaults for missing ones
func mergeWithDefaults(cfg Config) Config {
	defaults := DefaultKeyBindings()
	defaultColors := DefaultColors()

	if len(cfg.KeyBindings.Up.Keys) == 0 {
		cfg.KeyBindings.Up = defaults.Up
	}
	if len(cfg.KeyBindings.Down.Keys) == 0 {
		cfg.KeyBindings.Down = defaults.Down
	}
	if len(cfg.KeyBindings.NextPane.Keys) == 0 {
		cfg.KeyBindings.NextPane = defaults.NextPane
	}
	if len(cfg.KeyBindings.PrevPane.Keys) == 0 {
		cfg.KeyBindings.PrevPane = defaults.PrevPane
	}
	if len(cfg.KeyBindings.Select.Keys) == 0 {
		cfg.KeyBindings.Select = defaults.Select
	}
	if len(cfg.KeyBindings.Remove.Keys) == 0 {
		cfg.KeyBindings.Remove = defaults.Remove
	}
	if len(cfg.KeyBindings.Scan.Keys) == 0 {
		cfg.KeyBindings.Scan = defaults.Scan
	}
	if len(cfg.KeyBindings.ToggleAutoconnect.Keys) == 0 {
		cfg.KeyBindings.ToggleAutoconnect = defaults.ToggleAutoconnect
	}
	if len(cfg.KeyBindings.ToggleHidden.Keys) == 0 {
		cfg.KeyBindings.ToggleHidden = defaults.ToggleHidden
	}
	if len(cfg.KeyBindings.Edit.Keys) == 0 {
		cfg.KeyBindings.Edit = defaults.Edit
	}
	if len(cfg.KeyBindings.Quit.Keys) == 0 {
		cfg.KeyBindings.Quit = defaults.Quit
	}
	if len(cfg.KeyBindings.Cancel.Keys) == 0 {
		cfg.KeyBindings.Cancel = defaults.Cancel
	}

	// Ensure help text is set
	if cfg.KeyBindings.Up.Help == "" {
		cfg.KeyBindings.Up.Help = defaults.Up.Help
	}
	if cfg.KeyBindings.Down.Help == "" {
		cfg.KeyBindings.Down.Help = defaults.Down.Help
	}
	if cfg.KeyBindings.NextPane.Help == "" {
		cfg.KeyBindings.NextPane.Help = defaults.NextPane.Help
	}
	if cfg.KeyBindings.PrevPane.Help == "" {
		cfg.KeyBindings.PrevPane.Help = defaults.PrevPane.Help
	}
	if cfg.KeyBindings.Select.Help == "" {
		cfg.KeyBindings.Select.Help = defaults.Select.Help
	}
	if cfg.KeyBindings.Remove.Help == "" {
		cfg.KeyBindings.Remove.Help = defaults.Remove.Help
	}
	if cfg.KeyBindings.Scan.Help == "" {
		cfg.KeyBindings.Scan.Help = defaults.Scan.Help
	}
	if cfg.KeyBindings.ToggleAutoconnect.Help == "" {
		cfg.KeyBindings.ToggleAutoconnect.Help = defaults.ToggleAutoconnect.Help
	}
	if cfg.KeyBindings.ToggleHidden.Help == "" {
		cfg.KeyBindings.ToggleHidden.Help = defaults.ToggleHidden.Help
	}
	if cfg.KeyBindings.Edit.Help == "" {
		cfg.KeyBindings.Edit.Help = defaults.Edit.Help
	}
	if cfg.KeyBindings.Quit.Help == "" {
		cfg.KeyBindings.Quit.Help = defaults.Quit.Help
	}
	if cfg.KeyBindings.Cancel.Help == "" {
		cfg.KeyBindings.Cancel.Help = defaults.Cancel.Help
	}

	// Merge colors with defaults
	if cfg.Colors.Primary == "" {
		cfg.Colors.Primary = defaultColors.Primary
	}
	if cfg.Colors.Active == "" {
		cfg.Colors.Active = defaultColors.Active
	}
	if cfg.Colors.ActiveText == "" {
		cfg.Colors.ActiveText = defaultColors.ActiveText
	}
	if cfg.Colors.SelectionBg == "" {
		cfg.Colors.SelectionBg = defaultColors.SelectionBg
	}
	if cfg.Colors.Inactive == "" {
		cfg.Colors.Inactive = defaultColors.Inactive
	}
	if cfg.Colors.Error == "" {
		cfg.Colors.Error = defaultColors.Error
	}
	if cfg.Colors.ErrorText == "" {
		cfg.Colors.ErrorText = defaultColors.ErrorText
	}

	return cfg
}

// ToKeyBinding converts a KeyBinding config to a bubbles key.Binding
func (kb KeyBinding) ToKeyBinding() key.Binding {
	return key.NewBinding(
		key.WithKeys(kb.Keys...),
		key.WithHelp(formatKeysHelp(kb.Keys), kb.Help),
	)
}

// formatKeysHelp creates a help string from keys
func formatKeysHelp(keys []string) string {
	if len(keys) == 0 {
		return ""
	}

	// Map common keys to symbols for compact display
	symbolMap := map[string]string{
		"up":        "↑",
		"down":      "↓",
		"left":      "←",
		"right":     "→",
		"enter":     "⤶",
		" ":         "␣",
		"space":     "␣",
		"tab":       "⇥",
		"shift+tab": "⇤",
		"backspace": "⌫",
		"delete":    "⌦",
		"esc":       "⎋",
		"ctrl+c":    "^C",
		"ctrl+q":    "^Q",
		"ctrl+w":    "^W",
	}

	// Show first two keys max for compact display
	result := ""
	shown := 0
	for _, k := range keys {
		if shown >= 2 {
			break
		}
		if shown > 0 {
			result += "/"
		}
		if sym, ok := symbolMap[k]; ok {
			result += sym
		} else {
			result += k
		}
		shown++
	}

	return result
}

// Matches checks if a key message matches this keybinding
func (kb KeyBinding) Matches(keyStr string) bool {
	for _, k := range kb.Keys {
		if k == keyStr {
			return true
		}
	}
	return false
}

// AppKeyMap holds the application's key bindings in bubbles format
type AppKeyMap struct {
	Up                key.Binding
	Down              key.Binding
	NextPane          key.Binding
	PrevPane          key.Binding
	Select            key.Binding
	Remove            key.Binding
	Scan              key.Binding
	ToggleAutoconnect key.Binding
	ToggleHidden      key.Binding
	Edit              key.Binding
	Quit              key.Binding
	Cancel            key.Binding
}

// NewAppKeyMap creates a new AppKeyMap from a Config
func NewAppKeyMap(cfg *Config) AppKeyMap {
	return AppKeyMap{
		Up:                cfg.KeyBindings.Up.ToKeyBinding(),
		Down:              cfg.KeyBindings.Down.ToKeyBinding(),
		NextPane:          cfg.KeyBindings.NextPane.ToKeyBinding(),
		PrevPane:          cfg.KeyBindings.PrevPane.ToKeyBinding(),
		Select:            cfg.KeyBindings.Select.ToKeyBinding(),
		Remove:            cfg.KeyBindings.Remove.ToKeyBinding(),
		Scan:              cfg.KeyBindings.Scan.ToKeyBinding(),
		ToggleAutoconnect: cfg.KeyBindings.ToggleAutoconnect.ToKeyBinding(),
		ToggleHidden:      cfg.KeyBindings.ToggleHidden.ToKeyBinding(),
		Edit:              cfg.KeyBindings.Edit.ToKeyBinding(),
		Quit:              cfg.KeyBindings.Quit.ToKeyBinding(),
		Cancel:            cfg.KeyBindings.Cancel.ToKeyBinding(),
	}
}

// ShortHelp returns key bindings to show in short help
func (k AppKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Up, k.Down, k.Select, k.Remove,
		k.Scan, k.ToggleAutoconnect, k.ToggleHidden, k.Edit, k.NextPane, k.Quit,
	}
}

// FullHelp returns the full set of key bindings
func (k AppKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.NextPane, k.PrevPane},
		{k.Select, k.Remove, k.Scan, k.ToggleAutoconnect, k.ToggleHidden, k.Edit, k.Quit},
	}
}