package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/thriftllm/backend/internal/model"
	"github.com/thriftllm/backend/internal/store"
)

type FallbackChainsHandler struct {
	DB *store.Postgres
}

func (h *FallbackChainsHandler) List(w http.ResponseWriter, r *http.Request) {
	chains, err := h.DB.ListFallbackChains(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list fallback chains")
		return
	}
	writeJSON(w, http.StatusOK, chains)
}

func (h *FallbackChainsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateFallbackChainRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "Name is required")
		return
	}
	if len(req.ModelConfigIDs) == 0 {
		writeError(w, http.StatusBadRequest, "At least one model is required")
		return
	}

	chain, err := h.DB.CreateFallbackChain(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create fallback chain: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, chain)
}

func (h *FallbackChainsHandler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid chain ID")
		return
	}

	var req model.CreateFallbackChainRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "Name is required")
		return
	}
	if len(req.ModelConfigIDs) == 0 {
		writeError(w, http.StatusBadRequest, "At least one model is required")
		return
	}

	chain, err := h.DB.UpdateFallbackChain(r.Context(), id, req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update fallback chain: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, chain)
}

func (h *FallbackChainsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid chain ID")
		return
	}

	if err := h.DB.DeleteFallbackChain(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete fallback chain")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
