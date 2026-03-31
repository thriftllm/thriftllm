package proxy

import (
	"fmt"
	"os"
	"strings"

	"github.com/thriftllm/backend/internal/model"
)

// ProviderConfig holds the endpoint and auth info for a provider
type ProviderConfig struct {
	BaseURL    string
	APIKey     string
	Provider   model.Provider
	ModelName  string
	InputCost  float64
	OutputCost float64
}

// ResolveProviderConfig builds the config needed to call a specific model
func ResolveProviderConfig(mc *model.ModelConfig) (*ProviderConfig, error) {
	apiKey := os.Getenv(mc.APIKeyEnvName)
	if apiKey == "" {
		return nil, fmt.Errorf("API key env var %s is not set", mc.APIKeyEnvName)
	}

	baseURL := getProviderBaseURL(mc)

	return &ProviderConfig{
		BaseURL:    baseURL,
		APIKey:     apiKey,
		Provider:   mc.ProviderType,
		ModelName:  mc.ProviderModel,
		InputCost:  mc.InputCostPer1K,
		OutputCost: mc.OutputCostPer1K,
	}, nil
}

func getProviderBaseURL(mc *model.ModelConfig) string {
	if mc.APIBaseURL != nil && *mc.APIBaseURL != "" {
		return strings.TrimSuffix(*mc.APIBaseURL, "/")
	}

	switch mc.ProviderType {
	case model.ProviderOpenAI:
		return "https://api.openai.com/v1"
	case model.ProviderAnthropic:
		return "https://api.anthropic.com"
	case model.ProviderGemini:
		return "https://generativelanguage.googleapis.com"
	case model.ProviderGroq:
		return "https://api.groq.com/openai/v1"
	case model.ProviderTogether:
		return "https://api.together.xyz/v1"
	case model.ProviderOpenRouter:
		return "https://openrouter.ai/api/v1"
	default:
		return "https://api.openai.com/v1"
	}
}
