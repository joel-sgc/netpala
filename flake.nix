{
  description = "Netpala - A lightweight terminal-friendly NetworkManager wrapper";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};

        netpala = pkgs.buildGoModule {
          pname = "netpala";
          version = "1.3.0";

          src = pkgs.lib.cleanSourceWith {
            src = ./.;
            filter =
              path: type:
              let
                baseName = baseNameOf path;
              in
              # Exclude vendor directory, result symlinks, and other non-essential files
              !(
                baseName == "vendor"
                || baseName == "result"
                || pkgs.lib.hasPrefix "result-" baseName
                || baseName == ".direnv"
                || baseName == ".git"
                || baseName == "flake.lock"
              );
          };

          vendorHash = "sha256-nSLOvVn4gtpUOmi+msKSHMBU+5ly9QEENQEeFrEbuII=";

          meta = with pkgs.lib; {
            description = "A lightweight terminal-friendly NetworkManager wrapper written in Go";
            homepage = "https://github.com/joel-sgc/netpala";
            license = licenses.wtfpl;
            maintainers = [ ];
            platforms = platforms.linux;
            mainProgram = "netpala";
          };

          ldflags = [
            "-s"
            "-w"
          ];

          # Runtime dependencies
          nativeBuildInputs = with pkgs; [
            pkg-config
          ];

          buildInputs = with pkgs; [
            dbus
          ];
        };
      in
      {
        packages = {
          default = netpala;
          netpala = netpala;
        };

        apps = {
          default = flake-utils.lib.mkApp {
            drv = netpala;
          };
          netpala = flake-utils.lib.mkApp {
            drv = netpala;
          };
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            gopls
            go-tools
            golangci-lint
            dbus
            pkg-config
          ];

          shellHook = ''
            echo "Netpala development environment"
            echo "Go version: $(go version)"
            echo ""
            echo "Available commands:"
            echo "  go build    - Build the application"
            echo "  go run .    - Run the application"
            echo "  go test     - Run tests"
            echo "  nix build   - Build with Nix"
          '';
        };
      }
    )
    // {
      # NixOS module for system-wide installation
      nixosModules.default =
        {
          config,
          lib,
          pkgs,
          ...
        }:
        let
          cfg = config.programs.netpala;
        in
        {
          options.programs.netpala = {
            enable = lib.mkEnableOption "netpala network manager TUI";

            package = lib.mkOption {
              type = lib.types.package;
              default = self.packages.${pkgs.stdenv.hostPlatform.system}.default;
              description = "The netpala package to use";
            };
          };

          config = lib.mkIf cfg.enable {
            environment.systemPackages = [ cfg.package ];

            # Ensure NetworkManager and dbus are available
            services.dbus.enable = true;
          };
        };

      # Home Manager module for per-user installation + config.toml generation
      homeManagerModules.default =
        {
          config,
          lib,
          pkgs,
          ...
        }:
        let
          cfg = config.programs.netpala;
          tomlFormat = pkgs.formats.toml { };
        in
        {
          options.programs.netpala = {
            enable = lib.mkEnableOption "netpala network manager TUI";

            package = lib.mkOption {
              type = lib.types.package;
              default = self.packages.${pkgs.system}.default;
              description = "The netpala package to use";
            };

            settings = lib.mkOption {
              type = tomlFormat.type;
              default = { };
              description = ''
                Configuration written to
                `$XDG_CONFIG_HOME/netpala/config.toml`. Any keys you omit
                fall back to netpala's built-in defaults. See
                `config/keybindings.go` for the full set of `[colors]` and
                `[keybindings]` keys.
              '';
              example = lib.literalExpression ''
                {
                  colors = {
                    primary = "#a7abca";
                    active = "#9cca69";
                  };
                  keybindings = {
                    quit.keys = [ "q" "ctrl+c" ];
                    scan.keys = [ "s" ];
                  };
                }
              '';
            };
          };

          config = lib.mkIf cfg.enable {
            home.packages = [ cfg.package ];

            xdg.configFile."netpala/config.toml" = lib.mkIf (cfg.settings != { }) {
              source = tomlFormat.generate "netpala-config.toml" cfg.settings;
            };
          };
        };

      # Overlay for use with other flakes
      overlays.default = final: prev: {
        netpala = self.packages.${prev.stdenv.hostPlatform.system}.default;
      };
    };
}
