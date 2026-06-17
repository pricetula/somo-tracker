package school

// School represents a school within a tenant.
type School struct {
	ID                string `db:"id"                  json:"id"`
	TenantID          string `db:"tenant_id"           json:"tenant_id"`
	EducationSystemID string `db:"education_system_id" json:"education_system_id"`
	Name              string `db:"name"                json:"name"`
	IsActive          bool   `db:"is_active"           json:"is_active"`
	IsDemo            bool   `db:"is_demo"             json:"is_demo"`
}

// CreateSchoolPayload is the request body for POST /schools.
type CreateSchoolPayload struct {
	Name              string `json:"name"`
	EducationSystemID string `json:"education_system_id"`
}

// UpdateSchoolPayload is the request body for PUT /schools/:id.
type UpdateSchoolPayload struct {
	Name string `json:"name"`
}

// ErrorBody is the JSON error response body.
type ErrorBody struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}
