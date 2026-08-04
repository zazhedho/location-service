package population

import (
	"errors"
	"time"
)

var ErrNotFound = errors.New("population not found")

const (
	Source        = "Ditjen Dukcapil Kemendagri"
	ReferenceDate = "2024-12-31"
)

type Population struct {
	Code          string    `json:"code"`
	Name          string    `json:"name"`
	Male          int64     `json:"male"`
	Female        int64     `json:"female"`
	Total         int64     `json:"total"`
	Source        string    `json:"source"`
	ReferenceDate string    `json:"reference_date"`
	ImportedAt    time.Time `json:"imported_at"`
}
