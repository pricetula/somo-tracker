package services

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"somotracker/backend/internal/database/sqlc"
)

func parseDate(s string) (pgtype.Date, error) {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return pgtype.Date{}, err
	}
	return pgtype.Date{Time: t, Valid: true}, nil
}

type TermInput struct {
	Name      string `json:"name"`
	StartDate string `json:"start_date"` // "2026-01-15"
	EndDate   string `json:"end_date"`
}

type AcademicPeriodRequest struct {
	Year  int         `json:"year"`
	Terms []TermInput `json:"terms"`
}

type AcademicPeriodService interface {
	CreateAcademicPeriod(ctx context.Context, schoolID string, req AcademicPeriodRequest) (string, error)
}

type academicPeriodService struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
	logger  *zap.Logger
}

func NewAcademicPeriodService(pool *pgxpool.Pool, queries *sqlc.Queries, logger *zap.Logger) AcademicPeriodService {
	return &academicPeriodService{
		pool:    pool,
		queries: queries,
		logger:  logger.With(zap.String("service", "academic_period")),
	}
}

func (s *academicPeriodService) CreateAcademicPeriod(ctx context.Context, schoolID string, req AcademicPeriodRequest) (string, error) {
	if schoolID == "" {
		return "", fmt.Errorf("bad_request: school_id is required")
	}
	if req.Year == 0 {
		return "", fmt.Errorf("bad_request: year is required")
	}
	if len(req.Terms) == 0 {
		return "", fmt.Errorf("bad_request: at least one term is required")
	}

	// Validate and parse term dates; derive year boundaries from sorted terms.
	terms := make([]TermInput, len(req.Terms))
	copy(terms, req.Terms)

	sorted := make([]TermInput, len(terms))
	copy(sorted, terms)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].StartDate < sorted[j].StartDate
	})

	minStart := sorted[0].StartDate
	maxEnd := sorted[0].EndDate
	for _, t := range sorted[1:] {
		if t.StartDate < minStart {
			minStart = t.StartDate
		}
		if t.EndDate > maxEnd {
			maxEnd = t.EndDate
		}
	}

	schoolUUID, err := uuid.Parse(schoolID)
	if err != nil {
		return "", fmt.Errorf("bad_request: invalid school_id")
	}

	yearName := fmt.Sprintf("%d", req.Year)

	minStartDate, minErr := parseDate(minStart)
	if minErr != nil {
		return "", fmt.Errorf("bad_request: invalid term start_date %s: %w", minStart, minErr)
	}
	maxEndDate, maxErr := parseDate(maxEnd)
	if maxErr != nil {
		return "", fmt.Errorf("bad_request: invalid term end_date %s: %w", maxEnd, maxErr)
	}

	yearRow, err := s.queries.CreateAcademicYear(ctx, sqlc.CreateAcademicYearParams{
		SchoolID:  pgtype.UUID{Bytes: schoolUUID, Valid: true},
		Name:      yearName,
		StartDate: minStartDate,
		EndDate:   maxEndDate,
	})
	if err != nil {
		s.logger.Error("academic_period: failed to create academic year", zap.Error(err))
		return "", fmt.Errorf("internal_error: failed to create academic year")
	}

	for _, t := range sorted {
		sd, sErr := parseDate(t.StartDate)
		if sErr != nil {
			return "", fmt.Errorf("bad_request: invalid term start_date %s: %w", t.StartDate, sErr)
		}
		ed, eErr := parseDate(t.EndDate)
		if eErr != nil {
			return "", fmt.Errorf("bad_request: invalid term end_date %s: %w", t.EndDate, eErr)
		}
		_, termErr := s.queries.CreateAcademicTerm(ctx, sqlc.CreateAcademicTermParams{
			AcademicYearID: pgtype.UUID{Bytes: yearRow.ID.Bytes, Valid: true},
			Name:           t.Name,
			StartDate:      sd,
			EndDate:        ed,
		})
		if termErr != nil {
			s.logger.Error("academic_period: failed to create academic term",
				zap.String("term_name", t.Name),
				zap.Error(termErr),
			)
			return "", fmt.Errorf("internal_error: failed to create academic term")
		}
	}

	return yearRow.ID.String(), nil
}
