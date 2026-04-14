package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	ServerPort       string
	PostgresHost     string
	PostgresPort     string
	PostgresDB       string
	PostgresUser     string
	PostgresPassword string
	FrontendOrigin   string
	AuthCookieSecure bool
	JWTSecret        string
	OllamaBaseURL    string
	OllamaEmbedModel string
	OllamaTimeout    time.Duration
}

func Load() Config {
	return Config{
		ServerPort:       getEnv("SERVER_PORT", "8080"),
		PostgresHost:     getEnv("POSTGRES_HOST", "localhost"),
		PostgresPort:     getEnv("POSTGRES_PORT", "5432"),
		PostgresDB:       getEnv("POSTGRES_DB", "knowledge_base"),
		PostgresUser:     getEnv("POSTGRES_USER", "kb_user"),
		PostgresPassword: getEnv("POSTGRES_PASSWORD", "kb_password"),
		FrontendOrigin:   getEnv("FRONTEND_ORIGIN", "http://localhost:5173"),
		AuthCookieSecure: getBoolEnv("AUTH_COOKIE_SECURE", false),
		JWTSecret:        getEnv("JWT_SECRET", "change-me"),
		OllamaBaseURL:    getEnv("OLLAMA_BASE_URL", ""),
		OllamaEmbedModel: getEnv("OLLAMA_EMBED_MODEL", "nomic-embed-text"),
		OllamaTimeout:    getDurationEnv("OLLAMA_TIMEOUT", 15*time.Second),
	}
}

func (c Config) DatabaseURL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		c.PostgresUser,
		c.PostgresPassword,
		c.PostgresHost,
		c.PostgresPort,
		c.PostgresDB,
	)
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		parsed, err := time.ParseDuration(value)
		if err == nil {
			return parsed
		}
	}

	return fallback
}

func getBoolEnv(key string, fallback bool) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	switch value {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}
