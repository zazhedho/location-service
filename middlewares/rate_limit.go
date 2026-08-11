package middlewares

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"location-service/utils"
)

const fixedWindowScript = `
local count = redis.call("INCR", KEYS[1])
if count == 1 then
  redis.call("EXPIRE", KEYS[1], ARGV[1])
end
return count
`

const (
	defaultRateLimitWindow   = time.Minute
	defaultGeneralRateLimit  = 60
	defaultSearchRateLimit   = 20
	defaultBoundaryRateLimit = 30
	// ponytail: fixed retry cooldown avoids a Redis timeout per request; add adaptive backoff only if Redis flaps.
	redisRetryCooldown = 30 * time.Second
)

type rateLimitRoute string

const (
	generalRoute  rateLimitRoute = "general"
	searchRoute   rateLimitRoute = "search"
	boundaryRoute rateLimitRoute = "boundary"
)

type rateLimitConfig struct {
	generalLimit  int
	searchLimit   int
	boundaryLimit int
	window        time.Duration
}

type memoryRateLimit struct {
	windowID int64
	count    int64
}

type rateLimiter struct {
	redis *redis.Client
	cfg   rateLimitConfig

	mu                    sync.Mutex
	memory                map[string]memoryRateLimit
	memoryCleanupWindowID int64
	redisUnavailableUntil time.Time
}

type rateLimitResult struct {
	allowed   bool
	limit     int64
	remaining int64
	resetAt   time.Time
}

// RateLimit wraps public GET/HEAD routes with fixed-window request limits.
func RateLimit(redisClient *redis.Client, next http.Handler) http.Handler {
	return newRateLimitHandler(redisClient, rateLimitConfigFromEnv(), next)
}

