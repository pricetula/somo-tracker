package students

import (
	"context"
)

// Student represents a full student record returned by the listing endpoint.
type Student struct {
	ID                   string  `json:"id"`
	FullName             string  `json:"full_name"`
	Gender               string  `json:"gender"`
	DateOfBirth          *string `json:"date_of_birth,omitempty"`
	UPINumber            *string `json:"upi_number,omitempty"`
	KNECAssessmentNumber *string `json:"knec_assessment_number,omitempty"`
	ClassName            *string `json:"class_name,omitempty"`
	ClassID              *string `json:"class_id,omitempty"`
	IsActive             bool    `json:"is_active"`
	CreatedAt            string  `json:"created_at"`
}

// ListStudentsResponse is the paginated response body.
type ListStudentsResponse struct {
	Students []Student `json:"students"`
	Total    int       `json:"total"`
	Page     int       `json:"page"`
	Limit    int       `json:"limit"`
}

// ListFilter holds query parameters for listing students.
type ListFilter struct {
	TenantID string
	SchoolID string
	Page     int
	Limit    int
	Search   string
	ClassID  string
	Gender   string
}

// StudentRepository defines the data access contract.
type StudentRepository interface {
	List(ctx context.Context, filter ListFilter) ([]Student, int, error)
}
