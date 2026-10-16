package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
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
	GetTemplate(ctx context.Context, templateID uuid.UUID) (sqlc.TimetableTemplate, error)
	UpdateTemplate(ctx context.Context, templateID uuid.UUID, name string, description string) error
	ListTemplates(ctx context.Context, schoolID uuid.UUID) ([]sqlc.TimetableTemplate, error)
	ListTimeSlotsByTemplate(ctx context.Context, templateID uuid.UUID) ([]sqlc.TimeSlot, error)
	SetupClassTimetableSlot(ctx context.Context, schoolID uuid.UUID, classRoomID, timeSlotID, subjectID, teacherMembershipID string, dayOfWeek int, roomID string) error
	GetClassTimetableSlotsByTemplate(ctx context.Context, schoolID uuid.UUID, classRoomID uuid.UUID, templateID uuid.UUID) ([]sqlc.GetClassTimetableSlotsByTemplateWithDetailsRow, error)
	DeleteClassTimetableSlot(ctx context.Context, slotID uuid.UUID) error
}

type timetableService struct {
	queries *sqlc.Queries
	pool    *pgxpool.Pool
	logger  *zap.Logger
}

func NewTimetableService(pool *pgxpool.Pool, queries *sqlc.Queries) TimetableService {
	return &timetableService{queries: queries, pool: pool, logger: zap.L().With(zap.String("service", "timetable"))}
}

func (s *timetableService) GetTemplate(ctx context.Context, templateID uuid.UUID) (sqlc.TimetableTemplate, error) {
	return s.queries.GetTimetableTemplate(ctx, pgtype.UUID{Bytes: templateID, Valid: true})
}

func (s *timetableService) UpdateTemplate(ctx context.Context, templateID uuid.UUID, name string, description string) error {
	desc := pgtype.Text{}
	if description != "" {
		desc.String = description
		desc.Valid = true
	}
	return s.queries.UpdateTimetableTemplate(ctx, sqlc.UpdateTimetableTemplateParams{
		ID:          pgtype.UUID{Bytes: templateID, Valid: true},
		Name:        name,
		Description: desc,
	})
}

func (s *timetableService) ListTemplates(ctx context.Context, schoolID uuid.UUID) ([]sqlc.TimetableTemplate, error) {
	return s.queries.ListTimetableTemplatesBySchool(ctx, pgtype.UUID{Bytes: schoolID, Valid: true})
}

func (s *timetableService) ListTimeSlotsByTemplate(ctx context.Context, templateID uuid.UUID) ([]sqlc.TimeSlot, error) {
	return s.queries.ListTimeSlotsByTemplate(ctx, pgtype.UUID{Bytes: templateID, Valid: true})
}

