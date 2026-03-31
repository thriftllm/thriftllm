package transformer

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/thriftllm/backend/internal/model"
)

// AnthropicTransformer handles Anthropic Claude API
type AnthropicTransformer struct{}

type anthropicRequest struct {
	Model       string              `json:"model"`
	MaxTokens   int                 `json:"max_tokens"`
	System      string              `json:"system,omitempty"`
	Messages    []anthropicMessage  `json:"messages"`
	Stream      bool                `json:"stream"`
	Temperature *float64            `json:"temperature,omitempty"`
	TopP        *float64            `json:"top_p,omitempty"`
	StopSequences []string          `json:"stop_sequences,omitempty"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Model     string `json:"model"`
	StopReason string `json:"stop_reason"`
	Usage     struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

type anthropicStreamEvent struct {
	Type  string          `json:"type"`
	Index int             `json:"index,omitempty"`
	Delta json.RawMessage `json:"delta,omitempty"`
	Usage json.RawMessage `json:"usage,omitempty"`
}

func (t *AnthropicTransformer) TransformRequest(req *model.ChatCompletionRequest, cfg *ProviderCfg) (*http.Request, error) {
	ar := anthropicRequest{
		Model:     cfg.ModelName,
		MaxTokens: 4096,
		Stream:    req.Stream,
	}

	if req.MaxTokens != nil {
		ar.MaxTokens = *req.MaxTokens
	}
	if req.Temperature != nil {
		ar.Temperature = req.Temperature
	}
	if req.TopP != nil {
		ar.TopP = req.TopP
	}

	// Extract system message and convert messages
	for _, msg := range req.Messages {
		if msg.Role == "system" {
			ar.System = msg.ContentString()
			continue
		}
		ar.Messages = append(ar.Messages, anthropicMessage{
			Role:    msg.Role,
			Content: msg.ContentString(),
		})
	}

	if len(ar.Messages) == 0 {
		ar.Messages = []anthropicMessage{{Role: "user", Content: ""}}
	}

	url := fmt.Sprintf("%s/v1/messages", cfg.BaseURL)
	headers := map[string]string{
		"x-api-key":         cfg.APIKey,
		"anthropic-version": "2023-06-01",
	}

	return newJSONRequest("POST", url, ar, headers)
}

func (t *AnthropicTransformer) TransformResponse(body []byte, requestModel string) (*model.ChatCompletionResponse, error) {
	var ar anthropicResponse
	if err := json.Unmarshal(body, &ar); err != nil {
		return nil, fmt.Errorf("failed to parse Anthropic response: %w", err)
	}

	content := ""
	for _, block := range ar.Content {
		if block.Type == "text" {
			content += block.Text
		}
	}

	finishReason := "stop"
	if ar.StopReason == "max_tokens" {
		finishReason = "length"
	}

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
		Usage: &model.Usage{
			PromptTokens:     ar.Usage.InputTokens,
			CompletionTokens: ar.Usage.OutputTokens,
			TotalTokens:      ar.Usage.InputTokens + ar.Usage.OutputTokens,
		},
	}, nil
}

func (t *AnthropicTransformer) TransformStreamChunk(data []byte, requestModel string) ([]byte, bool, error) {
	var event anthropicStreamEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, false, nil // skip unparseable chunks
	}

	switch event.Type {
	case "content_block_delta":
		var delta struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}
		if err := json.Unmarshal(event.Delta, &delta); err != nil {
			return nil, false, nil
		}

		chunk := model.ChatCompletionChunk{
			ID:      "chatcmpl-" + uuid.New().String()[:8],
			Object:  "chat.completion.chunk",
			Created: time.Now().Unix(),
			Model:   requestModel,
			Choices: []model.ChatCompletionChoice{
				{
					Index: 0,
					Delta: &model.ChatMessage{
						Content: delta.Text,
					},
					FinishReason: nil,
				},
			},
		}
		chunkData, _ := json.Marshal(chunk)
		return []byte(fmt.Sprintf("data: %s", chunkData)), false, nil

	case "message_stop":
		finishReason := "stop"
		chunk := model.ChatCompletionChunk{
			ID:      "chatcmpl-" + uuid.New().String()[:8],
			Object:  "chat.completion.chunk",
			Created: time.Now().Unix(),
			Model:   requestModel,
			Choices: []model.ChatCompletionChoice{
				{
					Index:        0,
					Delta:        &model.ChatMessage{},
					FinishReason: &finishReason,
				},
			},
		}
		chunkData, _ := json.Marshal(chunk)
		return []byte(fmt.Sprintf("data: %s", chunkData)), false, nil

	case "message_delta":
		// May contain usage info; skip for now
		return nil, false, nil

	case "ping", "message_start", "content_block_start", "content_block_stop":
		return nil, false, nil

	default:
		return nil, false, nil
	}
}
