// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package opencage

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
	APIEndpoint = "https://api.opencagedata.com/geocode/v1/json"
	APITimeout  = time.Second * 10
	name        = "opencage"
)

type OpenCage struct {
	apikey string
	http   *http.Client
	lang   language.Tag
}

type Response struct {
	Results      []Result `json:"results"`
	TotalResults int      `json:"total_results"`
}

type Result struct {
	Components  Components `json:"components"`
	DisplayName string     `json:"formatted"`
	Geometry    Geometry   `json:"geometry"`
}

type Components struct {
	NomalizedCity  string `json:"_normalized_city"`
	City           string `json:"city"`
	CityDistrict   string `json:"city_district"`
	Continent      string `json:"continent"`
	Country        string `json:"country"`
	CountryCode    string `json:"country_code"`
	HouseNumber    string `json:"house_number"`
	PoliticalUnion string `json:"political_union"`
	Municipality   string `json:"municipality"`
	Postcode       string `json:"postcode"`
	Road           string `json:"road"`
	State          string `json:"state"`
	StateCode      string `json:"state_code"`
	Suburb         string `json:"suburb"`
}

type Geometry struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lng"`
}

func New(client *http.Client, lang language.Tag, apikey string) *OpenCage {
	return &OpenCage{
		apikey: apikey,
		lang:   lang,
		http:   client,
	}
}

func (o *OpenCage) Name() string {
	return name
}

func (o *OpenCage) Reverse(ctx context.Context, lat, lon float64) (geocode.Address, error) {
	var response Response
	apiUrl, err := url.Parse(APIEndpoint)
	if err != nil {
		return geocode.Address{}, fmt.Errorf("failed to parse API endpoint: %w", err)
	}

	query := apiUrl.Query()
	query.Set("key", o.apikey)
	query.Set("q", fmt.Sprintf("%f,%f", lat, lon))
	query.Set("no_annotations", "1")
	query.Set("no_record", "1")
	query.Set("language", o.lang.String())
	apiUrl.RawQuery = query.Encode()

	if _, err = o.http.GetWithTimeout(ctx, apiUrl.String(), &response, nil, APITimeout); err != nil {
		return geocode.Address{}, fmt.Errorf("failed to address details from OpenCage API: %w", err)
	}
	if response.TotalResults != 1 {
		return geocode.Address{}, fmt.Errorf("unambigous amount of results returned for coordinates: %d",
			response.TotalResults)
	}

	// Fill the geocode.Address struct
	result := response.Results[0].Components
	address := geocode.Address{
		AddressFound: true,
		Latitude:     response.Results[0].Geometry.Lat,
		Longitude:    response.Results[0].Geometry.Lon,
		DisplayName:  response.Results[0].DisplayName,
		Country:      result.Country,
		State:        result.State,
		Municipality: result.Municipality,
		CityDistrict: result.CityDistrict,
		Postcode:     result.Postcode,
		City:         result.City,
		Suburb:       result.Suburb,
		Street:       result.Road,
		HouseNumber:  result.HouseNumber,
	}

	return address, nil
}
