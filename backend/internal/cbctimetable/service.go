package cbctimetable

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// ─── Domain error markers ──────────────────────────────────────────────────

var (
	ErrNotFound         = fmt.Errorf("not found")
	ErrOverlap          = fmt.Errorf("time overlap conflict")
	ErrInvalidDayOfWeek = fmt.Errorf("day_of_week must be between 1 and 7")
	ErrInvalidTimeRange = fmt.Errorf("end_time must be after start_time")
	ErrNoAcademicYear   = fmt.Errorf("no current academic year set")
	ErrNoCurrentTerm    = fmt.Errorf("no current academic term set")
)

// Service contains business logic for the CBC timetable domain.
type Service struct {
	repo *Repository
}

// NewService creates a new Service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// ─── Slot CRUD ─────────────────────────────────────────────────────────────

// FetchSlots returns all slots for a class.
func (s *Service) FetchSlots(ctx context.Context, classID string) ([]TimetableSlot, error) {
	return s.repo.FetchSlotsByClass(ctx, classID)
}

// CreateSlot validates and creates a new timetable slot.
// It does NOT check conflicts — the DB exclusion constraints are the authority.
// The frontend runs its own pre-check; this accepts the DB's verdict.
func (s *Service) CreateSlot(ctx context.Context, schoolID, tenantID string, req *CreateSlotRequest) (*TimetableSlot, error) {
	if err := validateSlot(req.DayOfWeek, req.StartTime, req.EndTime); err != nil {
		return nil, err
	}

	// Resolve academic_year_id from the school via the class
	academicYearID, err := s.resolveAcademicYearID(ctx, req.ClassID)
	if err != nil {
		return nil, err
	}

	slot := &TimetableSlot{
		TenantID:       tenantID,
		SchoolID:       schoolID,
		AcademicYearID: academicYearID,
		ClassID:        req.ClassID,
		TeacherID:      req.TeacherID,
		LearningAreaID: req.LearningAreaID,
		RoomIdentifier: req.RoomIdentifier,
		DayOfWeek:      req.DayOfWeek,
		StartTime:      req.StartTime,
		EndTime:        req.EndTime,
	}

	if err := s.repo.CreateSlot(ctx, slot); err != nil {
		return nil, annotateDBError(err)
	}

	return slot, nil
}

// UpdateSlot validates and updates an existing timetable slot.
func (s *Service) UpdateSlot(ctx context.Context, slotID, schoolID string, req *UpdateSlotRequest) (*TimetableSlot, error) {
	if err := validateSlot(req.DayOfWeek, req.StartTime, req.EndTime); err != nil {
		return nil, err
	}

	existing, err := s.repo.FetchSlotByID(ctx, slotID)
	if err != nil {
		return nil, fmt.Errorf("fetch existing slot: %w", err)
	}
	if existing == nil {
		return nil, fmt.Errorf("slot %s: %w", slotID, ErrNotFound)
	}

	existing.TeacherID = req.TeacherID
	existing.LearningAreaID = req.LearningAreaID
	existing.RoomIdentifier = req.RoomIdentifier
	existing.DayOfWeek = req.DayOfWeek
	existing.StartTime = req.StartTime
	existing.EndTime = req.EndTime

	if err := s.repo.UpdateSlot(ctx, existing); err != nil {
		return nil, annotateDBError(err)
	}

	return existing, nil
}

// DeleteSlot removes a slot by ID.
func (s *Service) DeleteSlot(ctx context.Context, slotID string) error {
	return s.repo.DeleteSlot(ctx, slotID)
}

// ─── Conflict pre-check ────────────────────────────────────────────────────

