package testdb

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5/pgxpool"
)

func LoadSeeds(ctx context.Context, pool *pgxpool.Pool) error {
	base := filepath.Join("db", "seeds")
	entries, err := os.ReadDir(base)
	if err != nil {
		return nil
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(base, e.Name()))
		if err != nil {
			continue
		}
		var queries []string
		if json.Unmarshal(data, &queries) == nil {
			for _, q := range queries {
				_, _ = pool.Exec(ctx, q)
			}
		}
	}
	return nil
}
