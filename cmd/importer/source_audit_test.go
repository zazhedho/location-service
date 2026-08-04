package importer

import (
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	domainisland "location-service/internal/domain/island"
)

func TestLocalSourceAudit(t *testing.T) {
	root := filepath.Join("..", "..", "..", "wilayah-indonesia-api", "init-db")
	islandFile, err := os.Open(filepath.Join(root, "02-data.sql"))
	if err != nil {
		t.Skip(err)
	}
	islandCodes := make(map[string]struct{})
	duplicateCodes := 0
	islandStats, err := ParseIslandTuples(islandFile, func(item domainisland.Island) error {
		if _, exists := islandCodes[item.Code]; exists {
			duplicateCodes++
		}
		islandCodes[item.Code] = struct{}{}
		return nil
	})
	islandFile.Close()
	if err != nil || islandStats.RowsRead != 17374 || islandStats.RowsSkipped != 1 || duplicateCodes != 80 {
		t.Fatalf("islands: stats=%+v duplicates=%d err=%v", islandStats, duplicateCodes, err)
	}

	paths, err := BoundaryFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	rows := 0
	boundaryCodes := make(map[string]struct{})
	for _, path := range paths {
		file, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		reader, err := gzip.NewReader(file)
		if err != nil {
			t.Fatal(err)
		}
		err = parseBoundarySQL(reader, func(row boundaryRow) error {
			rows++
			if _, exists := boundaryCodes[row.Code]; exists {
				t.Fatalf("duplicate boundary code %s", row.Code)
			}
			boundaryCodes[row.Code] = struct{}{}
			return nil
		})
		reader.Close()
		file.Close()
		if err != nil {
			t.Fatal(err)
		}
	}
	if rows != 91241 {
		t.Fatalf("boundaries: rows=%d", rows)
	}
}
