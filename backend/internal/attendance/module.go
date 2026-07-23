package attendance

import (
	"context"
	"time"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"

	"somotracker/backend/internal/database"
)

// ─── Concrete Deduplicator (Redis-backed) ─────────────────────────────────

// redisDeduplicator wraps *redis.Client to implement Deduplicator.
type redisDeduplicator struct {
	client *redis.Client
}

func (d *redisDeduplicator) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	return d.client.SetNX(ctx, key, value, expiration).Result()
}

func (d *redisDeduplicator) Del(ctx context.Context, key string) error {
	return d.client.Del(ctx, key).Err()
}

// ─── Concrete TaskEnqueuer (Asynq-backed) ─────────────────────────────────

// asynqTaskEnqueuer wraps an *asynq.Client to implement TaskEnqueuer.
type asynqTaskEnqueuer struct {
	client *asynq.Client
}

func (e *asynqTaskEnqueuer) EnqueueTask(payload []byte, opts ...asynq.Option) error {
	task := asynq.NewTask(TypeRecomputeClassSummaries, payload)
	_, err := e.client.Enqueue(task, opts...)
	return err
}

// ─── Module ───────────────────────────────────────────────────────────────

// Module is an fx-compatible module for the attendance domain.
var Module = fx.Module("attendance",
	fx.Provide(
		fx.Annotate(NewRepository, fx.As(new(Repository))),
		NewRedisDeduplicator,
		NewAsynqTaskEnqueuer,
		NewService,
		NewHandler,
		NewWorker,
	),
)

// NewRedisDeduplicator is a provider that creates a Deduplicator from the pools.
func NewRedisDeduplicator(pools *database.Pools) Deduplicator {
	return &redisDeduplicator{client: pools.Redis}
}

// NewAsynqTaskEnqueuer is a provider that creates a TaskEnqueuer from the pools.
func NewAsynqTaskEnqueuer(pools *database.Pools) TaskEnqueuer {
	return &asynqTaskEnqueuer{
		client: asynq.NewClient(asynq.RedisClientOpt{Addr: pools.Redis.Options().Addr}),
	}
}
