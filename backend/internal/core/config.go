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
	LLMProvider  string // "claude" (default) or "gemini"
	AnthropicKey string
	GeminiKey    string
	OpenAIKey    string
	GroqKey      string
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
		LLMProvider:         envOrDefault("LLM_PROVIDER", "claude"),
		AnthropicKey:        os.Getenv("ANTHROPIC_API_KEY"),
		GeminiKey:           os.Getenv("GEMINI_API_KEY"),
		OpenAIKey:           os.Getenv("OPENAI_API_KEY"),
		GroqKey:             requireEnv("GROQ_API_KEY"),
		StripeKey:           os.Getenv("STRIPE_SECRET_KEY"),
		StripeWebhookSecret: os.Getenv("STRIPE_WEBHOOK_SECRET"),
	}

	dim, err := strconv.Atoi(envOrDefault("EMBEDDING_DIM", "1536"))
	if err != nil {
		return nil, fmt.Errorf("EMBEDDING_DIM must be an integer: %w", err)
	}
	cfg.EmbeddingDim = dim

	if cfg.DatabaseURL == "" || cfg.JWTSecret == "" || cfg.GroqKey == "" {
		return nil, fmt.Errorf("DATABASE_URL, JWT_SECRET and GROQ_API_KEY are required")
	}
	switch cfg.LLMProvider {
	case "claude":
		if cfg.AnthropicKey == "" {
			return nil, fmt.Errorf("ANTHROPIC_API_KEY is required when LLM_PROVIDER=claude")
		}
	case "gemini":
		if cfg.GeminiKey == "" {
			return nil, fmt.Errorf("GEMINI_API_KEY is required when LLM_PROVIDER=gemini")
		}
	case "groq":
		// reuses GROQ_API_KEY already validated above
	default:
		return nil, fmt.Errorf("LLM_PROVIDER must be 'claude', 'gemini' or 'groq', got %q", cfg.LLMProvider)
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
