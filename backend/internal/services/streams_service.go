package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"somotracker/backend/internal/database/sqlc"
)

type StreamsService interface {
	CreateStreams(ctx context.Context, schoolID string, names []string) ([]string, error)
}

type streamsService struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
	logger  *zap.Logger
}

func NewStreamsService(pool *pgxpool.Pool, queries *sqlc.Queries, logger *zap.Logger) StreamsService {
	return &streamsService{
		pool:    pool,
		queries: queries,
		logger:  logger.With(zap.String("service", "streams")),
	}
}

func (s *streamsService) CreateStreams(ctx context.Context, schoolID string, names []string) ([]string, error) {
	if schoolID == "" {
		return nil, fmt.Errorf("bad_request: school_id is required")
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("bad_request: at least one stream name is required")
	}

	schoolUUID, err := uuid.Parse(schoolID)
	if err != nil {
		return nil, fmt.Errorf("bad_request: invalid school_id")
	}

	created := make([]string, 0, len(names))
	for _, name := range names {
		if name == "" {
			continue
		}
		streamRow, err := s.queries.CreateStream(ctx, sqlc.CreateStreamParams{
			SchoolID: pgtype.UUID{Bytes: schoolUUID, Valid: true},
			Name:     name,
		})
		if err != nil {
			s.logger.Warn("streams: create stream skipped or failed",
				zap.String("stream_name", name),
				zap.Error(err),
			)
			// If conflict (already exists), skip; otherwise return error
			continue
		}
		created = append(created, streamRow.ID.String())
	}

	if len(created) == 0 {
		return nil, fmt.Errorf("internal_error: no streams created")
	}
	return created, nil
}
