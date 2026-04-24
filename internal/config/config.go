package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppAddr               string
	ReadTimeout           time.Duration
	WriteTimeout          time.Duration
	VictoriaLogsBaseURL   string
	VictoriaLogsAccountID string
	VictoriaLogsProjectID string
	VictoriaLogsUsername  string
	VictoriaLogsPassword  string
	VictoriaLogsTimeout   time.Duration
	DefaultLookback       time.Duration
	MaxLogLines           int
	MaxRawLines           int
}

func Load() Config {
	loadEnvFiles(
		"config/app.env",
		"config/app.local.env",
		".env",
		".env.local",
	)

	return Config{
		AppAddr:               getEnv("APP_ADDR", "127.0.0.1:8080"),
		ReadTimeout:           getDuration("APP_READ_TIMEOUT", 15*time.Second),
		WriteTimeout:          getDuration("APP_WRITE_TIMEOUT", 15*time.Second),
		VictoriaLogsBaseURL:   os.Getenv("VICTORIALOGS_BASE_URL"),
		VictoriaLogsAccountID: getEnv("VICTORIALOGS_ACCOUNT_ID", "0"),
		VictoriaLogsProjectID: os.Getenv("VICTORIALOGS_PROJECT_ID"),
		VictoriaLogsUsername:  os.Getenv("VICTORIALOGS_USERNAME"),
		VictoriaLogsPassword:  os.Getenv("VICTORIALOGS_PASSWORD"),
		VictoriaLogsTimeout:   getDuration("VICTORIALOGS_TIMEOUT", 20*time.Second),
		DefaultLookback:       getDuration("TRACE_DEFAULT_LOOKBACK", 6*time.Hour),
		MaxLogLines:           getInt("TRACE_MAX_LOG_LINES", 500),
		MaxRawLines:           getInt("TRACE_MAX_RAW_LINES", 500),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func getDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := parseFlexibleDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

func parseFlexibleDuration(value string) (time.Duration, error) {
	value = strings.TrimSpace(strings.ToLower(value))
	if strings.HasSuffix(value, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(value, "d"))
		if err != nil {
			return 0, err
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	return time.ParseDuration(value)
}
