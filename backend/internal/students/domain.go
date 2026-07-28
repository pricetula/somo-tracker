package students

import (
	"context"

	"somotracker/backend/internal/xerrors"
)

// ============================================================================
// Sentinel Domain Errors
// ============================================================================

var (
	ErrNotFound         = xerrors.NotFound("student not found")
	ErrAlreadyExists    = xerrors.AlreadyExists("student already exists")
	ErrInvalidInput     = xerrors.InvalidInput("invalid student input")
	ErrUnauthorized     = xerrors.Unauthorized("unauthorized")
	ErrForbidden        = xerrors.Forbidden("forbidden")
	ErrConflict         = xerrors.Conflict("student conflict")
	ErrDuplicateUPI     = xerrors.AlreadyExists("duplicate UPI number")
	ErrDuplicateEnroll  = xerrors.Conflict("student already enrolled in this term")
	ErrStudentNotActive = xerrors.InvalidInput("student is not active")
)

// ============================================================================
// Domain Models
// ============================================================================

// Student represents a full student record.
type Student struct {
	ID                   string  `json:"id"`
	FullName             string  `json:"full_name"`
	Gender               string  `json:"gender"`
	DateOfBirth          *string `json:"date_of_birth,omitempty"`
	UPINumber            *string `json:"upi_number,omitempty"`
	KNECAssessmentNumber *string `json:"knec_assessment_number,omitempty"`
	AdmissionNumber      *string `json:"admission_number,omitempty"`
	ClassName            *string `json:"class_name,omitempty"`
	ClassID              *string `json:"class_id,omitempty"`
	IsActive             bool    `json:"is_active"`
	CreatedAt            string  `json:"created_at"`
}

// StudentDetail extends Student with enrollment history, behavior notes,
// attendance summaries, and linked parent/guardian profiles.
type StudentDetail struct {
	Student
	Enrollments   []Enrollment            `json:"enrollments"`
	Behavior      []BehaviorNoteItem      `json:"behavior"`
	Attendance    []AttendanceSummaryItem `json:"attendance,omitempty"`
	LinkedParents []LinkedParent          `json:"linked_parents"`
}

// LinkedParent represents a parent/guardian linked to a student.
type LinkedParent struct {
	ParentID     string  `json:"parent_id"`
	FullName     string  `json:"full_name"`
	Email        string  `json:"email"`
	PhoneNumber  string  `json:"phone_number"`
	Relationship *string `json:"relationship,omitempty"`
	IsPrimary    bool    `json:"is_primary"`
}

// BehaviorNoteItem is a lightweight behavior note for the student detail page.
type BehaviorNoteItem struct {
	ID           string `json:"id"`
	CategoryName string `json:"category_name"`
	Description  string `json:"description"`
	Date         string `json:"date"`
	Status       string `json:"status"`
	IsUrgent     bool   `json:"is_urgent"`
}

// AttendanceSummaryItem is a lightweight attendance summary for the student detail page.
type AttendanceSummaryItem struct {
	LearningAreaID   string  `json:"learning_area_id"`
	LearningAreaName string  `json:"learning_area_name"`
	PeriodsTotal     int     `json:"periods_total"`
	PeriodsPresent   int     `json:"periods_present"`
	PeriodsAbsent    int     `json:"periods_absent"`
	PeriodsLate      int     `json:"periods_late"`
	PeriodsExcused   int     `json:"periods_excused"`
	Percentage       float64 `json:"percentage"`
}

// Enrollment represents a single term enrollment record.
type Enrollment struct {
	ID             string `json:"id"`
	StudentID      string `json:"student_id"`
	ClassID        string `json:"class_id"`
	AcademicTermID string `json:"academic_term_id"`
	AcademicYearID string `json:"academic_year_id"`
	TermName       string `json:"term_name"`
	TermNumber     int    `json:"term_number"`
	AcademicYear   string `json:"academic_year"`
	ClassName      string `json:"class_name"`
	Status         string `json:"status"`
	CreatedAt      string `json:"created_at"`
}

