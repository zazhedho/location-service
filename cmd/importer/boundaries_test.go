package importer

import (
	"strings"
	"testing"
)

func TestParseBoundarySQL(t *testing.T) {
	input := `INSERT INTO wilayah_boundaries(kode,nama,lat,lng,path) VALUES
('11.01','Kabupaten D''Aceh',3.1,97.4,'[[[1,2],[3,4],[1,2]]]'),
('11.01.01','Bakongan',-2,100,'[[[0,0],[0,1],[1,1],[0,0]]]');`

	var rows []boundaryRow
	if err := parseBoundarySQL(strings.NewReader(input), func(row boundaryRow) error {
		rows = append(rows, row)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("parsed %d rows, want 2", len(rows))
	}
	if rows[0].Code != "11.01" || rows[0].Coordinates.Latitude != 3.1 || rows[0].Coordinates.Longitude != 97.4 {
		t.Fatalf("unexpected first row: %+v", rows[0])
	}
}

func TestValidateBoundaryRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name string
		code string
		lat  string
		lng  string
		path string
	}{
		{name: "code", code: "11.1", lat: "1", lng: "2", path: "[[[1,2]]]"},
		{name: "latitude", code: "11.01", lat: "91", lng: "2", path: "[[[1,2]]]"},
		{name: "json", code: "11.01", lat: "1", lng: "2", path: "not-json"},
		{name: "path coordinate", code: "11.01", lat: "1", lng: "2", path: "[[[91,2]]]"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := validateBoundary(test.code, test.lat, test.lng, test.path); err == nil {
				t.Fatal("validateBoundary returned nil")
			}
		})
	}
}
