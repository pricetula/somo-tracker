package classes

// ListClassesParams holds optional filter criteria for listing classes.
type ListClassesParams struct {
	GradeIDs []string `json:"grade_ids"`
	Search   string   `json:"search"`
	IsActive *bool    `json:"is_active"`
}

// Class represents a single classroom entity (e.g., "Grade 1 East").
type Class struct {
	ID                string `json:"id,omitempty"`
	TenantID          string `json:"tenant_id,omitempty"`
	SchoolID          string `json:"school_id,omitempty"`
	AcademicYearID    string `json:"academic_year_id,omitempty"`
	EducationSystemID string `json:"education_system_id,omitempty"`
	GradeID           string `json:"grade_id,omitempty"`
	Name              string `json:"name"`
	Stream            string `json:"stream,omitempty"`
	IsActive          bool   `json:"is_active"`
	CreatedAt         string `json:"created_at,omitempty"`
}

// GradeInfo is a lightweight projection of a grade record.
type GradeInfo struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	SequenceOrder int    `json:"sequence_order"`
}

// ClassroomPreview is a lightweight projection returned for the onboarding preview grid.
type ClassroomPreview struct {
	Name string `json:"name"`
}

// GeneratePayload is the request body for POST /api/v1/schools/classes/generate.
// The user provides stream names; the backend cross-multiplies with the
// school's education system grades and atomic-inserts all classrooms.
type GeneratePayload struct {
	Streams []string `json:"streams"`
}

// GenerateResult is the response body for the generate endpoint.
type GenerateResult struct {
	Classes      []Class  `json:"classes"`
	TotalCreated int      `json:"total_created"`
	Streams      []string `json:"streams"`
	GradeNames   []string `json:"grade_names"`
}

// ErrorBody is the JSON error response body.
type ErrorBody struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// ─── Internal helpers ──────────────────────────────────────────────────────

// gradeAssignment holds a resolved grade record from the database.
type gradeAssignment struct {
	ID   string
	Name string
}

// yearAssignment holds the current academic year record.
type yearAssignment struct {
	ID                string
	EducationSystemID string
}

// classInput is an internal struct for passing bulk-insert parameters.
type classInput struct {
	GradeID string
	Name    string
	Stream  string
}
