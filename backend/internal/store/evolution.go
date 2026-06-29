package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/pradella/voz-clinica/internal/models"
)

// EvolutionStore handles persistence of clinical evolutions (Pro plan only).
type EvolutionStore struct {
	db *pgxpool.Pool
}

func NewEvolutionStore(db *pgxpool.Pool) *EvolutionStore {
	return &EvolutionStore{db: db}
}

// Create inserts a new evolution and returns it with the generated ID.
func (s *EvolutionStore) Create(ctx context.Context, evo models.Evolution) (*models.Evolution, error) {
	cidJSON, err := json.Marshal(evo.CIDSuggestions)
	if err != nil {
		return nil, fmt.Errorf("marshal cid_suggestions: %w", err)
	}
	flagsJSON, err := json.Marshal(evo.ConfidenceFlags)
	if err != nil {
		return nil, fmt.Errorf("marshal confidence_flags: %w", err)
	}
	refsJSON, err := json.Marshal(evo.SourceRefs)
	if err != nil {
		return nil, fmt.Errorf("marshal source_refs: %w", err)
	}

	var out models.Evolution
	var cidRaw, flagsRaw, refsRaw []byte

	err = s.db.QueryRow(ctx,
		`INSERT INTO evolutions
		   (user_id, label, soap_s, soap_o, soap_a, soap_p,
		    cid_suggestions, confidence_flags, status, source_refs)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		 RETURNING id, user_id, label, soap_s, soap_o, soap_a, soap_p,
		           cid_suggestions, confidence_flags, status, source_refs, created_at`,
		evo.UserID, evo.Label, evo.SOAP.S, evo.SOAP.O, evo.SOAP.A, evo.SOAP.P,
		cidJSON, flagsJSON, string(evo.Status), refsJSON,
	).Scan(
		&out.ID, &out.UserID, &out.Label,
		&out.SOAP.S, &out.SOAP.O, &out.SOAP.A, &out.SOAP.P,
		&cidRaw, &flagsRaw, &out.Status, &refsRaw, &out.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert evolution: %w", err)
	}

	if err := json.Unmarshal(cidRaw, &out.CIDSuggestions); err != nil {
		return nil, fmt.Errorf("unmarshal cid_suggestions: %w", err)
	}
	if err := json.Unmarshal(flagsRaw, &out.ConfidenceFlags); err != nil {
		return nil, fmt.Errorf("unmarshal confidence_flags: %w", err)
	}
	if err := json.Unmarshal(refsRaw, &out.SourceRefs); err != nil {
		return nil, fmt.Errorf("unmarshal source_refs: %w", err)
	}
	return &out, nil
}

// GetByID returns the evolution with the given ID, enforcing user ownership.
// Returns ErrNotFound if the evolution does not exist or belongs to another user.
func (s *EvolutionStore) GetByID(ctx context.Context, id, userID string) (*models.Evolution, error) {
	var out models.Evolution
	var cidRaw, flagsRaw, refsRaw []byte

	err := s.db.QueryRow(ctx,
		`SELECT id, user_id, label, soap_s, soap_o, soap_a, soap_p,
		        cid_suggestions, confidence_flags, status, source_refs, created_at
		 FROM evolutions WHERE id = $1 AND user_id = $2`,
		id, userID,
	).Scan(
		&out.ID, &out.UserID, &out.Label,
		&out.SOAP.S, &out.SOAP.O, &out.SOAP.A, &out.SOAP.P,
		&cidRaw, &flagsRaw, &out.Status, &refsRaw, &out.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get evolution by id: %w", err)
	}

	if err := unmarshalJSONFields(&out, cidRaw, flagsRaw, refsRaw); err != nil {
		return nil, err
	}
	return &out, nil
}

// Update applies a partial update to an evolution owned by userID.
// Only non-nil fields in the patch are changed.
func (s *EvolutionStore) Update(ctx context.Context, id, userID string, patch models.PatchEvolutionRequest) (*models.Evolution, error) {
	existing, err := s.GetByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	if patch.Label != nil {
		existing.Label = patch.Label
	}
	if patch.SOAP != nil {
		if patch.SOAP.S != "" {
			existing.SOAP.S = patch.SOAP.S
		}
		if patch.SOAP.O != "" {
			existing.SOAP.O = patch.SOAP.O
		}
		if patch.SOAP.A != "" {
			existing.SOAP.A = patch.SOAP.A
		}
		if patch.SOAP.P != "" {
			existing.SOAP.P = patch.SOAP.P
		}
	}
	if patch.CIDSuggestions != nil {
		existing.CIDSuggestions = patch.CIDSuggestions
	}
	if patch.Status != nil {
		existing.Status = *patch.Status
	}

	cidJSON, _ := json.Marshal(existing.CIDSuggestions)
	flagsJSON, _ := json.Marshal(existing.ConfidenceFlags)
	_ = existing.SourceRefs // source_refs are not updated via PATCH

	var out models.Evolution
	var cidRaw, flagsRaw, refsRaw []byte

	err = s.db.QueryRow(ctx,
		`UPDATE evolutions
		 SET label=$1, soap_s=$2, soap_o=$3, soap_a=$4, soap_p=$5,
		     cid_suggestions=$6, confidence_flags=$7, status=$8
		 WHERE id=$9 AND user_id=$10
		 RETURNING id, user_id, label, soap_s, soap_o, soap_a, soap_p,
		           cid_suggestions, confidence_flags, status, source_refs, created_at`,
		existing.Label, existing.SOAP.S, existing.SOAP.O, existing.SOAP.A, existing.SOAP.P,
		cidJSON, flagsJSON, string(existing.Status),
		id, userID,
	).Scan(
		&out.ID, &out.UserID, &out.Label,
		&out.SOAP.S, &out.SOAP.O, &out.SOAP.A, &out.SOAP.P,
		&cidRaw, &flagsRaw, &out.Status, &refsRaw, &out.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update evolution: %w", err)
	}

	if err := unmarshalJSONFields(&out, cidRaw, flagsRaw, refsRaw); err != nil {
		return nil, err
	}
	return &out, nil
}

