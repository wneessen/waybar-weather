{
  description = "A waybar weather module with automatic geolocation lookup";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs = { self, nixpkgs }:
    let
      lib = nixpkgs.lib;

      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "i686-linux"
        "armv6l-linux"
        "armv7l-linux"
      ];

      forAllSystems = lib.genAttrs systems;
    in {
      packages = forAllSystems (system:
        let
          pkgs = import nixpkgs {
            inherit system;
          };
        in {
          stable = pkgs.callPackage ./package-stable.nix { };

          development = pkgs.callPackage ./package-development.nix {
            source = self;
          };

          default = self.packages.${system}.stable;
        });

      apps = forAllSystems (system: {
        stable = {
          type = "app";
          program = "${self.packages.${system}.stable}/bin/waybar-weather";
        };

        development = {
          type = "app";
          program = "${self.packages.${system}.development}/bin/waybar-weather";
        };

        default = self.apps.${system}.stable;
      });

      devShells = forAllSystems (system:
        let
          pkgs = import nixpkgs {
            inherit system;
          };
        in {
          default = pkgs.mkShell {
            packages = with pkgs; [
              go
              gopls
              golangci-lint
            ];
          };
        });
    };
}
