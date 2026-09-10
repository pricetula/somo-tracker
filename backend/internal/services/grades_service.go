package services

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"somotracker/backend/internal/database/sqlc"
)

type GradesService interface {
	GetGradeLevels(ctx context.Context, schoolID string) ([]sqlc.GradeLevel, error)
}

type gradesService struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
	logger  *zap.Logger
}

func NewGradesService(pool *pgxpool.Pool, queries *sqlc.Queries, logger *zap.Logger) GradesService {
	return &gradesService{
		pool:    pool,
		queries: queries,
		logger:  logger.With(zap.String("service", "grades")),
	}
}

func (s *gradesService) GetGradeLevels(ctx context.Context, schoolID string) ([]sqlc.GradeLevel, error) {
	if schoolID == "" {
		return nil, fmt.Errorf("bad_request: school_id is required")
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, education_system_id, country_id, tier_stage, local_label, sequence_index, created_at, updated_at
		FROM grade_levels
		ORDER BY sequence_index ASC
	`)
	if err != nil {
		s.logger.Error("grades: query failed", zap.Error(err))
		return nil, fmt.Errorf("internal_error: failed to fetch grades")
	}
	defer rows.Close()

	var results []sqlc.GradeLevel
	for rows.Next() {
		var gl sqlc.GradeLevel
		if err := rows.Scan(
			&gl.ID, &gl.EducationSystemID, &gl.CountryID, &gl.TierStage,
			&gl.LocalLabel, &gl.SequenceIndex, &gl.CreatedAt, &gl.UpdatedAt,
		); err != nil {
			s.logger.Error("grades: scan failed", zap.Error(err))
			continue
		}
		results = append(results, gl)
	}
	return results, nil
}
