package importer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	domainarea "location-service/internal/domain/area"
)

func TestParseAreaTuplesOnlyReadsWilayahLuas(t *testing.T) {
	input := `
INSERT INTO wilayah_penduduk(kode,nama,jumlah)
VALUES ('96','Papua Barat Oaya','623186');
INSERT INTO wilayah_luas(kode,nama,luas)
VALUES
('11.1','Kabupaten Aceh Singkil',1851.615),
('53.02','Kab Timor Tengah Selatan',3931.747);
`

	var rows []domainarea.Area
	stats, err := ParseAreaTuples(strings.NewReader(input), func(item domainarea.Area) error {
		rows = append(rows, item)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if stats.RowsRead != 2 || len(rows) != 2 {
		t.Fatalf("rows = %d, emitted = %d; want 2", stats.RowsRead, len(rows))
	}
	if stats.CodeCorrections != 1 || stats.NameCorrections != 1 {
		t.Fatalf("corrections = code:%d name:%d; want code:1 name:1", stats.CodeCorrections, stats.NameCorrections)
	}
	if rows[0].Code != "11.10" || rows[0].AreaKM2 != 1851.615 {
		t.Fatalf("first area = %+v", rows[0])
	}
	if rows[1].Name != "Kabupaten Timor Tengah Selatan" {
		t.Fatalf("abbreviated name was not corrected: %+v", rows[1])
	}
}

func TestParseAreaTuplesRejectsInvalidArea(t *testing.T) {
	for _, value := range []string{"-1", "NaN", "Infinity", "-Infinity", "NULL"} {
		t.Run(value, func(t *testing.T) {
			input := "INSERT INTO wilayah_luas(kode,nama,luas) VALUES ('11','Aceh'," + value + ");"
			if _, err := ParseAreaTuples(strings.NewReader(input), func(domainarea.Area) error { return nil }); err == nil {
				t.Fatal("ParseAreaTuples() returned nil for invalid area")
			}
		})
	}
}

func TestLocalAreaSourceAudit(t *testing.T) {
	paths := []string{
		os.Getenv("WILAYAH_SOURCE_FILE"),
		filepath.Join("..", "..", "..", "wilayah-indonesia-api", "init-db", "02-data.sql"),
	}
	var path string
	for _, candidate := range paths {
		if candidate == "" {
			continue
		}
		if _, err := os.Stat(candidate); err == nil {
			path = candidate
			break
		}
	}
	if path == "" {
		t.Skip("wilayah-indonesia-api source not available")
	}

	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	var codes []string
	var names []string
	stats, err := ParseAreaTuples(file, func(item domainarea.Area) error {
		codes = append(codes, item.Code)
		names = append(names, item.Name)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if stats.RowsRead != 552 || len(codes) != 552 {
		t.Fatalf("area audit rows = %+v emitted=%d; want 552", stats, len(codes))
	}
	if stats.CodeCorrections != 5 || stats.NameCorrections != 1 {
		t.Fatalf("area audit corrections = %+v; want code=5 name=1", stats)
	}
	for _, code := range codes {
		if code == "11.1" {
			t.Fatal("uncorrected source code 11.1 emitted")
		}
	}
	for _, name := range names {
		if name == "Papua Barat Oaya" {
			t.Fatal("population-only source typo emitted from area tuples")
		}
	}
}
