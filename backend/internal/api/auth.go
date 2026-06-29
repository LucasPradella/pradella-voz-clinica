package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/pradella/voz-clinica/internal/core"
	"github.com/pradella/voz-clinica/internal/models"
	"github.com/pradella/voz-clinica/internal/services"
	"github.com/pradella/voz-clinica/internal/store"
)

// AuthHandler handles register and login endpoints.
type AuthHandler struct {
	userStore *store.UserStore
	auditSvc  *services.AuditService
	jwtSecret string
}

// Register handles POST /api/auth/register.
// Creates a Free account and returns a JWT.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}

	if req.Email == "" || req.Password == "" {
		WriteError(w, http.StatusBadRequest, "bad_request", "email and password are required")
		return
	}

	user, err := h.userStore.Create(r.Context(), req.Email, req.Password)
	if errors.Is(err, store.ErrEmailTaken) {
		WriteError(w, http.StatusConflict, "email_taken", "this email is already registered")
		return
	}
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", "could not create account")
		return
	}

	token, err := core.IssueToken(h.jwtSecret, user.ID, user.Email, string(user.Plan))
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", "could not issue token")
		return
	}

	h.auditSvc.Log(r.Context(), user.ID, "auth.register", "", nil)

	WriteJSON(w, http.StatusCreated, models.AuthResponse{
		Token: token,
		User: models.UserInfo{
			ID:    user.ID,
			Email: user.Email,
			Plan:  user.Plan,
		},
	})
}

// Login handles POST /api/auth/login.
// Returns a JWT on valid credentials.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}

	user, err := h.userStore.GetByEmail(r.Context(), req.Email)
	if errors.Is(err, store.ErrNotFound) || (err == nil && !store.CheckPassword(user.PasswordHash, req.Password)) {
		WriteError(w, http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
		return
	}
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", "login failed")
		return
	}

	token, err := core.IssueToken(h.jwtSecret, user.ID, user.Email, string(user.Plan))
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", "could not issue token")
		return
	}

	h.auditSvc.Log(r.Context(), user.ID, "auth.login", "", nil)

	WriteJSON(w, http.StatusOK, models.AuthResponse{
		Token: token,
		User: models.UserInfo{
			ID:    user.ID,
			Email: user.Email,
			Plan:  user.Plan,
		},
	})
}
