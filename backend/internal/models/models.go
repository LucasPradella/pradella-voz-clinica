package models

import (
	"time"

	"github.com/pgvector/pgvector-go"
)

// User represents a healthcare professional account.
type User struct {
	ID           string    `json:"id" db:"id"`
	Email        string    `json:"email" db:"email"`
	PasswordHash string    `json:"-" db:"password_hash"`
	Plan         Plan      `json:"plan" db:"plan"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

type Plan string

const (
	PlanFree Plan = "free"
	PlanPro  Plan = "pro"
)

// Subscription links a user to their billing cycle.
type Subscription struct {
	ID                   string     `json:"id" db:"id"`
	UserID               string     `json:"user_id" db:"user_id"`
	Type                 Plan       `json:"type" db:"type"`
	Status               SubStatus  `json:"status" db:"status"`
	StripeCustomerID     *string    `json:"stripe_customer_id,omitempty" db:"stripe_customer_id"`
	StripeSubscriptionID *string    `json:"stripe_subscription_id,omitempty" db:"stripe_subscription_id"`
	CurrentPeriodEnd     *time.Time `json:"current_period_end,omitempty" db:"current_period_end"`
}

type SubStatus string

const (
	SubStatusActive   SubStatus = "active"
	SubStatusCanceled SubStatus = "canceled"
	SubStatusPastDue  SubStatus = "past_due"
)

// UsageQuota tracks monthly usage for Free users.
type UsageQuota struct {
	ID     string `json:"id" db:"id"`
	UserID string `json:"user_id" db:"user_id"`
	Period string `json:"period" db:"period"` // "YYYY-MM"
	Count  int    `json:"count" db:"count"`
}

// CIDSuggestion is a diagnostic code suggestion (never an automatic assignment).
type CIDSuggestion struct {
	Code        string `json:"code"`
	Description string `json:"description"`
}

// ConfidenceFlag marks a span of text with low transcription confidence.
type ConfidenceFlag struct {
	Span   string `json:"span"`
	Reason string `json:"reason"`
}

// SourceRef identifies a clinical guideline source used for RAG.
type SourceRef struct {
	Origin  string `json:"origin"`
	Version string `json:"version"`
}

// Evolution is the structured clinical note produced from the audio.
// No patient PII is stored (FR-017a).
type Evolution struct {
	ID              string           `json:"id" db:"id"`
	UserID          string           `json:"user_id" db:"user_id"`
	Label           *string          `json:"label,omitempty" db:"label"`
	SOAP            SOAP             `json:"soap"`
	CIDSuggestions  []CIDSuggestion  `json:"cid_suggestions" db:"cid_suggestions"`
	ConfidenceFlags []ConfidenceFlag `json:"confidence_flags" db:"confidence_flags"`
	Status          EvoStatus        `json:"status" db:"status"`
	SourceRefs      []SourceRef      `json:"source_refs" db:"source_refs"`
	CreatedAt       time.Time        `json:"created_at" db:"created_at"`
}

type SOAP struct {
	S string `json:"s" db:"soap_s"`
	O string `json:"o" db:"soap_o"`
	A string `json:"a" db:"soap_a"`
	P string `json:"p" db:"soap_p"`
}

type EvoStatus string

const (
	EvoStatusDraft     EvoStatus = "draft"
	EvoStatusFinalized EvoStatus = "finalized"
)

// ClinicalSource is a RAG knowledge base chunk.
type ClinicalSource struct {
	ID        string          `json:"id" db:"id"`
	Title     string          `json:"title" db:"title"`
	Origin    string          `json:"origin" db:"origin"`
	Version   string          `json:"version" db:"version"`
	ChunkText string          `json:"chunk_text" db:"chunk_text"`
	Embedding pgvector.Vector `json:"-" db:"embedding"`
	UpdatedAt time.Time       `json:"updated_at" db:"updated_at"`
}

// AuditLog records access events for LGPD compliance.
type AuditLog struct {
	ID         string                 `json:"id" db:"id"`
	UserID     *string                `json:"user_id,omitempty" db:"user_id"`
	Action     string                 `json:"action" db:"action"`
	ResourceID *string                `json:"resource_id,omitempty" db:"resource_id"`
	Metadata   map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt  time.Time              `json:"created_at" db:"created_at"`
}

// --- DTOs ---

// RegisterRequest is the payload for POST /api/auth/register.
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginRequest is the payload for POST /api/auth/login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthResponse is returned after register or login.
type AuthResponse struct {
	Token string   `json:"token"`
	User  UserInfo `json:"user"`
}

// UserInfo is the public representation of a user (no sensitive fields).
type UserInfo struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Plan  Plan   `json:"plan"`
}

// CreateEvolutionRequest is the metadata attached to POST /api/evolutions (multipart).
type CreateEvolutionRequest struct {
	Label *string `json:"label,omitempty"`
}

// PatchEvolutionRequest is the payload for PATCH /api/evolutions/{id}.
type PatchEvolutionRequest struct {
	Label          *string          `json:"label,omitempty"`
	SOAP           *SOAP            `json:"soap,omitempty"`
	CIDSuggestions []CIDSuggestion  `json:"cid_suggestions,omitempty"`
	Status         *EvoStatus       `json:"status,omitempty"`
}

// EvolutionResponse is returned by POST and GET /api/evolutions/{id}.
type EvolutionResponse struct {
	ID              *string          `json:"id"` // null for Free (ephemeral)
	SOAP            SOAP             `json:"soap"`
	CIDSuggestions  []CIDSuggestion  `json:"cid_suggestions"`
	ConfidenceFlags []ConfidenceFlag `json:"confidence_flags"`
	SourceRefs      []SourceRef      `json:"source_refs"`
	Status          EvoStatus        `json:"status"`
}

// ErrorResponse is the standard API error envelope.
type ErrorResponse struct {
	Error   APIError `json:"error"`
	Upgrade bool     `json:"upgrade,omitempty"`
}

// APIError contains the structured error payload.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// EvolutionListItem is the lightweight shape returned in GET /api/evolutions.
type EvolutionListItem struct {
	ID        string    `json:"id"`
	Label     *string   `json:"label"`
	CreatedAt string    `json:"created_at"`
	Status    EvoStatus `json:"status"`
}

// EvolutionListResponse is the paginated list returned by GET /api/evolutions.
type EvolutionListResponse struct {
	Items []EvolutionListItem `json:"items"`
	Page  int                 `json:"page"`
	Total int                 `json:"total"`
}

// SubscriptionResponse is returned by GET /api/subscription.
type SubscriptionResponse struct {
	Plan             Plan      `json:"plan"`
	Status           SubStatus `json:"status"`
	CurrentPeriodEnd *string   `json:"current_period_end,omitempty"`
	Quota            Quota     `json:"quota"`
}

// Quota is the usage summary within SubscriptionResponse.
type Quota struct {
	Used  int  `json:"used"`
	Limit *int `json:"limit"` // null for Pro (unlimited)
}

// CheckoutResponse is returned by POST /api/subscription/checkout.
type CheckoutResponse struct {
	CheckoutURL string `json:"checkout_url"`
}
