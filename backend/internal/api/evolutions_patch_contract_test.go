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

// setupProEvoServer creates a handler with a Pro user and a seeded evolution.
func setupProEvoServer(t *testing.T) (http.Handler, string, string) {
	t.Helper()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set; skipping PATCH contract test")
	}

	pool, err := pgxpool.New(context.Background(), dbURL)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	cfg := &core.Config{JWTSecret: "test-secret-at-least-32-characters-long"}
	us := store.NewUserStore(pool)
	auditSvc := services.NewAuditService(pool)
	evoStore := store.NewEvolutionStore(pool)
	quotaSvc := services.NewQuotaService(pool)

	// Create a Pro user by inserting directly into DB.
	email := "pro_" + randomSuffix() + "@example.com"
	user, err := us.Create(context.Background(), email, "password")
	require.NoError(t, err)

	_, err = pool.Exec(context.Background(),
		`UPDATE users SET plan = 'pro' WHERE id = $1`,
		user.ID,
	)
	require.NoError(t, err)
	_, err = pool.Exec(context.Background(),
		`UPDATE subscriptions SET type = 'pro' WHERE user_id = $1`,
		user.ID,
	)
	require.NoError(t, err)

	token, err := core.IssueToken(cfg.JWTSecret, user.ID, user.Email, "pro")
	require.NoError(t, err)

	// Seed an evolution for this Pro user.
	s := "Paciente relata dor."
	o := "Avaliação normal."
	a := "Lombalgia."
	p := "Exercícios."
	evo, err := evoStore.Create(context.Background(), models.Evolution{
		UserID: user.ID,
		SOAP:   models.SOAP{S: s, O: o, A: a, P: p},
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
		QuotaSvc:  quotaSvc,
	}
	return api.New(deps), token, evo.ID
}

func TestPatchEvolution_UpdatesFields(t *testing.T) {
	handler, token, evoID := setupProEvoServer(t)

	newStatus := models.EvoStatusFinalized
	body, err := json.Marshal(models.PatchEvolutionRequest{
		Status: &newStatus,
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPatch, "/api/evolutions/"+evoID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp models.EvolutionResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, models.EvoStatusFinalized, resp.Status)
	assert.NotNil(t, resp.ID)
	assert.Equal(t, evoID, *resp.ID)
}

func TestPatchEvolution_UpdateSOAPFields(t *testing.T) {
	handler, token, evoID := setupProEvoServer(t)

	updatedSOAP := models.SOAP{
		S: "Atualizado S",
		O: "Atualizado O",
		A: "Atualizado A",
		P: "Atualizado P",
	}
	body, err := json.Marshal(models.PatchEvolutionRequest{SOAP: &updatedSOAP})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPatch, "/api/evolutions/"+evoID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp models.EvolutionResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "Atualizado A", resp.SOAP.A)
}

func TestPatchEvolution_NotFound(t *testing.T) {
	handler, token, _ := setupProEvoServer(t)

	body, _ := json.Marshal(models.PatchEvolutionRequest{})

	req := httptest.NewRequest(http.MethodPatch, "/api/evolutions/00000000-0000-0000-0000-000000000000", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	var errResp models.ErrorResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &errResp))
	assert.Equal(t, "not_found", errResp.Error.Code)
}

func TestPatchEvolution_FreeUserForbidden(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set")
	}

	pool, err := pgxpool.New(context.Background(), dbURL)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	cfg := &core.Config{JWTSecret: "test-secret-at-least-32-characters-long"}
	us := store.NewUserStore(pool)

	email := "free_" + randomSuffix() + "@example.com"
	user, err := us.Create(context.Background(), email, "password")
	require.NoError(t, err)
	token, err := core.IssueToken(cfg.JWTSecret, user.ID, user.Email, "free")
	require.NoError(t, err)

	deps := &api.Deps{
		Config:    cfg,
		DB:        pool,
		UserStore: us,
		AuditSvc:  services.NewAuditService(pool),
		Guardrail: services.NewGuardrailChecker(),
		EvoStore:  store.NewEvolutionStore(pool),
		QuotaSvc:  services.NewQuotaService(pool),
	}
	handler := api.New(deps)

	body, _ := json.Marshal(models.PatchEvolutionRequest{})
	req := httptest.NewRequest(http.MethodPatch, "/api/evolutions/any-id", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Free users can't patch (evolutions are ephemeral).
	assert.Equal(t, http.StatusForbidden, rr.Code)
}
