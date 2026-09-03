// Package config provides a strongly-typed configuration loader for the
// Somotracker backend service. Values are sourced from environment variables
// (typically injected by Doppler in non-local environments) and validated on
// load. Missing values fall back to safe defaults.
package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// Config is the strongly-typed application configuration.
//
// Values are populated by [Load] from environment variables. Direct construction
// of Config is reserved for tests; production code should always use [Load] or
// the Fx provider in [AsFxOption].
type Config struct {
	// Host is the interface address the HTTP server binds to.
	// Empty string means "all interfaces" (Fiber's ":port" form).
	Host string

	// Port is the TCP port the HTTP server listens on.
	// Defaults to 3030 when unset or invalid.
	Port int

	// Environment describes the deployment environment (e.g. "local",
	// "development", "staging", "production"). Populated from APP_ENV.
	Environment string

	// LogLevel is the zap log level ("debug", "info", "warn", "error").
	// Defaults to "info" when unset.
	LogLevel string
}

// ListenAddr returns the address string passed to fiber.App.Listen.
// Empty host yields ":port"; otherwise it returns "host:port".
func (c Config) ListenAddr() string {
	// Treat empty, 0.0.0.0, or localhost as binding to all interfaces for the server
	if c.Host == "" || c.Host == "0.0.0.0" || c.Host == "localhost" {
		return fmt.Sprintf(":%d", c.Port)
	}
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// IsProduction reports whether the service is running in a production-like
// environment. Used to switch between zap.NewProduction and zap.NewDevelopment.
func (c Config) IsProduction() bool {
	switch strings.ToLower(c.Environment) {
	case "production", "prod":
		return true
	default:
		return false
	}
}

// Load reads configuration from environment variables, validates required
// fields, and applies defaults. It is safe to call from package init or from
// an Fx provider.
//
// Recognized variables:
//
//	BACKEND_URL  Full URL of the backend (e.g. "http://api.example.com:8080").
//	             Host and port are derived from this when set.
//	APP_ENV      Environment name. Defaults to "local".
//	LOG_LEVEL    Zap log level. Defaults to "info".
//	BACKEND_HOST Optional host override (e.g. "127.0.0.1"). Takes precedence
//	             over the host portion of BACKEND_URL.
//	BACKEND_PORT Optional port override. Takes precedence over the port portion
//	             of BACKEND_URL. Defaults to 3030.
func Load() (*Config, error) {
	cfg := &Config{
		Host:        "",
		Port:        defaultPort,
		Environment: getEnv("APP_ENV", "local"),
		LogLevel:    strings.ToLower(getEnv("LOG_LEVEL", "info")),
	}

	if raw := os.Getenv("BACKEND_URL"); raw != "" {
		host, port, err := parseBackendURL(raw)
		if err != nil {
			return nil, fmt.Errorf("config.Load: invalid BACKEND_URL %q: %w", raw, err)
		}
		if host != "" {
			cfg.Host = host
		}
		if port != 0 {
			cfg.Port = port
		}
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

const defaultPort = 3030

func (c *Config) validate() error {
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("config.Load: port %d out of range (1-65535)", c.Port)
	}
	switch c.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("config.Load: invalid log level %q (want debug|info|warn|error)", c.LogLevel)
	}
	return nil
}

// parseBackendURL extracts host and port from a URL string. The scheme is
// optional. A missing port returns (host, 0, nil) so the caller can apply its
// own default.
func parseBackendURL(raw string) (host string, port int, err error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", 0, err
	}
	host = parsed.Hostname()
	if p := parsed.Port(); p != "" {
		port, err = strconv.Atoi(p)
		if err != nil {
			return "", 0, fmt.Errorf("non-numeric port %q", p)
		}
	}
	return host, port, nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
