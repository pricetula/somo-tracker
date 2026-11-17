package redis

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// Client wraps Redis client for Asynq queue and ephemeral data (BE-028).
type Client struct {
	*redis.Client
}

// New creates Redis client. AOF enabled, noeviction for queue.
// addr accepts a full Redis URL (redis://[:password@]host:port/db) or bare host:port.
func New(ctx context.Context, addr string) (*Client, error) {
	var opt *redis.Options
	var err error

	if strings.HasPrefix(addr, "redis://") {
		opt, err = redis.ParseURL(addr)
		if err != nil {
			return nil, fmt.Errorf("parse redis URL: %w", err)
		}
	} else {
		opt = &redis.Options{
			Addr: addr,
		}
	}

	opt.ReadTimeout = 3 * time.Second
	opt.WriteTimeout = 3 * time.Second
	opt.PoolSize = 10
	opt.MinIdleConns = 2

	rdb := redis.NewClient(opt)

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
