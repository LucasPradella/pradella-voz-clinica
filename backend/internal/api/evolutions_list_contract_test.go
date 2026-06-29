package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pradella/voz-clinica/internal/api"
	"github.com/pradella/voz-clinica/internal/core"
	"github.com/pradella/voz-clinica/internal/models"
	"github.com/pradella/voz-clinica/internal/services"
	"github.com/pradella/voz-clinica/internal/store"
)

func setupListServer(t *testing.T) (proHandler http.Handler, proToken string, freeToken string, evoID string) {
	t.Helper()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set; skipping list contract test")
	}

	pool, err := pgxpool.New(context.Background(), dbURL)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	cfg := &core.Config{JWTSecret: "test-secret-at-least-32-characters-long"}
	us := store.NewUserStore(pool)
	auditSvc := services.NewAuditService(pool)
	evoStore := store.NewEvolutionStore(pool)

	// Pro user
	proEmail := "list_pro_" + randomSuffix() + "@example.com"
	proUser, err := us.Create(context.Background(), proEmail, "password")
	require.NoError(t, err)
	_, err = pool.Exec(context.Background(), `UPDATE users SET plan='pro' WHERE id=$1`, proUser.ID)
	require.NoError(t, err)
	proTok, err := core.IssueToken(cfg.JWTSecret, proUser.ID, proUser.Email, "pro")
	require.NoError(t, err)

	// Free user
	freeEmail := "list_free_" + randomSuffix() + "@example.com"
	freeUser, err := us.Create(context.Background(), freeEmail, "password")
	require.NoError(t, err)
	freeTok, err := core.IssueToken(cfg.JWTSecret, freeUser.ID, freeUser.Email, "free")
	require.NoError(t, err)

	// Seed one evolution for Pro user.
	evo, err := evoStore.Create(context.Background(), models.Evolution{
		UserID: proUser.ID,
		SOAP:   models.SOAP{S: "S", O: "O", A: "A", P: "P"},
		Status: models.EvoStatusDraft,
	})
	require.NoError(t, err)

	deps := &api.Deps{
		Config:    cfg,
		DB:        pool,
		UserStore: us,
		AuditSvc:  auditSvc,
		Guardrail: services.NewGuardrailChecker(),
		EvoStore:  evoStore,
		QuotaSvc:  services.NewQuotaService(pool),
	}
	return api.New(deps), proTok, freeTok, evo.ID
}

func TestListEvolutions_ProOK(t *testing.T) {
	handler, proToken, _, _ := setupListServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/evolutions?page=1&limit=10", nil)
	req.Header.Set("Authorization", "Bearer "+proToken)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp models.EvolutionListResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.GreaterOrEqual(t, resp.Total, 1)
	assert.NotEmpty(t, resp.Items)
}

func TestListEvolutions_FreeForbidden(t *testing.T) {
	handler, _, freeToken, _ := setupListServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/evolutions", nil)
	req.Header.Set("Authorization", "Bearer "+freeToken)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
	var errResp models.ErrorResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &errResp))
	assert.Equal(t, "pro_required", errResp.Error.Code)
}

func TestGetEvolution_ProOK(t *testing.T) {
	handler, proToken, _, evoID := setupListServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/evolutions/"+evoID, nil)
	req.Header.Set("Authorization", "Bearer "+proToken)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp models.EvolutionResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.NotNil(t, resp.ID)
	assert.Equal(t, evoID, *resp.ID)
}

func TestGetEvolution_FreeForbidden(t *testing.T) {
	handler, _, freeToken, evoID := setupListServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/evolutions/"+evoID, nil)
	req.Header.Set("Authorization", "Bearer "+freeToken)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestListEvolutions_AuditLog(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set")
	}

	pool, err := pgxpool.New(context.Background(), dbURL)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	cfg := &core.Config{JWTSecret: "test-secret-at-least-32-characters-long"}
	us := store.NewUserStore(pool)

	email := "audit_list_" + randomSuffix() + "@example.com"
	user, err := us.Create(context.Background(), email, "password")
	require.NoError(t, err)
	_, err = pool.Exec(context.Background(), `UPDATE users SET plan='pro' WHERE id=$1`, user.ID)
	require.NoError(t, err)
	token, err := core.IssueToken(cfg.JWTSecret, user.ID, user.Email, "pro")
	require.NoError(t, err)

	auditSvc := services.NewAuditService(pool)
	evoStore := store.NewEvolutionStore(pool)

	deps := &api.Deps{
		Config:    cfg,
		DB:        pool,
		UserStore: us,
		AuditSvc:  auditSvc,
		Guardrail: services.NewGuardrailChecker(),
		EvoStore:  evoStore,
		QuotaSvc:  services.NewQuotaService(pool),
	}
	handler := api.New(deps)

	req := httptest.NewRequest(http.MethodGet, "/api/evolutions", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Verify that an audit log entry was created.
	var count int
	err = pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM audit_logs WHERE user_id=$1 AND action='evolution.list'`,
		user.ID,
	).Scan(&count)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, 1, "evolution.list action must be audited")

	_ = bytes.NewBuffer(nil) // avoid unused import
}
