package media

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"location-service/pkg/storage"
	"location-service/utils"
)

func ConfigFromEnv() (*storage.Config, error) {
	provider := strings.ToLower(strings.TrimSpace(utils.Env("STORAGE_PROVIDER", "")))
	if provider == "" {
		return nil, nil
	}
	if provider != "r2" && provider != "s3" && provider != "minio" {
		return nil, fmt.Errorf("STORAGE_PROVIDER must be one of: r2, s3, minio")
	}
	config := &storage.Config{
		Provider:       provider,
		Endpoint:       utils.Env("STORAGE_ENDPOINT", ""),
		AccessKey:      utils.Env("STORAGE_ACCESS_KEY", ""),
		SecretKey:      utils.Env("STORAGE_SECRET_KEY", ""),
		BucketName:     utils.Env("STORAGE_BUCKET_NAME", ""),
		BaseURL:        utils.Env("STORAGE_BASE_URL", ""),
		Region:         utils.Env("STORAGE_REGION", "auto"),
		UseSSL:         parseBoolEnv("STORAGE_USE_SSL", provider != "minio"),
		ForcePathStyle: parseBoolEnv("STORAGE_FORCE_PATH_STYLE", provider != "s3"),
	}
	for name, value := range map[string]string{
		"STORAGE_ENDPOINT":    config.Endpoint,
		"STORAGE_ACCESS_KEY":  config.AccessKey,
		"STORAGE_SECRET_KEY":  config.SecretKey,
		"STORAGE_BUCKET_NAME": config.BucketName,
	} {
		if strings.TrimSpace(value) == "" {
			return nil, fmt.Errorf("%s is required when storage is configured", name)
		}
	}
	return config, nil
}

func InitStorage() (storage.Provider, error) {
	config, err := ConfigFromEnv()
	if err != nil {
		return nil, err
	}
	if config == nil {
		return nil, nil
	}
	return storage.NewProvider(config)
}

func RequiredStorage() (storage.Provider, error) {
	provider, err := InitStorage()
	if err != nil {
		return nil, err
	}
	if provider == nil {
		return nil, errors.New("storage is required for boundary import")
	}
	return provider, nil
}

func parseBoolEnv(key string, fallback bool) bool {
	raw := strings.TrimSpace(utils.Env(key, ""))
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return parsed
}