func newRateLimitHandler(redisClient *redis.Client, cfg rateLimitConfig, next http.Handler) http.Handler {
	cfg = normalizeRateLimitConfig(cfg)
	limiter := &rateLimiter{
		redis:  redisClient,
		cfg:    cfg,
		memory: make(map[string]memoryRateLimit),
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if shouldSkipRateLimit(r) {
			next.ServeHTTP(w, r)
			return
		}

		class := rateLimitClass(r.URL.Path)
		limit := limiter.limit(class)
		now := time.Now()
		result := limiter.take(r.Context(), class, clientIP(r), limit, now)
		setRateLimitHeaders(w, result)
		if !result.allowed {
			w.Header().Set("Retry-After", strconv.FormatInt(retryAfterSeconds(result.resetAt, now), 10))
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func normalizeRateLimitConfig(cfg rateLimitConfig) rateLimitConfig {
	if cfg.generalLimit <= 0 {
		cfg.generalLimit = defaultGeneralRateLimit
	}
	if cfg.searchLimit <= 0 {
		cfg.searchLimit = defaultSearchRateLimit
	}
	if cfg.boundaryLimit <= 0 {
		cfg.boundaryLimit = defaultBoundaryRateLimit
	}
	if cfg.window <= 0 {
		cfg.window = defaultRateLimitWindow
	}
	return cfg
}

func rateLimitConfigFromEnv() rateLimitConfig {
	return normalizeRateLimitConfig(rateLimitConfig{
		generalLimit:  utils.GetEnv("RATE_LIMIT_GENERAL_LIMIT", defaultGeneralRateLimit),
		searchLimit:   utils.GetEnv("RATE_LIMIT_SEARCH_LIMIT", defaultSearchRateLimit),
		boundaryLimit: utils.GetEnv("RATE_LIMIT_BOUNDARY_LIMIT", defaultBoundaryRateLimit),
		window:        utils.DurationFromEnv([]string{"RATE_LIMIT_WINDOW"}, defaultRateLimitWindow),
	})
}

func shouldSkipRateLimit(r *http.Request) bool {
	return r.Method == http.MethodOptions || r.URL.Path == "/healthz" ||
		(r.Method != http.MethodGet && r.Method != http.MethodHead)
}

func rateLimitClass(path string) rateLimitRoute {
	switch {
	case path == "/api/locations/search":
		return searchRoute
	case strings.HasPrefix(path, "/api/locations/") && strings.HasSuffix(path, "/boundary"):
		return boundaryRoute
	default:
		return generalRoute
	}
}

func (l *rateLimiter) limit(class rateLimitRoute) int {
	switch class {
	case searchRoute:
		return l.cfg.searchLimit
	case boundaryRoute:
		return l.cfg.boundaryLimit
	default:
		return l.cfg.generalLimit
	}
}

func (l *rateLimiter) take(ctx context.Context, class rateLimitRoute, ip string, limit int, now time.Time) rateLimitResult {
	windowID, resetAt := rateLimitWindow(now, l.cfg.window)
	baseKey := fmt.Sprintf("location:rate-limit:%s:%s", class, ip)

	if l.redis != nil && !l.redisIsUnavailable(now) {
		redisContext, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
		count, err := l.redis.Eval(redisContext, fixedWindowScript, []string{
			fmt.Sprintf("%s:%d", baseKey, windowID),
		}, rateLimitWindowSeconds(l.cfg.window)).Int64()
		cancel()
		if err == nil {
			l.enableRedis()
			return rateLimitResult{
				allowed:   count <= int64(limit),
				limit:     int64(limit),
				remaining: maxInt64(int64(limit)-count, 0),
				resetAt:   resetAt,
			}
		}
		l.disableRedis(now)
	}

	return l.takeMemory(baseKey, windowID, resetAt, limit)
}

func (l *rateLimiter) takeMemory(key string, windowID int64, resetAt time.Time, limit int) rateLimitResult {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.memoryCleanupWindowID != windowID {
		for staleKey, entry := range l.memory {
			if entry.windowID != windowID {
				delete(l.memory, staleKey)
			}
		}
		l.memoryCleanupWindowID = windowID
	}

	entry := l.memory[key]
	if entry.windowID != windowID {
		entry = memoryRateLimit{windowID: windowID}
	}
	entry.count++
	l.memory[key] = entry

	count := entry.count
	return rateLimitResult{
		allowed:   count <= int64(limit),
		limit:     int64(limit),
		remaining: maxInt64(int64(limit)-count, 0),
		resetAt:   resetAt,
	}
}

func (l *rateLimiter) redisIsUnavailable(now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return now.Before(l.redisUnavailableUntil)
}

func (l *rateLimiter) disableRedis(now time.Time) {
	l.mu.Lock()
	l.redisUnavailableUntil = now.Add(redisRetryCooldown)
	l.mu.Unlock()
}

func (l *rateLimiter) enableRedis() {
	l.mu.Lock()
	l.redisUnavailableUntil = time.Time{}
	l.mu.Unlock()
}

func rateLimitWindow(now time.Time, window time.Duration) (int64, time.Time) {
	start := now.Truncate(window)
	return start.UnixNano() / window.Nanoseconds(), start.Add(window)
}

func rateLimitWindowSeconds(window time.Duration) int64 {
	seconds := int64(window / time.Second)
	if window%time.Second != 0 {
		seconds++
	}
	if seconds < 1 {
		return 1
	}
	return seconds
}

func retryAfterSeconds(resetAt, now time.Time) int64 {
	remaining := resetAt.Sub(now)
	seconds := int64(remaining / time.Second)
	if remaining%time.Second != 0 {
		seconds++
	}
	if seconds < 1 {
		return 1
	}
	return seconds
}

func setRateLimitHeaders(w http.ResponseWriter, result rateLimitResult) {
	w.Header().Set("X-RateLimit-Limit", strconv.FormatInt(result.limit, 10))
	w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(result.remaining, 10))
	w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(result.resetAt.Unix(), 10))
}

func clientIP(r *http.Request) string {
	remoteAddr := strings.TrimSpace(r.RemoteAddr)
	if host, _, err := net.SplitHostPort(remoteAddr); err == nil && host != "" {
		return host
	}
	if ip := net.ParseIP(remoteAddr); ip != nil {
		return ip.String()
	}
	if remoteAddr == "" {
		return "unknown"
	}
	return remoteAddr
}

func maxInt64(value, minimum int64) int64 {
	if value < minimum {
		return minimum
	}
	return value
}
