{
  lib,
  buildGoModule,
  source,
}:

buildGoModule rec {
  pname = "waybar-weather";
  version = "development";

  src = source;

  vendorHash = "sha256-abTh3uR0kC7BIFw2cw3iHenSLIR/76Df4LSJBB2uwn0=";

  subPackages = [ "cmd/waybar-weather" ];

  ldflags = [
    "-s"
    "-w"
    "-X main.version=development"
    "-X main.commit=main"
    "-X main.date=1970-01-01T00:00:00Z"
    "-X github.com/wneessen/waybar-weather/internal/http.version=development"
  ];

  postInstall = ''
    if [ -f etc/waybar-weather.toml ]; then
      install -Dm644 etc/waybar-weather.toml \
        $out/share/${pname}/waybar-weather.toml
    fi

    if [ -f etc/config.toml ]; then
      install -Dm644 etc/config.toml \
        $out/share/${pname}/config.toml
    fi

    if [ -f etc/geolocation ]; then
      install -Dm644 etc/geolocation \
        $out/share/${pname}/geolocation
    fi

    if [ -f etc/cityname ]; then
      install -Dm644 etc/cityname \
        $out/share/${pname}/cityname
    fi

    if [ -f contrib/style/waybar-weather.css ]; then
      install -Dm644 contrib/style/waybar-weather.css \
        $out/share/${pname}/waybar-weather.css
    fi

    if [ -d contrib/icons/meteocons ]; then
      mkdir -p $out/share/${pname}/weather-icons
      cp -r contrib/icons/meteocons/* \
        $out/share/${pname}/weather-icons/
    fi

    install -Dm644 README.md \
      $out/share/doc/${pname}/README.md

    install -Dm644 LICENSE \
      $out/share/licenses/${pname}/LICENSE
  '';

  meta = with lib; {
    description = "A waybar weather module with automatic geolocation lookup";
    homepage = "https://github.com/wneessen/waybar-weather";
    license = licenses.mit;
    mainProgram = "waybar-weather";
    platforms = platforms.linux;
  };
}
