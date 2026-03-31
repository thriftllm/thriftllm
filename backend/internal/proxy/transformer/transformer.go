package transformer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/thriftllm/backend/internal/model"
)

// Transformer converts between OpenAI format and provider-specific format
type Transformer interface {
	// TransformRequest converts an OpenAI-format request to provider format
	TransformRequest(req *model.ChatCompletionRequest, cfg *ProviderCfg) (*http.Request, error)
	// TransformResponse converts a provider response to OpenAI format
	TransformResponse(body []byte, requestModel string) (*model.ChatCompletionResponse, error)
	// TransformStreamChunk converts a provider SSE chunk to OpenAI format
	TransformStreamChunk(data []byte, requestModel string) ([]byte, bool, error) // returns (chunk, isDone, error)
}

type ProviderCfg struct {
	BaseURL   string
	APIKey    string
	ModelName string
}

// GetTransformer returns the appropriate transformer for the provider
func GetTransformer(provider model.Provider) Transformer {
	switch provider {
	case model.ProviderAnthropic:
		return &AnthropicTransformer{}
	case model.ProviderGemini:
		return &GeminiTransformer{}
	case model.ProviderOpenAI, model.ProviderGroq, model.ProviderTogether, model.ProviderOpenRouter, model.ProviderCustomOpenAI:
		return &OpenAITransformer{}
	default:
		return &OpenAITransformer{}
	}
}

// Helper to create HTTP request with JSON body
func newJSONRequest(method, url string, body interface{}, headers map[string]string) (*http.Request, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return req, nil
}
