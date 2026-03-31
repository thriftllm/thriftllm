package transformer

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/thriftllm/backend/internal/model"
)

// OpenAITransformer handles OpenAI and OpenAI-compatible providers (Groq, Together, OpenRouter, custom)
type OpenAITransformer struct{}

func (t *OpenAITransformer) TransformRequest(req *model.ChatCompletionRequest, cfg *ProviderCfg) (*http.Request, error) {
	// Build the OpenAI request body - near passthrough, just set the correct model
	body := map[string]interface{}{
		"model":    cfg.ModelName,
		"messages": req.Messages,
		"stream":   req.Stream,
	}

	if req.Temperature != nil {
		body["temperature"] = *req.Temperature
	}
	if req.MaxTokens != nil {
		body["max_tokens"] = *req.MaxTokens
	}
	if req.TopP != nil {
		body["top_p"] = *req.TopP
	}
	if req.FrequencyPenalty != nil {
		body["frequency_penalty"] = *req.FrequencyPenalty
	}
	if req.PresencePenalty != nil {
		body["presence_penalty"] = *req.PresencePenalty
	}
	if req.Stop != nil {
		body["stop"] = req.Stop
	}
	if req.N != nil {
		body["n"] = *req.N
	}

	url := fmt.Sprintf("%s/chat/completions", cfg.BaseURL)
	headers := map[string]string{
		"Authorization": "Bearer " + cfg.APIKey,
	}

	return newJSONRequest("POST", url, body, headers)
}

func (t *OpenAITransformer) TransformResponse(body []byte, requestModel string) (*model.ChatCompletionResponse, error) {
	var resp model.ChatCompletionResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse OpenAI response: %w", err)
	}
	return &resp, nil
}

func (t *OpenAITransformer) TransformStreamChunk(data []byte, requestModel string) ([]byte, bool, error) {
	text := string(data)
	if text == "[DONE]" {
		done := []byte(`data: [DONE]`)
		return done, true, nil
	}

	// OpenAI chunks are already in the right format, pass through
	chunk := fmt.Sprintf("data: %s", text)
	return []byte(chunk), false, nil
}

// generateOpenAIResponse creates a standard OpenAI response from raw content
func generateOpenAIResponse(content string, requestModel string, usage *model.Usage) *model.ChatCompletionResponse {
	finishReason := "stop"
	return &model.ChatCompletionResponse{
		ID:      "chatcmpl-" + uuid.New().String()[:8],
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   requestModel,
		Choices: []model.ChatCompletionChoice{
			{
				Index: 0,
				Message: model.ChatMessage{
					Role:    "assistant",
					Content: content,
				},
				FinishReason: &finishReason,
			},
		},
		Usage: usage,
	}
}