// ============================================================================
// Request / Response Payloads
// ============================================================================

type ListStudentsResponse struct {
	Items []Student `json:"items"`
	Total int       `json:"total"`
	Page  int       `json:"page"`
	Limit int       `json:"limit"`
}

type StudentDetailResponse struct {
	Data StudentDetail `json:"data"`
}

type CreateStudentPayload struct {
	FullName             string  `json:"full_name"`
	Gender               string  `json:"gender"`
	DateOfBirth          *string `json:"date_of_birth,omitempty"`
	UPINumber            *string `json:"upi_number,omitempty"`
	KNECAssessmentNumber *string `json:"knec_assessment_number,omitempty"`
	ClassID              *string `json:"class_id,omitempty"`
}

// CreateStudentsPayload is the request body for batch create (POST /api/v1/students).
// Accepts an array of students instead of a single object.
type CreateStudentsPayload struct {
	Students []CreateStudentPayload `json:"students"`
}

type UpdateStudentPayload struct {
	FullName             *string `json:"full_name,omitempty"`
	Gender               *string `json:"gender,omitempty"`
	DateOfBirth          *string `json:"date_of_birth,omitempty"`
	UPINumber            *string `json:"upi_number,omitempty"`
	KNECAssessmentNumber *string `json:"knec_assessment_number,omitempty"`
	IsActive             *bool   `json:"is_active,omitempty"`
}

type CreateStudentResponse struct {
	ID string `json:"id"`
}

type CreateStudentsResponse struct {
	IDs  []string `json:"ids"`
	Code string   `json:"code"`
}

type CreateEnrollmentPayload struct {
	AcademicTermID string `json:"academic_term_id"`
	ClassID        string `json:"class_id"`
	Status         string `json:"status,omitempty"` // defaults to ACTIVE
}

type CreateEnrollmentResponse struct {
	ID string `json:"id"`
}

// BatchEnrollItem represents one student+class pair in a batch enrollment request.
type BatchEnrollItem struct {
	StudentID string `json:"student_id"`
	ClassID   string `json:"class_id"`
}

// BatchEnrollRequest is the request body for POST /api/v1/students/enrollments.
// academic_term_id is resolved server-side from the current active term.
type BatchEnrollRequest struct {
	Enrollments []BatchEnrollItem `json:"enrollments"`
}

// BatchEnrollResponse returns the IDs of created enrollment records.
type BatchEnrollResponse struct {
	IDs  []string `json:"ids"`
	Code string   `json:"code"`
}

type ListEnrollmentsResponse struct {
	Items []Enrollment `json:"items"`
	Total int          `json:"total"`
	Page  int          `json:"page"`
	Limit int          `json:"limit"`
}

// ListFilter holds query parameters for listing students.
type ListFilter struct {
	TenantID         string
	SchoolID         string
	Page             int
	Limit            int
	Search           string
	ClassID          string
	Gender           string
	EducationLevels  []string // multi-select education level filter
	GradeLevels      []string // multi-select grade level filter
	EnrollmentStatus string   // ACTIVE, SUSPENDED, TRANSFERRED, or empty for all
}

// ============================================================================
// Import Types — Bulk Student Import
// ============================================================================

// ImportRow represents a single student row in a bulk import.
// ClassID is optional. When empty, the student is created without an
// enrollment (no class assignment for the term).
type ImportRow struct {
	FullName             string  `json:"full_name"`
	Gender               string  `json:"gender"`
	DateOfBirth          *string `json:"date_of_birth,omitempty"`
	UPINumber            *string `json:"upi_number,omitempty"`
	KNECAssessmentNumber *string `json:"knec_assessment_number,omitempty"`
	AdmissionNumber      *string `json:"admission_number,omitempty"`
	ClassID              string  `json:"class_id,omitempty"`
}

