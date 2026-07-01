package curriculum

import (
	"context"
	"fmt"

	"somotracker/backend/internal/middleware"
)

// Sentinel domain errors.
var (
	ErrNotFound      = fmt.Errorf("curriculum not found: %w", middleware.ErrNotFound)
	ErrAlreadyExists = fmt.Errorf("curriculum already exists: %w", middleware.ErrAlreadyExists)
	ErrInvalidInput  = fmt.Errorf("invalid curriculum input: %w", middleware.ErrInvalidInput)
	ErrUnauthorized  = fmt.Errorf("unauthorized: %w", middleware.ErrUnauthorized)
	ErrForbidden     = fmt.Errorf("forbidden: %w", middleware.ErrForbidden)
	ErrConflict      = fmt.Errorf("curriculum conflict: %w", middleware.ErrConflict)
	// ErrReferenceProtected is returned when a deletion is blocked by an FK constraint.
	ErrReferenceProtected = fmt.Errorf("curriculum reference protected: %w", middleware.ErrConflict)
)

// ── Repository ────────────────────────────────────────────────────────────

// Repository defines the contract for curriculum persistence.
type Repository interface {
	// Learning Areas
	CreateLearningArea(ctx context.Context, params CreateLearningAreaParams) (string, error)
	GetLearningAreaByID(ctx context.Context, id, tenantID, schoolID string) (*LearningArea, error)
	ListLearningAreas(ctx context.Context, tenantID, schoolID string, educationLevel *string) ([]LearningArea, error)
	UpdateLearningArea(ctx context.Context, params UpdateLearningAreaParams) error
	DeleteLearningArea(ctx context.Context, id, tenantID, schoolID string) error

	// Strands
	CreateStrand(ctx context.Context, params CreateStrandParams) (string, error)
	GetStrandByID(ctx context.Context, id string) (*Strand, error)
	ListStrandsByLearningArea(ctx context.Context, learningAreaID string) ([]Strand, error)
	UpdateStrand(ctx context.Context, params UpdateStrandParams) error
	DeleteStrand(ctx context.Context, id string) error

	// Sub-Strands
	CreateSubStrand(ctx context.Context, params CreateSubStrandParams) (string, error)
	GetSubStrandByID(ctx context.Context, id string) (*SubStrand, error)
	ListSubStrandsByStrand(ctx context.Context, strandID string) ([]SubStrand, error)
	UpdateSubStrand(ctx context.Context, params UpdateSubStrandParams) error
	DeleteSubStrand(ctx context.Context, id string) error

	// Performance Indicators
	CreatePerformanceIndicator(ctx context.Context, params CreatePerformanceIndicatorParams) (string, error)
	GetPerformanceIndicatorByID(ctx context.Context, id string) (*PerformanceIndicator, error)
	ListPerformanceIndicatorsBySubStrand(ctx context.Context, subStrandID string) ([]PerformanceIndicator, error)
	UpdatePerformanceIndicator(ctx context.Context, params UpdatePerformanceIndicatorParams) error
	DeletePerformanceIndicator(ctx context.Context, id string) error
	GetMaxSequenceOrder(ctx context.Context, subStrandID string) (int, error)

	// Tree
	GetTree(ctx context.Context, learningAreaID string) (*LearningAreaTree, error)

	// Parent-validation helpers (for tenant/school isolation)
	VerifyLearningAreaBelongsToTenant(ctx context.Context, id, tenantID, schoolID string) error
	VerifyStrandInTenantSchool(ctx context.Context, strandID, tenantID, schoolID string) (string, error)       // returns learning_area_id
	VerifySubStrandInTenantSchool(ctx context.Context, subStrandID, tenantID, schoolID string) (string, error) // returns strand_id

	// Cross-domain: resolves a performance indicator's education level by
	// traversing sub_strand → strand → learning_area → education_level.
	GetPerformanceIndicatorEducationLevel(ctx context.Context, indicatorID string) (string, error)
}

// ── Domain Models ─────────────────────────────────────────────────────────

// LearningArea represents a CBC learning area (subject) taught at a school.
type LearningArea struct {
	ID             string `json:"id"`
	TenantID       string `json:"-"`
	SchoolID       string `json:"-"`
	Name           string `json:"name"`
	Code           string `json:"code"`
	EducationLevel string `json:"education_level"`
}

// Strand represents a CBC strand within a learning area.
type Strand struct {
	ID             string `json:"id"`
	LearningAreaID string `json:"learning_area_id"`
	Name           string `json:"name"`
}

// SubStrand represents a CBC sub-strand within a strand.
type SubStrand struct {
	ID       string `json:"id"`
	StrandID string `json:"strand_id"`
	Name     string `json:"name"`
}

// PerformanceIndicator represents an atomic CBC learning outcome within a sub-strand.
type PerformanceIndicator struct {
	ID            string `json:"id"`
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
}

// UpdateLearningAreaParams holds fields that can be updated on a learning area.
type UpdateLearningAreaParams struct {
	ID             string
	TenantID       string
	SchoolID       string
	Name           *string
	Code           *string
	EducationLevel *string
}

// CreateStrandParams holds the fields needed to create a strand.
type CreateStrandParams struct {
	LearningAreaID string
	Name           string
}

// UpdateStrandParams holds fields that can be updated on a strand.
type UpdateStrandParams struct {
	ID   string
	Name *string
}

// CreateSubStrandParams holds the fields needed to create a sub-strand.
type CreateSubStrandParams struct {
	StrandID string
	Name     string
}

// UpdateSubStrandParams holds fields that can be updated on a sub-strand.
type UpdateSubStrandParams struct {
	ID   string
	Name *string
}

// CreatePerformanceIndicatorParams holds the fields needed to create a performance indicator.
type CreatePerformanceIndicatorParams struct {
	SubStrandID   string
	Description   string
	SequenceOrder *int // nil means auto-increment (last+1)
}

// UpdatePerformanceIndicatorParams holds fields that can be updated on a performance indicator.
type UpdatePerformanceIndicatorParams struct {
	ID            string
	Description   *string
	SequenceOrder *int
}

// ── Request Payloads (HTTP) ───────────────────────────────────────────────

// Learning Area payloads
type CreateLearningAreaPayload struct {
	Code           string `json:"code"`
	Name           string `json:"name"`
	EducationLevel string `json:"education_level"`
}

type UpdateLearningAreaPayload struct {
	Name           *string `json:"name,omitempty"`
	Code           *string `json:"code,omitempty"`
	EducationLevel *string `json:"education_level,omitempty"`
}

type ListLearningAreasResponse struct {
	LearningAreas []LearningArea `json:"learning_areas"`
	Total         int            `json:"total"`
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
	Strands []Strand `json:"strands"`
	Total   int      `json:"total"`
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
	SubStrands []SubStrand `json:"sub_strands"`
	Total      int         `json:"total"`
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
	PerformanceIndicators []PerformanceIndicator `json:"performance_indicators"`
	Total                 int                    `json:"total"`
}
