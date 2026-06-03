{
  description = "A waybar weather module with automatic geolocation lookup";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs = { self, nixpkgs }:
    let
      system = "x86_64-linux";
      pkgs = nixpkgs.legacyPackages.${system};

      owner = "wneessen";
      repo = "waybar-weather";
      version = "0.3.0";

      # Pre-built binary release (goreleaser tar.gz)
      binSrc = pkgs.fetchurl {
        url = "https://github.com/${owner}/${repo}/releases/download/v${version}/waybar-weather_${version}_linux_amd64.tar.gz";
        hash = "sha256-iK8llFf9LE+NiOeaJBizI66anx2F2l1g5JYMSRR5mlk=";
      };

      # Source tarball for config, icons, docs, and license
      sourceSrc = pkgs.fetchurl {
        url = "https://github.com/${owner}/${repo}/archive/refs/tags/v${version}.tar.gz";
        hash = "sha256-sXzquW/bQkefiawZZDPSDAzSXZfN8GxDHRIjGTxBpdk=";
      };
    in
    {
      packages.${system} = {
        default = self.packages.${system}.waybar-weather;

        waybar-weather = pkgs.stdenv.mkDerivation {
          pname = "waybar-weather";
          inherit version;

          srcs = [ binSrc sourceSrc ];
          sourceRoot = ".";

          unpackPhase = ''
            runHook preUnpack
            tar xzf ${binSrc}
            tar xzf ${sourceSrc}
            runHook postUnpack
          '';

          installPhase = ''
            runHook preInstall

            # Binary
            mkdir -p $out/bin
            install -Dm755 waybar-weather $out/bin/waybar-weather

            # Example config files
            mkdir -p $out/share/waybar-weather
            install -Dm644 waybar-weather-${version}/etc/config.toml \
               $out/share/waybar-weather/config.toml
            install -Dm644 waybar-weather-${version}/etc/geolocation \
               $out/share/waybar-weather/geolocation
            install -Dm644 waybar-weather-${version}/etc/cityname \
               $out/share/waybar-weather/cityname

            # Style and icons
            install -Dm644 waybar-weather-${version}/contrib/style/waybar-weather.css \
               $out/share/waybar-weather/waybar-weather.css
            cp -r waybar-weather-${version}/contrib/icons/meteocons \
               $out/share/waybar-weather/weather-icons

            # Documentation
            mkdir -p $out/share/doc/waybar-weather
            install -Dm644 waybar-weather-${version}/README.md \
               $out/share/doc/waybar-weather/README.md

            # License
            mkdir -p $out/share/licenses/waybar-weather
            install -Dm644 waybar-weather-${version}/LICENSE \
               $out/share/licenses/waybar-weather/LICENSE

            runHook postInstall
          '';

          meta = with pkgs.lib; {
            description = "A waybar weather module with automatic geolocation lookup";
            homepage = "https://github.com/wneessen/waybar-weather";
            license = licenses.mit;
            platforms = [ "x86_64-linux" ];
          };
        };
      };

      devShells.${system}.default = pkgs.mkShell {
        packages = [ self.packages.${system}.waybar-weather ];
      };
    };
}
