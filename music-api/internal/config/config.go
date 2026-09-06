package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	DatabaseURL string
	JWTSecret   string
	Port        string

	ResendAPIKey string
	EmailFrom    string
	FrontendURL  string

	CORSAllowedOrigins []string

	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration

	MaxJSONBodyBytes   int64
	MaxUploadBodyBytes int64
}

func Load() (*Config, error) {
	cfg := &Config{
		DatabaseURL: strings.TrimSpace(
			os.Getenv("DATABASE_URL"),
		),

		JWTSecret: strings.TrimSpace(
			os.Getenv("JWT_SECRET"),
		),

		Port: getEnv(
			"PORT",
			"7000",
		),

		ResendAPIKey: strings.TrimSpace(
			os.Getenv("RESEND_API_KEY"),
		),

		EmailFrom: strings.TrimSpace(
			os.Getenv("EMAIL_FROM"),
		),

		FrontendURL: strings.TrimSpace(
			os.Getenv("FRONTEND_URL"),
		),

		CORSAllowedOrigins: getCSVEnv(
			"CORS_ALLOWED_ORIGINS",
			"http://localhost:5173",
		),

		ReadTimeout: getDurationEnv(
			"HTTP_READ_TIMEOUT",
			15*time.Second,
		),

		WriteTimeout: getDurationEnv(
			"HTTP_WRITE_TIMEOUT",
			30*time.Second,
		),

		IdleTimeout: getDurationEnv(
			"HTTP_IDLE_TIMEOUT",
			60*time.Second,
		),

		ShutdownTimeout: getDurationEnv(
			"SHUTDOWN_TIMEOUT",
			10*time.Second,
		),

		MaxJSONBodyBytes: getInt64Env(
			"MAX_JSON_BODY_BYTES",
			1<<20,
		),

		MaxUploadBodyBytes: getInt64Env(
			"MAX_UPLOAD_BODY_BYTES",
			50<<20,
		),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf(
			"DATABASE_URL is not set",
		)
	}

	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf(
			"JWT_SECRET is not set",
		)
	}

	return cfg, nil
}

func getEnv(
	key string,
	fallback string,
) string {
	value := strings.TrimSpace(
		os.Getenv(key),
	)

	if value == "" {
		return fallback
	}

	return value
}

func getDurationEnv(
	key string,
	fallback time.Duration,
) time.Duration {
	value := strings.TrimSpace(
		os.Getenv(key),
	)

	if value == "" {
		return fallback
	}

	duration, err := time.ParseDuration(
		value,
	)
	if err != nil {
		return fallback
	}

	return duration
}

func getInt64Env(
	key string,
	fallback int64,
) int64 {
	value := strings.TrimSpace(
		os.Getenv(key),
	)

	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseInt(
		value,
		10,
		64,
	)
	if err != nil || parsed <= 0 {
		return fallback
	}

	return parsed
}

func getCSVEnv(
	key string,
	fallback string,
) []string {
	value := getEnv(
		key,
		fallback,
	)

	parts := strings.Split(
		value,
		",",
	)

	result := make(
		[]string,
		0,
		len(parts),
	)

	for _, part := range parts {
		part = strings.TrimSpace(
			part,
		)

		if part != "" {
			result = append(
				result,
				part,
			)
		}
	}

	return result
}
