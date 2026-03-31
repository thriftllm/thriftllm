package proxy

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/thriftllm/backend/internal/model"
	"github.com/thriftllm/backend/internal/proxy/transformer"
)

// Executor handles the actual HTTP call to a provider
type Executor struct {
	Client *http.Client
}

func NewExecutor() *Executor {
	return &Executor{
		Client: &http.Client{
			Timeout: 120 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 20,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

// ExecuteResult contains the result of a provider call
type ExecuteResult struct {
	Response    *model.ChatCompletionResponse
	RawBody     []byte
	StatusCode  int
	LatencyMs   int
	TtfbMs      int
	InputTokens int
	OutputTokens int
	Error       error
}

// Execute calls a provider (non-streaming) and returns the transformed response
func (e *Executor) Execute(ctx context.Context, req *model.ChatCompletionRequest, cfg *ProviderConfig) (*ExecuteResult, error) {
	xformer := transformer.GetTransformer(cfg.Provider)
	providerCfg := &transformer.ProviderCfg{
		BaseURL:   cfg.BaseURL,
		APIKey:    cfg.APIKey,
		ModelName: cfg.ModelName,
	}

	start := time.Now()

	httpReq, err := xformer.TransformRequest(req, providerCfg)
	if err != nil {
		return nil, fmt.Errorf("transform request: %w", err)
	}
	httpReq = httpReq.WithContext(ctx)

	resp, err := e.Client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("provider request failed: %w", err)
	}
	defer resp.Body.Close()

	ttfb := int(time.Since(start).Milliseconds())

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	latency := int(time.Since(start).Milliseconds())

	if resp.StatusCode != http.StatusOK {
		return &ExecuteResult{
			StatusCode: resp.StatusCode,
			LatencyMs:  latency,
			TtfbMs:     ttfb,
			RawBody:    body,
			Error:      fmt.Errorf("provider returned %d: %s", resp.StatusCode, string(body)),
		}, nil
	}

	openAIResp, err := xformer.TransformResponse(body, req.Model)
	if err != nil {
		return nil, fmt.Errorf("transform response: %w", err)
	}

	result := &ExecuteResult{
		Response:   openAIResp,
		RawBody:    body,
		StatusCode: resp.StatusCode,
		LatencyMs:  latency,
		TtfbMs:     ttfb,
	}

	if openAIResp.Usage != nil {
		result.InputTokens = openAIResp.Usage.PromptTokens
		result.OutputTokens = openAIResp.Usage.CompletionTokens
	}

	return result, nil
}

// StreamMeta contains metadata headers to set on the response before streaming begins
type StreamMeta struct {
	Provider      string
	Model         string
	FallbackDepth int
}

// ExecuteStream calls a provider with streaming and writes SSE chunks to the writer
func (e *Executor) ExecuteStream(ctx context.Context, req *model.ChatCompletionRequest, cfg *ProviderConfig, w http.ResponseWriter, meta *StreamMeta) (*ExecuteResult, error) {
	xformer := transformer.GetTransformer(cfg.Provider)
	providerCfg := &transformer.ProviderCfg{
		BaseURL:   cfg.BaseURL,
		APIKey:    cfg.APIKey,
		ModelName: cfg.ModelName,
	}

	start := time.Now()

	httpReq, err := xformer.TransformRequest(req, providerCfg)
	if err != nil {
		return nil, fmt.Errorf("transform request: %w", err)
	}
	httpReq = httpReq.WithContext(ctx)

	resp, err := e.Client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("provider request failed: %w", err)
	}
	defer resp.Body.Close()

	ttfb := int(time.Since(start).Milliseconds())

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return &ExecuteResult{
			StatusCode: resp.StatusCode,
			LatencyMs:  int(time.Since(start).Milliseconds()),
			TtfbMs:     ttfb,
			RawBody:    body,
			Error:      fmt.Errorf("provider returned %d: %s", resp.StatusCode, string(body)),
		}, nil
	}

	// Set Thrift metadata headers and SSE headers (committed with first flush)
	if meta != nil {
		w.Header().Set("X-Thrift-Cache", "miss")
		w.Header().Set("X-Thrift-Provider", meta.Provider)
		w.Header().Set("X-Thrift-Model", meta.Model)
		w.Header().Set("X-Thrift-Fallback-Depth", fmt.Sprintf("%d", meta.FallbackDepth))
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("streaming not supported")
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var fullContent strings.Builder
	firstChunk := true

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Remove "data: " prefix
		data := strings.TrimPrefix(line, "data: ")
		if data == line && !strings.HasPrefix(line, "data:") {
			// Not a data line (could be event: or other SSE fields)
			if strings.HasPrefix(line, "event:") {
				continue
			}
			continue
		}
		data = strings.TrimPrefix(data, "data:")
		data = strings.TrimSpace(data)

		if data == "[DONE]" {
			fmt.Fprintf(w, "data: [DONE]\n\n")
			flusher.Flush()
			break
		}

		if data == "" {
			continue
		}

		chunk, isDone, err := xformer.TransformStreamChunk([]byte(data), req.Model)
		if err != nil {
			log.Printf("Error transforming stream chunk: %v", err)
			continue
		}

		if chunk == nil {
			continue
		}

		if firstChunk {
			ttfb = int(time.Since(start).Milliseconds())
			firstChunk = false
		}

		// Extract content for token counting
		extractContentFromChunk(chunk, &fullContent)

		fmt.Fprintf(w, "%s\n\n", chunk)
		flusher.Flush()

		if isDone {
			break
		}
	}

	latency := int(time.Since(start).Milliseconds())

	// Rough token estimate for streaming (4 chars ≈ 1 token)
	estimatedOutputTokens := len(fullContent.String()) / 4

	return &ExecuteResult{
		StatusCode:   200,
		LatencyMs:    latency,
		TtfbMs:       ttfb,
		OutputTokens: estimatedOutputTokens,
	}, nil
}

func extractContentFromChunk(chunk []byte, builder *strings.Builder) {
	// Try to extract content from the chunk for token estimation
	str := string(chunk)
	if !strings.HasPrefix(str, "data: ") {
		return
	}
	jsonStr := strings.TrimPrefix(str, "data: ")
	var cc model.ChatCompletionChunk
	if err := json.Unmarshal([]byte(jsonStr), &cc); err != nil {
		return
	}
	if len(cc.Choices) > 0 && cc.Choices[0].Delta != nil {
		if c, ok := cc.Choices[0].Delta.Content.(string); ok {
			builder.WriteString(c)
		}
	}
}

// ShouldRetry returns true if the error is retryable (rate limit, server error, timeout)
func ShouldRetry(result *ExecuteResult) bool {
	if result == nil {
		return true
	}
	switch result.StatusCode {
	case http.StatusTooManyRequests, // 429
		http.StatusInternalServerError,  // 500
		http.StatusBadGateway,           // 502
		http.StatusServiceUnavailable,   // 503
		http.StatusGatewayTimeout:       // 504
		return true
	}
	return false
}
