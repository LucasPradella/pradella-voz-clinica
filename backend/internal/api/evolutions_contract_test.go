package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
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

// setupEvoServer creates a full handler with stub SOAP service (no AI calls needed).
func setupEvoServer(t *testing.T) (http.Handler, string) {
	t.Helper()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set; skipping evolution contract test")
	}

	pool, err := pgxpool.New(context.Background(), dbURL)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	cfg := &core.Config{JWTSecret: "test-secret-at-least-32-characters-long"}
	us := store.NewUserStore(pool)
	auditSvc := services.NewAuditService(pool)
	evoStore := store.NewEvolutionStore(pool)
	quotaSvc := services.NewQuotaService(pool)

	email := "evo_" + randomSuffix() + "@example.com"
	user, err := us.Create(context.Background(), email, "password")
	require.NoError(t, err)
	token, err := core.IssueToken(cfg.JWTSecret, user.ID, user.Email, string(user.Plan))
	require.NoError(t, err)

	// BillingSvc is nil in test to avoid Stripe calls.
	deps := &api.Deps{
		Config:    cfg,
		DB:        pool,
		UserStore: us,
		AuditSvc:  auditSvc,
		SOAPSvc:   nil, // No real AI in contract tests; tested via audio validation only
		Guardrail: services.NewGuardrailChecker(),
		EvoStore:  evoStore,
		QuotaSvc:  quotaSvc,
	}
	return api.New(deps), token
}

func multipartAudio(t *testing.T, sizeBytes int) (*bytes.Buffer, string) {
	t.Helper()
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	fw, err := w.CreateFormFile("audio", "recording.webm")
	require.NoError(t, err)
	_, err = io.Copy(fw, io.LimitReader(infiniteZeros{}, int64(sizeBytes)))
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return body, w.FormDataContentType()
}

// infiniteZeros is a reader that always returns zero bytes.
type infiniteZeros struct{}

func (infiniteZeros) Read(p []byte) (n int, err error) {
	for i := range p {
		p[i] = 0
	}
	return len(p), nil
}

func TestEvolution_NoAudioField_Returns422(t *testing.T) {
	handler, token := setupEvoServer(t)

	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	w.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/evolutions", body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
	var errResp models.ErrorResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &errResp))
	assert.Equal(t, "audio_empty", errResp.Error.Code)
}

func TestEvolution_AudioTooShort_Returns422(t *testing.T) {
	handler, token := setupEvoServer(t)

	// 512 bytes < 1024 threshold → too short.
	body, contentType := multipartAudio(t, 512)

	req := httptest.NewRequest(http.MethodPost, "/api/evolutions", body)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
	var errResp models.ErrorResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &errResp))
	assert.Equal(t, "audio_too_short", errResp.Error.Code)
}

func TestEvolution_AudioTooLarge_Returns413(t *testing.T) {
	handler, token := setupEvoServer(t)

	// 26 MB > 25 MB limit.
	body, contentType := multipartAudio(t, 26<<20)

	req := httptest.NewRequest(http.MethodPost, "/api/evolutions", body)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusRequestEntityTooLarge, rr.Code)
	var errResp models.ErrorResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &errResp))
	assert.Equal(t, "audio_too_long", errResp.Error.Code)
}

func TestEvolution_MissingAuth_Returns401(t *testing.T) {
	handler, _ := setupEvoServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/evolutions", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}
