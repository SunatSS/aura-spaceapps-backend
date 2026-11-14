// Package config loads runtime configuration from environment variables.
package config

import (
	"fmt"
	"log/slog"
	"os"
)

// Config holds all runtime settings. Every field has a working local default,
// so the server starts with zero configuration.
type Config struct {
	HTTPAddr string     // AURA_HTTP_ADDR, default ":8080"
	DBPath   string     // AURA_DB_PATH, default "aura.db"
	LogLevel slog.Level // AURA_LOG_LEVEL: debug|info|warn|error, default "info"
}

// Load reads configuration from the environment.
func Load() (Config, error) {
	cfg := Config{
		HTTPAddr: getenv("AURA_HTTP_ADDR", ":8080"),
		DBPath:   getenv("AURA_DB_PATH", "aura.db"),
	}
	if err := cfg.LogLevel.UnmarshalText([]byte(getenv("AURA_LOG_LEVEL", "info"))); err != nil {
		return Config{}, fmt.Errorf("config: AURA_LOG_LEVEL: %w", err)
	}
	return cfg, nil
}

func getenv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
