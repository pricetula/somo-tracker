package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"somotracker/backend/internal/database/sqlc"
)

type CreateTimeSlot struct {
	Name            string
	StartTime       string
	EndTime         string
	IsInstructional bool
	SequenceIndex   int
}

type CreateTemplateRequest struct {
	Name        string
	Description string
	TimeSlots   []CreateTimeSlot
}

type TimetableService interface {
	CreateTemplate(ctx context.Context, schoolID uuid.UUID, req CreateTemplateRequest) (uuid.UUID, error)
	ListTemplates(ctx context.Context, schoolID uuid.UUID) ([]sqlc.TimetableTemplate, error)
	ListTimeSlotsByTemplate(ctx context.Context, templateID uuid.UUID) ([]sqlc.TimeSlot, error)
}

type timetableService struct {
	queries *sqlc.Queries
	pool    *pgxpool.Pool
}

func NewTimetableService(pool *pgxpool.Pool, queries *sqlc.Queries) TimetableService {
	return &timetableService{queries: queries, pool: pool}
}

func (s *timetableService) ListTemplates(ctx context.Context, schoolID uuid.UUID) ([]sqlc.TimetableTemplate, error) {
	return s.queries.ListTimetableTemplatesBySchool(ctx, pgtype.UUID{Bytes: schoolID, Valid: true})
}

func (s *timetableService) ListTimeSlotsByTemplate(ctx context.Context, templateID uuid.UUID) ([]sqlc.TimeSlot, error) {
	return s.queries.ListTimeSlotsByTemplate(ctx, pgtype.UUID{Bytes: templateID, Valid: true})
}

func (s *timetableService) CreateTemplate(ctx context.Context, schoolID uuid.UUID, req CreateTemplateRequest) (uuid.UUID, error) {
	templateID := uuid.New()

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback(ctx)

	qtx := s.queries.WithTx(tx)

	desc := pgtype.Text{}
	if req.Description != "" {
		desc.String = req.Description
		desc.Valid = true
	}
	_, err = qtx.CreateTimetableTemplate(ctx, sqlc.CreateTimetableTemplateParams{
		ID:          pgtype.UUID{Bytes: templateID, Valid: true},
		SchoolID:    pgtype.UUID{Bytes: schoolID, Valid: true},
		Name:        req.Name,
		Description: desc,
	})
	if err != nil {
		return uuid.Nil, err
	}

	for i, slot := range req.TimeSlots {
		slotID := uuid.New()
		start, err := parseTime(slot.StartTime)
		if err != nil {
			return uuid.Nil, err
		}
		end, err := parseTime(slot.EndTime)
		if err != nil {
			return uuid.Nil, err
		}
		_, err = qtx.CreateTimeSlot(ctx, sqlc.CreateTimeSlotParams{
			ID:                  pgtype.UUID{Bytes: slotID, Valid: true},
			SchoolID:            pgtype.UUID{Bytes: schoolID, Valid: true},
			TimetableTemplateID: pgtype.UUID{Bytes: templateID, Valid: true},
			Name:                slot.Name,
			StartTime:           start,
			EndTime:             end,
			SequenceIndex:       int32(i),
			IsInstructional:     slot.IsInstructional,
		})
		if err != nil {
			return uuid.Nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, err
	}
	return templateID, nil
}

func parseTime(v string) (pgtype.Time, error) {
	t, err := time.Parse("15:04:05", v)
	if err != nil {
		t, err = time.Parse("15:04", v)
		if err != nil {
			return pgtype.Time{}, err
		}
	}
	var pt pgtype.Time
	pt.Microseconds = int64(t.Hour()*3600+int(t.Minute())*60+int(t.Second()))*1e6 + int64(t.Nanosecond()/1000)
	pt.Valid = true
	return pt, nil
}