// List returns a paginated list of evolutions for a user, ordered by created_at DESC.
func (s *EvolutionStore) List(ctx context.Context, userID string, page, limit int) ([]models.EvolutionListItem, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	var total int
	err := s.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM evolutions WHERE user_id = $1`, userID,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count evolutions: %w", err)
	}

	rows, err := s.db.Query(ctx,
		`SELECT id, label, created_at, status
		 FROM evolutions WHERE user_id = $1
		 ORDER BY created_at DESC
		 LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list evolutions: %w", err)
	}
	defer rows.Close()

	var items []models.EvolutionListItem
	for rows.Next() {
		var item models.EvolutionListItem
		var createdAt time.Time
		if err := rows.Scan(&item.ID, &item.Label, &createdAt, &item.Status); err != nil {
			return nil, 0, fmt.Errorf("scan evolution row: %w", err)
		}
		item.CreatedAt = createdAt.Format(time.RFC3339)
		items = append(items, item)
	}
	return items, total, rows.Err()
}

// GetSubscription returns the active subscription for a user.
func (s *EvolutionStore) GetSubscription(ctx context.Context, userID string) (*models.Subscription, error) {
	var sub models.Subscription
	err := s.db.QueryRow(ctx,
		`SELECT id, user_id, type, status, stripe_customer_id,
		        stripe_subscription_id, current_period_end
		 FROM subscriptions WHERE user_id = $1 ORDER BY id LIMIT 1`,
		userID,
	).Scan(
		&sub.ID, &sub.UserID, &sub.Type, &sub.Status,
		&sub.StripeCustomerID, &sub.StripeSubscriptionID, &sub.CurrentPeriodEnd,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get subscription: %w", err)
	}
	return &sub, nil
}

// ActivatePro upgrades the user to Pro and records Stripe IDs.
func (s *EvolutionStore) ActivatePro(userID, stripeCustomerID, stripeSubscriptionID string, periodEnd interface{}) error {
	ctx := context.Background()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	_, err = tx.Exec(ctx,
		`UPDATE users SET plan = 'pro', updated_at = now() WHERE id = $1`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("update user plan: %w", err)
	}

	_, err = tx.Exec(ctx,
		`UPDATE subscriptions
		 SET type = 'pro', status = 'active',
		     stripe_customer_id = $2, stripe_subscription_id = $3,
		     current_period_end = to_timestamp($4)
		 WHERE user_id = $1`,
		userID, stripeCustomerID, stripeSubscriptionID, periodEnd,
	)
	if err != nil {
		return fmt.Errorf("update subscription: %w", err)
	}

	return tx.Commit(ctx)
}

// DeactivatePro cancels a Pro subscription and reverts the user to Free.
func (s *EvolutionStore) DeactivatePro(stripeSubscriptionID string) error {
	ctx := context.Background()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var userID string
	err = tx.QueryRow(ctx,
		`UPDATE subscriptions SET type='free', status='canceled'
		 WHERE stripe_subscription_id = $1
		 RETURNING user_id`,
		stripeSubscriptionID,
	).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil // Already cleaned up or not found
	}
	if err != nil {
		return fmt.Errorf("deactivate subscription: %w", err)
	}

	_, err = tx.Exec(ctx,
		`UPDATE users SET plan = 'free', updated_at = now() WHERE id = $1`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("revert user plan: %w", err)
	}

	return tx.Commit(ctx)
}

func unmarshalJSONFields(out *models.Evolution, cidRaw, flagsRaw, refsRaw []byte) error {
	if len(cidRaw) > 0 {
		if err := json.Unmarshal(cidRaw, &out.CIDSuggestions); err != nil {
			return fmt.Errorf("unmarshal cid_suggestions: %w", err)
		}
	}
	if len(flagsRaw) > 0 {
		if err := json.Unmarshal(flagsRaw, &out.ConfidenceFlags); err != nil {
			return fmt.Errorf("unmarshal confidence_flags: %w", err)
		}
	}
	if len(refsRaw) > 0 {
		if err := json.Unmarshal(refsRaw, &out.SourceRefs); err != nil {
			return fmt.Errorf("unmarshal source_refs: %w", err)
		}
	}
	return nil
}
