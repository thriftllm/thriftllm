package server

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/thriftllm/backend/internal/cache"
	"github.com/thriftllm/backend/internal/config"
	"github.com/thriftllm/backend/internal/handler"
	"github.com/thriftllm/backend/internal/middleware"
	"github.com/thriftllm/backend/internal/proxy"
	"github.com/thriftllm/backend/internal/store"
)

type Server struct {
	cfg   *config.Config
	db    *store.Postgres
	redis *store.Redis
	cache *cache.SemanticCache
}

func New(cfg *config.Config, db *store.Postgres, redis *store.Redis, semanticCache *cache.SemanticCache) *Server {
	return &Server{
		cfg:   cfg,
		db:    db,
		redis: redis,
		cache: semanticCache,
	}
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(middleware.Logger)
	r.Use(chimw.Recoverer)
	// CORS configuration — use THRIFT_CORS_ORIGINS env var in production
	corsOrigins := []string{"http://localhost:3000", "http://localhost:8080"}
	if s.cfg.CORSOrigins != "" {
		corsOrigins = strings.Split(s.cfg.CORSOrigins, ",")
		for i := range corsOrigins {
			corsOrigins[i] = strings.TrimSpace(corsOrigins[i])
		}
	}
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   corsOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Thrift-Tags"},
		ExposedHeaders:   []string{"X-Thrift-Cache", "X-Thrift-Provider", "X-Thrift-Model", "X-Thrift-Fallback-Depth", "X-RateLimit-Limit", "X-RateLimit-Remaining"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// ---- Setup & Auth (no auth required) ----
	setupHandler := &handler.SetupHandler{DB: s.db, JWTSecret: s.cfg.JWTSecret, SecureCookies: s.cfg.SecureCookies}
	authHandler := &handler.AuthHandler{DB: s.db, JWTSecret: s.cfg.JWTSecret, SecureCookies: s.cfg.SecureCookies}

	r.Get("/api/setup/status", setupHandler.Status)
	r.Post("/api/setup", setupHandler.Setup)
	r.Post("/api/auth/login", authHandler.Login)
	r.Post("/api/auth/logout", authHandler.Logout)

	// ---- Dashboard API (JWT auth) ----
	r.Group(func(r chi.Router) {
		r.Use(middleware.JWTAuth(s.cfg.JWTSecret))

		// Auth
		r.Get("/api/auth/me", authHandler.Me)
		r.Put("/api/settings/profile", authHandler.UpdateProfile)
		r.Put("/api/settings/password", authHandler.ChangePassword)

		// Models
		modelsHandler := &handler.ModelsHandler{DB: s.db}
		r.Get("/api/models", modelsHandler.List)
		r.Post("/api/models", modelsHandler.Create)
		r.Put("/api/models/{id}", modelsHandler.Update)
		r.Delete("/api/models/{id}", modelsHandler.Delete)
		r.Patch("/api/models/{id}", modelsHandler.Toggle)

		// API Keys
		keysHandler := &handler.APIKeysHandler{DB: s.db}
		r.Get("/api/keys", keysHandler.List)
		r.Post("/api/keys", keysHandler.Create)
		r.Delete("/api/keys/{id}", keysHandler.Delete)
		r.Patch("/api/keys/{id}", keysHandler.Toggle)

		// Dashboard
		dashHandler := &handler.DashboardHandler{DB: s.db}
		r.Get("/api/dashboard/overview", dashHandler.Overview)
		r.Get("/api/dashboard/usage", dashHandler.Usage)
		r.Get("/api/dashboard/models", dashHandler.ModelBreakdown)
		r.Get("/api/requests", dashHandler.Requests)

		// Cache
		cacheHandler := &handler.CacheHandler{DB: s.db, Redis: s.redis, Cache: s.cache}
		r.Get("/api/cache/stats", cacheHandler.Stats)
		r.Post("/api/cache/flush", cacheHandler.Flush)

		// Fallback Chains
		chainsHandler := &handler.FallbackChainsHandler{DB: s.db}
		r.Get("/api/chains", chainsHandler.List)
		r.Post("/api/chains", chainsHandler.Create)
		r.Put("/api/chains/{id}", chainsHandler.Update)
		r.Delete("/api/chains/{id}", chainsHandler.Delete)
	})

	// ---- Proxy API (API Key auth) ----
	proxyRouter := proxy.NewRouter(s.db)
	proxyExecutor := proxy.NewExecutor()
	proxyHandler := &handler.ProxyHandler{
		DB:       s.db,
		Router:   proxyRouter,
		Executor: proxyExecutor,
		Cache:    s.cache,
	}
	modelsHandler := &handler.ModelsHandler{DB: s.db}

	r.Group(func(r chi.Router) {
		r.Use(middleware.APIKeyAuth(s.db))
		r.Use(middleware.RateLimit(s.redis))

		r.Post("/v1/chat/completions", proxyHandler.ChatCompletions)
		r.Get("/v1/models", modelsHandler.ListOpenAI)
	})

	return r
}
