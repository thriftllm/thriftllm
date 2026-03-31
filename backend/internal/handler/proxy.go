package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/thriftllm/backend/internal/cache"
	"github.com/thriftllm/backend/internal/middleware"
	"github.com/thriftllm/backend/internal/model"
	"github.com/thriftllm/backend/internal/proxy"
	"github.com/thriftllm/backend/internal/store"
)

type ProxyHandler struct {
	DB       *store.Postgres
	Router   *proxy.Router
	Executor *proxy.Executor
	Cache    *cache.SemanticCache
}

func (h *ProxyHandler) ChatCompletions(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// Parse the OpenAI-format request
	var req model.ChatCompletionRequest
	if err := readJSON(r, &req); err != nil {
		writeOpenAIError(w, http.StatusBadRequest, "invalid_request_error", "Invalid request body")
		return
	}

	// Get API key from context
	apiKeyID := middleware.GetAPIKeyID(r.Context())

	// Parse tags from header
	tags := proxy.ParseTags(r.Header.Get("X-Thrift-Tags"))

	// Resolve model chain (primary + fallbacks)
	chain, err := h.Router.ResolveChain(r.Context(), req.Model, tags)
	if err != nil || len(chain) == 0 {
		writeOpenAIError(w, http.StatusNotFound, "model_not_found", "No models available for this request")
		return
	}

	// ---- Semantic Cache Check (non-streaming only) ----
	if !req.Stream && h.Cache != nil {
		temp := 0.0
		if req.Temperature != nil {
			temp = *req.Temperature
		}
		if temp <= 0.5 {
			cached, similarity, err := h.Cache.Lookup(r.Context(), &req, chain[0].ProviderModel)
			if err == nil && cached != nil {
				latency := int(time.Since(startTime).Milliseconds())

				// Log cache hit
				go h.logRequest(apiKeyID, &req, chain[0], cached, latency, 0, true, similarity, 0)

				w.Header().Set("X-Thrift-Cache", "hit")
				w.Header().Set("X-Thrift-Provider", string(chain[0].ProviderType))
				w.Header().Set("X-Thrift-Model", chain[0].ProviderModel)
				writeJSON(w, http.StatusOK, cached)
				return
			}
		}
	}

	// ---- Execute with fallback ----
	var lastResult *proxy.ExecuteResult
	var lastErr error
	fallbackDepth := 0

	for i, mc := range chain {
		if i > 2 { // max 3 attempts
			break
		}

		cfg, err := proxy.ResolveProviderConfig(&mc)
		if err != nil {
			log.Printf("Skipping model %s: %v", mc.ProviderModel, err)
			fallbackDepth++
			continue
		}

		if req.Stream {
			// ---- Streaming path ----
			meta := &proxy.StreamMeta{
				Provider:      string(mc.ProviderType),
				Model:         mc.ProviderModel,
				FallbackDepth: fallbackDepth,
			}
			result, err := h.Executor.ExecuteStream(r.Context(), &req, cfg, w, meta)
			if err != nil {
				log.Printf("Stream attempt %d failed for %s/%s: %v", i+1, mc.ProviderType, mc.ProviderModel, err)
				lastErr = err
				fallbackDepth++
				continue
			}
			if result != nil && result.Error != nil {
				// 400 = client error, request itself is bad — all providers will reject it
				if result.StatusCode == http.StatusBadRequest {
					writeOpenAIError(w, result.StatusCode, "invalid_request_error", result.Error.Error())
					return
				}
				// Any other provider error (401, 403, 429, 5xx) — try next model
				log.Printf("Stream attempt %d provider error for %s/%s: [%d] %v", i+1, mc.ProviderType, mc.ProviderModel, result.StatusCode, result.Error)
				lastResult = result
				lastErr = result.Error
				fallbackDepth++
				continue
			}
			// Streaming succeeded — headers were written by ExecuteStream
			if result != nil {
				go h.logRequest(apiKeyID, &req, mc, nil, result.LatencyMs, result.TtfbMs, false, nil, fallbackDepth)
			}
			return
		}

		// ---- Non-streaming path ----
		result, err := h.Executor.Execute(r.Context(), &req, cfg)
		if err != nil {
			log.Printf("Attempt %d failed for %s/%s: %v", i+1, mc.ProviderType, mc.ProviderModel, err)
			lastErr = err
			fallbackDepth++
			continue
		}

		if result != nil && result.Error != nil {
			// 400 = client error, request itself is bad — all providers will reject it
			if result.StatusCode == http.StatusBadRequest {
				writeOpenAIError(w, result.StatusCode, "invalid_request_error", result.Error.Error())
				go h.logRequest(apiKeyID, &req, mc, nil, result.LatencyMs, result.TtfbMs, false, nil, fallbackDepth)
				return
			}
			// Any other provider error (401 bad key, 403, 429 rate limit, 5xx) — try next model
			log.Printf("Attempt %d provider error for %s/%s: [%d] %v", i+1, mc.ProviderType, mc.ProviderModel, result.StatusCode, result.Error)
			lastResult = result
			lastErr = result.Error
			fallbackDepth++
			continue
		}

		if result != nil && result.Response != nil {
			// Success! Store in cache if applicable
			if h.Cache != nil {
				temp := 0.0
				if req.Temperature != nil {
					temp = *req.Temperature
				}
				if temp <= 0.5 {
					go func() {
						if err := h.Cache.Store(context.Background(), &req, mc.ProviderModel, result.Response); err != nil {
							log.Printf("Cache store error: %v", err)
						}
					}()
				}
			}

			// Calculate cost
			inputCost := float64(result.InputTokens) / 1000.0 * mc.InputCostPer1K
			outputCost := float64(result.OutputTokens) / 1000.0 * mc.OutputCostPer1K

			go h.logRequest(apiKeyID, &req, mc, result.Response, result.LatencyMs, result.TtfbMs, false, nil, fallbackDepth)

			// Set response headers
			w.Header().Set("X-Thrift-Cache", "miss")
			w.Header().Set("X-Thrift-Provider", string(mc.ProviderType))
			w.Header().Set("X-Thrift-Model", mc.ProviderModel)
			w.Header().Set("X-Thrift-Fallback-Depth", fmt.Sprintf("%d", fallbackDepth))
			w.Header().Set("X-Thrift-Input-Cost", formatFloat(inputCost))
			w.Header().Set("X-Thrift-Output-Cost", formatFloat(outputCost))

			writeJSON(w, http.StatusOK, result.Response)
			return
		}
	}

	// All attempts failed
	statusCode := http.StatusBadGateway
	errMsg := "all providers failed"
	if lastResult != nil {
		statusCode = lastResult.StatusCode
	}
	if lastErr != nil {
		errMsg = lastErr.Error()
	}
	writeOpenAIError(w, statusCode, "provider_error", errMsg)
}