// ImportRequest is the request body for POST /students/import.
// academic_term_id is resolved server-side from the current active term.
type ImportRequest struct {
	IDempotencyKey *string     `json:"idempotency_key,omitempty"`
	Rows           []ImportRow `json:"rows"`
}

// ImportResponse is returned after creating a student import job.
// IsReplay indicates the response reflects a pre-existing job (idempotent
// replay) rather than a newly created one. When IsReplay is true the
// HTTP status should be 200 instead of 201.
type ImportResponse struct {
	JobID        string `json:"job_id"`
	TotalRecords int    `json:"total_records"`
	TotalChunks  int    `json:"total_chunks"`
	Status       string `json:"status"`
	IsReplay     bool   `json:"is_replay"`
}

// ============================================================================
// Import Repository Interface (for the student Importer)
// ============================================================================

// ============================================================================
// Check Duplicates Types
// ============================================================================

// CheckDuplicatesRequest is the body for POST /api/v1/students/check-duplicates.
// All fields are optional; only provided values are checked.
type CheckDuplicatesRequest struct {
	AdmissionNumbers      []string `json:"admission_numbers,omitempty"`
	UPINumbers            []string `json:"upi_numbers,omitempty"`
	KNECAssessmentNumbers []string `json:"knec_assessment_numbers,omitempty"`
}

// CheckDuplicatesResponse returns only the values that already exist for the
// caller's tenant/school.
type CheckDuplicatesResponse struct {
	ExistingAdmissionNumbers      []string `json:"existing_admission_numbers"`
	ExistingUPINumbers            []string `json:"existing_upi_numbers"`
	ExistingKNECAssessmentNumbers []string `json:"existing_knec_assessment_numbers"`
}

// ============================================================================
// Import Repository Interface (for the student Importer)
// ============================================================================

// ImportRepository defines what the student Importer needs from the DB.
type ImportRepository interface {
	// CheckSchoolAdminMembership verifies the caller has SCHOOL_ADMIN for the school.
	CheckSchoolAdminMembership(ctx context.Context, userID, tenantID, schoolID string) (bool, error)

	// ValidateClassExists checks that a class exists and belongs to the given
	// tenant and school. Used in ResolveReferences to fail-fast on invalid or
	// cross-tenant class references rather than allowing a FK violation deeper
	// in the insert path.
	ValidateClassExists(ctx context.Context, tenantID, schoolID, classID string) (bool, error)

	// CheckExistingFieldValues returns subsets of the input lists that already
	// exist in cbc_students for the given tenant/school. Each returned list
	// contains only the values that were found in the database. Empty/nil lists
	// are returned when no values match.
	CheckExistingFieldValues(ctx context.Context, tenantID, schoolID string,
		admissionNumbers, upiNumbers, knecNumbers []string) (
		existingAdmissionNumbers, existingUPINumbers, existingKnecNumbers []string, err error)
}

// ============================================================================
// Repository Interface
// ============================================================================

// StudentRepository defines the data access contract.
type StudentRepository interface {
	List(ctx context.Context, filter ListFilter) ([]Student, int, error)
	GetByID(ctx context.Context, id, tenantID, schoolID string) (*Student, error)
	Create(ctx context.Context, student *Student) (string, error)
	CreateBatch(ctx context.Context, students []*Student) ([]string, error)
	Update(ctx context.Context, student *Student) error
	Delete(ctx context.Context, id, tenantID, schoolID string) error
	GetDetail(ctx context.Context, id, tenantID, schoolID string) (*StudentDetail, error)

	// Enrollments
	CreateEnrollment(ctx context.Context, enrollment *Enrollment) (string, error)
	CreateBatchEnrollments(ctx context.Context, enrollments []*Enrollment, tenantID, schoolID string) ([]string, error)
	ListEnrollments(ctx context.Context, studentID, tenantID string) ([]Enrollment, error)
	IsEnrolledInTerm(ctx context.Context, studentID, academicTermID, tenantID string) (bool, error)
}
