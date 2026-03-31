package cache

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/thriftllm/backend/internal/model"
	"github.com/thriftllm/backend/internal/store"
)

const (
	cachePrefix     = "sc:"
	indexName       = "idx:sem_cache"
	cacheTTL        = 24 * time.Hour
	embeddingDim    = 64 // simplified hash-based embedding for portability
)

// SemanticCache provides semantic caching for LLM responses using Redis
type SemanticCache struct {
	redis     *store.Redis
	threshold float64
}

func NewSemanticCache(rdb *store.Redis, threshold float64) (*SemanticCache, error) {
	if rdb == nil {
		return nil, fmt.Errorf("redis client is nil")
	}
	return &SemanticCache{
		redis:     rdb,
		threshold: threshold,
	}, nil
}

// InitIndex creates the RediSearch index for vector similarity search
func (sc *SemanticCache) InitIndex(ctx context.Context) error {
	// Try to create the index; if it already exists, that's fine
	_, err := sc.redis.Client.Do(ctx,
		"FT.CREATE", indexName,
		"ON", "HASH",
		"PREFIX", "1", cachePrefix,
		"SCHEMA",
		"embedding", "VECTOR", "FLAT", "6",
		"TYPE", "FLOAT32",
		"DIM", fmt.Sprintf("%d", embeddingDim),
		"DISTANCE_METRIC", "COSINE",
		"model", "TAG",
		"temperature", "NUMERIC",
		"response", "TEXT",
		"created_at", "NUMERIC", "SORTABLE",
	).Result()

	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "Index already exists") {
			return nil
		}
		return fmt.Errorf("failed to create search index: %w", err)
	}

	log.Println("Redis search index created successfully")
	return nil
}

// Lookup searches the cache for a similar request
func (sc *SemanticCache) Lookup(ctx context.Context, req *model.ChatCompletionRequest, modelName string) (*model.ChatCompletionResponse, *float64, error) {
	embedding := sc.computeEmbedding(req)
	embBytes := float32SliceToBytes(embedding)
	embStr := string(embBytes) // go-redis needs string for binary PARAMS

	// KNN search with model filter
	result, err := sc.redis.Client.Do(ctx,
		"FT.SEARCH", indexName,
		fmt.Sprintf("(@model:{%s})=>[KNN 1 @embedding $vec AS score]", escapeTag(modelName)),
		"PARAMS", "2", "vec", embStr,
		"RETURN", "3", "response", "score", "model",
		"SORTBY", "score",
		"LIMIT", "0", "1",
		"DIALECT", "2",
	).Result()

	if err != nil {
		log.Printf("Cache lookup error: %v", err)
		return nil, nil, fmt.Errorf("cache lookup failed: %w", err)
	}

	// Parse the FT.SEARCH result
	results, ok := result.([]interface{})
	if !ok || len(results) < 2 {
		log.Printf("Cache lookup: no results or unexpected type: %T", result)
		return nil, nil, nil // no results
	}

	// First element is total count
	count, ok := results[0].(int64)
	if !ok || count == 0 {
		log.Printf("Cache lookup: count=0 or unexpected count type: %T", results[0])
		return nil, nil, nil
	}

	// Parse result fields
	if len(results) < 3 {
		return nil, nil, nil
	}

	fields, ok := results[2].([]interface{})
	if !ok {
		return nil, nil, nil
	}

	fieldMap := make(map[string]string)
	for i := 0; i < len(fields)-1; i += 2 {
		key, _ := fields[i].(string)
		val, _ := fields[i+1].(string)
		fieldMap[key] = val
	}

	// Check similarity score (cosine distance: 0 = identical, 2 = opposite)
	scoreStr, ok := fieldMap["score"]
	if !ok {
		log.Printf("Cache lookup: no score field in result")
		return nil, nil, nil
	}

	var score float64
	fmt.Sscanf(scoreStr, "%f", &score)

	// Convert distance to similarity: similarity = 1 - distance
	similarity := 1.0 - score
	log.Printf("Cache lookup: distance=%.6f similarity=%.6f threshold=%.6f", score, similarity, sc.threshold)

	if similarity < sc.threshold {
		return nil, nil, nil // not similar enough
	}

	// Parse the cached response
	responseStr, ok := fieldMap["response"]
	if !ok {
		return nil, nil, nil
	}

	var resp model.ChatCompletionResponse
	if err := json.Unmarshal([]byte(responseStr), &resp); err != nil {
		return nil, nil, nil
	}

	return &resp, &similarity, nil
}

