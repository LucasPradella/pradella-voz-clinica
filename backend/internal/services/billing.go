package services

import (
	"fmt"

	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/checkout/session"
	"github.com/stripe/stripe-go/v79/customer"
	"github.com/stripe/stripe-go/v79/webhook"
)

// BillingService handles Stripe Checkout and webhook processing.
type BillingService struct {
	secretKey      string
	webhookSecret  string
	priceID        string // Stripe Price ID for the Pro subscription
	successURL     string
	cancelURL      string
}

func NewBillingService(secretKey, webhookSecret, priceID, appBaseURL string) *BillingService {
	stripe.Key = secretKey
	return &BillingService{
		secretKey:     secretKey,
		webhookSecret: webhookSecret,
		priceID:       priceID,
		successURL:    appBaseURL + "/account?checkout=success",
		cancelURL:     appBaseURL + "/account?checkout=cancel",
	}
}

// CreateCheckoutSession creates a Stripe Checkout session for a Pro upgrade.
// Returns the checkout URL that the client should redirect to.
func (s *BillingService) CreateCheckoutSession(userID, email, stripeCustomerID string) (string, error) {
	params := &stripe.CheckoutSessionParams{
		Mode: stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(s.priceID),
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL: stripe.String(s.successURL),
		CancelURL:  stripe.String(s.cancelURL),
		Metadata: map[string]string{
			"user_id": userID,
		},
	}

	// Reuse Stripe customer if one already exists.
	if stripeCustomerID != "" {
		params.Customer = stripe.String(stripeCustomerID)
	} else if email != "" {
		cust, err := customer.New(&stripe.CustomerParams{Email: stripe.String(email)})
		if err != nil {
			return "", fmt.Errorf("create stripe customer: %w", err)
		}
		params.Customer = stripe.String(cust.ID)
	}

	sess, err := session.New(params)
	if err != nil {
		return "", fmt.Errorf("create checkout session: %w", err)
	}
	return sess.URL, nil
}

// VerifyWebhookSignature validates the Stripe-Signature header and returns the event.
// Returns an error if the signature is invalid.
func (s *BillingService) VerifyWebhookSignature(payload []byte, sigHeader string) (*stripe.Event, error) {
	event, err := webhook.ConstructEvent(payload, sigHeader, s.webhookSecret)
	if err != nil {
		return nil, fmt.Errorf("webhook signature invalid: %w", err)
	}
	return &event, nil
}