// CheckConflicts is the pre-check endpoint called by the frontend while the
// side panel is open. It returns both teacher and room overlaps as human-
// readable ConflictErrors. This is best-effort — the DB EXCLUDE constraint is
// the source of truth.
func (s *Service) CheckConflicts(ctx context.Context, req *ConflictCheckRequest) ([]*ConflictError, error) {
	var errors []*ConflictError
	var mu sync.Mutex
	var wg sync.WaitGroup

	// ── Teacher overlap check ──────────────────────────────────────────
	wg.Add(1)
	go func() {
		defer wg.Done()
		overlaps, err := s.repo.FindTeacherOverlaps(
			ctx, req.TeacherID, req.DayOfWeek,
			req.StartTime, req.EndTime,
			req.AcademicYearID, req.SchoolID,
			req.ExcludeSlotID, req.ExcludeClassID,
		)
		if err != nil {
			return
		}
		for _, o := range overlaps {
			mu.Lock()
			errors = append(errors, &ConflictError{
				Type:       "teacher",
				EntityName: o.TeacherName,
				ClassName:  o.ClassName,
				DayOfWeek:  req.DayOfWeek,
				StartTime:  o.StartTime,
				EndTime:    o.EndTime,
			})
			mu.Unlock()
		}
	}()

	// ── Room overlap check ─────────────────────────────────────────────
	if req.RoomIdentifier != nil && *req.RoomIdentifier != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			overlaps, err := s.repo.FindRoomOverlaps(
				ctx, *req.RoomIdentifier, req.DayOfWeek,
				req.StartTime, req.EndTime,
				req.AcademicYearID, req.SchoolID,
				req.ExcludeSlotID, req.ExcludeClassID,
			)
			if err != nil {
				return
			}
			for _, o := range overlaps {
				mu.Lock()
				errors = append(errors, &ConflictError{
					Type:       "room",
					EntityName: *req.RoomIdentifier,
					ClassName:  o.ClassName,
					DayOfWeek:  req.DayOfWeek,
					StartTime:  o.StartTime,
					EndTime:    o.EndTime,
				})
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// Sort by type for deterministic output
	sort.Slice(errors, func(i, j int) bool {
		if errors[i].Type != errors[j].Type {
			return errors[i].Type < errors[j].Type
		}
		return errors[i].StartTime < errors[j].StartTime
	})

	return errors, nil
}

// ─── Bulk operations ───────────────────────────────────────────────────────

// DuplicateDay copies all slots from a source day to one or more target days.
// Skips individual slots that would conflict (teacher or room overlap).
func (s *Service) DuplicateDay(ctx context.Context, req *DuplicateDayRequest) (*BulkOperationResult, error) {
	if req.SourceDay < 1 || req.SourceDay > 7 {
		return nil, ErrInvalidDayOfWeek
	}
	for _, d := range req.TargetDays {
		if d < 1 || d > 7 {
			return nil, ErrInvalidDayOfWeek
		}
	}

	// Fetch source slots
	slots, err := s.repo.FetchSlotsByClassAndDay(ctx, req.ClassID, req.SourceDay, req.AcademicYearID)
	if err != nil {
		return nil, fmt.Errorf("fetch source slots: %w", err)
	}
	if len(slots) == 0 {
		return &BulkOperationResult{}, nil
	}

	result := &BulkOperationResult{}

	// De-duplicate target days
	targetSet := make(map[int]bool)
	for _, d := range req.TargetDays {
		if d != req.SourceDay {
			targetSet[d] = true
		}
	}

	for targetDay := range targetSet {
		for _, slot := range slots {
			// Check for teacher conflict on target day
			teacherOverlaps, err := s.repo.FindTeacherOverlaps(
				ctx, slot.TeacherID, targetDay,
				slot.StartTime, slot.EndTime,
				req.AcademicYearID, slot.SchoolID,
				nil, &req.ClassID,
			)
			if err != nil {
				return nil, fmt.Errorf("check teacher overlap: %w", err)
			}

			roomOverlaps := false
			if slot.RoomIdentifier != nil && *slot.RoomIdentifier != "" {
				roomOverlapsList, err := s.repo.FindRoomOverlaps(
					ctx, *slot.RoomIdentifier, targetDay,
					slot.StartTime, slot.EndTime,
					req.AcademicYearID, slot.SchoolID,
					nil, &req.ClassID,
				)
				if err != nil {
					return nil, fmt.Errorf("check room overlap: %w", err)
				}
				roomOverlaps = len(roomOverlapsList) > 0
			}

			if len(teacherOverlaps) > 0 || roomOverlaps {
				reason := ""
				if len(teacherOverlaps) > 0 {
					reason = "Teacher " + teacherOverlaps[0].TeacherName + " already has a slot at this time"
				} else {
					reason = "Room " + *slot.RoomIdentifier + " is already in use at this time"
				}
				result.Skipped = append(result.Skipped, SlotSkipReason{
					DayOfWeek: targetDay,
					StartTime: slot.StartTime,
					Reason:    reason,
				})
				continue
			}

			// Create new slot with updated day
			newSlot := slot
			newSlot.ID = ""
			newSlot.DayOfWeek = targetDay
			if err := s.repo.CreateSlot(ctx, &newSlot); err != nil {
				result.Skipped = append(result.Skipped, SlotSkipReason{
					DayOfWeek: targetDay,
					StartTime: slot.StartTime,
					Reason:    "Database constraint violation: " + err.Error(),
				})
				continue
			}
			result.TotalCopied++
		}
	}

	return result, nil
}

// CopyFromClass copies all slots from a source class to the target class.
func (s *Service) CopyFromClass(ctx context.Context, req *CopyFromClassRequest) (*BulkOperationResult, error) {
	// Fetch all slots from the source class
	sourceSlots, err := s.repo.FetchSlotsByClass(ctx, req.SourceClassID)
	if err != nil {
		return nil, fmt.Errorf("fetch source class slots: %w", err)
	}
	if len(sourceSlots) == 0 {
		return &BulkOperationResult{}, nil
	}

	// Resolve target class info to get tenant/school IDs
	targetClass, err := s.repo.FetchClassBrief(ctx, req.TargetClassID)
	if err != nil {
		return nil, fmt.Errorf("fetch target class: %w", err)
	}
	if targetClass == nil {
		return nil, fmt.Errorf("target class %s: %w", req.TargetClassID, ErrNotFound)
	}

	// We need the tenant_id and school_id for the target class.
	// These are fetched from the target class via a separate query.
	// For now, we reuse the source slot's tenant/school since they share the same school.
	// In a real cross-school scenario, this would need mapping.

	result := &BulkOperationResult{}

	for _, slot := range sourceSlots {
		// Check teacher overlap — excludes source class so existing source slots
		// don't block the copy. Only conflicts with target class slots matter.
		sourceClassID := req.SourceClassID
		teacherOverlaps, err := s.repo.FindTeacherOverlaps(
			ctx, slot.TeacherID, slot.DayOfWeek,
			slot.StartTime, slot.EndTime,
			slot.AcademicYearID, slot.SchoolID,
			nil, &sourceClassID,
		)
		if err != nil {
			return nil, fmt.Errorf("check teacher overlap: %w", err)
		}

		roomOverlaps := false
		if slot.RoomIdentifier != nil && *slot.RoomIdentifier != "" {
			roomOverlapsList, err := s.repo.FindRoomOverlaps(
				ctx, *slot.RoomIdentifier, slot.DayOfWeek,
				slot.StartTime, slot.EndTime,
				slot.AcademicYearID, slot.SchoolID,
				nil, &sourceClassID,
			)
			if err != nil {
				return nil, fmt.Errorf("check room overlap: %w", err)
			}
			roomOverlaps = len(roomOverlapsList) > 0
		}

		if len(teacherOverlaps) > 0 || roomOverlaps {
			reason := ""
			if len(teacherOverlaps) > 0 {
				reason = "Teacher " + teacherOverlaps[0].TeacherName + " already has a slot at this time"
			} else {
				reason = "Room " + *slot.RoomIdentifier + " is already in use at this time"
			}
			result.Skipped = append(result.Skipped, SlotSkipReason{
				DayOfWeek: slot.DayOfWeek,
				StartTime: slot.StartTime,
				Reason:    reason,
			})
			continue
		}

		// Create new slot for the target class
		newSlot := slot
		newSlot.ID = ""
		newSlot.ClassID = req.TargetClassID
		if err := s.repo.CreateSlot(ctx, &newSlot); err != nil {
			result.Skipped = append(result.Skipped, SlotSkipReason{
				DayOfWeek: slot.DayOfWeek,
				StartTime: slot.StartTime,
				Reason:    "Database constraint violation: " + err.Error(),
			})
			continue
		}
		result.TotalCopied++
	}

	return result, nil
}

// ─── Learning areas ────────────────────────────────────────────────────────

// FetchLearningAreas returns learning areas for a grade.
func (s *Service) FetchLearningAreas(ctx context.Context, gradeID string) ([]LearningAreaBrief, error) {
	return s.repo.FetchLearningAreasByGrade(ctx, gradeID)
}

// ─── Teachers ──────────────────────────────────────────────────────────────

// FetchTeachers returns all teachers at a school.
func (s *Service) FetchTeachers(ctx context.Context, schoolID, tenantID string) ([]TeacherBrief, error) {
	return s.repo.FetchTeachersBySchool(ctx, schoolID, tenantID)
}

// FetchClassTeachers returns teachers scoped to a class (optionally by learning area).
func (s *Service) FetchClassTeachers(ctx context.Context, classID string, learningAreaID *string) ([]TeacherBrief, error) {
	return s.repo.FetchClassTeachers(ctx, classID, learningAreaID)
}

// ─── Room autocomplete ─────────────────────────────────────────────────────

// FetchRoomAutocomplete returns previously used room identifiers.
func (s *Service) FetchRoomAutocomplete(ctx context.Context, query string, schoolID, tenantID string) ([]string, error) {
	return s.repo.FetchRoomAutocomplete(ctx, query, schoolID, tenantID)
}

// ─── Slot attendance count ─────────────────────────────────────────────────

// FetchSlotAttendanceCount returns the number of attendance periods linked to a slot.
func (s *Service) FetchSlotAttendanceCount(ctx context.Context, slotID string) (*AttendanceCount, error) {
	count, err := s.repo.CountAttendancePeriodsForSlot(ctx, slotID)
	if err != nil {
		return nil, err
	}
	return &AttendanceCount{Count: count}, nil
}

// ─── Operating days ────────────────────────────────────────────────────────

// FetchOperatingDays returns the operating days for a school.
func (s *Service) FetchOperatingDays(ctx context.Context, schoolID, tenantID string) ([]int, error) {
	days, err := s.repo.FetchOperatingDays(ctx, schoolID, tenantID)
	if err != nil {
		return nil, err
	}
	// Always return 1-7 sorted
	sort.Ints(days)
	return days, nil
}

// ─── Attendance helpers ────────────────────────────────────────────────────

// FetchClassStudents returns students enrolled in a class for a term.
func (s *Service) FetchClassStudents(ctx context.Context, classID, termID string) ([]StudentAttendanceRow, error) {
	return s.repo.FetchClassStudents(ctx, classID, termID)
}

// FetchTodayTeacherSlots returns today's slots for a teacher.
func (s *Service) FetchTodayTeacherSlots(ctx context.Context, teacherID string) ([]SlotBrief, error) {
	return s.repo.FetchTodayTeacherSlots(ctx, teacherID)
}

// FetchCurrentTerm returns the current academic term for a school.
func (s *Service) FetchCurrentTerm(ctx context.Context, schoolID, tenantID string) (string, error) {
	return s.repo.FetchCurrentAcademicTerm(ctx, schoolID, tenantID)
}

// ═══════════════════════════════════════════════════════════════════════════
// ATTENDANCE — periods
// ═══════════════════════════════════════════════════════════════════════════

// CreateAttendancePeriod creates a new attendance period.
func (s *Service) CreateAttendancePeriod(ctx context.Context, classID string, tenantID, schoolID, userID string, req *CreatePeriodRequest) (*CbcAttendancePeriod, error) {
	// Resolve the current academic term for this school
	termID, err := s.repo.FetchCurrentAcademicTerm(ctx, schoolID, tenantID)
	if err != nil {
		return nil, ErrNoCurrentTerm
	}

	period := &CbcAttendancePeriod{
		TenantID:       tenantID,
		SchoolID:       schoolID,
		AcademicTermID: termID,
		ClassID:        classID,
		LearningAreaID: req.LearningAreaID,
		DateRecorded:   req.DateRecorded,
		RecordedBy:     userID,
	}

	if err := s.repo.CreateAttendancePeriod(ctx, period); err != nil {
		return nil, fmt.Errorf("create attendance period: %w", err)
	}

	return period, nil
}

// FetchAttendancePeriodsByDate returns periods for a class on a specific date.
func (s *Service) FetchAttendancePeriodsByDate(ctx context.Context, classID, date string) ([]CbcAttendancePeriod, error) {
	return s.repo.FetchAttendancePeriodsByDate(ctx, classID, date)
}

// FetchAttendancePeriodSummaries returns period summaries for a date range.
func (s *Service) FetchAttendancePeriodSummaries(ctx context.Context, classID, from, to string) ([]AttendancePeriodSummary, error) {
	return s.repo.FetchAttendancePeriodSummaries(ctx, classID, from, to)
}

// FetchAttendancePeriodSummary returns a single period summary by ID.
func (s *Service) FetchAttendancePeriodSummary(ctx context.Context, periodID string) (*AttendancePeriodSummary, error) {
	return s.repo.FetchAttendancePeriodSummary(ctx, periodID)
}

// ═══════════════════════════════════════════════════════════════════════════
// ATTENDANCE — logs
// ═══════════════════════════════════════════════════════════════════════════

// FetchAttendanceLogs returns all logs for a period.
func (s *Service) FetchAttendanceLogs(ctx context.Context, periodID string) ([]AttendanceLogDetail, error) {
	return s.repo.FetchAttendanceLogsByPeriod(ctx, periodID)
}

// SaveAttendanceLog upserts a single attendance log.
func (s *Service) SaveAttendanceLog(ctx context.Context, tenantID, userID string, req *SaveLogRequest) (*CbcAttendanceLog, error) {
	log := &CbcAttendanceLog{
		TenantID:   tenantID,
		PeriodID:   req.PeriodID,
		StudentID:  req.StudentID,
		Status:     req.Status,
		Remarks:    req.Remarks,
		RecordedBy: userID,
	}

	if err := s.repo.UpsertAttendanceLog(ctx, log); err != nil {
		return nil, fmt.Errorf("save attendance log: %w", err)
	}

	return log, nil
}

// BatchSaveAttendanceLogs upserts multiple logs for a period.
func (s *Service) BatchSaveAttendanceLogs(ctx context.Context, tenantID, userID string, req *BatchSaveLogsRequest) ([]CbcAttendanceLog, error) {
	return s.repo.BatchUpsertAttendanceLogs(ctx, tenantID, req.PeriodID, userID, req.Marks)
}

// ═══════════════════════════════════════════════════════════════════════════
// ATTENDANCE — analytics
// ═══════════════════════════════════════════════════════════════════════════

// FetchAttendanceHeatmap returns per-day heatmap data for a class/term.
func (s *Service) FetchAttendanceHeatmap(ctx context.Context, classID, termID string) ([]AttendanceHeatmapDay, error) {
	return s.repo.FetchAttendanceHeatmap(ctx, classID, termID)
}

// FetchAttendanceGaps returns timetable slots with no attendance coverage.
func (s *Service) FetchAttendanceGaps(ctx context.Context, classID, from, to string) ([]AttendanceGap, error) {
	return s.repo.FetchAttendanceGaps(ctx, classID, from, to)
}

// ─── Internal helpers ──────────────────────────────────────────────────────

func validateSlot(dayOfWeek int, startTime, endTime string) error {
	if dayOfWeek < 1 || dayOfWeek > 7 {
		return ErrInvalidDayOfWeek
	}
	if startTime >= endTime {
		return ErrInvalidTimeRange
	}
	return nil
}

func (s *Service) resolveAcademicYearID(ctx context.Context, classID string) (string, error) {
	return s.repo.ResolveAcademicYearID(ctx, classID)
}

// annotateDBError converts known PostgreSQL constraint violation errors into
// user-friendly ConflictErrors.
func annotateDBError(err error) error {
	errStr := err.Error()
	if containsAny(errStr,
		"excl_cbc_timetable_teacher",
		"excl_cbc_timetable_room",
		"conflicts with",
		"exclusion constraint",
	) {
		return fmt.Errorf("%w: %s", ErrOverlap, errStr)
	}
	return err
}

func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if idx := indexOf(s, sub); idx >= 0 {
			return true
		}
	}
	return false
}

// indexOf is a simple strings.Contains replacement to avoid importing strings
// just for this helper.
func indexOf(s, substr string) int {
	n := len(substr)
	if n == 0 {
		return 0
	}
	for i := 0; i <= len(s)-n; i++ {
		if s[i:i+n] == substr {
			return i
		}
	}
	return -1
}
