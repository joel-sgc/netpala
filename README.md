
<div align="center">
  <h2> Netpala (Impala Go Edition) </h2>
</div>


A lightweight (hopefully) terminal-friendly NetworkManager wrapper written in Go.
It’s a clone of Impala, because Impala’s UI made a single majestic white tear roll down my leg.

---

## 📸 Demo

![Netpala](./netpala.png "Netpala")

---

## 💡 Prerequisites

A Linux-based OS with NetworkManager and dbus running.
**Compatible with both wpa_supplicant and iwd backends.**

---

## 🚀 Installation

### Binary Release

Pre-built binaries coming soon.

---

### Install from AUR

```bash
# Using yay
yay -S netpala

# Or using paru
paru -S netpala
```

You'll need:

Go 1.25.4+ (only for building AUR package locally)
- NetworkManager running
- dbus available

---

### Install from Source (Go)

```bash
git clone https://github.com/joel-sgc/netpala.git
cd netpala
go build
./netpala
```

You'll need:

- Go 1.25.4+
- NetworkManager running
- dbus available

---

### Install with Nix

#### Run directly without installing

```bash
nix run github:joel-sgc/netpala
```

#### Install to profile

```bash
nix profile install github:joel-sgc/netpala
```

#### Add to NixOS configuration

Add to your `flake.nix` inputs:

```nix
{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    netpala.url = "github:joel-sgc/netpala";
  };
}
```

Then either add to `environment.systemPackages`:

```nix
{ inputs, pkgs, ... }:
{
  environment.systemPackages = [ inputs.netpala.packages.${pkgs.system}.default ];
}
```

Or use the provided NixOS module:

```nix
{ inputs, ... }:
{
  imports = [ inputs.netpala.nixosModules.default ];
  programs.netpala.enable = true;
}
```

#### Development shell

```bash
git clone https://github.com/joel-sgc/netpala.git
cd netpala
nix develop
```

---

### Omarchy / Waybar launcher example

```bash
#!/bin/bash
$TERMINAL --title=com.omarchy.netpala netpala
```

## Hyprland floating window rules
```bash
windowrule = float 1, match:title com.omarchy.netpala
```

---

## 🪄 Usage

### Global

- Tab / Shift+Tab : Switch between sections
- j / Down : Scroll down
- k / Up : Scroll up
- s : Force scan
- q / Ctrl+C: Quit

### Networks

- Space / Enter : Connect / Disconnect
- Delete / Backspace : Remove network

---

## ⚙️ Configuration

Netpala supports customizable keybindings through a configuration file located at `~/.config/netpala/config.toml`.

On first run, a default configuration file will be created automatically. You can also copy the example config:

```bash
mkdir -p ~/.config/netpala
cp config.example.toml ~/.config/netpala/config.toml
```

### Keybinding Configuration

All keybindings can be customized in the config file. Example:

```toml
[keybindings]

# Navigation
[keybindings.up]
keys = ["k", "up"]
help = "Up"

[keybindings.down]
keys = ["j", "down"]
help = "Down"

[keybindings.next_pane]
keys = ["tab"]
help = "Next"

[keybindings.prev_pane]
keys = ["shift+tab"]
help = "Prev"

# Actions
[keybindings.select]
keys = ["enter", " "]
help = "Dis/Connect"

[keybindings.remove]
keys = ["backspace", "delete"]
help = "Remove"

[keybindings.scan]
keys = ["s"]
help = "Scan"

[keybindings.toggle_autoconnect]
keys = ["a"]
help = "Auto"

[keybindings.toggle_hidden]
keys = ["h"]
help = "Hidden"

# Application
[keybindings.quit]
keys = ["q", "ctrl+c", "ctrl+q", "ctrl+w"]
help = "Quit"

[keybindings.cancel]
keys = ["esc"]
help = "Cancel"
```

### Available Modifiers and Keys

- **Modifiers**: `ctrl`, `alt`, `shift`
- **Special keys**: `enter`, `space` (use `" "`), `tab`, `backspace`, `delete`, `esc`, `up`, `down`, `left`, `right`
- **Examples**: `"ctrl+c"`, `"shift+tab"`, `"alt+enter"`, `"a"`, `"up"`

Multiple keys can be assigned to the same action by listing them in the `keys` array.

### Color Configuration

You can also customize the application colors in the same config file:

```toml
[colors]
# Default text and UI elements
primary = "#a7abca"        # Light blue-gray

# Active/selected borders
active = "#9cca69"         # Green

# Active/selected text
active_text = "#cda162"    # Orange

# Selection bar background
selection_bg = "#5a6988"   # Darker blue-gray for better contrast

# Inactive/dimmed elements
inactive = "#444a66"       # Dark gray

# Error states
error = "#ff0000"          # Red

# Error text
error_text = "#aa0000"     # Dark red
```

Colors can be specified as:
- Hex color codes: `"#a7abca"`
- Terminal color names: `"red"`, `"blue"`, `"green"`, etc.
- ANSI color numbers: `"1"` (red), `"2"` (green), etc.

---

## 🚀 Features (So Far)

- Lists available network devices
- Shows known and scanned networks
- Shows VPN connections
- Add & connect to:
  - WPA-PSK
  - WPA-SAE
  - WPA-Enterprise (EAP)
  - Open Networks
- Force network scan
- Enable/Disable devices
- Communicates with NetworkManager + wpa_supplicant over DBus

---

## ⚠️ Missing / TODO

- VPN manager UI Testing (Should work but haven't been able to test 100%)
- Probably some bugs (Hopefully there's nothing)

---

## 🧩 Implementation Notes

The DBus code was mostly vibe-coded.
It works. I don’t care. If you do care, fork it and figure it out.

---

## 🧾 License

Do What the Fuck You Want To Public License (WTFPL)

---

## ❤️ Closing Thoughts

I built this for myself because I wanted something that just works, and this just works.
If you like it, awesome. If not, feel free to improve it or ignore it entirely.