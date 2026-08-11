package middlewares

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestRateLimitBlocksAfterLimitAndSetsHeaders(t *testing.T) {
	calls := 0
	handler := newRateLimitHandler(nil, rateLimitConfig{
		generalLimit:  2,
		searchLimit:   2,
		boundaryLimit: 2,
		window:        time.Hour,
	}, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.WriteHeader(http.StatusNoContent)
	}))

	first := serveRateLimitRequest(handler, http.MethodGet, "/api/locations/provinces", "203.0.113.10:1000")
	second := serveRateLimitRequest(handler, http.MethodGet, "/api/locations/provinces", "203.0.113.10:1000")
	third := serveRateLimitRequest(handler, http.MethodGet, "/api/locations/provinces", "203.0.113.10:1000")

	if first.Code != http.StatusNoContent || second.Code != http.StatusNoContent {
		t.Fatalf("allowed statuses = %d, %d; want 204, 204", first.Code, second.Code)
	}
	if third.Code != http.StatusTooManyRequests {
		t.Fatalf("blocked status = %d; want 429", third.Code)
	}
	if calls != 2 {
		t.Fatalf("next called %d times; want 2", calls)
	}
	if got := third.Header().Get("Retry-After"); got == "" {
		t.Fatal("Retry-After header missing")
	}
	if got := third.Header().Get("X-RateLimit-Limit"); got != "2" {
		t.Fatalf("limit header = %q; want 2", got)
	}
	if got := third.Header().Get("X-RateLimit-Remaining"); got != "0" {
		t.Fatalf("remaining header = %q; want 0", got)
	}
	if got := third.Header().Get("X-RateLimit-Reset"); got == "" {
		t.Fatal("reset header missing")
	}
	if _, err := strconv.ParseInt(third.Header().Get("Retry-After"), 10, 64); err != nil {
		t.Fatalf("Retry-After = %q; want integer seconds", third.Header().Get("Retry-After"))
	}
}

func TestRateLimitSeparatesClientIPs(t *testing.T) {
	handler := newRateLimitHandler(nil, rateLimitConfig{
		generalLimit:  1,
		searchLimit:   1,
		boundaryLimit: 1,
		window:        time.Hour,
	}, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	firstClient := serveRateLimitRequest(handler, http.MethodGet, "/api/locations/provinces", "203.0.113.10:1000")
	secondClient := serveRateLimitRequest(handler, http.MethodGet, "/api/locations/provinces", "203.0.113.11:1000")
	repeated := serveRateLimitRequest(handler, http.MethodGet, "/api/locations/provinces", "203.0.113.10:1000")

	if firstClient.Code != http.StatusNoContent || secondClient.Code != http.StatusNoContent {
		t.Fatalf("client statuses = %d, %d; want 204, 204", firstClient.Code, secondClient.Code)
	}
	if repeated.Code != http.StatusTooManyRequests {
		t.Fatalf("repeated client status = %d; want 429", repeated.Code)
	}
}

func TestRateLimitSeparatesRouteClasses(t *testing.T) {
	handler := newRateLimitHandler(nil, rateLimitConfig{
		generalLimit:  1,
		searchLimit:   1,
		boundaryLimit: 1,
		window:        time.Hour,
	}, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	paths := []string{
		"/api/locations/provinces",
		"/api/locations/search?q=aceh",
		"/api/locations/11.01/boundary",
	}

	for _, path := range paths {
		if response := serveRateLimitRequest(handler, http.MethodGet, path, "203.0.113.10:1000"); response.Code != http.StatusNoContent {
			t.Fatalf("first %s status = %d; want 204", path, response.Code)
		}
		if response := serveRateLimitRequest(handler, http.MethodGet, path, "203.0.113.10:1000"); response.Code != http.StatusTooManyRequests {
			t.Fatalf("second %s status = %d; want 429", path, response.Code)
		}
	}
}

func TestRateLimitSkipsOptionsAndHealth(t *testing.T) {
	calls := 0
	handler := newRateLimitHandler(nil, rateLimitConfig{
		generalLimit:  1,
		searchLimit:   1,
		boundaryLimit: 1,
		window:        time.Hour,
	}, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.WriteHeader(http.StatusNoContent)
	}))

	options := serveRateLimitRequest(handler, http.MethodOptions, "/api/locations/provinces", "203.0.113.10:1000")
	health := serveRateLimitRequest(handler, http.MethodGet, "/healthz", "203.0.113.10:1000")
	firstAPI := serveRateLimitRequest(handler, http.MethodGet, "/api/locations/provinces", "203.0.113.10:1000")
	secondAPI := serveRateLimitRequest(handler, http.MethodGet, "/api/locations/provinces", "203.0.113.10:1000")

	if options.Code != http.StatusNoContent || health.Code != http.StatusNoContent || firstAPI.Code != http.StatusNoContent {
		t.Fatalf("skipped/first statuses = %d, %d, %d; want 204", options.Code, health.Code, firstAPI.Code)
	}
	if secondAPI.Code != http.StatusTooManyRequests {
		t.Fatalf("second API status = %d; want 429", secondAPI.Code)
	}
	if calls != 3 {
		t.Fatalf("next called %d times; want 3", calls)
	}
}

