package students

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

// Service implements student business logic.
type Service struct {
	repo       StudentRepository
	importRepo ImportRepository
}

// NewService creates a new Service.
func NewService(repo StudentRepository) *Service {
	return &Service{repo: repo}
}

// SetImportRepo sets the import repository (called during DI wiring).
func (s *Service) SetImportRepo(repo ImportRepository) {
	s.importRepo = repo
}

// ─── Check Duplicates ────────────────────────────────────────────────────

// CheckDuplicates checks which of the provided field values already exist in
// cbc_students for the given tenant/school. Delegates to the import repo.
func (s *Service) CheckDuplicates(ctx context.Context, tenantID, schoolID string,
	admissionNumbers, upiNumbers, knecNumbers []string) ([]string, []string, []string, error) {
	return s.importRepo.CheckExistingFieldValues(ctx, tenantID, schoolID,
		admissionNumbers, upiNumbers, knecNumbers)
}

// ─── List ─────────────────────────────────────────────────────────────────

// ListStudents returns a paginated list of students.
func (s *Service) ListStudents(ctx context.Context, filter ListFilter) (ListStudentsResponse, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 || filter.Limit > 200 {
		filter.Limit = 50
	}

	students, total, err := s.repo.List(ctx, filter)
	if err != nil {
		return ListStudentsResponse{}, fmt.Errorf("students.Service.ListStudents: %w", err)
	}

	return ListStudentsResponse{
		Items: students,
		Total: total,
		Page:  filter.Page,
		Limit: filter.Limit,
	}, nil
}

// ─── Create ───────────────────────────────────────────────────────────────

// Create creates a new student record.
func (s *Service) Create(ctx context.Context, tenantID, schoolID string, payload CreateStudentPayload) (*Student, error) {
	if tenantID == "" || schoolID == "" {
		return nil, fmt.Errorf("students.Service.Create: %w", ErrInvalidInput)
	}

	payload.FullName = strings.TrimSpace(payload.FullName)
	if payload.FullName == "" {
		return nil, fmt.Errorf("students.Service.Create: full_name is required: %w", ErrInvalidInput)
	}

	// Validate gender
	if payload.Gender != "" && payload.Gender != "M" && payload.Gender != "F" {
		return nil, fmt.Errorf("students.Service.Create: invalid gender %q: %w", payload.Gender, ErrInvalidInput)
	}

	student := &Student{
		FullName:             payload.FullName,
		Gender:               payload.Gender,
		DateOfBirth:          payload.DateOfBirth,
		UPINumber:            payload.UPINumber,
		KNECAssessmentNumber: payload.KNECAssessmentNumber,
	}

	id, err := s.repo.Create(ctx, student)
	if err != nil {
		return nil, fmt.Errorf("students.Service.Create: %w", err)
	}
	student.ID = id

	// If class_id is provided, also create an enrollment
	if payload.ClassID != nil && *payload.ClassID != "" {
		// We create a placeholder enrollment. In production, academic_term_id
		// would come from the current active term.
		slog.Warn("students.Service.Create: class_id provided but enrollment not yet supported without academic_term_id",
			"class_id", *payload.ClassID)
	}

	slog.Info("student.created",
		"tenant_id", tenantID,
		"school_id", schoolID,
		"resource_id", id,
		"full_name", payload.FullName,
	)

	return student, nil
}

// ─── Get Detail ───────────────────────────────────────────────────────────

// Delete hard-deletes a student record.
func (s *Service) Delete(ctx context.Context, id, tenantID, schoolID string) error {
	if id == "" || tenantID == "" || schoolID == "" {
		return fmt.Errorf("students.Service.Delete: %w", ErrInvalidInput)
	}
	return s.repo.Delete(ctx, id, tenantID, schoolID)
}

