package handler

import (
	"context"
	"net/http"

	"github.com/thriftllm/backend/internal/cache"
	"github.com/thriftllm/backend/internal/store"
)

type CacheHandler struct {
	DB    *store.Postgres
	Redis *store.Redis
	Cache *cache.SemanticCache
}

func (h *CacheHandler) Stats(w http.ResponseWriter, r *http.Request) {
	overview, err := h.DB.GetCacheOverview(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get cache stats")
		return
	}

	// Get Redis cache entry count
	var entryCount int64
	if h.Cache != nil {
		entryCount, _ = h.Cache.EntryCount(r.Context())
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"overview":    overview,
		"entry_count": entryCount,
	})
}

func (h *CacheHandler) Flush(w http.ResponseWriter, r *http.Request) {
	if h.Cache != nil {
		if err := h.Cache.FlushAll(context.Background()); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to flush cache")
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "cache flushed"})
}
