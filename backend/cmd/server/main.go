package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/thriftllm/backend/internal/aggregator"
	"github.com/thriftllm/backend/internal/cache"
	"github.com/thriftllm/backend/internal/config"
	"github.com/thriftllm/backend/internal/server"
	"github.com/thriftllm/backend/internal/store"
)

func main() {
	cfg := config.Load()

	// Connect to PostgreSQL
	db, err := store.NewPostgres(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	// Connect to Redis
	rdb, err := store.NewRedis(cfg.RedisURL)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer rdb.Close()

	// Initialize semantic cache
	semanticCache, err := cache.NewSemanticCache(rdb, cfg.CacheThreshold)
	if err != nil {
		log.Printf("Warning: Semantic cache initialization failed (will operate without cache): %v", err)
		semanticCache = nil
	}

	// Initialize Redis search index for semantic cache
	if semanticCache != nil {
		if err := semanticCache.InitIndex(context.Background()); err != nil {
			log.Printf("Warning: Failed to create Redis search index: %v", err)
		}
	}

	// Start background aggregator
	aggCtx, aggCancel := context.WithCancel(context.Background())
	defer aggCancel()
	agg := aggregator.New(db)
	go agg.Start(aggCtx)

	// Build HTTP server
	srv := server.New(cfg, db, rdb, semanticCache)
	httpServer := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      srv.Router(),
		ReadTimeout:  120 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("ThriftLLM backend starting on :%s", cfg.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	<-done
	log.Println("Shutting down...")

	// Cancel aggregator
	aggCancel()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}
	log.Println("Server stopped")
}
