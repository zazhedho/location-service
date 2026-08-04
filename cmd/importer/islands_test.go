package importer

import (
	"strings"
	"testing"

	domainisland "location-service/internal/domain/island"
)

func TestParseIslandTuples(t *testing.T) {
	input := `
INSERT INTO wilayah (kode, nama) VALUES ('11','Aceh');
INSERT INTO wilayah_pulau(kode,nama,lat,lng,status,luas,notes)
VALUES
('11.01.40001','Pulau Da''A','-3.3176','97.1283','TBP',NULL,'note, one'),
('12.00.40002','Pulau Nias','-1.0679','97.6009','BP',1.2,'');
INSERT INTO wilayah_pulau(kode,nama,lat,lng,status)
VALUES
('75.02.40001','Pulau Asiangi','-0.4903','122.3483','TBP');
`

	items := make([]domainisland.Island, 0, 3)
	stats, err := ParseIslandTuples(strings.NewReader(input), func(item domainisland.Island) error {
		items = append(items, item)
		return nil
	})
	if err != nil {
		t.Fatalf("ParseIslandTuples() error = %v", err)
	}
	if stats.RowsRead != 3 || len(items) != 3 {
		t.Fatalf("rows = %d, items = %d; want 3", stats.RowsRead, len(items))
	}
	if items[0].ProvinceCode != "11" || items[0].Name != "Pulau Da'A" || items[0].Area != nil {
		t.Fatalf("first island = %+v", items[0])
	}
	if items[1].Area == nil || *items[1].Area != 1.2 {
		t.Fatalf("second island area = %v; want 1.2", items[1].Area)
	}
	if items[2].Area != nil || items[2].Notes != "" {
		t.Fatalf("five-column island = %+v", items[2])
	}
}

func TestParseIslandTuplesRejectsInvalidCode(t *testing.T) {
	input := `INSERT INTO wilayah_pulau(kode,nama,lat,lng,status)
VALUES ('11.01.bad','Pulau','0','0','TBP');`
	if _, err := ParseIslandTuples(strings.NewReader(input), func(domainisland.Island) error { return nil }); err == nil {
		t.Fatal("ParseIslandTuples() error = nil; want invalid code error")
	}
}
