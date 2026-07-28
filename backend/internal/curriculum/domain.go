package curriculum

import (
	"context"

	"somotracker/backend/internal/xerrors"
)

// Sentinel domain errors.
var (
	ErrNotFound      = xerrors.NotFound("curriculum not found")
	ErrAlreadyExists = xerrors.AlreadyExists("curriculum already exists")
	ErrInvalidInput  = xerrors.InvalidInput("invalid curriculum input")
	ErrUnauthorized  = xerrors.Unauthorized("unauthorized")
	ErrForbidden     = xerrors.Forbidden("forbidden")
	ErrConflict      = xerrors.Conflict("curriculum conflict")
	// ErrReferenceProtected is returned when a deletion is blocked by an FK constraint.
	ErrReferenceProtected = xerrors.Conflict("curriculum reference protected")
)

// ── Repository ────────────────────────────────────────────────────────────

// Repository defines the contract for curriculum persistence.
type Repository interface {
	// Learning Areas
	CreateLearningArea(ctx context.Context, params CreateLearningAreaParams) (string, error)
	GetLearningAreaByID(ctx context.Context, id, tenantID, schoolID string) (*LearningArea, error)
	ListLearningAreas(ctx context.Context, tenantID, schoolID string, educationLevels, gradeLevels []string, search string, page, limit int) ([]LearningArea, int, error)
	UpdateLearningArea(ctx context.Context, params UpdateLearningAreaParams) error
	DeleteLearningArea(ctx context.Context, id, tenantID, schoolID string) error

	// Strands
	CreateStrand(ctx context.Context, params CreateStrandParams) (string, error)
	GetStrandByID(ctx context.Context, id, tenantID string) (*Strand, error)
	ListStrandsByLearningArea(ctx context.Context, learningAreaID, tenantID string) ([]Strand, error)
	UpdateStrand(ctx context.Context, params UpdateStrandParams) error
	DeleteStrand(ctx context.Context, id, tenantID string) error

	// Sub-Strands
	CreateSubStrand(ctx context.Context, params CreateSubStrandParams) (string, error)
	GetSubStrandByID(ctx context.Context, id, tenantID string) (*SubStrand, error)
	ListSubStrandsByStrand(ctx context.Context, strandID, tenantID string) ([]SubStrand, error)
	UpdateSubStrand(ctx context.Context, params UpdateSubStrandParams) error
	DeleteSubStrand(ctx context.Context, id, tenantID string) error

	// Performance Indicators
	CreatePerformanceIndicator(ctx context.Context, params CreatePerformanceIndicatorParams) (string, error)
	GetPerformanceIndicatorByID(ctx context.Context, id, tenantID string) (*PerformanceIndicator, error)
	ListPerformanceIndicatorsBySubStrand(ctx context.Context, subStrandID, tenantID string) ([]PerformanceIndicator, error)
	UpdatePerformanceIndicator(ctx context.Context, params UpdatePerformanceIndicatorParams) error
	DeletePerformanceIndicator(ctx context.Context, id, tenantID string) error
	GetMaxSequenceOrder(ctx context.Context, subStrandID string) (int, error)

	// Tree
	GetTree(ctx context.Context, learningAreaID, tenantID string) (*LearningAreaTree, error)

	// Parent-validation helpers (for tenant/school isolation)
	VerifyLearningAreaBelongsToTenant(ctx context.Context, id, tenantID, schoolID string) error
	VerifyStrandInTenantSchool(ctx context.Context, strandID, tenantID, schoolID string) (string, error)       // returns learning_area_id
	VerifySubStrandInTenantSchool(ctx context.Context, subStrandID, tenantID, schoolID string) (string, error) // returns strand_id

	// Cross-domain: resolves a performance indicator's education level by
	// traversing sub_strand → strand → learning_area → education_level.
	GetPerformanceIndicatorEducationLevel(ctx context.Context, indicatorID string) (string, error)
}

// ── Domain Models ─────────────────────────────────────────────────────────

// LearningArea represents a CBC learning area (subject) taught at a school
// for a specific grade level.
type LearningArea struct {
	ID             string `json:"id"`
	TenantID       string `json:"-"`
	SchoolID       string `json:"-"`
	Name           string `json:"name"`
	Code           string `json:"code"`
	EducationLevel string `json:"education_level"`
	GradeLevel     string `json:"grade_level"`
}

// Strand represents a CBC strand within a learning area.
type Strand struct {
	ID             string `json:"id"`
	TenantID       string `json:"-"`
	LearningAreaID string `json:"learning_area_id"`
	Name           string `json:"name"`
}

// SubStrand represents a CBC sub-strand within a strand.
type SubStrand struct {
	ID       string `json:"id"`
	TenantID string `json:"-"`
	StrandID string `json:"strand_id"`
	Name     string `json:"name"`
}

// PerformanceIndicator represents an atomic CBC learning outcome within a sub-strand.
type PerformanceIndicator struct {
	ID            string `json:"id"`
	TenantID      string `json:"-"`
	SubStrandID   string `json:"sub_strand_id"`
	Description   string `json:"description"`
	SequenceOrder int    `json:"sequence_order"`
}

// LearningAreaTree is the full hierarchy response for a learning area.
type LearningAreaTree struct {
	LearningArea
	Strands []StrandTree `json:"strands"`
}

// StrandTree is a strand with nested sub-strands.
type StrandTree struct {
	Strand
	SubStrands []SubStrandTree `json:"sub_strands"`
}

