# waybar-weather

An Open-Meteo module implementation for [Waybar](https://github.com/Alexays/Waybar)

## Table of Contents

* [About](#about)
* [Features](#features)
* [Requirements](#requirements)
* [Installation](#installation)
* [Configuration](#configuration)
* [Usage](#usage)
* [Development & Contributing](#development-contributing)
* [License](#license)

## About

`waybar-weather` is a small Go-based module that fetches weather data from the Open‑Meteo API 
and displays it in your Waybar status bar. It’s designed for users of Waybar on Linux who 
want a simple, local, and configurable weather widget without depending on heavier 
desktop-environment integrations.

## Features

* Uses the Geoclue as local geolocation provider.
* Fetch weather data from Open-Meteo (free, no API key required).
* Integrates with Waybar as a custom module.
* Display current weather conditions and temperature.
* Configurable via TOML, JSON or YAML.
* Lightweight, written in Go (single binary).

## Requirements

* A working installation of Waybar.
* Linux environment (Waybar compatible).
* Go environment (for building from source) **or** a pre-built binary
* Network connectivity to call the Open-Meteo API.

## Installation

### From Source

```bash
git clone https://github.com/wneessen/waybar-weather.git  
cd waybar-weather  
go build -o waybar-weather app
```

### Using Pre-Built Binary

TBD

## Configuration

TBD 

## License

This project is released under the MIT License - see [LICENSE](LICENSE) for details.
