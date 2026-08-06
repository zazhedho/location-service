package importer

import (
	"strings"
	"testing"
)

func TestLocationTruncateIncludesRawLocationDependents(t *testing.T) {
	for _, table := range []string{"location_boundaries", "location_population", "location_areas", "raw_locations"} {
		if !strings.Contains(truncateLocationsSQL, table) {
			t.Fatalf("truncate statement does not include %s", table)
		}
	}
}
