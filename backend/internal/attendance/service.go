package attendance

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"

	"somotracker/backend/internal/database"
)

// Service handles business logic for attendance operations.
type Service struct {
	repo   Repository
	redis  *redis.Client
	asynqc *asynq.Client
}

// NewService creates a new attendance Service.
func NewService(repo Repository, pools *database.Pools) *Service {
	asynqc := asynq.NewClient(asynq.RedisClientOpt{
		Addr: pools.Redis.Options().Addr,
	})
	return &Service{
		repo:   repo,
		redis:  pools.Redis,
		asynqc: asynqc,
	}
}

// GetRosterForSlot returns the class roster with existing attendance marks for a slot.
func (s *Service) GetRosterForSlot(ctx context.Context, tenantID, schoolID, timetableSlotID, date string) (*SlotRosterResponse, error) {
	if timetableSlotID == "" {
		return nil, fmt.Errorf("attendance.Service.GetRosterForSlot: timetable_slot_id is required: %w", ErrInvalidInput)
	}
	if date == "" {
		return nil, fmt.Errorf("attendance.Service.GetRosterForSlot: date is required: %w", ErrInvalidInput)
	}
	return s.repo.GetRosterForSlot(ctx, tenantID, schoolID, timetableSlotID, date)
}

// BulkMarkAttendance records attendance for all students in a given slot/date.
func (s *Service) BulkMarkAttendance(ctx context.Context, tenantID, schoolID string, payload BulkAttendancePayload, markedBy string) error {
	if payload.TimetableSlotID == "" {
		return fmt.Errorf("attendance.Service.BulkMarkAttendance: timetable_slot_id is required: %w", ErrInvalidInput)
	}
	if payload.Date == "" {
		return fmt.Errorf("attendance.Service.BulkMarkAttendance: date is required: %w", ErrInvalidInput)
	}
	if len(payload.Entries) == 0 {
		return fmt.Errorf("attendance.Service.BulkMarkAttendance: at least one entry is required: %w", ErrInvalidInput)
	}
	for _, entry := range payload.Entries {
		if entry.StudentID == "" {
			return fmt.Errorf("attendance.Service.BulkMarkAttendance: student_id is required in all entries: %w", ErrInvalidInput)
		}
		if entry.Status == "" {
			return fmt.Errorf("attendance.Service.BulkMarkAttendance: status is required for student %s: %w", entry.StudentID, ErrInvalidInput)
		}
	}

	classID, termID, err := s.repo.BulkUpsert(ctx, tenantID, schoolID, payload, markedBy)
	if err != nil {
		return err
	}

	// Enqueue a background recompute for this class's summaries.
	// Redis pending flag prevents duplicate enqueues for the same class+term
	// while a task is still outstanding.
	if err := s.enqueueClassRecompute(ctx, tenantID, schoolID, termID, classID); err != nil {
		// Non-fatal: log and continue. The materialised summary will be
		// recomputed on the fly the next time it's queried.
		slog.WarnContext(ctx, "attendance.Service.BulkMarkAttendance: enqueue recompute failed",
			slog.String("error", err.Error()),
			slog.String("tenant_id", tenantID),
			slog.String("school_id", schoolID),
			slog.String("term_id", termID),
			slog.String("class_id", classID),
		)
	}

	return nil
}

// enqueueClassRecompute enqueues an Asynq task to recompute attendance
// summaries for the given class+term. Uses a Redis SET NX key as a
// pending flag — if a task for this class+term is already queued
// (key exists), the enqueue is skipped. The worker DELETEs the key
// when it starts processing.
func (s *Service) enqueueClassRecompute(ctx context.Context, tenantID, schoolID, termID, classID string) error {
	dedupKey := fmt.Sprintf("attendance:pending:%s:%s", termID, classID)

	// Attempt to claim the pending flag. NX ensures only one caller succeeds.
	set, err := s.redis.SetNX(ctx, dedupKey, "1", 5*time.Minute).Result()
	if err != nil {
		return fmt.Errorf("attendance.Service.enqueueClassRecompute: setnx: %w", err)
	}
	if !set {
		// Another task for this class+term is already pending.
		return nil
	}

	payload := recomputeClassPayload{
		TenantID: tenantID,
		SchoolID: schoolID,
		TermID:   termID,
		ClassID:  classID,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		// Clean up the pending flag so a future mark doesn't get silently dropped.
		_ = s.redis.Del(ctx, dedupKey).Err()
		return fmt.Errorf("attendance.Service.enqueueClassRecompute: marshal: %w", err)
	}

	task := asynq.NewTask(TypeRecomputeClassSummaries, data)
	if _, err := s.asynqc.Enqueue(task, asynq.MaxRetry(3), asynq.Queue("attendance")); err != nil {
		// Clean up the pending flag so a future enqueue can retry.
		_ = s.redis.Del(ctx, dedupKey).Err()
		return fmt.Errorf("attendance.Service.enqueueClassRecompute: enqueue: %w", err)
	}

	return nil
}

