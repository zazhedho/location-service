package media

import "testing"

func TestConfigFromEnvAcceptsOnlySupportedProviders(t *testing.T) {
	for _, provider := range []string{"r2", "s3", "minio"} {
		t.Setenv("STORAGE_PROVIDER", provider)
		t.Setenv("STORAGE_ENDPOINT", "localhost:9000")
		t.Setenv("STORAGE_ACCESS_KEY", "access")
		t.Setenv("STORAGE_SECRET_KEY", "secret")
		t.Setenv("STORAGE_BUCKET_NAME", "location-assets")
		if _, err := ConfigFromEnv(); err != nil {
			t.Fatalf("provider %q: %v", provider, err)
		}
	}
	t.Setenv("STORAGE_PROVIDER", "supabase")
	if _, err := ConfigFromEnv(); err == nil {
		t.Fatal("expected supabase to be configured through provider s3")
	}
}

func TestConfigFromEnvIsDisabledWhenProviderIsEmpty(t *testing.T) {
	t.Setenv("STORAGE_PROVIDER", "")
	config, err := ConfigFromEnv()
	if err != nil || config != nil {
		t.Fatalf("got config=%v err=%v, want disabled storage", config, err)
	}
}

func TestRequiredStorageRejectsDisabledStorage(t *testing.T) {
	t.Setenv("STORAGE_PROVIDER", "")
	if _, err := RequiredStorage(); err == nil {
		t.Fatal("expected required storage error")
	}
}
