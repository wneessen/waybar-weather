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

	"golang.org/x/text/language"

	"github.com/wneessen/waybar-weather/internal/geobus"
	"github.com/wneessen/waybar-weather/internal/geocode"
	"github.com/wneessen/waybar-weather/internal/http"
)

const (
	APIEndpoint = "https://nominatim.openstreetmap.org/reverse"
	APITimeout  = time.Second * 10
	name        = "osm-nominatim"
)

type Nominatim struct {
	http *http.Client
	lang language.Tag
}

type Result struct {
	APILat      string  `json:"lat"`
	APILon      string  `json:"lon"`
	Name        string  `json:"name"`
	DisplayName string  `json:"display_name"`
	Address     Address `json:"address"`
}

type Address struct {
	DisplayName  string `json:"display_name"`
	HouseNumber  string `json:"house_number"`
	Road         string `json:"road"`
	Suburb       string `json:"suburb"`
	Municipality string `json:"municipality"`
	CityDistrict string `json:"city_district"`
	City         string `json:"city"`
	Town         string `json:"town"`
	Village      string `json:"village"`
	State        string `json:"state"`
	ISO31662Lvl4 string `json:"ISO3166-2-lvl4"`
	Postcode     string `json:"postcode"`
	Country      string `json:"country"`
}

func New(client *http.Client, lang language.Tag) *Nominatim {
	return &Nominatim{
		lang: lang,
		http: client,
	}
}

func (n *Nominatim) Name() string {
	return name
}

func (n *Nominatim) Reverse(ctx context.Context, coords geobus.Coordinate) (geocode.Address, error) {
	var result Result
	var err error

	query := url.Values{}
	query.Set("format", "jsonv2")
	query.Set("lat", fmt.Sprintf("%f", coords.Lat))
	query.Set("lon", fmt.Sprintf("%f", coords.Lon))
	query.Set("accept-language", n.lang.String())

	if _, err = n.http.GetWithTimeout(ctx, APIEndpoint, &result, query, nil, APITimeout); err != nil {
		return geocode.Address{}, fmt.Errorf("failed to address details from Nominatim API: %w", err)
	}

	// Fill the geocode.Address struct
	address := geocode.Address{
		AddressFound: true,
		DisplayName:  result.DisplayName,
		Country:      result.Address.Country,
		State:        result.Address.State,
		Municipality: result.Address.Municipality,
		CityDistrict: result.Address.CityDistrict,
		Postcode:     result.Address.Postcode,
		City:         result.Address.City,
		Suburb:       result.Address.Suburb,
		Street:       result.Address.Road,
		HouseNumber:  result.Address.HouseNumber,
	}
	if result.Address.City == "" && result.Address.Town != "" {
		address.City = result.Address.Town
	}
	if result.Address.City == "" && result.Address.Town == "" && result.Address.Village != "" {
		address.City = result.Address.Village
	}
	address.Latitude, err = strconv.ParseFloat(result.APILat, 64)
	if err != nil {
		return geocode.Address{}, fmt.Errorf("failed to parse latitude from Nominatim API response: %w", err)
	}
	address.Longitude, err = strconv.ParseFloat(result.APILon, 64)
	if err != nil {
		return geocode.Address{}, fmt.Errorf("failed to parse longitude from Nominatim API response: %w", err)
	}

	return address, nil
}
