package utils

import (
	"testing"
	"time"
)

func TestGetEnvParsesTypedValues(t *testing.T) {
	t.Setenv("TEST_ENV_INT", "42")
	t.Setenv("TEST_ENV_BOOL", "true")
	t.Setenv("TEST_ENV_DURATION", "2m")

	if got := GetEnv("TEST_ENV_INT", 7); got != 42 {
		t.Fatalf("int env = %d; want 42", got)
	}
	if got := GetEnv("TEST_ENV_BOOL", false); !got {
		t.Fatal("bool env = false; want true")
	}
	if got := GetEnv("TEST_ENV_DURATION", time.Second); got != 2*time.Minute {
		t.Fatalf("duration env = %s; want 2m", got)
	}
}

func TestGetEnvUsesDefaultForMissingOrInvalidValues(t *testing.T) {
	t.Setenv("TEST_ENV_INVALID_INT", "not-an-int")
	t.Setenv("TEST_ENV_INVALID_DURATION", "not-a-duration")

	if got := GetEnv("TEST_ENV_MISSING", 7); got != 7 {
		t.Fatalf("missing env = %d; want 7", got)
	}
	if got := GetEnv("TEST_ENV_INVALID_INT", 7); got != 7 {
		t.Fatalf("invalid int env = %d; want 7", got)
	}
	if got := GetEnv("TEST_ENV_INVALID_DURATION", time.Second); got != time.Second {
		t.Fatalf("invalid duration env = %s; want 1s", got)
	}
}

func TestDurationFromEnvUsesFirstPositiveValue(t *testing.T) {
	t.Setenv("TEST_ENV_DURATION_INVALID", "invalid")
	t.Setenv("TEST_ENV_DURATION_VALID", "90")

	got := DurationFromEnv(
		[]string{"TEST_ENV_DURATION_INVALID", "TEST_ENV_DURATION_VALID"},
		30*time.Second,
	)
	if got != 90*time.Second {
		t.Fatalf("duration = %s; want 1m30s", got)
	}
}
