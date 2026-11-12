package seed

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"somotracker/internal/platform/db"
)

// Seed seeds reference data (countries, education systems, class levels)
// idempotently in a single transaction.
type Seed struct {
	pool *pgxpool.Pool
	log  *slog.Logger
}

// New creates a Seed.
func New(pool *pgxpool.Pool) *Seed {
	return &Seed{
		pool: pool,
		log:  slog.Default(),
	}
}

// Run executes the seed. Idempotent: uses ON CONFLICT DO NOTHING on all tables.
// Runs in a single transaction via TxManager for atomicity.
func (s *Seed) Run(ctx context.Context) error {
	txManager := db.NewTxManager(s.pool)
	return txManager.Run(ctx, func(ctx context.Context) error {
		if err := s.seedCountries(ctx); err != nil {
			return err
		}
		if err := s.seedEducationSystems(ctx); err != nil {
			return err
		}
		if err := s.seedClassLevels(ctx); err != nil {
			return err
		}
		return nil
	})
}

// seedCountries inserts reference countries idempotently.
func (s *Seed) seedCountries(ctx context.Context) error {
	countries := []struct {
		name        string
		code        string
	}{
		{"Kenya", "KE"},
		{"Tanzania", "TZ"},
		{"Uganda", "UG"},
	}

	for _, c := range countries {
		var id string
		err := s.pool.QueryRow(ctx, `
			INSERT INTO countries (country_name, country_code)
			VALUES ($1, $2)
			ON CONFLICT (country_code) DO NOTHING
			RETURNING id
		`, c.name, c.code).Scan(&id)
		if err != nil {
			if err == sql.ErrNoRows {
				s.log.Info("country already exists", "code", c.code)
				continue
			}
			return err
		}
		s.log.Info("country seeded", "code", c.code, "id", id)
	}
	return nil
}

// seedEducationSystems inserts reference education systems idempotently.
func (s *Seed) seedEducationSystems(ctx context.Context) error {
	systems := []struct {
		countryCode string
		systemName  string
	}{
		{"KE", "Kenya CBC"},
		{"TZ", "Tanzania NECTA"},
		{"UG", "Uganda UNEB"},
	}

	for _, sys := range systems {
		var id string
		err := s.pool.QueryRow(ctx, `
			INSERT INTO education_systems (country_id, system_name)
			SELECT c.id, $1
			FROM countries c
			WHERE c.country_code = $2
			ON CONFLICT (country_id, system_name) DO NOTHING
			RETURNING id
		`, sys.systemName, sys.countryCode).Scan(&id)
		if err != nil {
			if err == sql.ErrNoRows {
				s.log.Info("education system already exists", "system", sys.systemName)
				continue
			}
			return err
		}
		s.log.Info("education system seeded", "system", sys.systemName, "id", id)
	}
	return nil
}

// seedClassLevels inserts reference class levels idempotently.
func (s *Seed) seedClassLevels(ctx context.Context) error {
	levels := []struct {
		systemName string
		tierStage  string
		localLabel string
		seqIndex   int
	}{
		{"Kenya CBC", "Pre-Primary", "PP1", 1},
		{"Kenya CBC", "Pre-Primary", "PP2", 2},
		{"Kenya CBC", "Lower Primary", "Grade 1", 3},
		{"Kenya CBC", "Lower Primary", "Grade 2", 4},
		{"Kenya CBC", "Lower Primary", "Grade 3", 5},
		{"Kenya CBC", "Lower Primary", "Grade 4", 6},
		{"Kenya CBC", "Lower Primary", "Grade 5", 7},
		{"Kenya CBC", "Lower Primary", "Grade 6", 8},
		{"Kenya CBC", "Junior Secondary", "Grade 7", 9},
		{"Kenya CBC", "Junior Secondary", "Grade 8", 10},
		{"Kenya CBC", "Junior Secondary", "Grade 9", 11},
	}

	for _, lvl := range levels {
		var id string
		err := s.pool.QueryRow(ctx, `
			INSERT INTO class_levels (education_system_id, tier_stage, local_label, sequence_index)
			SELECT es.id, $1, $2, $3
			FROM education_systems es
			JOIN countries c ON c.id = es.country_id
			WHERE es.system_name = $4 AND c.country_code = 'KE'
			ON CONFLICT (education_system_id, sequence_index) DO NOTHING
			RETURNING id
		`, lvl.tierStage, lvl.localLabel, lvl.seqIndex, lvl.systemName).Scan(&id)
		if err != nil {
			if err == sql.ErrNoRows {
				s.log.Info("class level already exists", "label", lvl.localLabel)
				continue
			}
			return err
		}
		s.log.Info("class level seeded", "label", lvl.localLabel, "id", id)
	}
	return nil
}
