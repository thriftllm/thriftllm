package model

import (
	"time"

	"github.com/google/uuid"
)

// ---- Provider type ----

type Provider string

const (
	ProviderOpenAI      Provider = "openai"
	ProviderAnthropic   Provider = "anthropic"
	ProviderGemini      Provider = "gemini"
	ProviderGroq        Provider = "groq"
	ProviderTogether    Provider = "together"
	ProviderOpenRouter  Provider = "openrouter"
	ProviderCustomOpenAI Provider = "custom_openai"
)

// ---- Admin User ----

type AdminUser struct {
	ID           uuid.UUID  `db:"id" json:"id"`
	Name         string     `db:"name" json:"name"`
	Email        string     `db:"email" json:"email"`
	PasswordHash string     `db:"password_hash" json:"-"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
	LastLoginAt  *time.Time `db:"last_login_at" json:"last_login_at"`
}

// ---- Model Config ----

type ModelConfig struct {
	ID              uuid.UUID  `db:"id" json:"id"`
	ProviderType    Provider   `db:"provider" json:"provider"`
	ProviderModel   string     `db:"provider_model" json:"provider_model"`
	DisplayName     string     `db:"display_name" json:"display_name"`
	APIKeyEnvName   string     `db:"api_key_env_name" json:"api_key_env_name"`
	APIBaseURL      *string    `db:"api_base_url" json:"api_base_url,omitempty"`
	InputCostPer1K  float64    `db:"input_cost_per_1k" json:"input_cost_per_1k"`
	OutputCostPer1K float64    `db:"output_cost_per_1k" json:"output_cost_per_1k"`
	Tags            []string   `db:"tags" json:"tags"`
	IsActive        bool       `db:"is_active" json:"is_active"`
	Priority        int        `db:"priority" json:"priority"`
	CreatedAt       time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at" json:"updated_at"`
}

// ---- Fallback Chain ----

type FallbackChain struct {
	ID             uuid.UUID   `db:"id" json:"id"`
	Name           string      `db:"name" json:"name"`
	ModelConfigIDs []uuid.UUID `db:"model_config_ids" json:"model_config_ids"`
	TagSelector    *string     `db:"tag_selector" json:"tag_selector,omitempty"`
	IsDefault      bool        `db:"is_default" json:"is_default"`
	CreatedAt      time.Time   `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time   `db:"updated_at" json:"updated_at"`
}

// ---- API Key ----

type APIKey struct {
	ID           uuid.UUID  `db:"id" json:"id"`
	Name         string     `db:"name" json:"name"`
	KeyHash      string     `db:"key_hash" json:"-"`
	KeyPrefix    string     `db:"key_prefix" json:"key_prefix"`
	IsActive     bool       `db:"is_active" json:"is_active"`
	RateLimitRPM int        `db:"rate_limit_rpm" json:"rate_limit_rpm"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
	LastUsedAt   *time.Time `db:"last_used_at" json:"last_used_at"`
	ExpiresAt    *time.Time `db:"expires_at" json:"expires_at"`
}

// ---- Request Log ----

type RequestLog struct {
	ID              uuid.UUID  `db:"id" json:"id"`
	APIKeyID        *uuid.UUID `db:"api_key_id" json:"api_key_id"`
	RequestedModel  string     `db:"requested_model" json:"requested_model"`
	ActualProvider  string     `db:"actual_provider" json:"actual_provider"`
	ActualModel     string     `db:"actual_model" json:"actual_model"`
	InputTokens     int        `db:"input_tokens" json:"input_tokens"`
	OutputTokens    int        `db:"output_tokens" json:"output_tokens"`
	TotalTokens     int        `db:"total_tokens" json:"total_tokens"`
	InputCostCents  float64    `db:"input_cost_cents" json:"input_cost_cents"`
	OutputCostCents float64    `db:"output_cost_cents" json:"output_cost_cents"`
	TotalCostCents  float64    `db:"total_cost_cents" json:"total_cost_cents"`
	LatencyMs       int        `db:"latency_ms" json:"latency_ms"`
	TtfbMs          int        `db:"ttfb_ms" json:"ttfb_ms"`
	StatusCode      int        `db:"status_code" json:"status_code"`
	ErrorMessage    *string    `db:"error_message" json:"error_message,omitempty"`
	CacheHit        bool       `db:"cache_hit" json:"cache_hit"`
	CacheSimilarity *float64   `db:"cache_similarity" json:"cache_similarity,omitempty"`
	FallbackDepth   int        `db:"fallback_depth" json:"fallback_depth"`
	IsStreaming     bool       `db:"is_streaming" json:"is_streaming"`
	CreatedAt       time.Time  `db:"created_at" json:"created_at"`
}

// ---- Usage Daily ----

