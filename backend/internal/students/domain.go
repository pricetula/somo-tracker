package students

import "time"

// ─── Core domain types ────────────────────────────────────────────────────

// Student represents a single student record.
type Student struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	FirstName   string    `json:"first_name"`
	MiddleName  *string   `json:"middle_name,omitempty"`
	LastName    string    `json:"last_name"`
	Gender      string    `json:"gender"`
	DateOfBirth string    `json:"date_of_birth"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
}

// CreateStudentPayload is the request body for POST /api/v1/students.
type CreateStudentPayload struct {
	FirstName   string  `json:"first_name"`
	MiddleName  *string `json:"middle_name,omitempty"`
	LastName    string  `json:"last_name"`
	Gender      string  `json:"gender"`
	DateOfBirth string  `json:"date_of_birth"`
}

// ─── CSV Import types ─────────────────────────────────────────────────────

// ImportProgress represents a single SSE progress event.
type ImportProgress struct {
	Status  string `json:"status"`            // "processing" | "completed" | "error"
	Current int    `json:"current,omitempty"` // rows processed so far
	Total   int    `json:"total,omitempty"`   // total rows in CSV
	Success int    `json:"success,omitempty"` // rows inserted (terminal)
	Failed  int    `json:"failed,omitempty"`  // rows rejected (terminal)
	Error   string `json:"error,omitempty"`   // error download URL (terminal)
}

// CSVRawRow is an intermediate representation for parsed-but-unvalidated rows.
type CSVRawRow struct {
	FirstName   string
	MiddleName  string
	LastName    string
	Gender      string
	DateOfBirth string
	LineNumber  int
}

// ─── HTTP types ───────────────────────────────────────────────────────────

// ListResponse wraps a paginated student list.
type ListResponse struct {
	Students []Student `json:"students"`
	Total    int       `json:"total"`
}

// ImportResponse is returned by POST /api/v1/students/import (HTTP 202).
type ImportResponse struct {
	ImportID string `json:"import_id"`
}

// ErrorBody is the JSON error response body.
type ErrorBody struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}