// Store saves a response in the cache
func (sc *SemanticCache) Store(ctx context.Context, req *model.ChatCompletionRequest, modelName string, resp *model.ChatCompletionResponse) error {
	embedding := sc.computeEmbedding(req)
	embBytes := float32SliceToBytes(embedding)

	respJSON, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	key := cachePrefix + uuid.New().String()
	temp := 0.0
	if req.Temperature != nil {
		temp = *req.Temperature
	}

	pipe := sc.redis.Client.Pipeline()
	pipe.HSet(ctx, key,
		"embedding", embBytes,
		"model", modelName,
		"temperature", temp,
		"response", string(respJSON),
		"created_at", time.Now().Unix(),
	)
	pipe.Expire(ctx, key, cacheTTL)

	_, err = pipe.Exec(ctx)
	return err
}

// FlushAll removes all cached entries
func (sc *SemanticCache) FlushAll(ctx context.Context) error {
	// Scan and delete all cache entries
	var cursor uint64
	for {
		keys, nextCursor, err := sc.redis.Client.Scan(ctx, cursor, cachePrefix+"*", 100).Result()
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			sc.redis.Client.Del(ctx, keys...)
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return nil
}

// EntryCount returns the number of cached entries
func (sc *SemanticCache) EntryCount(ctx context.Context) (int64, error) {
	var cursor uint64
	var count int64
	for {
		keys, nextCursor, err := sc.redis.Client.Scan(ctx, cursor, cachePrefix+"*", 100).Result()
		if err != nil {
			return 0, err
		}
		count += int64(len(keys))
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return count, nil
}

// computeEmbedding generates a hash-based embedding vector for a request.
// This is a simplified approach that uses feature hashing to create a fixed-size vector.
// For production, replace with ONNX all-MiniLM-L6-v2 for true semantic similarity.
func (sc *SemanticCache) computeEmbedding(req *model.ChatCompletionRequest) []float32 {
	// Build a canonical string from the request
	var parts []string
	for _, msg := range req.Messages {
		parts = append(parts, msg.Role+":"+msg.ContentString())
	}
	text := strings.Join(parts, "|")
	text = strings.ToLower(strings.TrimSpace(text))

	// Feature hashing: hash n-grams into a fixed-size vector
	embedding := make([]float32, embeddingDim)

	// Character trigrams
	for i := 0; i <= len(text)-3; i++ {
		trigram := text[i : i+3]
		h := sha256.Sum256([]byte(trigram))
		idx := binary.BigEndian.Uint32(h[:4]) % uint32(embeddingDim)
		sign := float32(1.0)
		if h[4]%2 == 0 {
			sign = -1.0
		}
		embedding[idx] += sign
	}

	// Word-level hashing
	words := strings.Fields(text)
	for _, word := range words {
		h := sha256.Sum256([]byte("word:" + word))
		idx := binary.BigEndian.Uint32(h[:4]) % uint32(embeddingDim)
		sign := float32(1.0)
		if h[4]%2 == 0 {
			sign = -1.0
		}
		embedding[idx] += sign * 2.0 // weight words higher
	}

	// Also hash the full content for exact-match boost
	fullHash := sha256.Sum256([]byte(text))
	hashHex := hex.EncodeToString(fullHash[:])
	for i := 0; i < embeddingDim && i*2+2 <= len(hashHex); i++ {
		val := float32(hashHex[i*2]) + float32(hashHex[i*2+1])
		embedding[i] += val * 0.01
	}

	// L2 normalize
	var norm float32
	for _, v := range embedding {
		norm += v * v
	}
	norm = float32(math.Sqrt(float64(norm)))
	if norm > 0 {
		for i := range embedding {
			embedding[i] /= norm
		}
	}

	return embedding
}

func float32SliceToBytes(s []float32) []byte {
	buf := make([]byte, len(s)*4)
	for i, v := range s {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(v))
	}
	return buf
}

func escapeTag(s string) string {
	// Escape special characters for RediSearch tag queries
	replacer := strings.NewReplacer(
		"-", "\\-",
		".", "\\.",
		":", "\\:",
		"/", "\\/",
	)
	return replacer.Replace(s)
}

// RateLimiter provides sliding window rate limiting via Redis
type RateLimiter struct {
	redis *store.Redis
}

func NewRateLimiter(rdb *store.Redis) *RateLimiter {
	return &RateLimiter{redis: rdb}
}

func (rl *RateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	now := time.Now()
	windowStart := now.Add(-window)

	pipe := rl.redis.Client.Pipeline()
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart.UnixMilli()))
	countCmd := pipe.ZCard(ctx, key)
	pipe.ZAdd(ctx, key, redis.Z{
		Score:  float64(now.UnixMilli()),
		Member: fmt.Sprintf("%d", now.UnixNano()),
	})
	pipe.Expire(ctx, key, window*2)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return true, err // allow on error
	}

	return countCmd.Val() < int64(limit), nil
}
