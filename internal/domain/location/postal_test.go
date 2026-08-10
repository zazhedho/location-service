package location

import "testing"

func TestIsValidPostalCode(t *testing.T) {
	valid := []string{"00000", "23773", "99999"}
	for _, value := range valid {
		if !IsValidPostalCode(value) {
			t.Errorf("IsValidPostalCode(%q) = false, want true", value)
		}
	}
	for _, value := range []string{"", "2377", "237731", "23A73", " 23773"} {
		if IsValidPostalCode(value) {
			t.Errorf("IsValidPostalCode(%q) = true, want false", value)
		}
	}
}
