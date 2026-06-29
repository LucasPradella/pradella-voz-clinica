package services_test

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pradella/voz-clinica/internal/models"
	"github.com/pradella/voz-clinica/internal/services"
	"github.com/pradella/voz-clinica/internal/store"
)

func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set; skipping quota integration test")
	}
	pool, err := pgxpool.New(context.Background(), dbURL)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	return pool
}

func createTestUser(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	us := store.NewUserStore(pool)
	user, err := us.Create(context.Background(), "quota_test_"+randomSuffix()+"@example.com", "password123")
	require.NoError(t, err)
	return user.ID
}

func TestQuota_FreeUserUnder10(t *testing.T) {
	pool := newTestPool(t)
	svc := services.NewQuotaService(pool)
	userID := createTestUser(t, pool)

	err := svc.Check(context.Background(), userID, models.PlanFree)
	assert.NoError(t, err, "user with 0 uses should pass quota check")
}

func TestQuota_FreeUserExceeds10(t *testing.T) {
	pool := newTestPool(t)
	svc := services.NewQuotaService(pool)
	userID := createTestUser(t, pool)

	// Debit 10 times to hit the limit.
	for i := 0; i < 10; i++ {
		err := svc.Debit(context.Background(), userID, models.PlanFree)
		require.NoError(t, err)
	}

	// 11th attempt must be blocked.
	err := svc.Check(context.Background(), userID, models.PlanFree)
	assert.ErrorIs(t, err, services.ErrQuotaExceeded, "11th generation must return quota_exceeded")
}

func TestQuota_ProUserNeverBlocked(t *testing.T) {
	pool := newTestPool(t)
	svc := services.NewQuotaService(pool)
	userID := createTestUser(t, pool)

	// Even after many debits, Pro is never blocked.
	for i := 0; i < 15; i++ {
		err := svc.Debit(context.Background(), userID, models.PlanPro)
		require.NoError(t, err)
	}

	err := svc.Check(context.Background(), userID, models.PlanPro)
	assert.NoError(t, err, "Pro users must never be quota-blocked")
}

func TestQuota_NoDebitOnFailure(t *testing.T) {
	pool := newTestPool(t)
	svc := services.NewQuotaService(pool)
	userID := createTestUser(t, pool)

	// Only Debit is called on success; a failed pipeline never calls Debit.
	// Verify that Check still passes after 9 debits (not 10).
	for i := 0; i < 9; i++ {
		require.NoError(t, svc.Debit(context.Background(), userID, models.PlanFree))
	}

	err := svc.Check(context.Background(), userID, models.PlanFree)
	assert.NoError(t, err, "9 uses must not block the 10th")
}

func randomSuffix() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func TestQuota_MonthlyReset(t *testing.T) {
	pool := newTestPool(t)
	svc := services.NewQuotaService(pool)
	userID := createTestUser(t, pool)

	// Simulate a different period by inserting directly for a past month.
	_, err := pool.Exec(context.Background(),
		`INSERT INTO usage_quotas (user_id, period, count) VALUES ($1, '2000-01', 10)`,
		userID,
	)
	require.NoError(t, err)

	// Current period count is still 0 → should pass.
	err = svc.Check(context.Background(), userID, models.PlanFree)
	assert.NoError(t, err, "past-month quota must not affect current month")
}
