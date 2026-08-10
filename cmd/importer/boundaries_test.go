package importer

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"location-service/internal/boundary"
	domainlocation "location-service/internal/domain/location"
	"location-service/pkg/storage"
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

type memoryBoundaryProvider struct {
	objects map[string][]byte
	mu      sync.Mutex
}

func (p *memoryBoundaryProvider) Upload(_ context.Context, key string, body io.Reader, _ int64, _ string) error {
	data, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.objects[key] = data
	return nil
}
func (p *memoryBoundaryProvider) Download(context.Context, string) (io.ReadCloser, error) {
	return nil, storage.ErrObjectNotFound
}
func (p *memoryBoundaryProvider) Exists(context.Context, string) (bool, error) { return false, nil }
func (p *memoryBoundaryProvider) Delete(context.Context, string) error         { return nil }
func (p *memoryBoundaryProvider) URL(string) string                            { return "" }
func (p *memoryBoundaryProvider) List(context.Context, string) (map[string]struct{}, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	keys := make(map[string]struct{}, len(p.objects))
	for key := range p.objects {
		keys[key] = struct{}{}
	}
	return keys, nil
}

func TestBoundaryObjectKeyAndPayload(t *testing.T) {
	provider := &memoryBoundaryProvider{objects: make(map[string][]byte)}
	row := boundaryRow{
		Code:        "11.01",
		Coordinates: domainlocation.Coordinates{Latitude: 3.1, Longitude: 97.4},
		LeafletPath: []byte(`[[[1,2],[3,4],[1,2]]]`),
	}
	if err := uploadBoundaryObject(context.Background(), provider, row); err != nil {
		t.Fatal(err)
	}
	payload, ok := provider.objects["boundaries/11.01.json.gz"]
	if !ok {
		t.Fatal("boundary object key missing")
	}
	decoded, err := boundary.DecodeBoundaryPayload(bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	if string(decoded) != string(row.LeafletPath) {
		t.Fatalf("payload=%s", decoded)
	}
}

func TestUploadBoundaryFileUsesConcurrentObjectUploads(t *testing.T) {
	path := filepath.Join(t.TempDir(), "boundaries.sql.gz")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	compressed := gzip.NewWriter(file)
	_, _ = compressed.Write([]byte(`INSERT INTO wilayah_boundaries(kode,nama,lat,lng,path) VALUES
('11.01','Kabupaten',3.1,97.4,'[[[1,2],[3,4],[1,2]]]'),
('11.02','Unknown',3.2,97.5,'[[[1,2],[3,4],[1,2]]]');`))
	if err := compressed.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	provider := &memoryBoundaryProvider{objects: make(map[string][]byte)}
	if err := uploadBoundaryFile(context.Background(), provider, path, map[string]struct{}{"11.01": {}}, nil); err != nil {
		t.Fatal(err)
	}
	provider.mu.Lock()
	defer provider.mu.Unlock()
	if len(provider.objects) != 1 {
		t.Fatalf("uploaded objects=%d, want 1", len(provider.objects))
	}
	if _, ok := provider.objects["boundaries/11.01.json.gz"]; !ok {
		t.Fatal("known boundary object missing")
	}
}