// GetDetail returns a student with enrollment history.
func (s *Service) GetDetail(ctx context.Context, id, tenantID, schoolID string) (*StudentDetail, error) {
	if id == "" || tenantID == "" || schoolID == "" {
		return nil, fmt.Errorf("students.Service.GetDetail: %w", ErrInvalidInput)
	}
	return s.repo.GetDetail(ctx, id, tenantID, schoolID)
}

// ─── Update ───────────────────────────────────────────────────────────────

// Update applies partial updates to a student record.
func (s *Service) Update(ctx context.Context, id, tenantID, schoolID string, payload UpdateStudentPayload) error {
	if id == "" || tenantID == "" || schoolID == "" {
		return fmt.Errorf("students.Service.Update: %w", ErrInvalidInput)
	}

	// Fetch existing student
	student, err := s.repo.GetByID(ctx, id, tenantID, schoolID)
	if err != nil {
		return fmt.Errorf("students.Service.Update: %w", err)
	}

	// Apply partial updates
	if payload.FullName != nil {
		trimmed := strings.TrimSpace(*payload.FullName)
		if trimmed == "" {
			return fmt.Errorf("students.Service.Update: full_name cannot be empty: %w", ErrInvalidInput)
		}
		student.FullName = trimmed
	}
	if payload.Gender != nil {
		if *payload.Gender != "M" && *payload.Gender != "F" {
			return fmt.Errorf("students.Service.Update: invalid gender %q: %w", *payload.Gender, ErrInvalidInput)
		}
		student.Gender = *payload.Gender
	}
	if payload.DateOfBirth != nil {
		student.DateOfBirth = payload.DateOfBirth
	}
	if payload.UPINumber != nil {
		student.UPINumber = payload.UPINumber
	}
	if payload.KNECAssessmentNumber != nil {
		student.KNECAssessmentNumber = payload.KNECAssessmentNumber
	}
	if payload.IsActive != nil {
		student.IsActive = *payload.IsActive
	}

	if err := s.repo.Update(ctx, student); err != nil {
		return fmt.Errorf("students.Service.Update: %w", err)
	}

	slog.Info("student.updated",
		"tenant_id", tenantID,
		"school_id", schoolID,
		"resource_id", id,
	)

	return nil
}

// ─── Batch Create ─────────────────────────────────────────────────────────

// CreateBatch creates multiple students in a single transaction.
// Returns the IDs of all created students. On any validation failure
// the entire batch is rejected (all-or-nothing).
func (s *Service) CreateBatch(ctx context.Context, tenantID, schoolID string, payloads []CreateStudentPayload) (CreateStudentsResponse, error) {
	if tenantID == "" || schoolID == "" {
		return CreateStudentsResponse{}, fmt.Errorf("students.Service.CreateBatch: %w", ErrInvalidInput)
	}

	if len(payloads) == 0 {
		return CreateStudentsResponse{}, fmt.Errorf("students.Service.CreateBatch: students list is empty: %w", ErrInvalidInput)
	}

	students := make([]*Student, 0, len(payloads))
	for i, payload := range payloads {
		fullName := strings.TrimSpace(payload.FullName)
		if fullName == "" {
			return CreateStudentsResponse{}, fmt.Errorf("students.Service.CreateBatch: students[%d].full_name is required: %w", i, ErrInvalidInput)
		}

		if payload.Gender != "" && payload.Gender != "M" && payload.Gender != "F" {
			return CreateStudentsResponse{}, fmt.Errorf("students.Service.CreateBatch: students[%d] invalid gender %q: %w", i, payload.Gender, ErrInvalidInput)
		}

		students = append(students, &Student{
			FullName:             fullName,
			Gender:               payload.Gender,
			DateOfBirth:          payload.DateOfBirth,
			UPINumber:            payload.UPINumber,
			KNECAssessmentNumber: payload.KNECAssessmentNumber,
		})
	}

	ids, err := s.repo.CreateBatch(ctx, students)
	if err != nil {
		return CreateStudentsResponse{}, fmt.Errorf("students.Service.CreateBatch: %w", err)
	}

	slog.Info("students.batch.created",
		"tenant_id", tenantID,
		"school_id", schoolID,
		"count", len(ids),
	)

	return CreateStudentsResponse{IDs: ids, Code: "ok"}, nil
}

