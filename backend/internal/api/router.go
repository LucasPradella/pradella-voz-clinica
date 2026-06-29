package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/pradella/voz-clinica/internal/core"
	"github.com/pradella/voz-clinica/internal/services"
	"github.com/pradella/voz-clinica/internal/store"
)

// Deps bundles server-level dependencies injected into handlers.
type Deps struct {
	Config     *core.Config
	DB         *pgxpool.Pool
	UserStore  *store.UserStore
	AuditSvc   *services.AuditService
	SOAPSvc    *services.SOAPService
	Guardrail  *services.GuardrailChecker
	EvoStore   *store.EvolutionStore
	QuotaSvc   *services.QuotaService
	BillingSvc *services.BillingService
}

// New builds and returns the chi router with all routes registered.
func New(deps *Deps) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Logger)

	r.Get("/api/healthz", handleHealthz)

	// Public auth routes
	authHandler := &AuthHandler{
		userStore: deps.UserStore,
		auditSvc:  deps.AuditSvc,
		jwtSecret: deps.Config.JWTSecret,
	}
	r.Post("/api/auth/register", authHandler.Register)
	r.Post("/api/auth/login", authHandler.Login)

	// Stripe webhook — no JWT; uses Stripe-Signature header
	if deps.BillingSvc != nil {
		wh := &WebhookHandler{
			billingSvc: deps.BillingSvc,
			subStore:   deps.EvoStore,
		}
		r.Post("/api/webhooks/stripe", wh.StripeWebhook)
	}

	// Authenticated routes
	r.Group(func(r chi.Router) {
		r.Use(core.AuthMiddleware(deps.Config.JWTSecret))

		evoHandler := &EvolutionHandler{
			soapSvc:   deps.SOAPSvc,
			guardrail: deps.Guardrail,
			evoStore:  deps.EvoStore,
			auditSvc:  deps.AuditSvc,
			quotaSvc:  deps.QuotaSvc,
		}
		r.Post("/api/evolutions", evoHandler.Create)
		r.Patch("/api/evolutions/{id}", evoHandler.Patch)
		r.Get("/api/evolutions", evoHandler.List)
		r.Get("/api/evolutions/{id}", evoHandler.Get)

		subHandler := &SubscriptionHandler{
			evoStore:   deps.EvoStore,
			userStore:  deps.UserStore,
			quotaSvc:   deps.QuotaSvc,
			billingSvc: deps.BillingSvc,
		}
		r.Get("/api/subscription", subHandler.Get)
		r.Post("/api/subscription/checkout", subHandler.Checkout)
	})

	return r
}

// WriteError writes the standard error envelope { "error": { "code": "...", "message": "..." } }.
func WriteError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}

// WriteJSON writes a JSON response with the given status code.
func WriteJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
