package transformer

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/thriftllm/backend/internal/model"
)

// GeminiTransformer handles Google Gemini API
type GeminiTransformer struct{}

type geminiRequest struct {
	Contents          []geminiContent    `json:"contents"`
	SystemInstruction *geminiContent     `json:"systemInstruction,omitempty"`
	GenerationConfig  *geminiGenConfig   `json:"generationConfig,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenConfig struct {
	Temperature     *float64 `json:"temperature,omitempty"`
	MaxOutputTokens *int     `json:"maxOutputTokens,omitempty"`
	TopP            *float64 `json:"topP,omitempty"`
	StopSequences   []string `json:"stopSequences,omitempty"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
			Role string `json:"role"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		TotalTokenCount      int `json:"totalTokenCount"`
	} `json:"usageMetadata"`
}

func (t *GeminiTransformer) TransformRequest(req *model.ChatCompletionRequest, cfg *ProviderCfg) (*http.Request, error) {
	gr := geminiRequest{}

	// Convert messages
	for _, msg := range req.Messages {
		if msg.Role == "system" {
			gr.SystemInstruction = &geminiContent{
				Parts: []geminiPart{{Text: msg.ContentString()}},
			}
			continue
		}

		role := msg.Role
		if role == "assistant" {
			role = "model"
		}

		gr.Contents = append(gr.Contents, geminiContent{
			Role:  role,
			Parts: []geminiPart{{Text: msg.ContentString()}},
		})
	}

	// Generation config
	genConfig := &geminiGenConfig{}
	hasGenConfig := false
	if req.Temperature != nil {
		genConfig.Temperature = req.Temperature
		hasGenConfig = true
	}
	if req.MaxTokens != nil {
		genConfig.MaxOutputTokens = req.MaxTokens
		hasGenConfig = true
	}
	if req.TopP != nil {
		genConfig.TopP = req.TopP
		hasGenConfig = true
	}
	if hasGenConfig {
		gr.GenerationConfig = genConfig
	}

	// Build URL
	action := "generateContent"
	if req.Stream {
		action = "streamGenerateContent"
	}
	url := fmt.Sprintf("%s/v1beta/models/%s:%s?key=%s", cfg.BaseURL, cfg.ModelName, action, cfg.APIKey)
	if req.Stream {
		url += "&alt=sse"
	}

	// For Gemini, no auth header (key in URL)
	return newJSONRequest("POST", url, gr, nil)
}

func (t *GeminiTransformer) TransformResponse(body []byte, requestModel string) (*model.ChatCompletionResponse, error) {
	var gr geminiResponse
	if err := json.Unmarshal(body, &gr); err != nil {
		return nil, fmt.Errorf("failed to parse Gemini response: %w", err)
	}

	content := ""
	if len(gr.Candidates) > 0 && len(gr.Candidates[0].Content.Parts) > 0 {
		content = gr.Candidates[0].Content.Parts[0].Text
	}

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
		Usage: &model.Usage{
			PromptTokens:     gr.UsageMetadata.PromptTokenCount,
			CompletionTokens: gr.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      gr.UsageMetadata.TotalTokenCount,
		},
	}, nil
}

func (t *GeminiTransformer) TransformStreamChunk(data []byte, requestModel string) ([]byte, bool, error) {
	var gr geminiResponse
	if err := json.Unmarshal(data, &gr); err != nil {
		return nil, false, nil
	}

	if len(gr.Candidates) == 0 {
		return nil, false, nil
	}

	text := ""
	if len(gr.Candidates[0].Content.Parts) > 0 {
		text = gr.Candidates[0].Content.Parts[0].Text
	}

	isDone := gr.Candidates[0].FinishReason == "STOP"

	if isDone && text == "" {
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
		result := fmt.Sprintf("data: %s\n\ndata: [DONE]", string(chunkData))
		return []byte(result), true, nil
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
					Content: text,
				},
			},
		},
	}
	chunkData, _ := json.Marshal(chunk)
	return []byte(fmt.Sprintf("data: %s", chunkData)), false, nil
}
