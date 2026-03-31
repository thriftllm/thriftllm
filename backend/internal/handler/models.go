package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/thriftllm/backend/internal/model"
	"github.com/thriftllm/backend/internal/store"
)

type ModelsHandler struct {
	DB *store.Postgres
}

func (h *ModelsHandler) List(w http.ResponseWriter, r *http.Request) {
	models, err := h.DB.ListModelConfigs(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list models")
		return
	}
	writeJSON(w, http.StatusOK, models)
}

func (h *ModelsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateModelRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ProviderModel == "" || req.DisplayName == "" || req.APIKeyEnvName == "" {
		writeError(w, http.StatusBadRequest, "provider_model, display_name, and api_key_env_name are required")
		return
	}

	if req.Tags == nil {
		req.Tags = []string{}
	}

	mc, err := h.DB.CreateModelConfig(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create model: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, mc)
}

func (h *ModelsHandler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid model id")
		return
	}

	var req model.CreateModelRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Tags == nil {
		req.Tags = []string{}
	}

	mc, err := h.DB.UpdateModelConfig(r.Context(), id, req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update model")
		return
	}

	writeJSON(w, http.StatusOK, mc)
}

func (h *ModelsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid model id")
		return
	}

	if err := h.DB.DeleteModelConfig(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete model")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "model deleted"})
}

func (h *ModelsHandler) Toggle(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid model id")
		return
	}

	var req struct {
		Active bool `json:"is_active"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.DB.ToggleModelActive(r.Context(), id, req.Active); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to toggle model")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "model updated"})
}

// OpenAI-compatible /v1/models endpoint
func (h *ModelsHandler) ListOpenAI(w http.ResponseWriter, r *http.Request) {
	models, err := h.DB.ListModelConfigs(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list models")
		return
	}

	type openAIModel struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		OwnedBy string `json:"owned_by"`
	}

	var data []openAIModel
	for _, m := range models {
		if m.IsActive {
			data = append(data, openAIModel{
				ID:      m.ProviderModel,
				Object:  "model",
				Created: m.CreatedAt.Unix(),
				OwnedBy: string(m.ProviderType),
			})
		}
	}

	if data == nil {
		data = []openAIModel{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"object": "list",
		"data":   data,
	})
}
