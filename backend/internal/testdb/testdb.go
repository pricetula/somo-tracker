package testdb

import (
	"context"
	"fmt"
	"testing"

	"go.uber.org/zap"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"somotracker/backend/internal/database"
)

var TestPool *pgxpool.Pool

func Setup(t *testing.T) {
	pool, err := setupDockerPostgres(t)
	if err != nil {
		t.Fatalf("failed to setup test db: %v", err)
	}
	TestPool = pool
	if err := database.RunMigrations(TestPool, zap.NewNop()); err != nil {
		t.Fatalf("migrations failed: %v", err)
	}
}

func Teardown(t *testing.T) {
	if TestPool != nil {
		TestPool.Close()
	}
}

func setupDockerPostgres(t *testing.T) (*pgxpool.Pool, error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, fmt.Errorf("dockertest pool: %w", err)
	}
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "16-alpine",
		Env: []string{
			"POSTGRES_PASSWORD=postgres",
			"POSTGRES_DB=somotracker_test",
			"POSTGRES_USER=postgres",
		},
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		return nil, fmt.Errorf("run postgres: %w", err)
	}
	var pgURL string
	if err := pool.Retry(func() error {
		pgURL = fmt.Sprintf("postgres://postgres:postgres@localhost:%s/somotracker_test?sslmode=disable", resource.GetPort("5432/tcp"))
		TestPool, err = pgxpool.New(context.Background(), pgURL)
		if err != nil {
			return err
		}
		return TestPool.Ping(context.Background())
	}); err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	t.Cleanup(func() {
		if TestPool != nil {
			TestPool.Close()
		}
		_ = pool.Purge(resource)
	})
	return TestPool, nil
}
