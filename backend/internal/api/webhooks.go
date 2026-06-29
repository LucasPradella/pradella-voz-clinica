package api

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/stripe/stripe-go/v79"

	"github.com/pradella/voz-clinica/internal/services"
)

// WebhookHandler handles POST /api/webhooks/stripe.
// No JWT — Stripe signature is used for authentication.
type WebhookHandler struct {
	billingSvc *services.BillingService
	subStore   subscriptionStore
}

// subscriptionStore abstracts the DB writes needed during webhook processing.
type subscriptionStore interface {
	ActivatePro(userID, stripeCustomerID, stripeSubscriptionID string, periodEnd interface{}) error
	DeactivatePro(stripeSubscriptionID string) error
}

// StripeWebhook handles POST /api/webhooks/stripe.
func (h *WebhookHandler) StripeWebhook(w http.ResponseWriter, r *http.Request) {
	const maxBody = 65536 // 64 KB max webhook payload
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBody))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "bad_request", "could not read body")
		return
	}

	sigHeader := r.Header.Get("Stripe-Signature")
	event, err := h.billingSvc.VerifyWebhookSignature(body, sigHeader)
	if err != nil {
		slog.Warn("stripe webhook signature verification failed", "err", err)
		WriteError(w, http.StatusUnauthorized, "invalid_signature", "webhook signature verification failed")
		return
	}

	switch event.Type {
	case "customer.subscription.created", "customer.subscription.updated":
		h.handleSubscriptionUpsert(event)
	case "customer.subscription.deleted":
		h.handleSubscriptionDeleted(event)
	default:
		// Unhandled events are acknowledged but not processed.
		slog.Debug("unhandled stripe event", "type", event.Type)
	}

	WriteJSON(w, http.StatusOK, map[string]bool{"received": true})
}

func (h *WebhookHandler) handleSubscriptionUpsert(event *stripe.Event) {
	var sub stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
		slog.Error("unmarshal stripe subscription", "err", err)
		return
	}

	userID := sub.Metadata["user_id"]
	if userID == "" {
		slog.Warn("stripe subscription missing user_id metadata", "id", sub.ID)
		return
	}

	active := sub.Status == stripe.SubscriptionStatusActive || sub.Status == stripe.SubscriptionStatusTrialing
	if !active {
		return
	}

	var periodEnd interface{}
	if sub.CurrentPeriodEnd > 0 {
		periodEnd = sub.CurrentPeriodEnd
	}

	if err := h.subStore.ActivatePro(userID, sub.Customer.ID, sub.ID, periodEnd); err != nil {
		slog.Error("activate pro subscription", "user_id", userID, "err", err)
	}
}

func (h *WebhookHandler) handleSubscriptionDeleted(event *stripe.Event) {
	var sub stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
		slog.Error("unmarshal stripe subscription", "err", err)
		return
	}

	if err := h.subStore.DeactivatePro(sub.ID); err != nil {
		slog.Error("deactivate pro subscription", "id", sub.ID, "err", err)
	}
}
