package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/thriftllm/backend/internal/store"
)

func RateLimit(rdb *store.Redis) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey, ok := GetAPIKey(r.Context())
			if !ok {
				next.ServeHTTP(w, r)
				return
			}

			key := fmt.Sprintf("ratelimit:%s", apiKey.ID.String())
			now := time.Now()
			windowStart := now.Add(-1 * time.Minute)

			pipe := rdb.Client.Pipeline()
			// Remove old entries
			pipe.ZRemRangeByScore(r.Context(), key, "0", fmt.Sprintf("%d", windowStart.UnixMilli()))
			// Count current entries
			countCmd := pipe.ZCard(r.Context(), key)
			// Add current request
			pipe.ZAdd(r.Context(), key, redis.Z{
				Score:  float64(now.UnixMilli()),
				Member: fmt.Sprintf("%d", now.UnixNano()),
			})
			// Set expiry
			pipe.Expire(r.Context(), key, 2*time.Minute)

			_, err := pipe.Exec(r.Context())
			if err != nil {
				// On Redis error, allow the request
				next.ServeHTTP(w, r)
				return
			}

			count := countCmd.Val()
			if count >= int64(apiKey.RateLimitRPM) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", apiKey.RateLimitRPM))
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("Retry-After", "60")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error":{"message":"Rate limit exceeded","type":"rate_limit_error"}}`))
				return
			}

			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", apiKey.RateLimitRPM))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", int64(apiKey.RateLimitRPM)-count))

			next.ServeHTTP(w, r)
		})
	}
}
