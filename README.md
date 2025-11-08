<!--
SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>

SPDX-License-Identifier: MIT
-->

# waybar-weather
### A simple and elegant Waybar module to display weather data from Open-Meteo.

## About
waybar-weather is a simple programm written in Go that fetches weather data from Open-Meteo
and presents it in a format suitable to be used as custom Waybar module. It uses the Geoclue 
to determine your current location and fetches weather data for that location. 

## Features
* Uses the Geoclue as local geolocation provider.
* Fetch weather data from Open-Meteo (free, no API key required).
* Integrates with Waybar as a custom module.
* Display current weather conditions and temperature.
* Configurable via TOML, JSON or YAML.
* Lightweight, written in Go (single binary).

## Screenshots
![Full desktop view](assets/full.png)
![Detailed tooltip view](assets/detail.png)

## Requirements
* A working Linux installation with Waybar.
* Geoclue-2 installed and running.
* [Geoclue-2 demo agent installed and running.](#geoclue)
* Network connectivity to call the Open-Meteo API.

## Installation

### Using Pre-Built Binary
Pre-Built binaries are automatically built whenever a new release is created. Each release
holds binaries for several different Linux distributions. Each file is digitally signed via GPG. 
After downloading the corresponding file, make sure that the file is verified with the GPG 
signature. The public GPG key is: ["Winni Neessen" (Software signing key) <wn@neessen.dev>](https://keys.openpgp.org/vks/v1/by-fingerprint/10B5700F5ECCB06532CEC873C3D38948DA536E89)

### From Source
To build from source, you require a working Go environment. Go 1.25+ is required.
Run the following commands to build the binary:
```bash
git clone https://github.com/wneessen/waybar-weather.git
cd waybar-weather
go mod tidy
go mod download
go mod verify
go build -o waybar-weather app
```

## Configuration

### GeoClue
waybar-weather uses the [Geoclue-2](https://github.com/deepin-community/geoclue-2.0) to determine your current location. 
By default, waybar-weather will use the [Geoclue-2 demo agent](https://github.com/deepin-community/geoclue-2.0/tree/master/demo)
that is shipped with Geoclue-2. If your system doesn't have Geoclue-2 installed yet, you can do so using your 
distribution's package manager. Make sure to configure the `/etc/geoclue/geoclue.conf` according to your needs and
environment. For waybar-weather to properly work, it expects a minimum accuacy of "city".

The demo agent is not started automatically, but you can run it locally using systemd's user 
capabilities. Here is an example systemd unit file (put it into `~/.config/systemd/user/geoclue-agent.service`):
```systemd
[Unit]
Description=geoclue agent

[Service]
ExecStart=/usr/lib/geoclue-2.0/demos/agent

[Install]
WantedBy=default.target
```
You might have to adjust the path to the demo agent.

Once you have the service configured, you can start it with: `systemctl --user start --now geoclue-agent.service`

### waybar-weather
waybar-weather comes with defaults that should work out of the box for most users. You can however
provide a customer configuration file by appending the `-config` flag, followed by the path to your
configuration file. An example configuration file can be found in the [etc](etc) directory.

### Waybar integration
waybar-weather integrates with Waybar effortlessly. 

Add the following to your waybar config file (usually `.config/waybar/config.jsonc`):
```json
"custom/weather": {
    "exec": "<path_to_your>/waybar-weather",
    "restart-interval": 60,
    "return-type": "json",
    "hide-empty-text": true
}
```

Once you added that, add the module to your waybar module of choice, similar to this:
```json
"modules-right": [
    "cpu",
    "custom/weather",
    "battery",
    "clock"
],
```

waybar-weather always emits a custom CSS class to waybar, so you can apply your custom style to it. The class is
always `waybar-weather`. Add the following to your waybar config file (usually `.config/waybar/style.css`) to adjust
the style:
```css
.waybar-weather {
    <your_style_rules>
}
```

Once complete, restart Waybar and you should be good to go:
```bash
killall waybar && waybar
```

## License

This project is developed by Winni Neessen and released under the [MIT License](LICENSE).
