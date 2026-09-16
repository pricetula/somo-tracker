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
	ListStreams(ctx context.Context, schoolID string) ([]sqlc.Stream, error)
	CreateStreams(ctx context.Context, schoolID string, names []string, colors []string) ([]string, error)
	GetStream(ctx context.Context, streamID string) (*sqlc.Stream, error)
	UpdateStream(ctx context.Context, streamID string, name *string, color *string) (*sqlc.Stream, error)
	DeleteStreams(ctx context.Context, ids []string) error
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

func (s *streamsService) ListStreams(ctx context.Context, schoolID string) ([]sqlc.Stream, error) {
	if schoolID == "" {
		return nil, fmt.Errorf("bad_request: school_id is required")
	}
	schoolUUID, err := uuid.Parse(schoolID)
	if err != nil {
		return nil, fmt.Errorf("bad_request: invalid school_id")
	}
	rows, err := s.queries.ListStreamsBySchool(ctx, pgtype.UUID{Bytes: schoolUUID, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("internal_error: failed to list streams: %w", err)
	}
	return rows, nil
}

func (s *streamsService) CreateStreams(ctx context.Context, schoolID string, names []string, colors []string) ([]string, error) {
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
	for i, name := range names {
		if name == "" {
			continue
		}
		colorVal := ""
		if i < len(colors) && colors[i] != "" {
			colorVal = colors[i]
		}
		params := sqlc.CreateStreamParams{
			SchoolID: pgtype.UUID{Bytes: schoolUUID, Valid: true},
			Name:     name,
		}
		if colorVal != "" {
			params.Color = pgtype.Text{String: colorVal, Valid: true}
		} else {
			params.Color = pgtype.Text{Valid: false}
		}
		streamRow, err := s.queries.CreateStream(ctx, params)
		if err != nil {
			s.logger.Warn("streams: create stream skipped or failed",
				zap.String("stream_name", name),
				zap.Error(err),
			)
			continue
		}
		created = append(created, streamRow.ID.String())
	}

	if len(created) == 0 {
		return nil, fmt.Errorf("internal_error: no streams created")
	}
	return created, nil
}

func (s *streamsService) GetStream(ctx context.Context, streamID string) (*sqlc.Stream, error) {
	streamUUID, err := uuid.Parse(streamID)
	if err != nil {
		return nil, fmt.Errorf("bad_request: invalid stream_id")
	}
	var stream sqlc.Stream
	err = s.pool.QueryRow(ctx, `SELECT id, school_id, name, color, created_at, updated_at FROM streams WHERE id = $1`, streamUUID).Scan(
		&stream.ID, &stream.SchoolID, &stream.Name, &stream.Color, &stream.CreatedAt, &stream.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("internal_error: failed to get stream: %w", err)
	}
	return &stream, nil
}

func (s *streamsService) UpdateStream(ctx context.Context, streamID string, name *string, color *string) (*sqlc.Stream, error) {
	streamUUID, err := uuid.Parse(streamID)
	if err != nil {
		return nil, fmt.Errorf("bad_request: invalid stream_id")
	}
	var stream sqlc.Stream
	err = s.pool.QueryRow(ctx, `UPDATE streams SET name = COALESCE($2, name), color = COALESCE($3, color), updated_at = NOW() WHERE id = $1 RETURNING id, school_id, name, color, created_at, updated_at`, streamUUID, name, color).Scan(
		&stream.ID, &stream.SchoolID, &stream.Name, &stream.Color, &stream.CreatedAt, &stream.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("internal_error: failed to update stream: %w", err)
	}
	return &stream, nil
}

func (s *streamsService) DeleteStreams(ctx context.Context, ids []string) error {
	for _, idStr := range ids {
		uuidVal, err := uuid.Parse(idStr)
		if err != nil {
			continue
		}
		_, _ = s.pool.Exec(ctx, `DELETE FROM streams WHERE id = $1`, uuidVal)
	}
	return nil
}
