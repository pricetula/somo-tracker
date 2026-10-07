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

type TimetableSubstitution struct {
	ID                            uuid.UUID
	SchoolID                      uuid.UUID
	ClassTimetableSlotID          uuid.UUID
	SubstitutionDate              time.Time
	OriginalTeacherMembershipID   uuid.UUID
	SubstituteTeacherMembershipID *uuid.UUID
	Status                        string
	Reason                        string
	ClassName                     string
	SubjectName                   string
	OriginalTeacherName           string
	SubstituteTeacherName         string
	TimeSlotName                  string
	StartTime                     time.Time
	EndTime                       time.Time
	CreatedAt                     time.Time
	UpdatedAt                     time.Time
}

type TimetableSubstitutionsService interface {
	CreateSubstitution(ctx context.Context, schoolID uuid.UUID, classTimetableSlotID uuid.UUID, substitutionDate string, originalTeacherMembershipID uuid.UUID, substituteTeacherMembershipID *uuid.UUID, status string, reason *string) (uuid.UUID, error)
	GetSubstitution(ctx context.Context, schoolID uuid.UUID, id uuid.UUID) (*TimetableSubstitution, error)
	ListSubstitutions(ctx context.Context, schoolID uuid.UUID, page, limit int, dateFrom, dateTo *time.Time, statusFilter string) ([]*TimetableSubstitution, int, error)
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
	// Double-booking check before insert
	if substituteTeacherMembershipID != nil {
		conflictCount, err := s.queries.CountSubstitutionConflicts(ctx, sqlc.CountSubstitutionConflictsParams{
			SchoolID:                      pgtype.UUID{Bytes: schoolID, Valid: true},
			ClassTimetableSlotID:          pgtype.UUID{Bytes: classTimetableSlotID, Valid: true},
			SubstitutionDate:              subDate,
			SubstituteTeacherMembershipID: subTeacher,
			Column5:                       pgtype.UUID{Valid: false},
		})
		if err != nil {
			s.logger.Error("conflict check failed", zap.Error(err))
		} else if conflictCount > 0 {
			return uuid.Nil, fmt.Errorf("substitute teacher already assigned to this slot on this date")
		}
	}

	row, err := s.queries.CreateTimetableSubstitution(ctx, params)
	if err != nil {
		s.logger.Error("create timetable substitution failed", zap.Error(err))
		return uuid.Nil, err
	}
	return uuid.UUID(row.ID.Bytes), nil
}

func (s *timetableSubstitutionsService) GetSubstitution(ctx context.Context, schoolID uuid.UUID, id uuid.UUID) (*TimetableSubstitution, error) {
	row, err := s.queries.GetTimetableSubstitution(ctx, sqlc.GetTimetableSubstitutionParams{
		ID:       pgtype.UUID{Bytes: id, Valid: true},
		SchoolID: pgtype.UUID{Bytes: schoolID, Valid: true},
	})
	if err != nil {
		s.logger.Error("get timetable substitution failed", zap.Error(err))
		return nil, err
	}
	return s.mapTimetableSubstitution(row), nil
}

func (s *timetableSubstitutionsService) ListSubstitutions(ctx context.Context, schoolID uuid.UUID, page, limit int, dateFrom, dateTo *time.Time, statusFilter string) ([]*TimetableSubstitution, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}

	var dateFromPg, dateToPg pgtype.Date
	if dateFrom != nil {
		dateFromPg = pgtype.Date{Time: *dateFrom, Valid: true}
	}
	if dateTo != nil {
		dateToPg = pgtype.Date{Time: *dateTo, Valid: true}
	}

	total, err := s.queries.CountTimetableSubstitutions(ctx, sqlc.CountTimetableSubstitutionsParams{
		SchoolID: pgtype.UUID{Bytes: schoolID, Valid: true},
		Column2:  dateFromPg,
		Column3:  dateToPg,
		Column4:  statusFilter,
	})
	if err != nil {
		s.logger.Error("count timetable substitutions failed", zap.Error(err))
		return nil, 0, err
	}

	rows, err := s.queries.ListTimetableSubstitutions(ctx, sqlc.ListTimetableSubstitutionsParams{
		SchoolID: pgtype.UUID{Bytes: schoolID, Valid: true},
		Column2:  dateFromPg,
		Column3:  dateToPg,
		Column4:  statusFilter,
		Limit:    int32(limit),
		Offset:   int32((page - 1) * limit),
	})
	if err != nil {
		s.logger.Error("list timetable substitutions failed", zap.Error(err))
		return nil, 0, err
	}

	result := make([]*TimetableSubstitution, len(rows))
	for i, row := range rows {
		result[i] = s.mapListRowToSubstitution(row)
	}
	return result, int(total), nil
}

