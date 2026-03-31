package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/thriftllm/backend/internal/model"
	"github.com/thriftllm/backend/internal/store"
)

const (
	ContextKeyAPIKey contextKey = "api_key"
)

func APIKeyAuth(db *store.Postgres) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if !strings.HasPrefix(auth, "Bearer thr_") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":{"message":"Invalid API key. Keys must start with thr_","type":"invalid_request_error"}}`))
				return
			}

			rawKey := strings.TrimPrefix(auth, "Bearer ")
			keyHash := store.HashAPIKey(rawKey)

			apiKey, err := db.GetAPIKeyByHash(r.Context(), keyHash)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":{"message":"Invalid API key","type":"invalid_request_error"}}`))
				return
			}

			if !apiKey.IsActive {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(`{"error":{"message":"API key is disabled","type":"invalid_request_error"}}`))
				return
			}

			if apiKey.ExpiresAt != nil && apiKey.ExpiresAt.Before(time.Now()) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(`{"error":{"message":"API key has expired","type":"invalid_request_error"}}`))
				return
			}

			// Update last used (fire and forget)
			go func() {
				_ = db.UpdateAPIKeyLastUsed(context.Background(), apiKey.ID)
			}()

			ctx := context.WithValue(r.Context(), ContextKeyAPIKey, *apiKey)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetAPIKey(ctx context.Context) (model.APIKey, bool) {
	k, ok := ctx.Value(ContextKeyAPIKey).(model.APIKey)
	return k, ok
}

func GetAPIKeyID(ctx context.Context) *uuid.UUID {
	k, ok := ctx.Value(ContextKeyAPIKey).(model.APIKey)
	if !ok {
		return nil
	}
	return &k.ID
}
