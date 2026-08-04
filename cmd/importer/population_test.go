package importer

import (
	"strings"
	"testing"

	domainpopulation "location-service/internal/domain/population"
)

func TestParsePopulationTuples(t *testing.T) {
	input := `
INSERT INTO wilayah (kode, nama) VALUES ('11','Aceh');
INSERT INTO wilayah_pulau(kode,nama,lat,lng,status)
VALUES ('11.01.40001','Pulau','0','0','TBP');
INSERT INTO wilayah_penduduk(kode,nama,pria,wanita,total)
VALUES
('11','Aceh','2815060','2808419','5623479'),
('11.01','Kabupaten Aceh Selatan','120041','119588','239629'),
('12.01','Kabupaten D''Aceh, Barat','1',2,3);
`

	items := make([]domainpopulation.Population, 0, 3)
	stats, err := ParsePopulationTuples(strings.NewReader(input), func(item domainpopulation.Population) error {
		items = append(items, item)
		return nil
	})
	if err != nil {
		t.Fatalf("ParsePopulationTuples() error = %v", err)
	}
	if stats.RowsRead != 3 || stats.RowsParsed != 3 || len(items) != 3 {
		t.Fatalf("stats = %+v, items = %d; want 3 parsed rows", stats, len(items))
	}
	if items[2].Name != "Kabupaten D'Aceh, Barat" || items[2].Male != 1 || items[2].Female != 2 || items[2].Total != 3 {
		t.Fatalf("third population = %+v", items[2])
	}
	if items[2].Source != domainpopulation.Source || items[2].ReferenceDate != domainpopulation.ReferenceDate {
		t.Fatalf("provenance = %q/%q", items[2].Source, items[2].ReferenceDate)
	}
}

func TestParsePopulationTuplesRejectsInvalidCounts(t *testing.T) {
	for name, tuple := range map[string]string{
		"negative": "('11','Aceh','-1','2','1')",
		"mismatch": "('11','Aceh','1','2','4')",
		"null":     "('11','Aceh',NULL,'2','2')",
	} {
		t.Run(name, func(t *testing.T) {
			input := "INSERT INTO wilayah_penduduk(kode,nama,pria,wanita,total) VALUES " + tuple + ";"
			if _, err := ParsePopulationTuples(strings.NewReader(input), func(domainpopulation.Population) error { return nil }); err == nil {
				t.Fatal("ParsePopulationTuples() error = nil")
			}
		})
	}
}
