package importer

import (
	"strings"
	"testing"
)

func TestParsePostalTuples(t *testing.T) {
	input := `
INSERT INTO wilayah_kodepos(kode,kodepos)
VALUES
('11.01.01.2001', '23773'),
('11.01.01.2002', '23773');
`

	rows := make(map[string]string)
	count, err := ParsePostalTuples(strings.NewReader(input), func(code, postalCode string) error {
		rows[code] = postalCode
		return nil
	})
	if err != nil {
		t.Fatalf("ParsePostalTuples() error = %v", err)
	}
	if count != 2 || len(rows) != 2 || rows["11.01.01.2001"] != "23773" {
		t.Fatalf("rows = %d, parsed = %#v", count, rows)
	}
}

func TestParsePostalTuplesRejectsInvalidPostalCode(t *testing.T) {
	input := `INSERT INTO wilayah_kodepos(kode,kodepos) VALUES ('11.01.01.2001', '2377');`
	if _, err := ParsePostalTuples(strings.NewReader(input), func(string, string) error { return nil }); err == nil {
		t.Fatal("ParsePostalTuples() error = nil")
	}
}

func TestLoadPostalCodesSource(t *testing.T) {
	postalCodes, err := LoadPostalCodes("../../data/kodepos.sql")
	if err != nil {
		t.Fatal(err)
	}
	if len(postalCodes) != 83762 {
		t.Fatalf("postal rows = %d, want 83762", len(postalCodes))
	}
}
