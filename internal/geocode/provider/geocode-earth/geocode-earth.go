// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package geocodeearth

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"golang.org/x/text/language"

	"github.com/wneessen/waybar-weather/internal/geocode"
	"github.com/wneessen/waybar-weather/internal/http"
)

const (
	APIEndpoint = "https://api.geocode.earth/v1/reverse"
	APITimeout  = time.Second * 10
	name        = "geocode-earth"
)

type GeocodeEarth struct {
	apikey string
	http   *http.Client
	lang   language.Tag
}

type Response struct {
	Features []Feature `json:"features"`
	Type     string    `json:"type"`
}

type Feature struct {
	Properties Properties `json:"properties"`
	Type       string     `json:"type"`
}

type Properties struct {
	DisplayName    string `json:"label"`
	City           string `json:"locality"`
	CityDistrict   string `json:"county"`
	Continent      string `json:"continent"`
	Country        string `json:"country"`
	CountryCode    string `json:"country_code"`
	HouseNumber    string `json:"housenumber"`
	PoliticalUnion string `json:"political_union"`
	Municipality   string `json:"neighbourhood"`
	Postcode       string `json:"postalcode"`
	Road           string `json:"street"`
	State          string `json:"region"`
	StateCode      string `json:"region_a"`
}

func New(client *http.Client, lang language.Tag, apikey string) *GeocodeEarth {
	return &GeocodeEarth{
		apikey: apikey,
		lang:   lang,
		http:   client,
	}
}

func (o *GeocodeEarth) Name() string {
	return name
}

func (o *GeocodeEarth) Reverse(ctx context.Context, lat, lon float64) (geocode.Address, error) {
	var response Response

	query := url.Values{}
	query.Set("api_key", o.apikey)
	query.Set("point.lat", fmt.Sprintf("%f", lat))
	query.Set("point.lon", fmt.Sprintf("%f", lon))
	query.Set("lang", o.lang.String())

	code, err := o.http.GetWithTimeout(ctx, APIEndpoint, &response, query, nil, APITimeout)
	if err != nil {
		return geocode.Address{}, fmt.Errorf("failed to retrieve address details from geocode.earth API: %w", err)
	}
	if code != 200 {
		return geocode.Address{}, fmt.Errorf("received non-positive response code from geocode.earth API: %d", code)
	}
	if len(response.Features) < 1 {
		return geocode.Address{}, fmt.Errorf("no address found for coordinates")
	}

	// Fill the geocode.Address struct
	result := response.Features[0].Properties
	address := geocode.Address{
		AddressFound: true,
		Latitude:     lat,
		Longitude:    lon,
		DisplayName:  result.DisplayName,
		Country:      result.Country,
		State:        result.State,
		Municipality: result.Municipality,
		CityDistrict: result.CityDistrict,
		Postcode:     result.Postcode,
		City:         result.City,
		Street:       result.Road,
		HouseNumber:  result.HouseNumber,
	}

	return address, nil
}
