package area

import (
	"errors"
	"time"
)

const (
	Source        = "Badan Informasi Geospasial (BIG)"
	ReferenceDate = "2024-12-16"
)

var (
	ErrNotFound     = errors.New("area not found")
	ErrCodeRequired = errors.New("code is required")
	ErrCodeInvalid  = errors.New("code is invalid")
)

type Area struct {
	Code          string    `json:"code"`
	Name          string    `json:"name"`
	AreaKM2       float64   `json:"area_km2"`
	Source        string    `json:"source"`
	ReferenceDate string    `json:"reference_date"`
	ImportedAt    time.Time `json:"imported_at"`
}