// GetStudentHistory returns attendance history for a specific student.
func (s *Service) GetStudentHistory(ctx context.Context, tenantID, schoolID, studentID string, filter StudentHistoryFilter) ([]AttendanceRecord, error) {
	if studentID == "" {
		return nil, fmt.Errorf("attendance.Service.GetStudentHistory: student_id is required: %w", ErrInvalidInput)
	}
	return s.repo.GetStudentHistory(ctx, tenantID, schoolID, studentID, filter)
}

// UpdateAttendanceRecord updates a single attendance record (admin correction).
// Same-day records can be edited by the teacher; older records require admin.
func (s *Service) UpdateAttendanceRecord(ctx context.Context, id, tenantID string, payload UpdateAttendanceEntryPayload) error {
	if id == "" {
		return fmt.Errorf("attendance.Service.UpdateAttendanceRecord: id is required: %w", ErrInvalidInput)
	}
	if payload.Status == "" {
		return fmt.Errorf("attendance.Service.UpdateAttendanceRecord: status is required: %w", ErrInvalidInput)
	}
	// Enforce same-day edit window: records dated before today cannot be edited.
	record, err := s.repo.GetRecordByID(ctx, id, tenantID)
	if err != nil {
		return fmt.Errorf("attendance.Service.UpdateAttendanceRecord: %w", err)
	}
	today := time.Now().Format("2006-01-02")
	if record.Date != today {
		return fmt.Errorf("attendance.Service.UpdateAttendanceRecord: can only edit same-day records (record date: %s, today: %s): %w", record.Date, today, ErrForbidden)
	}
	return s.repo.UpdateRecord(ctx, id, tenantID, payload)
}

// GetAdminDashboard returns the school-wide attendance completion view.
func (s *Service) GetAdminDashboard(ctx context.Context, tenantID, schoolID, date string, filter DashboardFilter) (*AdminDashboardResponse, error) {
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	if schoolID == "" {
		return nil, fmt.Errorf("attendance.Service.GetAdminDashboard: school_id is required: %w", ErrInvalidInput)
	}
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 || filter.Limit > 100 {
		filter.Limit = 50
	}
	return s.repo.GetAdminDashboard(ctx, tenantID, schoolID, date, filter)
}

// GetChildAttendanceSummary returns a parent-facing summary for their child.
func (s *Service) GetChildAttendanceSummary(ctx context.Context, tenantID, schoolID, studentID, termID string) (*ChildAttendanceSummary, error) {
	if studentID == "" {
		return nil, fmt.Errorf("attendance.Service.GetChildAttendanceSummary: student_id is required: %w", ErrInvalidInput)
	}
	if termID == "" {
		return nil, fmt.Errorf("attendance.Service.GetChildAttendanceSummary: term_id is required: %w", ErrInvalidInput)
	}
	return s.repo.GetChildAttendanceSummary(ctx, tenantID, schoolID, studentID, termID)
}

// ComputeTermSummaries triggers a recalculation of attendance_term_summaries
// for all students in the given school and term.
func (s *Service) ComputeTermSummaries(ctx context.Context, tenantID, schoolID, termID string) (int, error) {
	if termID == "" {
		return 0, fmt.Errorf("attendance.Service.ComputeTermSummaries: term_id is required: %w", ErrInvalidInput)
	}
	return s.repo.ComputeTermSummaries(ctx, tenantID, schoolID, termID)
}

// ComputeClassSummaries triggers a recalculation of attendance_term_summaries
// for students in a single class within the given term.
func (s *Service) ComputeClassSummaries(ctx context.Context, tenantID, schoolID, termID, classID string) (int, error) {
	if termID == "" {
		return 0, fmt.Errorf("attendance.Service.ComputeClassSummaries: term_id is required: %w", ErrInvalidInput)
	}
	if classID == "" {
		return 0, fmt.Errorf("attendance.Service.ComputeClassSummaries: class_id is required: %w", ErrInvalidInput)
	}
	return s.repo.ComputeClassSummaries(ctx, tenantID, schoolID, termID, classID)
}
