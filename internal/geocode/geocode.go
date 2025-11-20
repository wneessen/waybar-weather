// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package geocode

import "context"

type Address struct {
	AddressFound bool
	Latitude     float64
	Longitude    float64
	Altitude     float64
	DisplayName  string
	Country      string
	State        string
	Municipality string
	CityDistrict string
	Postcode     string
	City         string
	Suburb       string
	Street       string
	HouseNumber  string
}

type Geocoder interface {
	Name() string
	Reverse(ctx context.Context, lat, lon float64) (Address, error)
}