// ─── Enrollments ──────────────────────────────────────────────────────────

// CreateEnrollment enrolls a student in a class for a specific academic term.
func (s *Service) CreateEnrollment(ctx context.Context, studentID, tenantID, schoolID string, payload CreateEnrollmentPayload) (*Enrollment, error) {
	if studentID == "" || tenantID == "" || schoolID == "" {
		return nil, fmt.Errorf("students.Service.CreateEnrollment: %w", ErrInvalidInput)
	}

	status := strings.TrimSpace(payload.Status)
	if status == "" {
		status = "ACTIVE"
	}

	enrollment := &Enrollment{
		StudentID:      studentID,
		ClassID:        payload.ClassID,
		AcademicTermID: payload.AcademicTermID,
		Status:         status,
	}

	id, err := s.repo.CreateEnrollment(ctx, enrollment)
	if err != nil {
		return nil, fmt.Errorf("students.Service.CreateEnrollment: %w", err)
	}
	enrollment.ID = id

	slog.Info("student.enrollment.created",
		"tenant_id", tenantID,
		"school_id", schoolID,
		"student_id", studentID,
		"term_id", payload.AcademicTermID,
		"class_id", payload.ClassID,
		"resource_id", id,
	)

	return enrollment, nil
}

// ListEnrollments returns all enrollments for a student.
func (s *Service) ListEnrollments(ctx context.Context, studentID, tenantID string) ([]Enrollment, error) {
	if studentID == "" || tenantID == "" {
		return nil, fmt.Errorf("students.Service.ListEnrollments: %w", ErrInvalidInput)
	}
	return s.repo.ListEnrollments(ctx, studentID, tenantID)
}

// ─── Batch Enrollments ───────────────────────────────────────────────────

// CreateBatchEnrollments enrolls multiple students in the given class for the
// specified academic term. Status defaults to ACTIVE. Skips students already
// enrolled in the term.
func (s *Service) CreateBatchEnrollments(ctx context.Context, tenantID, schoolID, academicTermID string, items []BatchEnrollItem) (BatchEnrollResponse, error) {
	if tenantID == "" || schoolID == "" || academicTermID == "" {
		return BatchEnrollResponse{}, fmt.Errorf("students.Service.CreateBatchEnrollments: %w", ErrInvalidInput)
	}
	if len(items) == 0 {
		return BatchEnrollResponse{}, fmt.Errorf("students.Service.CreateBatchEnrollments: enrollments list is empty: %w", ErrInvalidInput)
	}

	enrollments := make([]*Enrollment, 0, len(items))
	for _, item := range items {
		if item.StudentID == "" || item.ClassID == "" {
			return BatchEnrollResponse{}, fmt.Errorf("students.Service.CreateBatchEnrollments: student_id and class_id are required: %w", ErrInvalidInput)
		}
		enrollments = append(enrollments, &Enrollment{
			StudentID:      item.StudentID,
			ClassID:        item.ClassID,
			AcademicTermID: academicTermID,
			Status:         "ACTIVE",
		})
	}

	ids, err := s.repo.CreateBatchEnrollments(ctx, enrollments, tenantID, schoolID)
	if err != nil {
		return BatchEnrollResponse{}, fmt.Errorf("students.Service.CreateBatchEnrollments: %w", err)
	}

	slog.Info("students.batch.enrolled",
		"tenant_id", tenantID,
		"school_id", schoolID,
		"academic_term_id", academicTermID,
		"count", len(ids),
		"total_requested", len(items),
	)

	return BatchEnrollResponse{IDs: ids, Code: "ok"}, nil
}
