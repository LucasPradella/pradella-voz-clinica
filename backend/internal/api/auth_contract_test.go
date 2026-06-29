package api_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
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

func newTestServer(t *testing.T) http.Handler {
	t.Helper()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set; skipping contract test")
	}

	pool, err := pgxpool.New(context.Background(), dbURL)
	require.NoError(t, err)
	t.Cleanup(func() { pool.Close() })

	cfg := &core.Config{
		JWTSecret: "test-secret-at-least-32-characters-long",
	}
	userStore := store.NewUserStore(pool)
	auditSvc := services.NewAuditService(pool)

	return api.New(&api.Deps{
		Config:    cfg,
		DB:        pool,
		UserStore: userStore,
		AuditSvc:  auditSvc,
	})
}

func postJSON(t *testing.T, handler http.Handler, path string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	b, err := json.Marshal(body)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

func TestRegister_Created(t *testing.T) {
	handler := newTestServer(t)

	email := "test_register_" + randomSuffix() + "@example.com"
	rr := postJSON(t, handler, "/api/auth/register", models.RegisterRequest{
		Email:    email,
		Password: "secure-password-123",
	})

	assert.Equal(t, http.StatusCreated, rr.Code)

	var resp models.AuthResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.Token)
	assert.Equal(t, email, resp.User.Email)
	assert.Equal(t, models.PlanFree, resp.User.Plan)
}

func TestRegister_EmailTaken(t *testing.T) {
	handler := newTestServer(t)

	email := "test_taken_" + randomSuffix() + "@example.com"
	payload := models.RegisterRequest{Email: email, Password: "password"}

	rr := postJSON(t, handler, "/api/auth/register", payload)
	require.Equal(t, http.StatusCreated, rr.Code)

	rr2 := postJSON(t, handler, "/api/auth/register", payload)
	assert.Equal(t, http.StatusConflict, rr2.Code)

	var errResp models.ErrorResponse
	require.NoError(t, json.Unmarshal(rr2.Body.Bytes(), &errResp))
	assert.Equal(t, "email_taken", errResp.Error.Code)
}

func TestLogin_Success(t *testing.T) {
	handler := newTestServer(t)

	email := "test_login_" + randomSuffix() + "@example.com"
	password := "my-secure-password"

	// Register first
	rr := postJSON(t, handler, "/api/auth/register", models.RegisterRequest{
		Email:    email,
		Password: password,
	})
	require.Equal(t, http.StatusCreated, rr.Code)

	// Login
	rr2 := postJSON(t, handler, "/api/auth/login", models.LoginRequest{
		Email:    email,
		Password: password,
	})
	assert.Equal(t, http.StatusOK, rr2.Code)

	var resp models.AuthResponse
	require.NoError(t, json.Unmarshal(rr2.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.Token)
	assert.Equal(t, email, resp.User.Email)
}

func TestLogin_InvalidCredentials(t *testing.T) {
	handler := newTestServer(t)

	rr := postJSON(t, handler, "/api/auth/login", models.LoginRequest{
		Email:    "nonexistent@example.com",
		Password: "wrongpassword",
	})
	assert.Equal(t, http.StatusUnauthorized, rr.Code)

	var errResp models.ErrorResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &errResp))
	assert.Equal(t, "invalid_credentials", errResp.Error.Code)
}

func TestLogin_WrongPassword(t *testing.T) {
	handler := newTestServer(t)

	email := "test_wrongpw_" + randomSuffix() + "@example.com"

	rr := postJSON(t, handler, "/api/auth/register", models.RegisterRequest{
		Email:    email,
		Password: "correct-password",
	})
	require.Equal(t, http.StatusCreated, rr.Code)

	rr2 := postJSON(t, handler, "/api/auth/login", models.LoginRequest{
		Email:    email,
		Password: "wrong-password",
	})
	assert.Equal(t, http.StatusUnauthorized, rr2.Code)
}

func TestHealthz(t *testing.T) {
	handler := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/healthz", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

// randomSuffix returns a short random string to avoid test email collisions.
func randomSuffix() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}
