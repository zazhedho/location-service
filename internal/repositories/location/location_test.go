package location

import (
	"bytes"
	"context"
	"io"
	"testing"

	"location-service/internal/boundary"
	domainlocation "location-service/internal/domain/location"
	"location-service/pkg/storage"
)

type boundaryStorageStub struct {
	body []byte
	err  error
}

func (s boundaryStorageStub) Upload(context.Context, string, io.Reader, int64, string) error {
	return nil
}
func (s boundaryStorageStub) Download(context.Context, string) (io.ReadCloser, error) {
	if s.err != nil {
		return nil, s.err
	}
	return io.NopCloser(bytes.NewReader(s.body)), nil
}
func (s boundaryStorageStub) Exists(context.Context, string) (bool, error) { return true, nil }
func (s boundaryStorageStub) Delete(context.Context, string) error         { return nil }
func (s boundaryStorageStub) URL(string) string                            { return "" }

func TestLoadBoundaryPayloadFromObject(t *testing.T) {
	payload, err := boundary.EncodeBoundaryPayload([]byte(`[[[1,2],[3,4],[1,2]]]`))
	if err != nil {
		t.Fatal(err)
	}
	got, err := loadBoundaryPayload(context.Background(), boundaryStorageStub{body: payload}, "boundaries/11.01.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "[[[1,2],[3,4],[1,2]]]" {
		t.Fatalf("payload=%s", got)
	}
}

func TestLoadBoundaryPayloadMapsMissingObject(t *testing.T) {
	_, err := loadBoundaryPayload(context.Background(), boundaryStorageStub{err: storage.ErrObjectNotFound}, "missing")
	if err != domainlocation.ErrBoundaryNotFound {
		t.Fatalf("err=%v, want %v", err, domainlocation.ErrBoundaryNotFound)
	}
}
