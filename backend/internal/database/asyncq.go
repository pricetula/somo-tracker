package database

import (
	"fmt"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

// asynqRedisOpt maps the shared Redis client's connection options onto the
// Asynq RedisConnOpt shape. This is the single source of truth for how Asynq
// reaches Redis — every field the Redis pool was created with (Addr, Password,
// DB index, TLS) must be mirrored here so producers and consumers survive
// auth, non-default DB indexes, and TLS-enabled connections.
func asynqRedisOpt(pools *Pools) asynq.RedisClientOpt {
	redisOpt := pools.Redis.Options()
	return asynq.RedisClientOpt{
		Addr:      redisOpt.Addr,
		Password:  redisOpt.Password,  // Fixes NOAUTH error
		DB:        redisOpt.DB,        // Keeps the same database index
		TLSConfig: redisOpt.TLSConfig, // Required if using TLS/SSL (rediss://)
	}
}

// NewAsynqClient creates the single shared Asynq client from the Redis pool.
// One client is provided through fx (database.Module) and injected into every
// enqueuer — never construct per-package clients. This mirrors the psql
// pattern: database.Connect builds one *Pools and fx injects it everywhere.
func NewAsynqClient(pools *Pools) *asynq.Client {
	return asynq.NewClient(asynqRedisOpt(pools))
}

// NewAsynqServer creates an Asynq server bound to the shared Redis pool.
// Each worker passes its own per-worker asynq.Config (concurrency + queue
// map); the Redis connection options and zap log adapter are applied here
// once instead of being re-derived in every worker package.
func NewAsynqServer(pools *Pools, logger *zap.SugaredLogger, cfg asynq.Config) *asynq.Server {
	cfg.Logger = asynqLogger{logger: logger}
	return asynq.NewServer(asynqRedisOpt(pools), cfg)
}

// NewAsynqScheduler creates an Asynq periodic-task scheduler bound to the
// shared Redis pool. opts may be nil; the zap log adapter is always applied.
func NewAsynqScheduler(pools *Pools, logger *zap.SugaredLogger, opts *asynq.SchedulerOpts) *asynq.Scheduler {
	if opts == nil {
		opts = &asynq.SchedulerOpts{}
	}
	opts.Logger = asynqLogger{logger: logger}
	return asynq.NewScheduler(asynqRedisOpt(pools), opts)
}

// asynqLogger wraps zap to implement asynq.Logger. Shared by every server and
// scheduler so background job logs flow through the same zap instance.
type asynqLogger struct {
	logger *zap.SugaredLogger
}

func (l asynqLogger) Debug(args ...interface{}) { l.logger.Debug(fmt.Sprint(args...)) }
func (l asynqLogger) Info(args ...interface{})  { l.logger.Info(fmt.Sprint(args...)) }
func (l asynqLogger) Warn(args ...interface{})  { l.logger.Warn(fmt.Sprint(args...)) }
func (l asynqLogger) Error(args ...interface{}) { l.logger.Error(fmt.Sprint(args...)) }
func (l asynqLogger) Fatal(args ...interface{}) { l.logger.Error(fmt.Sprint(args...)) }
