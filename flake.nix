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

          src = ./.;

          vendorHash = null; # Set to null if using go.sum, or use pkgs.lib.fakeHash to get the real hash

          # If vendorHash = null doesn't work, you may need to calculate the hash:
          # Run: nix build and it will tell you the correct hash
          # vendorHash = "sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=";

          meta = with pkgs.lib; {
            description = "A lightweight terminal-friendly NetworkManager wrapper written in Go";
            homepage = "https://github.com/joel-sgc/netpala";
            license = licenses.wtfpl;
            maintainers = [ ];
            platforms = platforms.linux;
            mainProgram = "netpala";
          };

          # Runtime dependencies
          buildInputs = with pkgs; [
            dbus
          ];

          # Ensure the binary can find dbus at runtime
          nativeBuildInputs = with pkgs; [
            pkg-config
          ];

          ldflags = [
            "-s"
            "-w"
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