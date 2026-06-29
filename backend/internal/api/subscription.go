package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/pradella/voz-clinica/internal/core"
	"github.com/pradella/voz-clinica/internal/models"
	"github.com/pradella/voz-clinica/internal/services"
	"github.com/pradella/voz-clinica/internal/store"
)

// SubscriptionHandler serves /api/subscription routes.
type SubscriptionHandler struct {
	evoStore   *store.EvolutionStore
	userStore  *store.UserStore
	quotaSvc   *services.QuotaService
	billingSvc *services.BillingService
}

// Get handles GET /api/subscription.
func (h *SubscriptionHandler) Get(w http.ResponseWriter, r *http.Request) {
	claims, ok := core.ClaimsFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "unauthorized", "missing auth")
		return
	}

	sub, err := h.evoStore.GetSubscription(r.Context(), claims.UserID)
	if errors.Is(err, store.ErrNotFound) {
		WriteError(w, http.StatusNotFound, "not_found", "subscription not found")
		return
	}
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", "could not fetch subscription")
		return
	}

	plan := models.Plan(claims.Plan)

	resp := models.SubscriptionResponse{
		Plan:   plan,
		Status: sub.Status,
	}

	if sub.CurrentPeriodEnd != nil {
		formatted := sub.CurrentPeriodEnd.Format(time.RFC3339)
		resp.CurrentPeriodEnd = &formatted
	}

	if plan == models.PlanPro {
		resp.Quota = models.Quota{Used: 0, Limit: nil}
	} else {
		used, limit, err := h.quotaSvc.GetUsage(r.Context(), claims.UserID)
		if err == nil {
			resp.Quota = models.Quota{Used: used, Limit: limit}
		}
	}

	WriteJSON(w, http.StatusOK, resp)
}

// Checkout handles POST /api/subscription/checkout — initiates Stripe upgrade flow.
func (h *SubscriptionHandler) Checkout(w http.ResponseWriter, r *http.Request) {
	claims, ok := core.ClaimsFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "unauthorized", "missing auth")
		return
	}

	sub, _ := h.evoStore.GetSubscription(r.Context(), claims.UserID)
	var stripeCustomerID string
	if sub != nil && sub.StripeCustomerID != nil {
		stripeCustomerID = *sub.StripeCustomerID
	}

	checkoutURL, err := h.billingSvc.CreateCheckoutSession(claims.UserID, claims.Email, stripeCustomerID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "billing_error", "could not create checkout session")
		return
	}

	WriteJSON(w, http.StatusOK, models.CheckoutResponse{CheckoutURL: checkoutURL})
}
