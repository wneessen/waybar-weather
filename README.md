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
* Geoclue installed and running.
* Geoclue-2 demo agent installed and running.
* Network connectivity to call the Open-Meteo API.

## Installation

### Using Pre-Built Binary
Pre-Built binaries are automatically built whenever a new release is created. Each release
holds binaries for several different Linux distributions.

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

To style the module, add the following to your waybar config file (usually `.config/waybar/style.css`):
```css
#waybar-weather {
    <your_style_rules>
}
```

Once complete, restart Waybar and you should be good to go:
```bash
killall waybar && waybar
```

## License

This project is developed by Winni Neessen and released under the [MIT License](LICENSE).
