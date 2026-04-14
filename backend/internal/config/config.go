package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	ServerPort       string
	PostgresHost     string
	PostgresPort     string
	PostgresDB       string
	PostgresUser     string
	PostgresPassword string
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
