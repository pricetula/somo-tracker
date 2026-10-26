package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// Client wraps Redis client for Asynq queue and ephemeral data (BE-028).
type Client struct {
	*redis.Client
}

// New creates Redis client. AOF enabled, noeviction for queue.
func New(ctx context.Context, addr string) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 2,
	})

	// Verify connection
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &Client{Client: rdb}, nil
}

// Ping checks Redis connectivity.
func (c *Client) Ping(ctx context.Context) error {
	return c.Client.Ping(ctx).Err()
}