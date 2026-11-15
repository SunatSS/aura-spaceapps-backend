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
	HTTPAddr  string     // AURA_HTTP_ADDR; else ":$PORT" (Render); else ":8080"
	DBPath    string     // AURA_DB_PATH, default "aura.db"
	LogLevel  slog.Level // AURA_LOG_LEVEL: debug|info|warn|error, default "info"
	StaticDir string     // AURA_STATIC_DIR: built frontend to serve at "/"; empty = API only
	Seed      bool       // AURA_SEED: "auto" (default) fills an empty DB with demo data; "off" disables
	// AURA_INGEST_TOKEN: bearer token for POST /api/v1/ingest (onboard systems).
	// Empty = ingest disabled, except in demo mode, which uses DemoIngestToken.
	IngestToken string
}

// DemoIngestToken is the ingest token of the public demo (documented in the README).
const DemoIngestToken = "aura-demo-ingest"

// Load reads configuration from the environment.
func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:  getenv("AURA_HTTP_ADDR", ""),
		DBPath:    getenv("AURA_DB_PATH", "aura.db"),
		StaticDir: getenv("AURA_STATIC_DIR", ""),
	}
	if cfg.HTTPAddr == "" {
		cfg.HTTPAddr = ":" + getenv("PORT", "8080")
	}
	if err := cfg.LogLevel.UnmarshalText([]byte(getenv("AURA_LOG_LEVEL", "info"))); err != nil {
		return Config{}, fmt.Errorf("config: AURA_LOG_LEVEL: %w", err)
	}
	switch seed := getenv("AURA_SEED", "auto"); seed {
	case "auto":
		cfg.Seed = true
	case "off":
	default:
		return Config{}, fmt.Errorf("config: AURA_SEED must be auto or off, got %q", seed)
	}
	cfg.IngestToken = getenv("AURA_INGEST_TOKEN", "")
	if cfg.IngestToken == "" && cfg.Seed {
		cfg.IngestToken = DemoIngestToken
	}
	return cfg, nil
}

func getenv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
