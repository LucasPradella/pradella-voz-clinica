package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	openai "github.com/sashabaranov/go-openai"

	"github.com/pradella/voz-clinica/internal/api"
	"github.com/pradella/voz-clinica/internal/core"
	"github.com/pradella/voz-clinica/internal/rag"
	"github.com/pradella/voz-clinica/internal/services"
	"github.com/pradella/voz-clinica/internal/store"
)

func main() {
	cfg, err := core.Load()
	if err != nil {
		slog.Error("load config", "err", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := core.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("connect database", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	transcriptionSvc, err := services.NewTranscriptionService(cfg.GroqKey)
	if err != nil {
		slog.Error("init transcription service", "err", err)
		os.Exit(1)
	}

	openaiClient := openai.NewClient(cfg.OpenAIKey)
	ragStore := rag.NewStore(pool)

	var llmClient services.SOAPGenerator
	switch cfg.LLMProvider {
	case "gemini":
		llmClient = services.NewGeminiSOAPClient(cfg.GeminiKey)
		slog.Info("LLM backend: Gemini", "model", "gemini-2.0-flash")
	case "groq":
		llmClient = services.NewGroqSOAPClient(cfg.GroqKey)
		slog.Info("LLM backend: Groq", "model", "llama-3.3-70b-versatile")
	default:
		llmClient = services.NewClaudeSOAPClient(cfg.AnthropicKey)
		slog.Info("LLM backend: Claude", "model", "claude-sonnet-4-6")
	}

	soapSvc := services.NewSOAPService(transcriptionSvc, llmClient, ragStore, openaiClient)

	var billingSvc *services.BillingService
	if cfg.StripeKey != "" {
		priceID := os.Getenv("STRIPE_PRICE_ID")
		appBaseURL := os.Getenv("APP_BASE_URL")
		if appBaseURL == "" {
			appBaseURL = "http://localhost:" + cfg.Port
		}
		billingSvc = services.NewBillingService(cfg.StripeKey, cfg.StripeWebhookSecret, priceID, appBaseURL)
	}

	deps := &api.Deps{
		Config:     cfg,
		DB:         pool,
		UserStore:  store.NewUserStore(pool),
		AuditSvc:   services.NewAuditService(pool),
		SOAPSvc:    soapSvc,
		Guardrail:  services.NewGuardrailChecker(),
		EvoStore:   store.NewEvolutionStore(pool),
		QuotaSvc:   services.NewQuotaService(pool),
		BillingSvc: billingSvc,
	}

	handler := api.New(deps)
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		slog.Info("server starting", "port", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "err", err)
	}
}
