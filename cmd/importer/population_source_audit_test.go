package importer

import (
	"bufio"
	"os"
	"path/filepath"
	"testing"

	domainpopulation "location-service/internal/domain/population"
)

func TestPopulationSourceAudit(t *testing.T) {
	path := os.Getenv("WILAYAH_DATA_SQL")
	if path == "" {
		path = "/Users/zaqiakhana/Code/Project/wilayah-indonesia-api/init-db/02-data.sql"
	}
	file, err := os.Open(path)
	if err != nil {
		t.Skip(err)
	}
	defer file.Close()

	codes := make(map[string]struct{})
	national, provinces, regencies := 0, 0, 0
	stats, err := ParsePopulationTuples(file, func(item domainpopulation.Population) error {
		if _, exists := codes[item.Code]; exists {
			t.Errorf("duplicate population code %s", item.Code)
		}
		codes[item.Code] = struct{}{}
		switch {
		case item.Code == "0":
			national++
		case len(item.Code) == 2:
			provinces++
		case len(item.Code) == 5 && item.Code[2] == '.':
			regencies++
		default:
			t.Errorf("unexpected population code %s", item.Code)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if stats.RowsRead != 553 || stats.RowsParsed != 553 || national != 1 || provinces != 38 || regencies != 514 {
		t.Fatalf("source audit: stats=%+v national=%d provinces=%d regency_city=%d", stats, national, provinces, regencies)
	}

	rawPath := filepath.Join("..", "..", "data", "wilayah.sql")
	rawFile, err := os.Open(rawPath)
	if err != nil {
		t.Fatal(err)
	}
	defer rawFile.Close()
	rawCodes := make(map[string]struct{})
	scanner := bufio.NewScanner(rawFile)
	for scanner.Scan() {
		for _, match := range tuplePattern.FindAllStringSubmatch(scanner.Text(), -1) {
			rawCodes[unescapeSQL(match[1])] = struct{}{}
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	for code := range codes {
		if code != "0" {
			if _, ok := rawCodes[code]; !ok {
				t.Errorf("population code %s absent from raw location source", code)
			}
		}
	}
}
