package core

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all runtime configuration loaded from environment variables.
// Secrets are never hardcoded; they must be supplied via env or a managed secrets provider.
type Config struct {
	DatabaseURL  string
	JWTSecret    string
	Port         string
	AnthropicKey string
	OpenAIKey    string
	StripeKey    string
	StripeWebhookSecret string

	// Embedding model dimension for pgvector
	EmbeddingDim int
}

// Load reads configuration from environment variables and returns an error if any
// required value is missing.
func Load() (*Config, error) {
	cfg := &Config{
		DatabaseURL:         requireEnv("DATABASE_URL"),
		JWTSecret:           requireEnv("JWT_SECRET"),
		Port:                envOrDefault("PORT", "8080"),
		AnthropicKey:        requireEnv("ANTHROPIC_API_KEY"),
		OpenAIKey:           requireEnv("OPENAI_API_KEY"),
		StripeKey:           os.Getenv("STRIPE_SECRET_KEY"),
		StripeWebhookSecret: os.Getenv("STRIPE_WEBHOOK_SECRET"),
	}

	dim, err := strconv.Atoi(envOrDefault("EMBEDDING_DIM", "1536"))
	if err != nil {
		return nil, fmt.Errorf("EMBEDDING_DIM must be an integer: %w", err)
	}
	cfg.EmbeddingDim = dim

	if cfg.DatabaseURL == "" || cfg.JWTSecret == "" {
		return nil, fmt.Errorf("DATABASE_URL and JWT_SECRET are required")
	}

	return cfg, nil
}

func requireEnv(key string) string {
	return os.Getenv(key)
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
