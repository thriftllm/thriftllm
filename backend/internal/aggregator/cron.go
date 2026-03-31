package aggregator

import (
	"context"
	"log"
	"time"

	"github.com/thriftllm/backend/internal/store"
)

// Aggregator runs periodic aggregation of request logs
type Aggregator struct {
	DB *store.Postgres
}

func New(db *store.Postgres) *Aggregator {
	return &Aggregator{DB: db}
}

// Start begins the aggregation loop (runs every hour)
func (a *Aggregator) Start(ctx context.Context) {
	// Run immediately for yesterday and today
	a.runAggregation(ctx)

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.runAggregation(ctx)
		}
	}
}

func (a *Aggregator) runAggregation(ctx context.Context) {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	yesterday := today.AddDate(0, 0, -1)

	for _, date := range []time.Time{yesterday, today} {
		if err := a.DB.AggregateUsageDaily(ctx, date); err != nil {
			log.Printf("Failed to aggregate usage for %s: %v", date.Format("2006-01-02"), err)
		}
		if err := a.DB.AggregateCacheStatsDaily(ctx, date); err != nil {
			log.Printf("Failed to aggregate cache stats for %s: %v", date.Format("2006-01-02"), err)
		}
	}

	// Also ensure future partitions exist
	_, err := a.DB.DB.ExecContext(ctx, "SELECT create_request_logs_partition()")
	if err != nil {
		log.Printf("Failed to create future partitions: %v", err)
	}
}