type UsageDaily struct {
	ID               int       `db:"id" json:"id"`
	Date             time.Time `db:"date" json:"date"`
	APIKeyID         *uuid.UUID `db:"api_key_id" json:"api_key_id"`
	ModelAlias       string    `db:"model_alias" json:"model_alias"`
	ProviderName     string    `db:"provider" json:"provider"`
	RequestCount     int       `db:"request_count" json:"request_count"`
	TotalInputTokens int64     `db:"total_input_tokens" json:"total_input_tokens"`
	TotalOutputTokens int64    `db:"total_output_tokens" json:"total_output_tokens"`
	TotalCostCents   float64   `db:"total_cost_cents" json:"total_cost_cents"`
	CacheHits        int       `db:"cache_hits" json:"cache_hits"`
	CacheMisses      int       `db:"cache_misses" json:"cache_misses"`
}

// ---- Cache Stats Daily ----

type CacheStatsDaily struct {
	ID             int       `db:"id" json:"id"`
	Date           time.Time `db:"date" json:"date"`
	TotalRequests  int       `db:"total_requests" json:"total_requests"`
	CacheHits      int       `db:"cache_hits" json:"cache_hits"`
	CacheMisses    int       `db:"cache_misses" json:"cache_misses"`
	TokensSaved    int64     `db:"tokens_saved" json:"tokens_saved"`
	CostSavedCents float64  `db:"cost_saved_cents" json:"cost_saved_cents"`
}

// ---- OpenAI-compatible types ----

type ChatCompletionRequest struct {
	Model            string          `json:"model"`
	Messages         []ChatMessage   `json:"messages"`
	Temperature      *float64        `json:"temperature,omitempty"`
	MaxTokens        *int            `json:"max_tokens,omitempty"`
	TopP             *float64        `json:"top_p,omitempty"`
	FrequencyPenalty *float64        `json:"frequency_penalty,omitempty"`
	PresencePenalty  *float64        `json:"presence_penalty,omitempty"`
	Stream           bool            `json:"stream"`
	N                *int            `json:"n,omitempty"`
	Stop             interface{}     `json:"stop,omitempty"`
	User             string          `json:"user,omitempty"`
}

type ChatMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"` // string or array for vision
}

func (m ChatMessage) ContentString() string {
	if s, ok := m.Content.(string); ok {
		return s
	}
	return ""
}

type ChatCompletionResponse struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Created int64                  `json:"created"`
	Model   string                 `json:"model"`
	Choices []ChatCompletionChoice `json:"choices"`
	Usage   *Usage                 `json:"usage,omitempty"`
}

type ChatCompletionChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message,omitempty"`
	Delta        *ChatMessage `json:"delta,omitempty"`
	FinishReason *string     `json:"finish_reason"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type ChatCompletionChunk struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Created int64                  `json:"created"`
	Model   string                 `json:"model"`
	Choices []ChatCompletionChoice `json:"choices"`
	Usage   *Usage                 `json:"usage,omitempty"`
}

// ---- API Request/Response types ----

type SetupRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string    `json:"token"`
	User  AdminUser `json:"user"`
}

type CreateModelRequest struct {
	Provider        Provider `json:"provider"`
	ProviderModel   string   `json:"provider_model"`
	DisplayName     string   `json:"display_name"`
	APIKeyEnvName   string   `json:"api_key_env_name"`
	APIBaseURL      *string  `json:"api_base_url,omitempty"`
	InputCostPer1K  float64  `json:"input_cost_per_1k"`
	OutputCostPer1K float64  `json:"output_cost_per_1k"`
	Tags            []string `json:"tags"`
	Priority        int      `json:"priority"`
}

type CreateFallbackChainRequest struct {
	Name           string      `json:"name"`
	ModelConfigIDs []uuid.UUID `json:"model_config_ids"`
	TagSelector    *string     `json:"tag_selector,omitempty"`
	IsDefault      bool        `json:"is_default"`
}

type CreateAPIKeyRequest struct {
	Name         string `json:"name"`
	RateLimitRPM int    `json:"rate_limit_rpm"`
}

type CreateAPIKeyResponse struct {
	Key    string `json:"key"` // full key, shown once
	APIKey APIKey `json:"api_key"`
}

type DashboardOverview struct {
	TotalRequests24h int     `json:"total_requests_24h"`
	TotalCost24h     float64 `json:"total_cost_24h"`
	CacheHitRate     float64 `json:"cache_hit_rate"`
	ActiveModels     int     `json:"active_models"`
	TotalRequests    int     `json:"total_requests"`
	TotalCost        float64 `json:"total_cost"`
	TokensSaved      int64   `json:"tokens_saved"`
	CostSaved        float64 `json:"cost_saved"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}
