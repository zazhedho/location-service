package utils

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"time"
)

func LoadEnvFile(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		_ = os.Setenv(key, value)
	}
}

func Env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func GetEnv[T any](key string, def T) T {
	value, ok := os.LookupEnv(key)
	if !ok {
		return def
	}
	value = strings.TrimSpace(value)

	switch any(def).(type) {
	case int:
		if parsed, err := strconv.Atoi(value); err == nil {
			return any(parsed).(T)
		}
	case int32:
		if parsed, err := strconv.ParseInt(value, 10, 32); err == nil {
			return any(int32(parsed)).(T)
		}
	case int64:
		if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
			return any(parsed).(T)
		}
	case float32:
		if parsed, err := strconv.ParseFloat(value, 32); err == nil {
			return any(float32(parsed)).(T)
		}
	case float64:
		if parsed, err := strconv.ParseFloat(value, 64); err == nil {
			return any(parsed).(T)
		}
	case bool:
		if parsed, err := strconv.ParseBool(value); err == nil {
			return any(parsed).(T)
		}
	case string:
		return any(value).(T)
	case time.Duration:
		if strings.EqualFold(value, "eod") {
			location := time.FixedZone("WIB", 7*3600)
			now := time.Now().In(location)
			end := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, location)
			if end.After(now) {
				return any(end.Sub(now)).(T)
			}
			return def
		}
		if duration, err := time.ParseDuration(value); err == nil {
			return any(duration).(T)
		}
		if seconds, err := strconv.ParseInt(value, 10, 64); err == nil {
			return any(time.Duration(seconds) * time.Second).(T)
		}
	}

	return def
}

func DurationFromEnv(keys []string, fallback time.Duration) time.Duration {
	for _, key := range keys {
		if value := GetEnv(key, time.Duration(0)); value > 0 {
			return value
		}
	}
	return fallback
}
