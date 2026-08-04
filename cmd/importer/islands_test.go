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
('12.00.40002','Pulau Nias','-1.0679','97.6009','BP',1.2,''),
('12.04.40003','Pulau Semambawa','-0.9088','98.0191','',Perubahan nama pulau.,''),
('21.04.40002','Pulau Adu Besar','-0.2474','104.4388','Perubahan nama lama',0.1214,''),
('53.14.40073','Pulau Dengka',NULL,NULL,'0.5521',Perubahan nama Dengka.,'TBP');
INSERT INTO wilayah_pulau(kode,nama,lat,lng,status)
VALUES
('75.02.40001','Pulau Asiangi','-0.4903','122.3483','TBP');
`

	items := make([]domainisland.Island, 0, 6)
	stats, err := ParseIslandTuples(strings.NewReader(input), func(item domainisland.Island) error {
		items = append(items, item)
		return nil
	})
	if err != nil {
		t.Fatalf("ParseIslandTuples() error = %v", err)
	}
	if stats.RowsRead != 6 || len(items) != 6 {
		t.Fatalf("rows = %d, items = %d; want 6", stats.RowsRead, len(items))
	}
	if items[0].ProvinceCode != "11" || items[0].Name != "Pulau Da'A" || items[0].Area != nil {
		t.Fatalf("first island = %+v", items[0])
	}
	if items[1].Area == nil || *items[1].Area != 1.2 {
		t.Fatalf("second island area = %v; want 1.2", items[1].Area)
	}
	if items[2].Area != nil || items[2].Notes != "Perubahan nama pulau." {
		t.Fatalf("recovered island = %+v", items[2])
	}
	if items[3].Status != "" || items[3].Notes != "Perubahan nama lama" {
		t.Fatalf("recovered status island = %+v", items[3])
	}
	if items[4].Status != "TBP" || items[4].Area == nil || *items[4].Area != 0.5521 || items[4].Notes != "Perubahan nama Dengka." {
		t.Fatalf("shifted island = %+v", items[4])
	}
	if items[5].Area != nil || items[5].Notes != "" {
		t.Fatalf("five-column island = %+v", items[5])
	}
}

func TestParseIslandTuplesSkipsInvalidCode(t *testing.T) {
	input := `INSERT INTO wilayah_pulau(kode,nama,lat,lng,status)
VALUES ('11.01.bad','Pulau','0','0','TBP');`
	stats, err := ParseIslandTuples(strings.NewReader(input), func(domainisland.Island) error { return nil })
	if err != nil || stats.RowsRead != 1 || stats.RowsSkipped != 1 {
		t.Fatalf("ParseIslandTuples() stats = %+v, err = %v", stats, err)
	}
}