func (s *timetableSubstitutionsService) mapTimetableSubstitution(row sqlc.TimetableSubstitution) *TimetableSubstitution {
	sub := &TimetableSubstitution{
		ID:                          uuid.UUID(row.ID.Bytes),
		SchoolID:                    uuid.UUID(row.SchoolID.Bytes),
		ClassTimetableSlotID:        uuid.UUID(row.ClassTimetableSlotID.Bytes),
		SubstitutionDate:            row.SubstitutionDate.Time,
		OriginalTeacherMembershipID: uuid.UUID(row.OriginalTeacherMembershipID.Bytes),
		Status:                      string(row.Status),
		Reason:                      row.Reason.String,
		CreatedAt:                   row.CreatedAt.Time,
		UpdatedAt:                   row.UpdatedAt.Time,
	}
	if row.SubstituteTeacherMembershipID.Valid {
		id := uuid.UUID(row.SubstituteTeacherMembershipID.Bytes)
		sub.SubstituteTeacherMembershipID = &id
	}
	return sub
}

func (s *timetableSubstitutionsService) mapListRowToSubstitution(row sqlc.ListTimetableSubstitutionsRow) *TimetableSubstitution {
	var startTime, endTime time.Time
	if row.StartTime.Valid {
		startTime = time.UnixMicro(row.StartTime.Microseconds)
	}
	if row.EndTime.Valid {
		endTime = time.UnixMicro(row.EndTime.Microseconds)
	}

	sub := &TimetableSubstitution{
		ID:                          uuid.UUID(row.ID.Bytes),
		SchoolID:                    uuid.UUID(row.SchoolID.Bytes),
		ClassTimetableSlotID:        uuid.UUID(row.ClassTimetableSlotID.Bytes),
		SubstitutionDate:            row.SubstitutionDate.Time,
		OriginalTeacherMembershipID: uuid.UUID(row.OriginalTeacherMembershipID.Bytes),
		Status:                      string(row.Status),
		Reason:                      row.Reason.String,
		ClassName:                   row.ClassName,
		SubjectName:                 row.SubjectName,
		OriginalTeacherName:         row.OriginalTeacherName,
		SubstituteTeacherName:       row.SubstituteTeacherName.String,
		TimeSlotName:                row.TimeSlotName,
		StartTime:                   startTime,
		EndTime:                     endTime,
		CreatedAt:                   row.CreatedAt.Time,
		UpdatedAt:                   row.UpdatedAt.Time,
	}
	if row.SubstituteTeacherMembershipID.Valid {
		id := uuid.UUID(row.SubstituteTeacherMembershipID.Bytes)
		sub.SubstituteTeacherMembershipID = &id
	}
	return sub
}

func (s *timetableSubstitutionsService) UpdateSubstitution(ctx context.Context, schoolID uuid.UUID, id uuid.UUID, substituteTeacherMembershipID *uuid.UUID, status *string, reason *string) error {
	// Load existing to preserve fields not being updated
	existing, err := s.queries.GetTimetableSubstitution(ctx, sqlc.GetTimetableSubstitutionParams{
		ID:       pgtype.UUID{Bytes: id, Valid: true},
		SchoolID: pgtype.UUID{Bytes: schoolID, Valid: true},
	})
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

	// Double-booking check: if a substitute teacher is set, ensure no other
	// substitution conflicts on the same slot/date.
	if subTeacher.Valid {
		conflictCount, err := s.queries.CountSubstitutionConflicts(ctx, sqlc.CountSubstitutionConflictsParams{
			SchoolID:                      pgtype.UUID{Bytes: schoolID, Valid: true},
			ClassTimetableSlotID:          existing.ClassTimetableSlotID,
			SubstitutionDate:              existing.SubstitutionDate,
			SubstituteTeacherMembershipID: subTeacher,
			Column5:                       pgtype.UUID{Bytes: id, Valid: true},
		})
		if err != nil {
			s.logger.Error("conflict check failed", zap.Error(err))
		} else if conflictCount > 0 {
			return fmt.Errorf("substitute teacher already assigned to this slot on this date")
		}
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
