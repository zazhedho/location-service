package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

type s3Provider struct {
	client     *s3.Client
	bucketName string
	baseURL    string
}

func NewProvider(config *Config) (Provider, error) {
	if config == nil {
		return nil, errors.New("storage config is required")
	}
	endpoint, err := normalizeEndpoint(config.Endpoint, config.UseSSL)
	if err != nil {
		return nil, err
	}
	awsConfig := aws.Config{
		Region:      config.Region,
		Credentials: credentials.NewStaticCredentialsProvider(config.AccessKey, config.SecretKey, ""),
	}
	client := s3.NewFromConfig(awsConfig, func(options *s3.Options) {
		options.BaseEndpoint = aws.String(endpoint)
		options.UsePathStyle = config.ForcePathStyle
	})
	return &s3Provider{
		client:     client,
		bucketName: config.BucketName,
		baseURL:    strings.TrimRight(config.BaseURL, "/"),
	}, nil
}

func normalizeEndpoint(raw string, useSSL bool) (string, error) {
	endpoint := strings.TrimSpace(raw)
	if endpoint == "" {
		return "", errors.New("storage endpoint is required")
	}
	if !strings.Contains(endpoint, "://") {
		scheme := "https"
		if !useSSL {
			scheme = "http"
		}
		endpoint = scheme + "://" + endpoint
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid storage endpoint")
	}
	return endpoint, nil
}

func (p *s3Provider) Upload(ctx context.Context, key string, body io.Reader, size int64, contentType string) error {
	input := &s3.PutObjectInput{
		Bucket:        aws.String(p.bucketName),
		Key:           aws.String(key),
		Body:          body,
		ContentLength: aws.Int64(size),
	}
	if contentType != "" {
		input.ContentType = aws.String(contentType)
	}
	_, err := p.client.PutObject(ctx, input)
	return err
}

func (p *s3Provider) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	output, err := p.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(p.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, mapStorageError(err)
	}
	return output.Body, nil
}

func (p *s3Provider) Exists(ctx context.Context, key string) (bool, error) {
	_, err := p.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(p.bucketName),
		Key:    aws.String(key),
	})
	if err == nil {
		return true, nil
	}
	if errors.Is(mapStorageError(err), ErrObjectNotFound) {
		return false, nil
	}
	return false, err
}

func (p *s3Provider) Delete(ctx context.Context, key string) error {
	_, err := p.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(p.bucketName),
		Key:    aws.String(key),
	})
	return mapStorageError(err)
}

func (p *s3Provider) List(ctx context.Context, prefix string) (map[string]struct{}, error) {
	keys := make(map[string]struct{})
	paginator := s3.NewListObjectsV2Paginator(p.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(p.bucketName),
		Prefix: aws.String(prefix),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, mapStorageError(err)
		}
		for _, object := range page.Contents {
			if object.Key != nil {
				keys[*object.Key] = struct{}{}
			}
		}
	}
	return keys, nil
}

func (p *s3Provider) URL(key string) string {
	if p.baseURL == "" {
		return ""
	}
	return strings.TrimRight(p.baseURL, "/") + "/" + strings.TrimLeft(key, "/")
}

func mapStorageError(err error) error {
	if err == nil {
		return nil
	}
	var responseErr *smithyhttp.ResponseError
	if errors.As(err, &responseErr) && responseErr.HTTPStatusCode() == http.StatusNotFound {
		return fmt.Errorf("%w: %v", ErrObjectNotFound, err)
	}
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) && (apiErr.ErrorCode() == "NoSuchKey" || apiErr.ErrorCode() == "NotFound") {
		return fmt.Errorf("%w: %v", ErrObjectNotFound, err)
	}
	return err
}