func (h *ProxyHandler) logRequest(
	apiKeyID *uuid.UUID,
	req *model.ChatCompletionRequest,
	mc model.ModelConfig,
	resp *model.ChatCompletionResponse,
	latencyMs, ttfbMs int,
	cacheHit bool,
	cacheSimilarity *float64,
	fallbackDepth int,
) {
	rl := &model.RequestLog{
		ID:             uuid.New(),
		APIKeyID:       apiKeyID,
		RequestedModel: req.Model,
		ActualProvider: string(mc.ProviderType),
		ActualModel:    mc.ProviderModel,
		LatencyMs:      latencyMs,
		TtfbMs:         ttfbMs,
		StatusCode:     200,
		CacheHit:       cacheHit,
		CacheSimilarity: cacheSimilarity,
		FallbackDepth:  fallbackDepth,
		IsStreaming:    req.Stream,
		CreatedAt:      time.Now(),
	}

	if resp != nil && resp.Usage != nil {
		rl.InputTokens = resp.Usage.PromptTokens
		rl.OutputTokens = resp.Usage.CompletionTokens
		rl.TotalTokens = resp.Usage.TotalTokens
		rl.InputCostCents = float64(rl.InputTokens) / 1000.0 * mc.InputCostPer1K
		rl.OutputCostCents = float64(rl.OutputTokens) / 1000.0 * mc.OutputCostPer1K
		rl.TotalCostCents = rl.InputCostCents + rl.OutputCostCents
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := h.DB.InsertRequestLog(ctx, rl); err != nil {
		log.Printf("Failed to insert request log: %v", err)
	}
}

func writeOpenAIError(w http.ResponseWriter, status int, errType, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]interface{}{
			"message": message,
			"type":    errType,
		},
	})
}

func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}
