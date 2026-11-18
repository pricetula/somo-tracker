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
