package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	AppEnv     string
	HTTPPort   string
	DatabaseURL string
	RedisURL   string

	JWTAccessSecret      string
	JWTAccessTTLMinutes  int
	RefreshTokenTTLHours int

	CookieDomain string
	CookieSecure bool
	CrossSiteCookies bool // true when frontend and backend are on different domains (e.g. Vercel + Render)
	CSRFSecret   string
	AllowedOrigin string // exact frontend origin for CORS; required when CrossSiteCookies is true

	PlaceProvider    string
	PlanningProvider string
	TransportProvider string

	RateLimitAuthPerMin      int
	RateLimitGeneratePerHour int

	LogLevel string
}

func Load() (*Config, error) {
	cfg := &Config{
		AppEnv:               getEnv("APP_ENV", "development"),
		HTTPPort:             getEnv("PORT", getEnv("HTTP_PORT", "8080")), // most PaaS inject PORT
		DatabaseURL:          os.Getenv("DATABASE_URL"),
		RedisURL:             os.Getenv("REDIS_URL"),
		JWTAccessSecret:      os.Getenv("JWT_ACCESS_SECRET"),
		JWTAccessTTLMinutes:  getEnvInt("JWT_ACCESS_TTL_MINUTES", 15),
		RefreshTokenTTLHours: getEnvInt("REFRESH_TOKEN_TTL_HOURS", 720),
		CookieDomain:         os.Getenv("COOKIE_DOMAIN"),
		CookieSecure:         getEnvBool("COOKIE_SECURE", false),
		CrossSiteCookies:     getEnvBool("CROSS_SITE_COOKIES", false),
		CSRFSecret:           os.Getenv("CSRF_SECRET"),
		AllowedOrigin:        os.Getenv("ALLOWED_ORIGIN"),
		PlaceProvider:        getEnv("PLACE_PROVIDER", "mock"),
		PlanningProvider:     getEnv("PLANNING_PROVIDER", "mock"),
		TransportProvider:    getEnv("TRANSPORT_PROVIDER", "mock"),
		RateLimitAuthPerMin:      getEnvInt("RATE_LIMIT_AUTH_PER_MIN", 10),
		RateLimitGeneratePerHour: getEnvInt("RATE_LIMIT_GENERATE_PER_HOUR", 6),
		LogLevel:             getEnv("LOG_LEVEL", "info"),
	}

	if cfg.CrossSiteCookies && cfg.AllowedOrigin == "" {
		return nil, fmt.Errorf("ALLOWED_ORIGIN is required when CROSS_SITE_COOKIES=true")
	}

	if cfg.CrossSiteCookies {
		// Browsers require Secure when SameSite=None; there is no safe way
		// around this, so we enforce it rather than silently failing to
		// set cookies at all.
		cfg.CookieSecure = true
	}

	if cfg.AppEnv == "production" {
		if cfg.JWTAccessSecret == "" {
			return nil, fmt.Errorf("JWT_ACCESS_SECRET is required in production")
		}
		if cfg.CSRFSecret == "" {
			return nil, fmt.Errorf("CSRF_SECRET is required in production")
		}
		if !cfg.CookieSecure {
			return nil, fmt.Errorf("COOKIE_SECURE must be true in production")
		}
	}
	if cfg.JWTAccessSecret == "" {
		cfg.JWTAccessSecret = "dev-only-insecure-secret-change-me"
	}
	if cfg.CSRFSecret == "" {
		cfg.CSRFSecret = "dev-only-insecure-csrf-secret"
	}
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}
