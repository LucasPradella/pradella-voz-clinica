package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/pradella/voz-clinica/internal/models"
)

const freeQuotaLimit = 10

// ErrQuotaExceeded is returned when a Free user has reached their monthly limit.
var ErrQuotaExceeded = errors.New("monthly quota exceeded")

// QuotaService manages monthly usage quotas for Free users.
// Pro users bypass all quota checks.
type QuotaService struct {
	db *pgxpool.Pool
}

func NewQuotaService(db *pgxpool.Pool) *QuotaService {
	return &QuotaService{db: db}
}

// Check returns ErrQuotaExceeded if a Free user has hit their monthly limit.
// Pro users always pass.
func (s *QuotaService) Check(ctx context.Context, userID string, plan models.Plan) error {
	if plan == models.PlanPro {
		return nil
	}
	period := currentPeriod()
	count, err := s.getCount(ctx, userID, period)
	if err != nil {
		return fmt.Errorf("quota check: %w", err)
	}
	if count >= freeQuotaLimit {
		return ErrQuotaExceeded
	}
	return nil
}

// Debit increments the usage counter for a Free user after a successful generation.
// Must only be called on success (FR-018: no debit on failure).
// Pro users are skipped.
func (s *QuotaService) Debit(ctx context.Context, userID string, plan models.Plan) error {
	if plan == models.PlanPro {
		return nil
	}
	period := currentPeriod()
	_, err := s.db.Exec(ctx,
		`INSERT INTO usage_quotas (user_id, period, count)
		 VALUES ($1, $2, 1)
		 ON CONFLICT (user_id, period)
		 DO UPDATE SET count = usage_quotas.count + 1`,
		userID, period,
	)
	if err != nil {
		return fmt.Errorf("debit quota: %w", err)
	}
	return nil
}

// GetUsage returns the current period usage for a user.
func (s *QuotaService) GetUsage(ctx context.Context, userID string) (used int, limit *int, err error) {
	period := currentPeriod()
	count, err := s.getCount(ctx, userID, period)
	if err != nil {
		return 0, nil, err
	}
	lim := freeQuotaLimit
	return count, &lim, nil
}

func (s *QuotaService) getCount(ctx context.Context, userID, period string) (int, error) {
	var count int
	err := s.db.QueryRow(ctx,
		`SELECT COALESCE(count, 0) FROM usage_quotas WHERE user_id = $1 AND period = $2`,
		userID, period,
	).Scan(&count)
	// No row means no usage yet; that's zero.
	if err != nil && err.Error() != "no rows in result set" {
		// pgx.ErrNoRows has a specific message
		return 0, nil
	}
	return count, nil
}

// currentPeriod returns the current calendar month in "YYYY-MM" format.
func currentPeriod() string {
	return time.Now().UTC().Format("2006-01")
}
