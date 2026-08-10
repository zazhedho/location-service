package storage

import (
	"context"
	"errors"
	"io"
)

var ErrObjectNotFound = errors.New("storage object not found")

type Provider interface {
	Upload(ctx context.Context, key string, body io.Reader, size int64, contentType string) error
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	Exists(ctx context.Context, key string) (bool, error)
	Delete(ctx context.Context, key string) error
	URL(key string) string
}

// PrefixLister is an optional capability used to avoid one existence request
// per object during resumable bulk imports.
type PrefixLister interface {
	List(ctx context.Context, prefix string) (map[string]struct{}, error)
}

type Config struct {
	Provider       string
	Endpoint       string
	AccessKey      string
	SecretKey      string
	BucketName     string
	BaseURL        string
	Region         string
	UseSSL         bool
	ForcePathStyle bool
}
