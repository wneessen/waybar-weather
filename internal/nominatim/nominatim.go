// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package nominatim

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/wneessen/waybar-weather/internal/config"
	"github.com/wneessen/waybar-weather/internal/http"
)

const (
	APIEndpoint = "https://nominatim.openstreetmap.org/reverse"
	APITimeout  = time.Second * 10
)

type Nominatim struct {
	conf *config.Config
	http *http.Client
}

type Result struct {
	PlaceID     int    `json:"place_id"`
	Licence     string `json:"licence"`
	OSMType     string `json:"osm_type"`
	OSMID       int    `json:"osm_id"`
	APILat      string `json:"lat"`
	APILon      string `json:"lon"`
	Lat         float64
	Lon         float64
	Category    string   `json:"category"`
	Type        string   `json:"type"`
	PlaceRank   int      `json:"place_rank"`
	Importance  float64  `json:"importance"`
	Addresstype string   `json:"addresstype"`
	Name        string   `json:"name"`
	DisplayName string   `json:"display_name"`
	Address     *Address `json:"address,omitempty"`
}

type Address struct {
	DisplayName  string `json:"display_name"`
	HouseNumber  string `json:"house_number"`
	Road         string `json:"road"`
	Suburb       string `json:"suburb"`
	CityDistrict string `json:"city_district"`
	City         string `json:"city"`
	State        string `json:"state"`
	ISO31662Lvl4 string `json:"ISO3166-2-lvl4"`
	Postcode     string `json:"postcode"`
	Country      string `json:"country"`
}

func New(client *http.Client, conf *config.Config) *Nominatim {
	return &Nominatim{
		conf: conf,
		http: client,
	}
}

func (n *Nominatim) Reverse(ctx context.Context, lat, lon float64) (Result, error) {
	var result Result
	apiUrl, err := url.Parse(APIEndpoint)
	if err != nil {
		return result, fmt.Errorf("failed to parse API endpoint: %w", err)
	}
	query := apiUrl.Query()
	query.Set("format", "jsonv2")
	query.Set("lat", fmt.Sprintf("%f", lat))
	query.Set("lon", fmt.Sprintf("%f", lon))
	query.Set("accept-language", n.conf.Locale)
	apiUrl.RawQuery = query.Encode()

	if _, err = n.http.GetWithTimeout(ctx, apiUrl.String(), &result, nil, APITimeout); err != nil {
		return result, fmt.Errorf("failed to address details from Nominatim API: %w", err)
	}

	// Fill in or parse missing values
	result.Lat, err = strconv.ParseFloat(result.APILat, 64)
	if err != nil {
		return result, fmt.Errorf("failed to parse latitude from Nominatim API response: %w", err)
	}
	result.Lon, err = strconv.ParseFloat(result.APILon, 64)
	if err != nil {
		return result, fmt.Errorf("failed to parse longitude from Nominatim API response: %w", err)
	}
	if result.Address != nil {
		result.Address.DisplayName = result.DisplayName
	}

	return result, nil
}