func TestRateLimitDoesNotTrustForwardedIP(t *testing.T) {
	handler := newRateLimitHandler(nil, rateLimitConfig{
		generalLimit:  1,
		searchLimit:   1,
		boundaryLimit: 1,
		window:        time.Hour,
	}, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	first := httptest.NewRequest(http.MethodGet, "/api/locations/provinces", nil)
	first.RemoteAddr = "203.0.113.10:1000"
	first.Header.Set("X-Forwarded-For", "203.0.113.20")
	firstRecorder := httptest.NewRecorder()
	handler.ServeHTTP(firstRecorder, first)

	second := httptest.NewRequest(http.MethodGet, "/api/locations/provinces", nil)
	second.RemoteAddr = "203.0.113.10:1000"
	second.Header.Set("X-Forwarded-For", "203.0.113.21")
	secondRecorder := httptest.NewRecorder()
	handler.ServeHTTP(secondRecorder, second)

	if firstRecorder.Code != http.StatusNoContent {
		t.Fatalf("first status = %d; want 204", firstRecorder.Code)
	}
	if secondRecorder.Code != http.StatusTooManyRequests {
		t.Fatalf("second status = %d; want 429", secondRecorder.Code)
	}
}

func TestRateLimitFallsBackWhenRedisUnavailable(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr:         "127.0.0.1:1",
		DialTimeout:  10 * time.Millisecond,
		ReadTimeout:  10 * time.Millisecond,
		WriteTimeout: 10 * time.Millisecond,
	})
	defer client.Close()

	handler := newRateLimitHandler(client, rateLimitConfig{
		generalLimit:  1,
		searchLimit:   1,
		boundaryLimit: 1,
		window:        time.Hour,
	}, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	first := serveRateLimitRequest(handler, http.MethodGet, "/api/locations/provinces", "203.0.113.10:1000")
	second := serveRateLimitRequest(handler, http.MethodGet, "/api/locations/provinces", "203.0.113.10:1000")

	if first.Code != http.StatusNoContent {
		t.Fatalf("first status = %d; want 204", first.Code)
	}
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("fallback status = %d; want 429", second.Code)
	}
}

func serveRateLimitRequest(handler http.Handler, method, target, remoteAddr string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, target, nil)
	request.RemoteAddr = remoteAddr
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func TestRateLimitFallbackEvictsPreviousWindows(t *testing.T) {
	limiter := &rateLimiter{memory: make(map[string]memoryRateLimit)}
	resetAt := time.Unix(200, 0)

	limiter.takeMemory("first", 1, resetAt, 1)
	limiter.takeMemory("second", 1, resetAt, 1)
	limiter.takeMemory("first", 2, resetAt.Add(time.Hour), 1)

	if got := len(limiter.memory); got != 1 {
		t.Fatalf("memory entries = %d; want 1 after window cleanup", got)
	}
}

func TestRateLimitRetriesRedisAfterCooldown(t *testing.T) {
	limiter := &rateLimiter{}
	failedAt := time.Unix(100, 0)
	limiter.disableRedis(failedAt)

	if !limiter.redisIsUnavailable(failedAt.Add(redisRetryCooldown - time.Nanosecond)) {
		t.Fatal("Redis should remain disabled during retry cooldown")
	}
	if limiter.redisIsUnavailable(failedAt.Add(redisRetryCooldown)) {
		t.Fatal("Redis should be retried after retry cooldown")
	}
}
