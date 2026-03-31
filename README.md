# ThriftLLM

**Self-hosted LLM Proxy with Semantic Caching**

A single proxy endpoint for OpenAI, Anthropic, Gemini, Groq, Together, OpenRouter, and any OpenAI-compatible API — with automatic fallback, semantic caching, cost tracking, and a beautiful dashboard.

## Features

- **Single Proxy Endpoint** — Drop-in OpenAI-compatible `/v1/chat/completions` endpoint
- **Multi-Provider Support** — Route to OpenAI, Anthropic, Gemini, Groq, Together, OpenRouter, or custom providers
- **Automatic Fallback** — Configurable fallback chains when primary model/provider fails
- **Semantic Caching** — Reduce costs by caching similar requests using vector similarity search
- **Streaming Support** — Full SSE streaming support across all providers
- **Cost Tracking** — Per-model input/output cost tracking with daily aggregations
- **Rate Limiting** — Per-API-key sliding window rate limiting
- **Dashboard** — Real-time monitoring with charts for usage, costs, cache performance
- **API Key Management** — Create and manage proxy access keys from the dashboard

## Quick Start

### 1. Clone and configure

```bash
git clone https://github.com/thriftllm/thriftllm
cd thriftllm
cp .env.example .env
```

Edit `.env` and add your provider API keys:

```
OPENAI_API_KEY=sk-...
ANTHROPIC_API_KEY=sk-ant-...
GEMINI_API_KEY=AIza...
```

### 2. Start with Docker Compose

```bash
docker compose up -d
```

This starts 4 services:
- **PostgreSQL** (port 5432) — Data storage  
- **Redis Stack** (port 6379) — Caching & vector search
- **Backend** (port 8080) — Go API server
- **Frontend** (port 3000) — Next.js dashboard

### 3. Initial Setup

Open http://localhost:3000 — you'll be guided through creating your admin account.

### 4. Add Models

Go to **Dashboard → Models → Add Model** and configure your LLM providers:

| Provider | Model Name | API Key Env |
|----------|-----------|-------------|
| OpenAI | gpt-4o | OPENAI_API_KEY |
| Anthropic | claude-sonnet-4-20250514 | ANTHROPIC_API_KEY |
| Gemini | gemini-2.0-flash | GEMINI_API_KEY |

### 5. Create an API Key

Go to **Dashboard → API Keys → Create Key** and copy your key.

### 6. Use the Proxy

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer thr_your_key_here" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

Or with the OpenAI Python SDK:

```python
from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:8080/v1",
    api_key="thr_your_key_here"
)

response = client.chat.completions.create(
    model="gpt-4o",
    messages=[{"role": "user", "content": "Hello!"}]
)
print(response.choices[0].message.content)
```

## Architecture

```
┌─────────────────────────────────────────────────┐
│                  Docker Compose                  │
│                                                  │
│  ┌──────────┐  ┌──────────┐  ┌───────────────┐  │
│  │ Frontend │  │ Backend  │  │   Providers   │  │
│  │ Next.js  │→ │   Go     │→ │ OpenAI/Claude │  │
│  │ :3000    │  │  :8080   │  │ Gemini/Groq   │  │
│  └──────────┘  └────┬─────┘  └───────────────┘  │
│                     │                             │
│              ┌──────┴──────┐                      │
│              │             │                      │
│         ┌────▼────┐  ┌────▼────┐                  │
│         │PostgreSQL│  │  Redis  │                  │
│         │  :5432   │  │  :6379  │                  │
│         └─────────┘  └─────────┘                  │
└─────────────────────────────────────────────────┘
```

## Semantic Caching

ThriftLLM converts each request into a 64-dimensional embedding and performs KNN vector similarity search in Redis. If a cached entry has ≥95% cosine similarity, the cached response is returned instantly — saving both time and money.

**Cache behavior:**
- Requests with `temperature > 0.5` bypass the cache
- Streaming requests bypass the cache
- Cache entries expire after 24 hours (configurable)
- Flush all cache from the dashboard or API

## Fallback Chains

Configure models with tags (e.g., `fast`, `coding`, `cheap`) and priorities. When a request comes in:

1. ThriftLLM looks up the requested model
2. If it fails (429, 5xx), it falls back to the next model in priority order
3. Up to 3 fallback attempts per request
4. Use the `X-Thrift-Tags` header to route by tags

Response headers show what happened:
- `X-Thrift-Cache: hit/miss` — Cache status
- `X-Thrift-Provider` — Which provider was used
- `X-Thrift-Model` — Which model was used
- `X-Thrift-Fallback-Depth` — Number of fallback attempts

## Development

### Backend (Go)

```bash
cd backend
go run ./cmd/server
```

### Frontend (Next.js)

```bash
cd frontend
npm install
npm run dev
```

### Environment Variables

See `.env.example` for all configuration options.

## Tech Stack

- **Backend:** Go 1.26, Chi router, sqlx, go-redis
- **Frontend:** Next.js 16, React 19, shadcn/ui, Recharts, Tailwind CSS v4
- **Database:** PostgreSQL 16 with partitioned tables
- **Cache:** Redis Stack with RediSearch vector search
- **Deployment:** Docker Compose

## License

MIT
