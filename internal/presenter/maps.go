// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package presenter

import "github.com/vorlif/spreak/localize"

// MoonPhaseIcon is a map where moon phase names are keys and their corresponding emoji representations are values.
var MoonPhaseIcon = map[string]string{
	"New Moon":        "🌑",
	"Waxing Crescent": "🌒",
	"First Quarter":   "🌓",
	"Waxing Gibbous":  "🌔",
	"Full Moon":       "🌕",
	"Waning Gibbous":  "🌖",
	"Third Quarter":   "🌗",
	"Waning Crescent": "🌘",
}

// WMOWeatherCodes maps WMO weather code integers to their descriptions
var WMOWeatherCodes = map[int]localize.MsgID{
	0:  "Clear sky",
	1:  "Mainly clear",
	2:  "Partly cloudy",
	3:  "Overcast",
	45: "Fog",
	48: "Depositing rime fog",
	51: "Light drizzle",
	53: "Moderate drizzle",
	55: "Dense drizzle",
	56: "Light freezing drizzle",
	57: "Dense freezing drizzle",
	61: "Slight rain",
	63: "Moderate rain",
	65: "Heavy rain",
	66: "Light freezing rain",
	67: "Heavy freezing rain",
	71: "Slight snow fall",
	73: "Moderate snow fall",
	75: "Heavy snow fall",
	77: "Snow grains",
	80: "Slight rain showers",
	81: "Moderate rain showers",
	82: "Violent rain showers",
	85: "Slight snow showers",
	86: "Heavy snow showers",
	95: "Thunderstorm",
	96: "Thunderstorm with slight hail",
	99: "Thunderstorm with heavy hail",
}

// WMOWeatherIcons maps WMO weather codes to single emoji icons for day (1) and night (0)
var WMOWeatherIcons = map[int]map[bool]string{
	0: {
		true:  "☀️", // Clear sky (day)
		false: "🌙",
	},
	1: {
		true:  "🌤️", // Mainly clear (day)
		false: "🌙",
	},
	2: {
		true:  "⛅", // Partly cloudy
		false: "☁️",
	},
	3: {
		true:  "☁️", // Overcast
		false: "☁️",
	},
	45: {
		true:  "🌫️", // Fog
		false: "🌫️",
	},
	48: {
		true:  "🌫️", // Depositing rime fog
		false: "🌫️",
	},
	51: {
		true:  "🌦️", // Drizzle: Light
		false: "🌧️",
	},
	53: {
		true:  "🌧️", // Drizzle: Moderate
		false: "🌧️",
	},
	55: {
		true:  "🌧️", // Drizzle: Dense intensity
		false: "🌧️",
	},
	56: {
		true:  "🌨️", // Freezing drizzle: Light
		false: "🌨️",
	},
	57: {
		true:  "🌨️", // Freezing drizzle: Dense intensity
		false: "🌨️",
	},
	61: {
		true:  "🌦️", // Rain: Slight
		false: "🌧️",
	},
	63: {
		true:  "🌧️", // Rain: Moderate
		false: "🌧️",
	},
	65: {
		true:  "🌧️", // Rain: Heavy
		false: "🌧️",
	},
	66: {
		true:  "🌨️", // Freezing rain: Light
		false: "🌨️",
	},
	67: {
		true:  "🌨️", // Freezing rain: Heavy
		false: "🌨️",
	},
	71: {
		true:  "🌨️", // Snow fall: Slight
		false: "🌨️",
	},
	73: {
		true:  "🌨️", // Snow fall: Moderate
		false: "🌨️",
	},
	75: {
		true:  "🌨️", // Snow fall: Heavy
		false: "🌨️",
	},
	77: {
		true:  "🌨️", // Snow grains
		false: "🌨️",
	},
	80: {
		true:  "🌦️", // Rain showers: Slight
		false: "🌧️",
	},
	81: {
		true:  "🌧️", // Rain showers: Moderate
		false: "🌧️",
	},
	82: {
		true:  "🌧️", // Rain showers: Violent
		false: "🌧️",
	},
	85: {
		true:  "🌨️", // Snow showers: Slight
		false: "🌨️",
	},
	86: {
		true:  "🌨️", // Snow showers: Heavy
		false: "🌨️",
	},
	95: {
		true:  "🌩️", // Thunderstorm: Slight or moderate
		false: "🌩️",
	},
	96: {
		true:  "⛈️", // Thunderstorm with slight hail
		false: "⛈️",
	},
	99: {
		true:  "⛈️", // Thunderstorm with heavy hail
		false: "⛈️",
	},
}

var i18nVars = map[string]localize.MsgID{
	"temp":            "Temperature",
	"humidity":        "Humidity",
	"winddir":         "Wind direction",
	"windspeed":       "Wind speed",
	"wind":            "Wind",
	"pressure":        "Pressure",
	"apparent":        "Feels like",
	"weathercode":     "Weather code",
	"forecastfor":     "Forecast for",
	"weatherdatafor":  "Weather data for",
	"sunrise":         "Sunrise",
	"sunset":          "Sunset",
	"moonphase":       "Moonphase",
	"new moon":        "New moon",
	"waxing crescent": "Waxing crescent",
	"first quarter":   "First quarter",
	"waxing gibbous":  "Waxing gibbous",
	"full moon":       "Full moon",
	"waning gibbous":  "Waning gibbous",
	"third quarter":   "Third quarter",
	"waning crescent": "Waning crescent",
}

var windDirIcons = map[string]string{
	"N":  "↑",
	"NE": "↗",
	"E":  "→",
	"SE": "↘",
	"S":  "↓",
	"SW": "↙",
	"W":  "←",
	"NW": "↖",
}
