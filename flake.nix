{
  description = "Netpala - A lightweight terminal-friendly NetworkManager wrapper";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};

        netpala = pkgs.buildGoModule {
          pname = "netpala";
          version = "0.1.0";

          src = pkgs.lib.cleanSourceWith {
            src = ./.;
            filter = path: type:
              let
                baseName = baseNameOf path;
              in
              # Exclude vendor directory, result symlinks, and other non-essential files
              !(baseName == "vendor" ||
                baseName == "result" ||
                pkgs.lib.hasPrefix "result-" baseName ||
                baseName == ".direnv" ||
                baseName == ".git" ||
                baseName == "flake.lock");
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
    ) // {
      # NixOS module for system-wide installation
      nixosModules.default = { config, lib, pkgs, ... }:
        let
          cfg = config.programs.netpala;
        in
        {
          options.programs.netpala = {
            enable = lib.mkEnableOption "netpala network manager TUI";

            package = lib.mkOption {
              type = lib.types.package;
              default = self.packages.${pkgs.system}.default;
              description = "The netpala package to use";
            };
          };

          config = lib.mkIf cfg.enable {
            environment.systemPackages = [ cfg.package ];

            # Ensure NetworkManager and dbus are available
            services.dbus.enable = true;
          };
        };

      # Overlay for use with other flakes
      overlays.default = final: prev: {
        netpala = self.packages.${prev.system}.default;
      };
    };
}