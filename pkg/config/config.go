// Package config centralizes environment-based configuration shared by all services.
package config

import (
	"net"
	"net/url"
	"os"
)

// Config is the process-wide configuration, loaded once at startup.
type Config struct {
	ServiceName string
	Env         string

	HTTPAddr string

	LogLevel  string
	LogFormat string

	// Postgres: either a full DSN or discrete parts (used by k8s, where the
	// DSN is assembled from a ConfigMap + Secret).
	PostgresDSN      string
	PostgresHost     string
	PostgresPort     string
	PostgresUser     string
	PostgresPassword string
	PostgresDB       string

	NATSURL string
}

// Load reads configuration from the environment with sensible local defaults.
func Load() Config {
	return Config{
		ServiceName: env("SERVICE_NAME", "service"),
		Env:         env("ENV", "dev"),

		HTTPAddr: env("HTTP_ADDR", "0.0.0.0:8080"),

		LogLevel:  env("LOG_LEVEL", "info"),
		LogFormat: env("LOG_FORMAT", "text"),

		PostgresDSN:      os.Getenv("POSTGRES_DSN"),
		PostgresHost:     env("POSTGRES_HOST", "127.0.0.1"),
		PostgresPort:     env("POSTGRES_PORT", "5432"),
		PostgresUser:     env("POSTGRES_USER", "app"),
		PostgresPassword: env("POSTGRES_PASSWORD", "app"),
		PostgresDB:       env("POSTGRES_DB", "app"),

		NATSURL: env("NATS_URL", "nats://127.0.0.1:4222"),
	}
}

// DatabaseDSN returns the explicit DSN if set, otherwise assembles one
// from the discrete Postgres settings.
func (c Config) DatabaseDSN() string {
	if c.PostgresDSN != "" {
		return c.PostgresDSN
	}
	u := url.URL{Scheme: "postgres", User: url.UserPassword(c.PostgresUser, c.PostgresPassword), Host: net.JoinHostPort(c.PostgresHost, c.PostgresPort), Path: "/" + c.PostgresDB}
	u.RawQuery = "sslmode=disable"
	return u.String()
}

// IsProd reports whether the process runs in a production-like environment.
func (c Config) IsProd() bool { return c.Env == "prod" }

func env(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
