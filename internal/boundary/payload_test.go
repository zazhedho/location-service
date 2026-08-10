package boundary

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestBoundaryPayloadRoundTrip(t *testing.T) {
	input := json.RawMessage(`[{"type":"Feature","geometry":{"type":"Polygon","coordinates":[[[106.8,-6.2]]]}}]`)
	encoded, err := EncodeBoundaryPayload(input)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	decoded, err := DecodeBoundaryPayload(bytes.NewReader(encoded))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	var want, got any
	if err := json.Unmarshal(input, &want); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(decoded, &got); err != nil {
		t.Fatal(err)
	}
	if !jsonEqual(want, got) {
		t.Fatalf("round trip mismatch: %s", decoded)
	}
}

func TestBoundaryPayloadRejectsMalformedInput(t *testing.T) {
	if _, err := EncodeBoundaryPayload(json.RawMessage(`{"type":"Feature"}`)); err == nil {
		t.Fatal("expected non-array payload to fail")
	}
	if _, err := DecodeBoundaryPayload(bytes.NewReader([]byte("not gzip"))); err == nil {
		t.Fatal("expected malformed gzip to fail")
	}
}

func jsonEqual(a, b any) bool {
	left, _ := json.Marshal(a)
	right, _ := json.Marshal(b)
	return bytes.Equal(left, right)
}
