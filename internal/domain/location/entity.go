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
)

type Item struct {
	Code       string `json:"code"`
	FullCode   string `json:"full_code,omitempty"`
	Name       string `json:"name"`
	Level      string `json:"level,omitempty"`
	ParentCode string `json:"parent_code,omitempty"`
}

type Centroid struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type Boundary struct {
	Code        string          `json:"code"`
	Centroid    Centroid        `json:"centroid"`
	LeafletPath json.RawMessage `json:"leaflet_path"`
}

type Detail struct {
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Level       string    `json:"level"`
	Centroid    *Centroid `json:"centroid"`
	HasBoundary bool      `json:"has_boundary"`
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
