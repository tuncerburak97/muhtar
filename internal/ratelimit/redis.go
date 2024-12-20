package ratelimit

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog/log"
)

// RedisStore implements Store interface using Redis
type RedisStore struct {
	client *redis.Client
}

// NewRedisStore creates a new Redis-based store
func NewRedisStore(host string, port int, password string, db int, timeout time.Duration) (*RedisStore, error) {
	log.Info().
		Str("host", host).
		Int("port", port).
		Int("db", db).
		Dur("timeout", timeout).
		Msg("Attempting to connect to Redis")

	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", host, port),
		Password:     password,
		DB:           db,
		DialTimeout:  timeout,
		ReadTimeout:  timeout,
		WriteTimeout: timeout,
		Username:     "",
		MaxRetries:   3,
		PoolSize:     10,
		MinIdleConns: 5,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		log.Error().Err(err).Msg("Failed to connect to Redis")
		return nil, fmt.Errorf("failed to connect to Redis: %v", err)
	}

	log.Info().Msg("Successfully connected to Redis")
	return &RedisStore{client: client}, nil
}

func (s *RedisStore) Get(ctx context.Context, key string) (int, time.Time, error) {
	log.Debug().
		Str("key", key).
		Str("operation", "Get").
		Msg("Fetching rate limit data from Redis")

	pipe := s.client.Pipeline()
	countCmd := pipe.Get(ctx, key)
	ttlCmd := pipe.TTL(ctx, key)

	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		log.Error().
			Err(err).
			Str("key", key).
			Msg("Failed to get rate limit data from Redis")
		return 0, time.Now(), err
	}

	count := 0
	if val, err := countCmd.Result(); err == nil {
		count, _ = strconv.Atoi(val)
	}

	ttl := ttlCmd.Val()
	resetTime := time.Now().Add(ttl)

	log.Debug().
		Str("key", key).
		Int("count", count).
		Dur("ttl", ttl).
		Time("resetTime", resetTime).
		Msg("Retrieved rate limit data from Redis")

	return count, resetTime, nil
}

func (s *RedisStore) Increment(ctx context.Context, key string, resetTime time.Time) (int, error) {
	log.Debug().
		Str("key", key).
		Time("resetTime", resetTime).
		Str("operation", "Increment").
		Msg("Incrementing rate limit counter in Redis")

	pipe := s.client.Pipeline()

	// Increment the counter
	incr := pipe.Incr(ctx, key)

	// Set expiration if key is new
	ttl := time.Until(resetTime)
	pipe.PExpire(ctx, key, ttl)

	_, err := pipe.Exec(ctx)
	if err != nil {
		log.Error().
			Err(err).
			Str("key", key).
			Msg("Failed to increment rate limit counter in Redis")
		return 0, err
	}

	newCount := int(incr.Val())
	log.Debug().
		Str("key", key).
		Int("newCount", newCount).
		Dur("ttl", ttl).
		Msg("Successfully incremented rate limit counter")

	return newCount, nil
}

func (s *RedisStore) Reset(ctx context.Context, key string) error {
	log.Debug().
		Str("key", key).
		Str("operation", "Reset").
		Msg("Resetting rate limit counter in Redis")

	err := s.client.Del(ctx, key).Err()
	if err != nil {
		log.Error().
			Err(err).
			Str("key", key).
			Msg("Failed to reset rate limit counter in Redis")
		return err
	}

	log.Debug().
		Str("key", key).
		Msg("Successfully reset rate limit counter")

	return nil
}

func (s *RedisStore) Close() error {
	log.Info().Msg("Closing Redis connection")
	return s.client.Close()
}

// Helper methods for Redis key management
func (s *RedisStore) buildKey(parts ...string) string {
	key := fmt.Sprintf("ratelimit:%s", parts)
	log.Trace().
		Strs("parts", parts).
		Str("key", key).
		Msg("Built Redis key")
	return key
}

func (s *RedisStore) buildWindowKey(key string, windowStart time.Time) string {
	windowKey := fmt.Sprintf("%s:%d", key, windowStart.Unix())
	log.Trace().
		Str("baseKey", key).
		Time("windowStart", windowStart).
		Str("windowKey", windowKey).
		Msg("Built window key")
	return windowKey
}

// Sliding window implementation
func (s *RedisStore) slidingWindowIncrement(ctx context.Context, key string, window time.Duration, limit int) (int, error) {
	log.Debug().
		Str("key", key).
		Dur("window", window).
		Int("limit", limit).
		Str("operation", "SlidingWindowIncrement").
		Msg("Processing sliding window increment")

	now := time.Now()
	windowStart := now.Add(-window)

	// Create a transaction
	pipe := s.client.Pipeline()

	// Add current timestamp to sorted set with score as timestamp
	pipe.ZAdd(ctx, key, &redis.Z{
		Score:  float64(now.Unix()),
		Member: now.Unix(),
	})

	// Remove old entries outside the window
	pipe.ZRemRangeByScore(ctx, key, "0", strconv.FormatInt(windowStart.Unix(), 10))

	// Count remaining entries in the window
	count := pipe.ZCard(ctx, key)

	// Set key expiration
	pipe.Expire(ctx, key, window)

	// Execute transaction
	_, err := pipe.Exec(ctx)
	if err != nil {
		log.Error().
			Err(err).
			Str("key", key).
			Msg("Failed to process sliding window increment")
		return 0, err
	}

	result := int(count.Val())
	log.Debug().
		Str("key", key).
		Int("count", result).
		Time("windowStart", windowStart).
		Time("now", now).
		Msg("Successfully processed sliding window increment")

	return result, nil
}

// Token bucket implementation
func (s *RedisStore) tokenBucketTake(ctx context.Context, key string, capacity int, fillRate float64, fillInterval time.Duration) (bool, error) {
	log.Debug().
		Str("key", key).
		Int("capacity", capacity).
		Float64("fillRate", fillRate).
		Dur("fillInterval", fillInterval).
		Str("operation", "TokenBucketTake").
		Msg("Processing token bucket take")

	script := `
		local key = KEYS[1]
		local capacity = tonumber(ARGV[1])
		local fillRate = tonumber(ARGV[2])
		local fillInterval = tonumber(ARGV[3])
		local now = tonumber(ARGV[4])
		
		-- Get the current bucket state
		local bucket = redis.call('HMGET', key, 'tokens', 'lastFill')
		local tokens = tonumber(bucket[1] or capacity)
		local lastFill = tonumber(bucket[2] or now)
		
		-- Calculate token refill
		local elapsed = now - lastFill
		local refill = math.floor(elapsed / fillInterval * fillRate)
		tokens = math.min(capacity, tokens + refill)
		
		-- Try to take a token
		if tokens > 0 then
			tokens = tokens - 1
			redis.call('HMSET', key, 'tokens', tokens, 'lastFill', now)
			redis.call('EXPIRE', key, fillInterval * 2)
			return 1
		end
		
		return 0
	`

	now := time.Now().Unix()
	result, err := s.client.Eval(ctx, script, []string{key},
		capacity,
		fillRate,
		int64(fillInterval.Seconds()),
		now,
	).Result()

	if err != nil {
		log.Error().
			Err(err).
			Str("key", key).
			Msg("Failed to process token bucket take")
		return false, err
	}

	success := result.(int64) == 1
	log.Debug().
		Str("key", key).
		Bool("success", success).
		Int64("now", now).
		Msg("Successfully processed token bucket take")

	return success, nil
}
