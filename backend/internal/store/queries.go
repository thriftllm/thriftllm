package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/thriftllm/backend/internal/model"
)

// ---- Setup ----

func (p *Postgres) IsSetupComplete(ctx context.Context) (bool, error) {
	var complete bool
	err := p.DB.GetContext(ctx, &complete, "SELECT is_complete FROM setup_status WHERE id = TRUE")
	if err != nil {
		return false, err
	}
	return complete, nil
}

func (p *Postgres) CompleteSetup(ctx context.Context) error {
	_, err := p.DB.ExecContext(ctx, "UPDATE setup_status SET is_complete = TRUE, completed_at = NOW() WHERE id = TRUE")
	return err
}

// ---- Admin Users ----

func (p *Postgres) CreateAdminUser(ctx context.Context, name, email, passwordHash string) (*model.AdminUser, error) {
	user := &model.AdminUser{}
	err := p.DB.QueryRowxContext(ctx,
		`INSERT INTO admin_users (name, email, password_hash) VALUES ($1, $2, $3)
		 RETURNING id, name, email, password_hash, created_at, last_login_at`,
		name, email, passwordHash,
	).StructScan(user)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (p *Postgres) GetAdminByEmail(ctx context.Context, email string) (*model.AdminUser, error) {
	user := &model.AdminUser{}
	err := p.DB.GetContext(ctx, user, "SELECT * FROM admin_users WHERE email = $1", email)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (p *Postgres) GetAdminByID(ctx context.Context, id uuid.UUID) (*model.AdminUser, error) {
	user := &model.AdminUser{}
	err := p.DB.GetContext(ctx, user, "SELECT * FROM admin_users WHERE id = $1", id)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (p *Postgres) UpdateAdminLogin(ctx context.Context, id uuid.UUID) error {
	_, err := p.DB.ExecContext(ctx, "UPDATE admin_users SET last_login_at = NOW() WHERE id = $1", id)
	return err
}

func (p *Postgres) UpdateAdminProfile(ctx context.Context, id uuid.UUID, name, email string) error {
	_, err := p.DB.ExecContext(ctx, "UPDATE admin_users SET name = $1, email = $2 WHERE id = $3", name, email, id)
	return err
}

func (p *Postgres) UpdateAdminPassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	_, err := p.DB.ExecContext(ctx, "UPDATE admin_users SET password_hash = $1 WHERE id = $2", passwordHash, id)
	return err
}

// ---- Model Configs ----

func (p *Postgres) ListModelConfigs(ctx context.Context) ([]model.ModelConfig, error) {
	rows, err := p.DB.QueryxContext(ctx,
		`SELECT id, provider, provider_model, display_name, api_key_env_name, api_base_url,
		        input_cost_per_1k, output_cost_per_1k, tags, is_active, priority, created_at, updated_at
		 FROM model_configs ORDER BY priority ASC, created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []model.ModelConfig
	for rows.Next() {
		var mc model.ModelConfig
		var tags pq.StringArray
		err := rows.Scan(&mc.ID, &mc.ProviderType, &mc.ProviderModel, &mc.DisplayName,
			&mc.APIKeyEnvName, &mc.APIBaseURL, &mc.InputCostPer1K, &mc.OutputCostPer1K,
			&tags, &mc.IsActive, &mc.Priority, &mc.CreatedAt, &mc.UpdatedAt)
		if err != nil {
			return nil, err
		}
		mc.Tags = []string(tags)
		if mc.Tags == nil {
			mc.Tags = []string{}
		}
		configs = append(configs, mc)
	}
	if configs == nil {
		configs = []model.ModelConfig{}
	}
	return configs, nil
}

func (p *Postgres) GetModelConfig(ctx context.Context, id uuid.UUID) (*model.ModelConfig, error) {
	row := p.DB.QueryRowxContext(ctx,
		`SELECT id, provider, provider_model, display_name, api_key_env_name, api_base_url,
		        input_cost_per_1k, output_cost_per_1k, tags, is_active, priority, created_at, updated_at
		 FROM model_configs WHERE id = $1`, id)

	var mc model.ModelConfig
	var tags pq.StringArray
	err := row.Scan(&mc.ID, &mc.ProviderType, &mc.ProviderModel, &mc.DisplayName,
		&mc.APIKeyEnvName, &mc.APIBaseURL, &mc.InputCostPer1K, &mc.OutputCostPer1K,
		&tags, &mc.IsActive, &mc.Priority, &mc.CreatedAt, &mc.UpdatedAt)
	if err != nil {
		return nil, err
	}
	mc.Tags = []string(tags)
	if mc.Tags == nil {
		mc.Tags = []string{}
	}
	return &mc, nil
}

func (p *Postgres) GetActiveModelByName(ctx context.Context, providerModel string) (*model.ModelConfig, error) {
	row := p.DB.QueryRowxContext(ctx,
		`SELECT id, provider, provider_model, display_name, api_key_env_name, api_base_url,
		        input_cost_per_1k, output_cost_per_1k, tags, is_active, priority, created_at, updated_at
		 FROM model_configs WHERE (provider_model = $1 OR display_name = $1) AND is_active = TRUE
		 ORDER BY priority ASC LIMIT 1`, providerModel)

	var mc model.ModelConfig
	var tags pq.StringArray
	err := row.Scan(&mc.ID, &mc.ProviderType, &mc.ProviderModel, &mc.DisplayName,
		&mc.APIKeyEnvName, &mc.APIBaseURL, &mc.InputCostPer1K, &mc.OutputCostPer1K,
		&tags, &mc.IsActive, &mc.Priority, &mc.CreatedAt, &mc.UpdatedAt)
	if err != nil {
		return nil, err
	}
	mc.Tags = []string(tags)
	return &mc, nil
}

func (p *Postgres) GetActiveModelsByTags(ctx context.Context, tags []string) ([]model.ModelConfig, error) {
	rows, err := p.DB.QueryxContext(ctx,
		`SELECT id, provider, provider_model, display_name, api_key_env_name, api_base_url,
		        input_cost_per_1k, output_cost_per_1k, tags, is_active, priority, created_at, updated_at
		 FROM model_configs WHERE is_active = TRUE AND tags && $1
		 ORDER BY priority ASC`, pq.Array(tags))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []model.ModelConfig
	for rows.Next() {
		var mc model.ModelConfig
		var t pq.StringArray
		err := rows.Scan(&mc.ID, &mc.ProviderType, &mc.ProviderModel, &mc.DisplayName,
			&mc.APIKeyEnvName, &mc.APIBaseURL, &mc.InputCostPer1K, &mc.OutputCostPer1K,
			&t, &mc.IsActive, &mc.Priority, &mc.CreatedAt, &mc.UpdatedAt)
		if err != nil {
			return nil, err
		}
		mc.Tags = []string(t)
		configs = append(configs, mc)
	}
	return configs, nil
}

func (p *Postgres) CreateModelConfig(ctx context.Context, req model.CreateModelRequest) (*model.ModelConfig, error) {
	mc := &model.ModelConfig{}
	var tags pq.StringArray
	err := p.DB.QueryRowxContext(ctx,
		`INSERT INTO model_configs (provider, provider_model, display_name, api_key_env_name, api_base_url,
		 input_cost_per_1k, output_cost_per_1k, tags, priority)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING id, provider, provider_model, display_name, api_key_env_name, api_base_url,
		           input_cost_per_1k, output_cost_per_1k, tags, is_active, priority, created_at, updated_at`,
		req.Provider, req.ProviderModel, req.DisplayName, req.APIKeyEnvName, req.APIBaseURL,
		req.InputCostPer1K, req.OutputCostPer1K, pq.Array(req.Tags), req.Priority,
	).Scan(&mc.ID, &mc.ProviderType, &mc.ProviderModel, &mc.DisplayName,
		&mc.APIKeyEnvName, &mc.APIBaseURL, &mc.InputCostPer1K, &mc.OutputCostPer1K,
		&tags, &mc.IsActive, &mc.Priority, &mc.CreatedAt, &mc.UpdatedAt)
	if err != nil {
		return nil, err
	}
	mc.Tags = []string(tags)
	if mc.Tags == nil {
		mc.Tags = []string{}
	}
	return mc, nil
}

func (p *Postgres) UpdateModelConfig(ctx context.Context, id uuid.UUID, req model.CreateModelRequest) (*model.ModelConfig, error) {
	mc := &model.ModelConfig{}
	var tags pq.StringArray
	err := p.DB.QueryRowxContext(ctx,
		`UPDATE model_configs SET provider = $1, provider_model = $2, display_name = $3,
		 api_key_env_name = $4, api_base_url = $5, input_cost_per_1k = $6, output_cost_per_1k = $7,
		 tags = $8, priority = $9, updated_at = NOW()
		 WHERE id = $10
		 RETURNING id, provider, provider_model, display_name, api_key_env_name, api_base_url,
		           input_cost_per_1k, output_cost_per_1k, tags, is_active, priority, created_at, updated_at`,
		req.Provider, req.ProviderModel, req.DisplayName, req.APIKeyEnvName, req.APIBaseURL,
		req.InputCostPer1K, req.OutputCostPer1K, pq.Array(req.Tags), req.Priority, id,
	).Scan(&mc.ID, &mc.ProviderType, &mc.ProviderModel, &mc.DisplayName,
		&mc.APIKeyEnvName, &mc.APIBaseURL, &mc.InputCostPer1K, &mc.OutputCostPer1K,
		&tags, &mc.IsActive, &mc.Priority, &mc.CreatedAt, &mc.UpdatedAt)
	if err != nil {
		return nil, err
	}
	mc.Tags = []string(tags)
	return mc, nil
}

func (p *Postgres) DeleteModelConfig(ctx context.Context, id uuid.UUID) error {
	_, err := p.DB.ExecContext(ctx, "DELETE FROM model_configs WHERE id = $1", id)
	return err
}

func (p *Postgres) ToggleModelActive(ctx context.Context, id uuid.UUID, active bool) error {
	_, err := p.DB.ExecContext(ctx, "UPDATE model_configs SET is_active = $1, updated_at = NOW() WHERE id = $2", active, id)
	return err
}

func (p *Postgres) CountActiveModels(ctx context.Context) (int, error) {
	var count int
	err := p.DB.GetContext(ctx, &count, "SELECT COUNT(*) FROM model_configs WHERE is_active = TRUE")
	return count, err
}

// ---- API Keys ----

func HashAPIKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}

func (p *Postgres) CreateAPIKey(ctx context.Context, name string, rateLimitRPM int) (string, *model.APIKey, error) {
	rawKey := "thr_" + uuid.New().String()
	keyHash := HashAPIKey(rawKey)
	keyPrefix := rawKey[:12]

	ak := &model.APIKey{}
	err := p.DB.QueryRowxContext(ctx,
		`INSERT INTO api_keys (name, key_hash, key_prefix, rate_limit_rpm)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, name, key_hash, key_prefix, is_active, rate_limit_rpm, created_at, last_used_at, expires_at`,
		name, keyHash, keyPrefix, rateLimitRPM,
	).StructScan(ak)
	if err != nil {
		return "", nil, err
	}

	return rawKey, ak, nil
}

func (p *Postgres) GetAPIKeyByHash(ctx context.Context, keyHash string) (*model.APIKey, error) {
	ak := &model.APIKey{}
	err := p.DB.GetContext(ctx, ak, "SELECT * FROM api_keys WHERE key_hash = $1 AND is_active = TRUE", keyHash)
	if err != nil {
		return nil, err
	}
	return ak, nil
}

func (p *Postgres) ListAPIKeys(ctx context.Context) ([]model.APIKey, error) {
	var keys []model.APIKey
	err := p.DB.SelectContext(ctx, &keys, "SELECT * FROM api_keys ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	if keys == nil {
		keys = []model.APIKey{}
	}
	return keys, nil
}

func (p *Postgres) DeleteAPIKey(ctx context.Context, id uuid.UUID) error {
	_, err := p.DB.ExecContext(ctx, "DELETE FROM api_keys WHERE id = $1", id)
	return err
}

func (p *Postgres) ToggleAPIKey(ctx context.Context, id uuid.UUID, active bool) error {
	_, err := p.DB.ExecContext(ctx, "UPDATE api_keys SET is_active = $1 WHERE id = $2", active, id)
	return err
}

func (p *Postgres) UpdateAPIKeyLastUsed(ctx context.Context, id uuid.UUID) error {
	_, err := p.DB.ExecContext(ctx, "UPDATE api_keys SET last_used_at = NOW() WHERE id = $1", id)
	return err
}

// ---- Fallback Chains ----

func (p *Postgres) ListFallbackChains(ctx context.Context) ([]model.FallbackChain, error) {
	rows, err := p.DB.QueryxContext(ctx,
		`SELECT id, name, model_config_ids, tag_selector, is_default, created_at, updated_at
		 FROM fallback_chains ORDER BY is_default DESC, created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chains []model.FallbackChain
	for rows.Next() {
		var fc model.FallbackChain
		var ids pq.StringArray
		err := rows.Scan(&fc.ID, &fc.Name, &ids, &fc.TagSelector, &fc.IsDefault, &fc.CreatedAt, &fc.UpdatedAt)
		if err != nil {
			return nil, err
		}
		fc.ModelConfigIDs = make([]uuid.UUID, len(ids))
		for i, s := range ids {
			fc.ModelConfigIDs[i], _ = uuid.Parse(s)
		}
		chains = append(chains, fc)
	}
	if chains == nil {
		chains = []model.FallbackChain{}
	}
	return chains, nil
}

func (p *Postgres) GetFallbackChain(ctx context.Context, id uuid.UUID) (*model.FallbackChain, error) {
	row := p.DB.QueryRowxContext(ctx,
		`SELECT id, name, model_config_ids, tag_selector, is_default, created_at, updated_at
		 FROM fallback_chains WHERE id = $1`, id)

	var fc model.FallbackChain
	var ids pq.StringArray
	err := row.Scan(&fc.ID, &fc.Name, &ids, &fc.TagSelector, &fc.IsDefault, &fc.CreatedAt, &fc.UpdatedAt)
	if err != nil {
		return nil, err
	}
	fc.ModelConfigIDs = make([]uuid.UUID, len(ids))
	for i, s := range ids {
		fc.ModelConfigIDs[i], _ = uuid.Parse(s)
	}
	return &fc, nil
}

func (p *Postgres) GetDefaultFallbackChain(ctx context.Context) (*model.FallbackChain, error) {
	row := p.DB.QueryRowxContext(ctx,
		`SELECT id, name, model_config_ids, tag_selector, is_default, created_at, updated_at
		 FROM fallback_chains WHERE is_default = TRUE LIMIT 1`)

	var fc model.FallbackChain
	var ids pq.StringArray
	err := row.Scan(&fc.ID, &fc.Name, &ids, &fc.TagSelector, &fc.IsDefault, &fc.CreatedAt, &fc.UpdatedAt)
	if err != nil {
		return nil, err
	}
	fc.ModelConfigIDs = make([]uuid.UUID, len(ids))
	for i, s := range ids {
		fc.ModelConfigIDs[i], _ = uuid.Parse(s)
	}
	return &fc, nil
}

func (p *Postgres) GetFallbackChainByTag(ctx context.Context, tag string) (*model.FallbackChain, error) {
	row := p.DB.QueryRowxContext(ctx,
		`SELECT id, name, model_config_ids, tag_selector, is_default, created_at, updated_at
		 FROM fallback_chains WHERE tag_selector = $1 LIMIT 1`, tag)

	var fc model.FallbackChain
	var ids pq.StringArray
	err := row.Scan(&fc.ID, &fc.Name, &ids, &fc.TagSelector, &fc.IsDefault, &fc.CreatedAt, &fc.UpdatedAt)
	if err != nil {
		return nil, err
	}
	fc.ModelConfigIDs = make([]uuid.UUID, len(ids))
	for i, s := range ids {
		fc.ModelConfigIDs[i], _ = uuid.Parse(s)
	}
	return &fc, nil
}

func (p *Postgres) CreateFallbackChain(ctx context.Context, req model.CreateFallbackChainRequest) (*model.FallbackChain, error) {
	// If setting as default, clear existing default first
	if req.IsDefault {
		_, _ = p.DB.ExecContext(ctx, "UPDATE fallback_chains SET is_default = FALSE, updated_at = NOW() WHERE is_default = TRUE")
	}

	idStrs := make([]string, len(req.ModelConfigIDs))
	for i, id := range req.ModelConfigIDs {
		idStrs[i] = id.String()
	}

	var fc model.FallbackChain
	var ids pq.StringArray
	err := p.DB.QueryRowxContext(ctx,
		`INSERT INTO fallback_chains (name, model_config_ids, tag_selector, is_default)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, name, model_config_ids, tag_selector, is_default, created_at, updated_at`,
		req.Name, pq.Array(idStrs), req.TagSelector, req.IsDefault,
	).Scan(&fc.ID, &fc.Name, &ids, &fc.TagSelector, &fc.IsDefault, &fc.CreatedAt, &fc.UpdatedAt)
	if err != nil {
		return nil, err
	}
	fc.ModelConfigIDs = make([]uuid.UUID, len(ids))
	for i, s := range ids {
		fc.ModelConfigIDs[i], _ = uuid.Parse(s)
	}
	return &fc, nil
}

func (p *Postgres) UpdateFallbackChain(ctx context.Context, id uuid.UUID, req model.CreateFallbackChainRequest) (*model.FallbackChain, error) {
	// If setting as default, clear existing default first
	if req.IsDefault {
		_, _ = p.DB.ExecContext(ctx, "UPDATE fallback_chains SET is_default = FALSE, updated_at = NOW() WHERE is_default = TRUE AND id != $1", id)
	}

	idStrs := make([]string, len(req.ModelConfigIDs))
	for i, uid := range req.ModelConfigIDs {
		idStrs[i] = uid.String()
	}

	var fc model.FallbackChain
	var ids pq.StringArray
	err := p.DB.QueryRowxContext(ctx,
		`UPDATE fallback_chains SET name = $1, model_config_ids = $2, tag_selector = $3,
		 is_default = $4, updated_at = NOW()
		 WHERE id = $5
		 RETURNING id, name, model_config_ids, tag_selector, is_default, created_at, updated_at`,
		req.Name, pq.Array(idStrs), req.TagSelector, req.IsDefault, id,
	).Scan(&fc.ID, &fc.Name, &ids, &fc.TagSelector, &fc.IsDefault, &fc.CreatedAt, &fc.UpdatedAt)
	if err != nil {
		return nil, err
	}
	fc.ModelConfigIDs = make([]uuid.UUID, len(ids))
	for i, s := range ids {
		fc.ModelConfigIDs[i], _ = uuid.Parse(s)
	}
	return &fc, nil
}

func (p *Postgres) DeleteFallbackChain(ctx context.Context, id uuid.UUID) error {
	_, err := p.DB.ExecContext(ctx, "DELETE FROM fallback_chains WHERE id = $1", id)
	return err
}

// ---- Request Logs ----

func (p *Postgres) InsertRequestLog(ctx context.Context, log *model.RequestLog) error {
	_, err := p.DB.ExecContext(ctx,
		`INSERT INTO request_logs (id, api_key_id, requested_model, actual_provider, actual_model,
		 input_tokens, output_tokens, total_tokens, input_cost_cents, output_cost_cents, total_cost_cents,
		 latency_ms, ttfb_ms, status_code, error_message, cache_hit, cache_similarity, fallback_depth,
		 is_streaming, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20)`,
		log.ID, log.APIKeyID, log.RequestedModel, log.ActualProvider, log.ActualModel,
		log.InputTokens, log.OutputTokens, log.TotalTokens,
		log.InputCostCents, log.OutputCostCents, log.TotalCostCents,
		log.LatencyMs, log.TtfbMs, log.StatusCode, log.ErrorMessage,
		log.CacheHit, log.CacheSimilarity, log.FallbackDepth, log.IsStreaming, log.CreatedAt,
	)
	return err
}

type RequestLogFilter struct {
	Page     int
	Limit    int
	Search   string
	Model    string
	Provider string
	CacheHit *bool
	Status   *int
}

type PaginatedLogs struct {
	Logs       []model.RequestLog `json:"logs"`
	Total      int                `json:"total"`
	Page       int                `json:"page"`
	Limit      int                `json:"limit"`
	TotalPages int                `json:"total_pages"`
}

func (p *Postgres) ListRequestLogs(ctx context.Context, filter RequestLogFilter) (*PaginatedLogs, error) {
	where := []string{"1=1"}
	args := []interface{}{}
	argIdx := 1

	if filter.Search != "" {
		where = append(where, fmt.Sprintf("(requested_model ILIKE $%d OR actual_model ILIKE $%d OR actual_provider ILIKE $%d)", argIdx, argIdx, argIdx))
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}
	if filter.Model != "" {
		where = append(where, fmt.Sprintf("requested_model ILIKE $%d", argIdx))
		args = append(args, "%"+filter.Model+"%")
		argIdx++
	}
	if filter.Provider != "" {
		where = append(where, fmt.Sprintf("actual_provider = $%d", argIdx))
		args = append(args, filter.Provider)
		argIdx++
	}
	if filter.CacheHit != nil {
		where = append(where, fmt.Sprintf("cache_hit = $%d", argIdx))
		args = append(args, *filter.CacheHit)
		argIdx++
	}
	if filter.Status != nil {
		where = append(where, fmt.Sprintf("status_code = $%d", argIdx))
		args = append(args, *filter.Status)
		argIdx++
	}

	whereClause := strings.Join(where, " AND ")

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM request_logs WHERE %s", whereClause)
	err := p.DB.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, err
	}

	offset := (filter.Page - 1) * filter.Limit
	query := fmt.Sprintf(
		`SELECT id, api_key_id, requested_model, actual_provider, actual_model,
		 input_tokens, output_tokens, total_tokens, input_cost_cents, output_cost_cents, total_cost_cents,
		 latency_ms, ttfb_ms, status_code, error_message, cache_hit, cache_similarity, fallback_depth,
		 is_streaming, created_at
		 FROM request_logs WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		whereClause, argIdx, argIdx+1)
	args = append(args, filter.Limit, offset)

	var logs []model.RequestLog
	err = p.DB.SelectContext(ctx, &logs, query, args...)
	if err != nil {
		return nil, err
	}
	if logs == nil {
		logs = []model.RequestLog{}
	}

	totalPages := total / filter.Limit
	if total%filter.Limit != 0 {
		totalPages++
	}

	return &PaginatedLogs{
		Logs:       logs,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

// ---- Dashboard Queries ----

func (p *Postgres) GetOverviewStats(ctx context.Context) (*model.DashboardOverview, error) {
	overview := &model.DashboardOverview{}

	// 24h stats
	err := p.DB.GetContext(ctx, &overview.TotalRequests24h,
		"SELECT COUNT(*) FROM request_logs WHERE created_at > NOW() - INTERVAL '24 hours'")
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	err = p.DB.GetContext(ctx, &overview.TotalCost24h,
		"SELECT COALESCE(SUM(total_cost_cents), 0) FROM request_logs WHERE created_at > NOW() - INTERVAL '24 hours'")
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Cache hit rate (24h)
	var totalReqs, cacheHits int
	_ = p.DB.GetContext(ctx, &totalReqs,
		"SELECT COUNT(*) FROM request_logs WHERE created_at > NOW() - INTERVAL '24 hours'")
	_ = p.DB.GetContext(ctx, &cacheHits,
		"SELECT COUNT(*) FROM request_logs WHERE created_at > NOW() - INTERVAL '24 hours' AND cache_hit = TRUE")
	if totalReqs > 0 {
		overview.CacheHitRate = float64(cacheHits) / float64(totalReqs) * 100
	}

	// Active models
	overview.ActiveModels, _ = p.CountActiveModels(ctx)

	// All-time totals
	_ = p.DB.GetContext(ctx, &overview.TotalRequests, "SELECT COUNT(*) FROM request_logs")
	_ = p.DB.GetContext(ctx, &overview.TotalCost,
		"SELECT COALESCE(SUM(total_cost_cents), 0) FROM request_logs")

	// Tokens & cost saved
	_ = p.DB.GetContext(ctx, &overview.TokensSaved,
		"SELECT COALESCE(SUM(total_tokens), 0) FROM request_logs WHERE cache_hit = TRUE")
	_ = p.DB.GetContext(ctx, &overview.CostSaved,
		"SELECT COALESCE(SUM(total_cost_cents), 0) FROM request_logs WHERE cache_hit = TRUE")

	return overview, nil
}

type UsageDataPoint struct {
	Date         string  `db:"date" json:"date"`
	Requests     int     `db:"requests" json:"requests"`
	TotalCost    float64 `db:"total_cost" json:"total_cost"`
	InputTokens  int64   `db:"input_tokens" json:"input_tokens"`
	OutputTokens int64   `db:"output_tokens" json:"output_tokens"`
	CacheHits    int     `db:"cache_hits" json:"cache_hits"`
}

func (p *Postgres) GetUsageOverTime(ctx context.Context, days int) ([]UsageDataPoint, error) {
	var data []UsageDataPoint
	err := p.DB.SelectContext(ctx, &data,
		`SELECT date_trunc('day', created_at)::date::text AS date,
		 COUNT(*) AS requests,
		 COALESCE(SUM(total_cost_cents), 0) AS total_cost,
		 COALESCE(SUM(input_tokens), 0) AS input_tokens,
		 COALESCE(SUM(output_tokens), 0) AS output_tokens,
		 COUNT(*) FILTER (WHERE cache_hit = TRUE) AS cache_hits
		 FROM request_logs
		 WHERE created_at > NOW() - ($1 || ' days')::interval
		 GROUP BY date_trunc('day', created_at)::date
		 ORDER BY date ASC`, fmt.Sprintf("%d", days))
	if err != nil {
		return nil, err
	}
	if data == nil {
		data = []UsageDataPoint{}
	}
	return data, nil
}

type ModelUsageBreakdown struct {
	Model    string  `db:"model" json:"model"`
	Provider string  `db:"provider" json:"provider"`
	Requests int     `db:"requests" json:"requests"`
	Cost     float64 `db:"cost" json:"cost"`
	Tokens   int64   `db:"tokens" json:"tokens"`
}

func (p *Postgres) GetModelUsageBreakdown(ctx context.Context, days int) ([]ModelUsageBreakdown, error) {
	var data []ModelUsageBreakdown
	err := p.DB.SelectContext(ctx, &data,
		`SELECT actual_model AS model, actual_provider AS provider,
		 COUNT(*) AS requests,
		 COALESCE(SUM(total_cost_cents), 0) AS cost,
		 COALESCE(SUM(total_tokens), 0) AS tokens
		 FROM request_logs
		 WHERE created_at > NOW() - ($1 || ' days')::interval
		 GROUP BY actual_model, actual_provider
		 ORDER BY requests DESC`, fmt.Sprintf("%d", days))
	if err != nil {
		return nil, err
	}
	if data == nil {
		data = []ModelUsageBreakdown{}
	}
	return data, nil
}

type CacheOverview struct {
	TotalRequests int     `json:"total_requests"`
	CacheHits     int     `json:"cache_hits"`
	CacheMisses   int     `json:"cache_misses"`
	HitRate       float64 `json:"hit_rate"`
	TokensSaved   int64   `json:"tokens_saved"`
	CostSaved     float64 `json:"cost_saved"`
}

func (p *Postgres) GetCacheOverview(ctx context.Context) (*CacheOverview, error) {
	co := &CacheOverview{}
	_ = p.DB.GetContext(ctx, &co.TotalRequests, "SELECT COUNT(*) FROM request_logs")
	_ = p.DB.GetContext(ctx, &co.CacheHits, "SELECT COUNT(*) FROM request_logs WHERE cache_hit = TRUE")
	co.CacheMisses = co.TotalRequests - co.CacheHits
	if co.TotalRequests > 0 {
		co.HitRate = float64(co.CacheHits) / float64(co.TotalRequests) * 100
	}
	_ = p.DB.GetContext(ctx, &co.TokensSaved,
		"SELECT COALESCE(SUM(total_tokens), 0) FROM request_logs WHERE cache_hit = TRUE")
	_ = p.DB.GetContext(ctx, &co.CostSaved,
		"SELECT COALESCE(SUM(total_cost_cents), 0) FROM request_logs WHERE cache_hit = TRUE")
	return co, nil
}

// ---- Aggregation ----

func (p *Postgres) AggregateUsageDaily(ctx context.Context, date time.Time) error {
	dateStr := date.Format("2006-01-02")
	_, err := p.DB.ExecContext(ctx,
		`INSERT INTO usage_daily (date, api_key_id, model_alias, provider, request_count,
		 total_input_tokens, total_output_tokens, total_cost_cents, cache_hits, cache_misses)
		 SELECT $1::date, api_key_id, requested_model, actual_provider,
		   COUNT(*), COALESCE(SUM(input_tokens),0), COALESCE(SUM(output_tokens),0),
		   COALESCE(SUM(total_cost_cents),0),
		   COUNT(*) FILTER (WHERE cache_hit = TRUE),
		   COUNT(*) FILTER (WHERE cache_hit = FALSE)
		 FROM request_logs
		 WHERE created_at >= $1::date AND created_at < ($1::date + INTERVAL '1 day')
		 GROUP BY api_key_id, requested_model, actual_provider
		 ON CONFLICT (date, api_key_id, model_alias) DO UPDATE SET
		   request_count = EXCLUDED.request_count,
		   total_input_tokens = EXCLUDED.total_input_tokens,
		   total_output_tokens = EXCLUDED.total_output_tokens,
		   total_cost_cents = EXCLUDED.total_cost_cents,
		   cache_hits = EXCLUDED.cache_hits,
		   cache_misses = EXCLUDED.cache_misses`,
		dateStr)
	return err
}

func (p *Postgres) AggregateCacheStatsDaily(ctx context.Context, date time.Time) error {
	dateStr := date.Format("2006-01-02")
	_, err := p.DB.ExecContext(ctx,
		`INSERT INTO cache_stats_daily (date, total_requests, cache_hits, cache_misses, tokens_saved, cost_saved_cents)
		 SELECT $1::date,
		   COUNT(*),
		   COUNT(*) FILTER (WHERE cache_hit = TRUE),
		   COUNT(*) FILTER (WHERE cache_hit = FALSE),
		   COALESCE(SUM(total_tokens) FILTER (WHERE cache_hit = TRUE), 0),
		   COALESCE(SUM(total_cost_cents) FILTER (WHERE cache_hit = TRUE), 0)
		 FROM request_logs
		 WHERE created_at >= $1::date AND created_at < ($1::date + INTERVAL '1 day')
		 ON CONFLICT (date) DO UPDATE SET
		   total_requests = EXCLUDED.total_requests,
		   cache_hits = EXCLUDED.cache_hits,
		   cache_misses = EXCLUDED.cache_misses,
		   tokens_saved = EXCLUDED.tokens_saved,
		   cost_saved_cents = EXCLUDED.cost_saved_cents`,
		dateStr)
	return err
}
