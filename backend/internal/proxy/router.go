package proxy

import (
	"context"
	"database/sql"
	"strings"

	"github.com/thriftllm/backend/internal/model"
	"github.com/thriftllm/backend/internal/store"
)

// Router resolves which models to try for a given request
type Router struct {
	DB *store.Postgres
}

func NewRouter(db *store.Postgres) *Router {
	return &Router{DB: db}
}

// ResolveChain returns an ordered list of models to try for the given request.
// Resolution order:
//  1. If a specific model is requested, use it as primary.
//  2. Look for a fallback chain matching request tags.
//  3. Look for the default fallback chain.
//  4. If a fallback chain is found, resolve its model_config_ids to ordered models.
//  5. Append any remaining active models not in the chain as a last resort.
//  6. If no chain found, fall back to tag matching then all active models by priority.
func (r *Router) ResolveChain(ctx context.Context, requestedModel string, tags []string) ([]model.ModelConfig, error) {
	var primary *model.ModelConfig

	// 1. If a specific model is requested, find it
	if requestedModel != "" {
		m, err := r.DB.GetActiveModelByName(ctx, requestedModel)
		if err != nil && err != sql.ErrNoRows {
			return nil, err
		}
		primary = m
	}

	// 2. Try to find a fallback chain by tag
	var chain *model.FallbackChain
	for _, tag := range tags {
		fc, err := r.DB.GetFallbackChainByTag(ctx, tag)
		if err == nil && fc != nil {
			chain = fc
			break
		}
	}

	// 3. If no tag-matched chain, try default chain
	if chain == nil {
		fc, err := r.DB.GetDefaultFallbackChain(ctx)
		if err == nil && fc != nil {
			chain = fc
		}
	}

	// 4. If we have a chain, resolve model_config_ids to ordered models
	if chain != nil {
		var result []model.ModelConfig
		// Put the primary model first if it was requested and exists
		if primary != nil {
			result = append(result, *primary)
		}
		// Then add models from the chain in order
		for _, mcID := range chain.ModelConfigIDs {
			if primary != nil && mcID == primary.ID {
				continue // already added as primary
			}
			mc, err := r.DB.GetModelConfig(ctx, mcID)
			if err != nil || !mc.IsActive {
				continue
			}
			result = append(result, *mc)
		}
		// If we got models from the chain, return them
		if len(result) > 0 {
			return result, nil
		}
	}

	// 5. No chain matched — legacy fallback logic
	if primary != nil {
		result := []model.ModelConfig{*primary}
		// Find fallback models with same tags, ordered by priority
		fallbacks, _ := r.DB.GetActiveModelsByTags(ctx, primary.Tags)
		for _, fb := range fallbacks {
			if fb.ID != primary.ID {
				result = append(result, fb)
			}
		}
		// If no tag-based fallbacks, get all active models as fallback
		if len(result) == 1 {
			all, _ := r.DB.ListModelConfigs(ctx)
			for _, m := range all {
				if m.ID != primary.ID && m.IsActive {
					result = append(result, m)
				}
			}
		}
		return result, nil
	}

	// 6. If tags are specified, find models matching those tags
	if len(tags) > 0 {
		models, err := r.DB.GetActiveModelsByTags(ctx, tags)
		if err != nil {
			return nil, err
		}
		if len(models) > 0 {
			return models, nil
		}
	}

	// 7. Return all active models sorted by priority
	all, err := r.DB.ListModelConfigs(ctx)
	if err != nil {
		return nil, err
	}
	var active []model.ModelConfig
	for _, m := range all {
		if m.IsActive {
			active = append(active, m)
		}
	}
	return active, nil
}

// ParseTags extracts tags from the x-thrift-tags header
func ParseTags(header string) []string {
	if header == "" {
		return nil
	}
	var tags []string
	for _, t := range strings.Split(header, ",") {
		t = strings.TrimSpace(t)
		if t != "" {
			tags = append(tags, t)
		}
	}
	return tags
}