func (s *timetableService) SetupClassTimetableSlot(ctx context.Context, schoolID uuid.UUID, classRoomID, timeSlotID, subjectID, teacherMembershipID string, dayOfWeek int, roomID string) error {
	if dayOfWeek < 1 || dayOfWeek > 7 {
		return fmt.Errorf("day_of_week must be between 1 and 7")
	}

	// Parse IDs upfront with context
	classRoomUUID, err := uuid.Parse(classRoomID)
	if err != nil {
		return fmt.Errorf("invalid class_room_id: %w", err)
	}
	timeSlotUUID, err := uuid.Parse(timeSlotID)
	if err != nil {
		return fmt.Errorf("invalid time_slot_id: %w", err)
	}
	subjectUUID, err := uuid.Parse(subjectID)
	if err != nil {
		return fmt.Errorf("invalid subject_id: %w", err)
	}
	teacherUUID, err := uuid.Parse(teacherMembershipID)
	if err != nil {
		return fmt.Errorf("invalid teacher_membership_id: %w", err)
	}

	var roomUUID pgtype.UUID
	if roomID != "" {
		r, err := uuid.Parse(roomID)
		if err != nil {
			return fmt.Errorf("invalid room_id: %w", err)
		}
		roomUUID = pgtype.UUID{Bytes: r, Valid: true}
	}

	resolver := NewAcademicPeriodResolver(s.queries, s.logger)
	res, err := resolver.Resolve(ctx, schoolID, time.Now(), "current")
	if err != nil {
		return fmt.Errorf("failed to resolve academic period: %w", err)
	}
	if !res.IsWriteAllowed || res.WriteTermID == uuid.Nil {
		return fmt.Errorf("no_next_academic_term: cannot schedule timetable slot for current period")
	}
	termID := res.WriteTermID

	// Use transaction for atomicity
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			s.logger.Error("rollback failed", zap.Error(rbErr))
		}
	}()

	// Conflict detection
	exists := false
	// Teacher double-book
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM class_timetable_slots WHERE school_id=$1 AND academic_term_id=$2 AND day_of_week=$3 AND time_slot_id=$4 AND teacher_membership_id=$5)`, schoolID, termID, dayOfWeek, timeSlotUUID, teacherUUID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("conflict check teacher: %w", err)
	}
	if exists {
		return fmt.Errorf("timetable_conflict:teacher_double_book")
	}
	// Room double-book
	if roomID != "" {
		err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM class_timetable_slots WHERE school_id=$1 AND academic_term_id=$2 AND day_of_week=$3 AND time_slot_id=$4 AND room_id=$5)`, schoolID, termID, dayOfWeek, timeSlotUUID, roomUUID).Scan(&exists)
		if err != nil {
			return fmt.Errorf("conflict check room: %w", err)
		}
		if exists {
			return fmt.Errorf("timetable_conflict:room_double_book")
		}
	}
	// Class overlap
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM class_timetable_slots WHERE school_id=$1 AND academic_term_id=$2 AND day_of_week=$3 AND time_slot_id=$4 AND class_room_id=$5)`, schoolID, termID, dayOfWeek, timeSlotUUID, classRoomUUID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("conflict check class: %w", err)
	}
	if exists {
		return fmt.Errorf("timetable_conflict:class_overlap")
	}

	qtx := s.queries.WithTx(tx)

	_, err = qtx.CreateClassTimetableSlot(ctx, sqlc.CreateClassTimetableSlotParams{
		SchoolID:            pgtype.UUID{Bytes: schoolID, Valid: true},
		ClassRoomID:         pgtype.UUID{Bytes: classRoomUUID, Valid: true},
		AcademicTermID:      pgtype.UUID{Bytes: termID, Valid: true},
		DayOfWeek:           int32(dayOfWeek),
		TimeSlotID:          pgtype.UUID{Bytes: timeSlotUUID, Valid: true},
		SubjectID:           pgtype.UUID{Bytes: subjectUUID, Valid: true},
		TeacherMembershipID: pgtype.UUID{Bytes: teacherUUID, Valid: true},
		RoomID:              roomUUID,
	})
	if err != nil {
		return fmt.Errorf("create class timetable slot: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}

func (s *timetableService) CreateTemplate(ctx context.Context, schoolID uuid.UUID, req CreateTemplateRequest) (uuid.UUID, error) {
	templateID := uuid.New()

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return uuid.Nil, err
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			s.logger.Error("rollback failed", zap.Error(rbErr))
		}
	}()

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

func (s *timetableService) GetClassTimetableSlotsByTemplate(ctx context.Context, schoolID uuid.UUID, classRoomID uuid.UUID, templateID uuid.UUID) ([]sqlc.GetClassTimetableSlotsByTemplateWithDetailsRow, error) {
	// Resolve current academic term strictly on backend
	resolver := NewAcademicPeriodResolver(s.queries, s.logger)
	res, err := resolver.Resolve(ctx, schoolID, time.Now(), "current")
	if err != nil {
		return nil, fmt.Errorf("failed to resolve academic period: %w", err)
	}
	if res.ReadTermID == uuid.Nil {
		return nil, fmt.Errorf("no academic term found for school")
	}
	termID := pgtype.UUID{Bytes: res.ReadTermID, Valid: true}
	return s.queries.GetClassTimetableSlotsByTemplateWithDetails(ctx, sqlc.GetClassTimetableSlotsByTemplateWithDetailsParams{
		ClassRoomID:         pgtype.UUID{Bytes: classRoomID, Valid: true},
		TimetableTemplateID: pgtype.UUID{Bytes: templateID, Valid: true},
		AcademicTermID:      termID,
	})
}

func (s *timetableService) DeleteClassTimetableSlot(ctx context.Context, slotID uuid.UUID) error {
	return s.queries.DeleteClassTimetableSlot(ctx, pgtype.UUID{Bytes: slotID, Valid: true})
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
