{
  lib,
  buildGoModule,
  fetchFromGitHub,
}:

buildGoModule rec {
  pname = "waybar-weather";
  version = "0.2.6";

  src = fetchFromGitHub {
    owner = "wneessen";
    repo = "waybar-weather";
    rev = "v${version}";
    hash = "sha256-TSBTUBY/xB4UiYS7mgGp9nHCqp6VPjtUYcq1OrSmHuM=";
  };

  vendorHash = "sha256-nLLOnYNT3pN5rJlIUWuCeM4HVrluB2dBbg0MQD6FXB8=";

  subPackages = [ "cmd/waybar-weather" ];

  ldflags = [
    "-s"
    "-w"
    "-X main.version=${version}"
    "-X main.commit=release-${version}"
    "-X main.date=1970-01-01T00:00:00Z"
    "-X github.com/wneessen/waybar-weather/internal/http.version=${version}"
  ];

  postInstall = ''
    install -Dm644 etc/waybar-weather.toml \
      $out/share/${pname}/waybar-weather.toml

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