// SubStrandTree is a sub-strand with nested performance indicators.
type SubStrandTree struct {
	SubStrand
	PerformanceIndicators []PerformanceIndicator `json:"performance_indicators"`
}

// ── Params (internal) ─────────────────────────────────────────────────────

// CreateLearningAreaParams holds the fields needed to create a learning area.
type CreateLearningAreaParams struct {
	TenantID       string
	SchoolID       string
	Name           string
	Code           string
	EducationLevel string
	GradeLevel     string
}

// UpdateLearningAreaParams holds fields that can be updated on a learning area.
type UpdateLearningAreaParams struct {
	ID             string
	TenantID       string
	SchoolID       string
	Name           *string
	Code           *string
	EducationLevel *string
	GradeLevel     *string
}

// CreateStrandParams holds the fields needed to create a strand.
type CreateStrandParams struct {
	TenantID       string
	LearningAreaID string
	Name           string
}

// UpdateStrandParams holds fields that can be updated on a strand.
type UpdateStrandParams struct {
	ID       string
	TenantID string
	Name     *string
}

// CreateSubStrandParams holds the fields needed to create a sub-strand.
type CreateSubStrandParams struct {
	TenantID string
	StrandID string
	Name     string
}

// UpdateSubStrandParams holds fields that can be updated on a sub-strand.
type UpdateSubStrandParams struct {
	ID       string
	TenantID string
	Name     *string
}

// CreatePerformanceIndicatorParams holds the fields needed to create a performance indicator.
type CreatePerformanceIndicatorParams struct {
	TenantID      string
	SubStrandID   string
	Description   string
	SequenceOrder *int // nil means auto-increment (last+1)
}

// UpdatePerformanceIndicatorParams holds fields that can be updated on a performance indicator.
type UpdatePerformanceIndicatorParams struct {
	ID            string
	TenantID      string
	Description   *string
	SequenceOrder *int
}

// ── Request Payloads (HTTP) ───────────────────────────────────────────────

// Learning Area payloads
type CreateLearningAreaPayload struct {
	Code           string `json:"code"`
	Name           string `json:"name"`
	EducationLevel string `json:"education_level"`
	GradeLevel     string `json:"grade_level"`
}

type UpdateLearningAreaPayload struct {
	Name           *string `json:"name,omitempty"`
	Code           *string `json:"code,omitempty"`
	EducationLevel *string `json:"education_level,omitempty"`
	GradeLevel     *string `json:"grade_level,omitempty"`
}

type ListLearningAreasResponse struct {
	Items []LearningArea `json:"items"`
	Total int            `json:"total"`
	Page  int            `json:"page"`
	Limit int            `json:"limit"`
}

// Strand payloads
type CreateStrandPayload struct {
	LearningAreaID string `json:"learning_area_id"`
	Name           string `json:"name"`
}

type UpdateStrandPayload struct {
	Name *string `json:"name,omitempty"`
}

type ListStrandsResponse struct {
	Items []Strand `json:"items"`
	Total int      `json:"total"`
	Page  int      `json:"page"`
	Limit int      `json:"limit"`
}

// Sub-Strand payloads
type CreateSubStrandPayload struct {
	StrandID string `json:"strand_id"`
	Name     string `json:"name"`
}

type UpdateSubStrandPayload struct {
	Name *string `json:"name,omitempty"`
}

type ListSubStrandsResponse struct {
	Items []SubStrand `json:"items"`
	Total int         `json:"total"`
	Page  int         `json:"page"`
	Limit int         `json:"limit"`
}

// Performance Indicator payloads
type CreatePerformanceIndicatorPayload struct {
	SubStrandID   string `json:"sub_strand_id"`
	Description   string `json:"description"`
	SequenceOrder *int   `json:"sequence_order,omitempty"`
}

type UpdatePerformanceIndicatorPayload struct {
	Description   *string `json:"description,omitempty"`
	SequenceOrder *int    `json:"sequence_order,omitempty"`
}

type ListPerformanceIndicatorsResponse struct {
	Items []PerformanceIndicator `json:"items"`
	Total int                    `json:"total"`
	Page  int                    `json:"page"`
	Limit int                    `json:"limit"`
}

// ── Seeding Input Structs (JSON payload) ──────────────────────────────────

// CurriculumData wraps a slice of learning areas parsed from a single JSON file.
type CurriculumData []LearningAreaInput

// LearningAreaInput mirrors the JSON payload for a CBC learning area.
type LearningAreaInput struct {
	Name           string        `json:"name"`
	Code           string        `json:"code"`
	EducationLevel string        `json:"education_level"`
	Pathway        string        `json:"pathway,omitempty"`
	Strands        []StrandInput `json:"strands"`
}

// StrandInput mirrors the JSON payload for a strand within a learning area.
type StrandInput struct {
	Name       string           `json:"name"`
	SubStrands []SubStrandInput `json:"sub_strands"`
}

// SubStrandInput mirrors the JSON payload for a sub-strand within a strand.
type SubStrandInput struct {
	Name                  string   `json:"name"`
	PerformanceIndicators []string `json:"performance_indicators"`
}

// GradeFile maps a JSON file stem to a CBC grade level.
type GradeFile struct {
	Stem  string // file stem (e.g. "pp1", "grade4", "grade10.stem")
	Grade string // normalized grade level (e.g. "PP1", "G4", "G10")
}
