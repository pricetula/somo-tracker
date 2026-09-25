package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
	"somotracker/backend/internal/database/sqlc"
)

type TimetableSubstitutionsService interface {
	CreateSubstitution(ctx context.Context, schoolID uuid.UUID, classTimetableSlotID uuid.UUID, substitutionDate string, originalTeacherMembershipID uuid.UUID, substituteTeacherMembershipID *uuid.UUID, status string, reason *string) (uuid.UUID, error)
	UpdateSubstitution(ctx context.Context, schoolID uuid.UUID, id uuid.UUID, substituteTeacherMembershipID *uuid.UUID, status *string, reason *string) error
	DeleteSubstitution(ctx context.Context, id uuid.UUID) error
}

type timetableSubstitutionsService struct {
	queries *sqlc.Queries
	logger  *zap.Logger
}

func NewTimetableSubstitutionsService(queries *sqlc.Queries) TimetableSubstitutionsService {
	return &timetableSubstitutionsService{
		queries: queries,
		logger:  zap.L().With(zap.String("service", "timetable_substitutions")),
	}
}

func (s *timetableSubstitutionsService) CreateSubstitution(ctx context.Context, schoolID uuid.UUID, classTimetableSlotID uuid.UUID, substitutionDate string, originalTeacherMembershipID uuid.UUID, substituteTeacherMembershipID *uuid.UUID, status string, reason *string) (uuid.UUID, error) {
	if status == "" {
		status = "PENDING"
	}
	parsed, err := time.Parse("2006-01-02", substitutionDate)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid substitution_date: %w", err)
	}
	subDate := pgtype.Date{Time: parsed, Valid: true}

	var subTeacher pgtype.UUID
	if substituteTeacherMembershipID != nil {
		subTeacher = pgtype.UUID{Bytes: *substituteTeacherMembershipID, Valid: true}
	}
	var reasonText pgtype.Text
	if reason != nil {
		reasonText = pgtype.Text{String: *reason, Valid: true}
	}

	params := sqlc.CreateTimetableSubstitutionParams{
		SchoolID:                      pgtype.UUID{Bytes: schoolID, Valid: true},
		ClassTimetableSlotID:          pgtype.UUID{Bytes: classTimetableSlotID, Valid: true},
		SubstitutionDate:              subDate,
		OriginalTeacherMembershipID:   pgtype.UUID{Bytes: originalTeacherMembershipID, Valid: true},
		SubstituteTeacherMembershipID: subTeacher,
		Status:                        sqlc.SubstitutionStatus(status),
		Reason:                        reasonText,
	}
	row, err := s.queries.CreateTimetableSubstitution(ctx, params)
	if err != nil {
		s.logger.Error("create timetable substitution failed", zap.Error(err))
		return uuid.Nil, err
	}
	return uuid.UUID(row.ID.Bytes), nil
}

func (s *timetableSubstitutionsService) UpdateSubstitution(ctx context.Context, schoolID uuid.UUID, id uuid.UUID, substituteTeacherMembershipID *uuid.UUID, status *string, reason *string) error {
	// Load existing to preserve fields not being updated
	existing, err := s.queries.GetTimetableSubstitution(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		s.logger.Error("fetch substitution for update failed", zap.Error(err))
		return err
	}
	var subTeacher pgtype.UUID
	if substituteTeacherMembershipID != nil {
		subTeacher = pgtype.UUID{Bytes: *substituteTeacherMembershipID, Valid: true}
	} else {
		subTeacher = existing.SubstituteTeacherMembershipID
	}
	var subStatus sqlc.SubstitutionStatus
	if status != nil {
		subStatus = sqlc.SubstitutionStatus(*status)
	} else {
		subStatus = existing.Status
	}
	var reasonText pgtype.Text
	if reason != nil {
		reasonText = pgtype.Text{String: *reason, Valid: true}
	} else {
		reasonText = existing.Reason
	}
	params := sqlc.UpdateTimetableSubstitutionParams{
		ID:                            pgtype.UUID{Bytes: id, Valid: true},
		SchoolID:                      pgtype.UUID{Bytes: schoolID, Valid: true},
		SubstituteTeacherMembershipID: subTeacher,
		Status:                        subStatus,
		Reason:                        reasonText,
	}
	if err := s.queries.UpdateTimetableSubstitution(ctx, params); err != nil {
		s.logger.Error("update timetable substitution failed", zap.Error(err))
		return err
	}
	return nil
}

func (s *timetableSubstitutionsService) DeleteSubstitution(ctx context.Context, id uuid.UUID) error {
	if err := s.queries.DeleteTimetableSubstitution(ctx, pgtype.UUID{Bytes: id, Valid: true}); err != nil {
		s.logger.Error("delete timetable substitution failed", zap.Error(err))
		return err
	}
	return nil
}
