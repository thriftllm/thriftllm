package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/thriftllm/backend/internal/model"
	"github.com/thriftllm/backend/internal/store"
)

type APIKeysHandler struct {
	DB *store.Postgres
}

func (h *APIKeysHandler) List(w http.ResponseWriter, r *http.Request) {
	keys, err := h.DB.ListAPIKeys(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list API keys")
		return
	}
	writeJSON(w, http.StatusOK, keys)
}

func (h *APIKeysHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateAPIKeyRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	if req.RateLimitRPM <= 0 {
		req.RateLimitRPM = 60
	}

	rawKey, apiKey, err := h.DB.CreateAPIKey(r.Context(), req.Name, req.RateLimitRPM)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create API key")
		return
	}

	writeJSON(w, http.StatusCreated, model.CreateAPIKeyResponse{
		Key:    rawKey,
		APIKey: *apiKey,
	})
}

func (h *APIKeysHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid key id")
		return
	}

	if err := h.DB.DeleteAPIKey(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete API key")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "API key deleted"})
}

func (h *APIKeysHandler) Toggle(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid key id")
		return
	}

	var req struct {
		Active bool `json:"is_active"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.DB.ToggleAPIKey(r.Context(), id, req.Active); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to toggle API key")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "API key updated"})
}
