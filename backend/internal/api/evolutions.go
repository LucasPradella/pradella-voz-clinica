package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/pradella/voz-clinica/internal/core"
	"github.com/pradella/voz-clinica/internal/models"
	"github.com/pradella/voz-clinica/internal/services"
	"github.com/pradella/voz-clinica/internal/store"
)

const maxUploadBytes = 25 << 20 // 25 MB — generous ceiling for 120s audio

// EvolutionHandler serves all /api/evolutions routes.
type EvolutionHandler struct {
	soapSvc   *services.SOAPService
	guardrail *services.GuardrailChecker
	evoStore  *store.EvolutionStore
	auditSvc  *services.AuditService
	quotaSvc  *services.QuotaService
}

// Create handles POST /api/evolutions — full audio→SOAP pipeline.
func (h *EvolutionHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims, ok := core.ClaimsFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "unauthorized", "missing auth")
		return
	}

	plan := models.Plan(claims.Plan)

	// Check quota before processing (no debit yet).
	if err := h.quotaSvc.Check(r.Context(), claims.UserID, plan); err != nil {
		if errors.Is(err, services.ErrQuotaExceeded) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusPaymentRequired)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error":   map[string]string{"code": "quota_exceeded", "message": "monthly quota exceeded"},
				"upgrade": true,
			})
			return
		}
		WriteError(w, http.StatusInternalServerError, "internal_error", "quota check failed")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		WriteError(w, http.StatusRequestEntityTooLarge, "audio_too_long", "request body too large (max 120s audio)")
		return
	}

	audioFile, header, err := r.FormFile("audio")
	if err != nil {
		WriteError(w, http.StatusUnprocessableEntity, "audio_empty", "audio field is required")
		return
	}
	defer audioFile.Close()

	// Reject clearly empty files (< 1 KB → no real audio content).
	if header.Size < 1024 {
		WriteError(w, http.StatusUnprocessableEntity, "audio_too_short", "audio is too short or empty")
		return
	}

	filename := header.Filename
	if filename == "" {
		filename = fmt.Sprintf("recording.webm")
	}

	result, err := h.soapSvc.Process(r.Context(), audioFile, filename)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "processing_error", "could not process audio")
		return
	}

	// Post-generation guardrail check: merge additional flags from checker.
	extraFlags := h.guardrail.Check(result.Transcription, result.SOAP)
	result.SOAP.ConfidenceFlags = append(result.SOAP.ConfidenceFlags, extraFlags...)

	soap := models.SOAP{
		S: result.SOAP.S,
		O: result.SOAP.O,
		A: result.SOAP.A,
		P: result.SOAP.P,
	}

	label := r.FormValue("label")
	var labelPtr *string
	if label != "" {
		labelPtr = &label
	}

	status := models.EvoStatusDraft
	resp := models.EvolutionResponse{
		SOAP:            soap,
		CIDSuggestions:  result.SOAP.CIDSuggestions,
		ConfidenceFlags: result.SOAP.ConfidenceFlags,
		SourceRefs:      result.SourceRefs,
		Status:          status,
	}

	// Persist for Pro users; Free evolutions are ephemeral (no ID).
	if plan == models.PlanPro {
		evo := models.Evolution{
			UserID:          claims.UserID,
			Label:           labelPtr,
			SOAP:            soap,
			CIDSuggestions:  result.SOAP.CIDSuggestions,
			ConfidenceFlags: result.SOAP.ConfidenceFlags,
			SourceRefs:      result.SourceRefs,
			Status:          status,
		}
		saved, err := h.evoStore.Create(r.Context(), evo)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "internal_error", "could not save evolution")
			return
		}
		resp.ID = &saved.ID
	}

	// Debit quota only after successful generation (FR-018).
	if err := h.quotaSvc.Debit(r.Context(), claims.UserID, plan); err != nil {
		// Non-fatal: log but do not block the response.
		_ = err
	}

	h.auditSvc.Log(r.Context(), claims.UserID, "evolution.create", ptrStr(resp.ID), nil)

	WriteJSON(w, http.StatusOK, resp)
}

// Patch handles PATCH /api/evolutions/{id} — edit before finalizing (Pro only).
func (h *EvolutionHandler) Patch(w http.ResponseWriter, r *http.Request) {
	claims, ok := core.ClaimsFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "unauthorized", "missing auth")
		return
	}

	if models.Plan(claims.Plan) != models.PlanPro {
		WriteError(w, http.StatusForbidden, "pro_required", "editing persisted evolutions requires Pro")
		return
	}

	id := chi.URLParam(r, "id")

	var req models.PatchEvolutionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}

	updated, err := h.evoStore.Update(r.Context(), id, claims.UserID, req)
	if errors.Is(err, store.ErrNotFound) {
		WriteError(w, http.StatusNotFound, "not_found", "evolution not found or access denied")
		return
	}
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", "could not update evolution")
		return
	}

	h.auditSvc.Log(r.Context(), claims.UserID, "evolution.update", updated.ID, nil)

	WriteJSON(w, http.StatusOK, evolutionToResponse(updated))
}

// List handles GET /api/evolutions — paginated history (Pro only).
func (h *EvolutionHandler) List(w http.ResponseWriter, r *http.Request) {
	claims, ok := core.ClaimsFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "unauthorized", "missing auth")
		return
	}

	if models.Plan(claims.Plan) != models.PlanPro {
		WriteError(w, http.StatusForbidden, "pro_required", "history requires Pro")
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	items, total, err := h.evoStore.List(r.Context(), claims.UserID, page, limit)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", "could not list evolutions")
		return
	}

	h.auditSvc.Log(r.Context(), claims.UserID, "evolution.list", "", nil)

	if items == nil {
		items = []models.EvolutionListItem{}
	}

	WriteJSON(w, http.StatusOK, models.EvolutionListResponse{
		Items: items,
		Page:  max(page, 1),
		Total: total,
	})
}

// Get handles GET /api/evolutions/{id} (Pro only).
func (h *EvolutionHandler) Get(w http.ResponseWriter, r *http.Request) {
	claims, ok := core.ClaimsFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "unauthorized", "missing auth")
		return
	}

	if models.Plan(claims.Plan) != models.PlanPro {
		WriteError(w, http.StatusForbidden, "pro_required", "history requires Pro")
		return
	}

	id := chi.URLParam(r, "id")

	evo, err := h.evoStore.GetByID(r.Context(), id, claims.UserID)
	if errors.Is(err, store.ErrNotFound) {
		WriteError(w, http.StatusNotFound, "not_found", "evolution not found or access denied")
		return
	}
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", "could not fetch evolution")
		return
	}

	h.auditSvc.Log(r.Context(), claims.UserID, "evolution.view", evo.ID, nil)

	WriteJSON(w, http.StatusOK, evolutionToResponse(evo))
}

func evolutionToResponse(evo *models.Evolution) models.EvolutionResponse {
	return models.EvolutionResponse{
		ID:              &evo.ID,
		SOAP:            evo.SOAP,
		CIDSuggestions:  evo.CIDSuggestions,
		ConfidenceFlags: evo.ConfidenceFlags,
		SourceRefs:      evo.SourceRefs,
		Status:          evo.Status,
	}
}

func ptrStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
