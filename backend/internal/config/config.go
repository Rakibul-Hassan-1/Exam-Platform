package config

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

// Config holds all runtime configuration, sourced from environment
// variables (and optionally a .env file for local development).
type Config struct {
	Port              string
	DatabaseURL       string
	JWTSecret         string
	AnthropicAPIKey   string
	DocsStoragePath   string
	WorkerPollSeconds int
	AllowedOrigin     string
}

// Load reads a .env file (if present) into the process environment,
// then builds a Config from environment variables with sane defaults.
func Load() Config {
	loadDotEnv(".env")

	return Config{
		Port:              getEnv("PORT", "8080"),
		DatabaseURL:       getEnv("DATABASE_URL", "postgres://examuser:exampass@localhost:5432/examplatform?sslmode=disable"),
		JWTSecret:         getEnv("JWT_SECRET", "dev-secret-change-me"),
		AnthropicAPIKey:   getEnv("ANTHROPIC_API_KEY", ""),
		DocsStoragePath:   getEnv("DOCS_STORAGE_PATH", "./storage/documents"),
		WorkerPollSeconds: getEnvInt("WORKER_POLL_SECONDS", 3),
		AllowedOrigin:     getEnv("ALLOWED_ORIGIN", "http://localhost:3000"),
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

// loadDotEnv is a minimal .env parser (KEY=VALUE per line, '#' comments)
// so the project has no third-party dependency just for local dev config.
func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, val)
		}
	}
}
