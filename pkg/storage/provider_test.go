package storage

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestS3ProviderUsesPathStyleAndObjectOperations(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.Path)
		switch r.Method {
		case http.MethodPut:
			w.Header().Set("ETag", `"test-etag"`)
			w.WriteHeader(http.StatusOK)
		case http.MethodHead:
			w.Header().Set("Content-Length", "5")
			w.WriteHeader(http.StatusOK)
		case http.MethodGet:
			if r.URL.Query().Get("list-type") == "2" {
				_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><Name>location-assets</Name><KeyCount>1</KeyCount><MaxKeys>1000</MaxKeys><IsTruncated>false</IsTruncated><Contents><Key>boundaries/11.json.gz</Key></Contents></ListBucketResult>`))
			} else {
				_, _ = w.Write([]byte("hello"))
			}
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	provider, err := NewProvider(&Config{
		Provider:       "s3",
		Endpoint:       server.URL + "/storage/v1/s3",
		AccessKey:      "access",
		SecretKey:      "secret",
		BucketName:     "location-assets",
		BaseURL:        "https://cdn.example.com/location-assets",
		Region:         "us-east-1",
		ForcePathStyle: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	lister, ok := provider.(PrefixLister)
	if !ok {
		t.Fatal("provider does not support prefix listing")
	}
	keys, err := lister.List(t.Context(), "boundaries/")
	if err != nil || len(keys) != 1 {
		t.Fatalf("keys=%v err=%v", keys, err)
	}
	if err := provider.Upload(t.Context(), "boundaries/11.json.gz", bytes.NewReader([]byte("hello")), 5, "application/gzip"); err != nil {
		t.Fatal(err)
	}
	if exists, err := provider.Exists(t.Context(), "boundaries/11.json.gz"); err != nil || !exists {
		t.Fatalf("exists=%v err=%v", exists, err)
	}
	object, err := provider.Download(t.Context(), "boundaries/11.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(object)
	_ = object.Close()
	if err != nil || string(body) != "hello" {
		t.Fatalf("body=%q err=%v", body, err)
	}
	if err := provider.Delete(t.Context(), "boundaries/11.json.gz"); err != nil {
		t.Fatal(err)
	}
	if got := provider.URL("boundaries/11.json.gz"); got != "https://cdn.example.com/location-assets/boundaries/11.json.gz" {
		t.Fatalf("url=%q", got)
	}
	if !strings.Contains(strings.Join(paths, "\n"), "/storage/v1/s3/location-assets/boundaries/11.json.gz") {
		t.Fatalf("paths=%v", paths)
	}
}
