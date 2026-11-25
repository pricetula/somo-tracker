package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Client wraps go-redis client.
type Client struct {
	*redis.Client
}

// New creates a Redis client from a URL.
func New(ctx context.Context, url string) (*Client, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("parse redis URL: %w", err)
	}
	client := redis.NewClient(opts)
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("ping redis: %w", err)
	}
	return &Client{client}, nil
}

// Close releases the client.
func (c *Client) Close() error {
	return c.Client.Close()
}

// Increment increments a counter with a TTL, returning the new value.
func (c *Client) Increment(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	pipe := c.Pipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, fmt.Errorf("increment counter: %w", err)
	}
	return incr.Val(), nil
}

// Get returns the current value of a counter.
func (c *Client) Get(ctx context.Context, key string) (int64, error) {
	val, err := c.Client.Get(ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("get counter: %w", err)
	}
	return val, nil
}

// ============ Sorted Set Operations (for sliding window rate limiting) ============

// ZAdd adds a member with score to a sorted set.
func (c *Client) ZAdd(ctx context.Context, key, member string, score float64) error {
	return c.Client.ZAdd(ctx, key, redis.Z{Score: score, Member: member}).Err()
}

// ZRemRangeByScore removes members in a sorted set within the score range.
func (c *Client) ZRemRangeByScore(ctx context.Context, key, min, max string) (int64, error) {
	return c.Client.ZRemRangeByScore(ctx, key, min, max).Result()
}

// ZCard returns the cardinality of a sorted set.
func (c *Client) ZCard(ctx context.Context, key string) (int64, error) {
	return c.Client.ZCard(ctx, key).Result()
}

// ZRange returns members in a sorted set by index range.
func (c *Client) ZRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return c.Client.ZRange(ctx, key, start, stop).Result()
}

// ============ Lua Script Helpers ============

// EvalScript executes a Lua script directly.
func (c *Client) EvalScript(ctx context.Context, script string, keys []string, args ...interface{}) (interface{}, error) {
	return c.Client.Eval(ctx, script, keys, args...).Result()
}

// ScriptLoad loads a Lua script into Redis and returns its SHA.
func (c *Client) ScriptLoad(ctx context.Context, script string) (string, error) {
	return c.Client.ScriptLoad(ctx, script).Result()
}

// ScriptEvalSHA executes a loaded Lua script by its SHA.
func (c *Client) ScriptEvalSHA(ctx context.Context, sha string, keys []string, args ...interface{}) (interface{}, error) {
	return c.Client.EvalSha(ctx, sha, keys, args...).Result()
}

// PTTL returns the remaining TTL of a key in milliseconds.
func (c *Client) PTTL(ctx context.Context, key string) (int64, error) {
	ttl, err := c.Client.PTTL(ctx, key).Result()
	return int64(ttl.Milliseconds()), err
}

// TIME returns the current Redis server time as Unix milliseconds.
func (c *Client) TIME(ctx context.Context) (int64, error) {
	res, err := c.Client.Time(ctx).Result()
	if err != nil {
		return 0, err
	}
	return res.UnixMilli(), nil
}

// SetNX sets a key only if it doesn't exist, with TTL.
// Returns true if key was set, false if key already exists.
func (c *Client) SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
	return c.Client.SetNX(ctx, key, value, ttl).Result()
}

// Del deletes one or more keys.
func (c *Client) Del(ctx context.Context, keys ...string) (int64, error) {
	return c.Client.Del(ctx, keys...).Result()
}

// Exists checks if one or more keys exist.
func (c *Client) Exists(ctx context.Context, keys ...string) (int64, error) {
	return c.Client.Exists(ctx, keys...).Result()
}

// Expire sets a TTL on a key.
func (c *Client) Expire(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	return c.Client.Expire(ctx, key, ttl).Result()
}

// Incr increments a key and returns the new value.
func (c *Client) Incr(ctx context.Context, key string) (int64, error) {
	return c.Client.Incr(ctx, key).Result()
}

// GetString gets a string value.
func (c *Client) GetString(ctx context.Context, key string) (string, error) {
	return c.Client.Get(ctx, key).Result()
}

// SetString sets a string value with TTL.
func (c *Client) SetString(ctx context.Context, key string, value string, ttl time.Duration) error {
	return c.Client.Set(ctx, key, value, ttl).Err()
}

// Pipeline returns a new pipeline for batched operations.
func (c *Client) Pipeline() redis.Pipeliner {
	return c.Client.Pipeline()
}

// TxPipeline returns a new transaction pipeline.
func (c *Client) TxPipeline() redis.Pipeliner {
	return c.Client.TxPipeline()
}

// Subscribe subscribes to channels.
func (c *Client) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	return c.Client.Subscribe(ctx, channels...)
}