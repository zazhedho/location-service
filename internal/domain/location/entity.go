package location

import (
	"encoding/json"
	"errors"
	"regexp"
)

var (
	ErrNotFound         = errors.New("location not found")
	ErrBoundaryNotFound = errors.New("boundary not found")
	locationCodePattern = regexp.MustCompile(`^(?:[0-9]{2}|[0-9]{2}\.[0-9]{2}|[0-9]{2}\.[0-9]{2}\.[0-9]{2}|[0-9]{2}\.[0-9]{2}\.[0-9]{2}\.[0-9]{4})$`)
	postalCodePattern   = regexp.MustCompile(`^[0-9]{5}$`)
)

type Item struct {
	Code       string `json:"code"`
	FullCode   string `json:"full_code,omitempty"`
	Name       string `json:"name"`
	Level      string `json:"level,omitempty"`
	ParentCode string `json:"parent_code,omitempty"`
	PostalCode string `json:"postal_code,omitempty"`
}

type LocationRef struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type PostalLocation struct {
	PostalCode string      `json:"postal_code"`
	Village    LocationRef `json:"village"`
	District   LocationRef `json:"district"`
	Regency    LocationRef `json:"regency"`
	Province   LocationRef `json:"province"`
}

type Coordinates struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type Boundary struct {
	Code        string          `json:"code"`
	Name        string          `json:"name,omitempty"`
	Latitude    float64         `json:"latitude"`
	Longitude   float64         `json:"longitude"`
	LeafletPath json.RawMessage `json:"leaflet_path"`
}

type Detail struct {
	Code        string       `json:"code"`
	FullCode    string       `json:"full_code"`
	Name        string       `json:"name"`
	Level       string       `json:"level"`
	ParentCode  string       `json:"parent_code,omitempty"`
	PostalCode  string       `json:"postal_code,omitempty"`
	Coordinates *Coordinates `json:"coordinates,omitempty"`
	HasBoundary bool         `json:"has_boundary"`
}

type ImportStats struct {
	Raw       int
	Provinces int
	Regencies int
	Districts int
	Villages  int
}

type Stats struct {
	Raw       int `json:"raw"`
	Provinces int `json:"provinces"`
	Regencies int `json:"regencies"`
	Districts int `json:"districts"`
	Villages  int `json:"villages"`
	Total     int `json:"total"`
}

type StatsScope struct {
	Level string
	Code  string
}

func IsValidCode(code string) bool {
	return locationCodePattern.MatchString(code)
}

func IsValidPostalCode(value string) bool {
	return postalCodePattern.MatchString(value)
}
