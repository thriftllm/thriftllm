-- ThriftLLM Database Schema
-- PostgreSQL 16

-- Enable UUID generation
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================
-- Setup Status (single-row flag)
-- ============================================
CREATE TABLE setup_status (
    id          BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (id = TRUE), -- ensures single row
    is_complete BOOLEAN NOT NULL DEFAULT FALSE,
    completed_at TIMESTAMPTZ
);
INSERT INTO setup_status (is_complete) VALUES (FALSE);

-- ============================================
-- Admin Users
-- ============================================
CREATE TABLE admin_users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          VARCHAR(255) NOT NULL,
    email         VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_login_at TIMESTAMPTZ
);

-- ============================================
-- Provider Enum
-- ============================================
CREATE TYPE provider_type AS ENUM ('openai', 'anthropic', 'gemini', 'groq', 'together', 'openrouter', 'custom_openai');

-- ============================================
-- Model Configs
-- ============================================
CREATE TABLE model_configs (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider          provider_type NOT NULL,
    provider_model    VARCHAR(200) NOT NULL,
    display_name      VARCHAR(200) NOT NULL,
    api_key_env_name  VARCHAR(100) NOT NULL,
    api_base_url      VARCHAR(500),  -- only needed for custom_openai
    input_cost_per_1k  DECIMAL(10,6) NOT NULL DEFAULT 0,
    output_cost_per_1k DECIMAL(10,6) NOT NULL DEFAULT 0,
    tags              TEXT[] NOT NULL DEFAULT '{}',
    is_active         BOOLEAN NOT NULL DEFAULT TRUE,
    priority          INT NOT NULL DEFAULT 0,  -- lower = higher priority for fallback
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_model_configs_provider ON model_configs(provider);
CREATE INDEX idx_model_configs_tags ON model_configs USING GIN(tags);
CREATE INDEX idx_model_configs_active ON model_configs(is_active) WHERE is_active = TRUE;

-- ============================================
-- Fallback Chains
-- ============================================
CREATE TABLE fallback_chains (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name             VARCHAR(100) NOT NULL UNIQUE,
    model_config_ids UUID[] NOT NULL,
    tag_selector     VARCHAR(50),
    is_default       BOOLEAN NOT NULL DEFAULT FALSE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Only one default chain
CREATE UNIQUE INDEX idx_fallback_chains_default ON fallback_chains(is_default) WHERE is_default = TRUE;

-- ============================================
-- API Keys
-- ============================================
CREATE TABLE api_keys (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          VARCHAR(100) NOT NULL,
    key_hash      VARCHAR(255) NOT NULL UNIQUE,
    key_prefix    VARCHAR(12) NOT NULL,
    is_active     BOOLEAN NOT NULL DEFAULT TRUE,
    rate_limit_rpm INT NOT NULL DEFAULT 60,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at  TIMESTAMPTZ,
    expires_at    TIMESTAMPTZ
);

CREATE INDEX idx_api_keys_hash ON api_keys(key_hash);
CREATE INDEX idx_api_keys_active ON api_keys(is_active) WHERE is_active = TRUE;

-- ============================================
-- Request Logs (partitioned by month)
-- ============================================
CREATE TABLE request_logs (
    id              UUID NOT NULL DEFAULT gen_random_uuid(),
    api_key_id      UUID,
    requested_model VARCHAR(200),
    actual_provider VARCHAR(50),
    actual_model    VARCHAR(200),
    input_tokens    INT NOT NULL DEFAULT 0,
    output_tokens   INT NOT NULL DEFAULT 0,
    total_tokens    INT NOT NULL DEFAULT 0,
    input_cost_cents  DECIMAL(10,6) NOT NULL DEFAULT 0,
    output_cost_cents DECIMAL(10,6) NOT NULL DEFAULT 0,
    total_cost_cents  DECIMAL(10,6) NOT NULL DEFAULT 0,
    latency_ms      INT NOT NULL DEFAULT 0,
    ttfb_ms         INT NOT NULL DEFAULT 0,
    status_code     SMALLINT NOT NULL DEFAULT 200,
    error_message   TEXT,
    cache_hit       BOOLEAN NOT NULL DEFAULT FALSE,
    cache_similarity DECIMAL(5,4),
    fallback_depth  SMALLINT NOT NULL DEFAULT 0,
    is_streaming    BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Create partitions for current and next 6 months
DO $$
DECLARE
    start_date DATE;
    end_date DATE;
    partition_name TEXT;
BEGIN
    FOR i IN 0..6 LOOP
        start_date := date_trunc('month', CURRENT_DATE) + (i || ' months')::interval;
        end_date := start_date + '1 month'::interval;
        partition_name := 'request_logs_' || to_char(start_date, 'YYYY_MM');
        EXECUTE format(
            'CREATE TABLE IF NOT EXISTS %I PARTITION OF request_logs FOR VALUES FROM (%L) TO (%L)',
            partition_name, start_date, end_date
        );
    END LOOP;
END $$;

CREATE INDEX idx_request_logs_created ON request_logs(created_at DESC);
CREATE INDEX idx_request_logs_api_key ON request_logs(api_key_id, created_at DESC);
CREATE INDEX idx_request_logs_cache ON request_logs(cache_hit, created_at DESC);

-- ============================================
-- Usage Daily (aggregated)
-- ============================================
CREATE TABLE usage_daily (
    id                  SERIAL PRIMARY KEY,
    date                DATE NOT NULL,
    api_key_id          UUID,
    model_alias         VARCHAR(200),
    provider            VARCHAR(50),
    request_count       INT NOT NULL DEFAULT 0,
    total_input_tokens  BIGINT NOT NULL DEFAULT 0,
    total_output_tokens BIGINT NOT NULL DEFAULT 0,
    total_cost_cents    DECIMAL(12,6) NOT NULL DEFAULT 0,
    cache_hits          INT NOT NULL DEFAULT 0,
    cache_misses        INT NOT NULL DEFAULT 0,
    UNIQUE(date, api_key_id, model_alias)
);

CREATE INDEX idx_usage_daily_date ON usage_daily(date DESC);

-- ============================================
-- Cache Stats Daily (aggregated)
-- ============================================
CREATE TABLE cache_stats_daily (
    id               SERIAL PRIMARY KEY,
    date             DATE NOT NULL UNIQUE,
    total_requests   INT NOT NULL DEFAULT 0,
    cache_hits       INT NOT NULL DEFAULT 0,
    cache_misses     INT NOT NULL DEFAULT 0,
    tokens_saved     BIGINT NOT NULL DEFAULT 0,
    cost_saved_cents DECIMAL(12,6) NOT NULL DEFAULT 0
);

CREATE INDEX idx_cache_stats_daily_date ON cache_stats_daily(date DESC);

-- ============================================
-- Function to auto-create future partitions
-- ============================================
CREATE OR REPLACE FUNCTION create_request_logs_partition()
RETURNS void AS $$
DECLARE
    start_date DATE;
    end_date DATE;
    partition_name TEXT;
BEGIN
    FOR i IN 0..3 LOOP
        start_date := date_trunc('month', CURRENT_DATE) + (i || ' months')::interval;
        end_date := start_date + '1 month'::interval;
        partition_name := 'request_logs_' || to_char(start_date, 'YYYY_MM');
        EXECUTE format(
            'CREATE TABLE IF NOT EXISTS %I PARTITION OF request_logs FOR VALUES FROM (%L) TO (%L)',
            partition_name, start_date, end_date
        );
    END LOOP;
END;
$$ LANGUAGE plpgsql;
