package academiccalendar

// AcademicYear represents a school year with its constituent periods.
type AcademicYear struct {
	ID      string           `json:"id,omitempty"`
	Year    int              `json:"year"`
	Periods []AcademicPeriod `json:"periods"`
}

// AcademicPeriod represents a single term / period within an academic year.
type AcademicPeriod struct {
	ID        string `json:"id,omitempty"`
	Name      string `json:"name"`
	StartDate string `json:"start_date"` // "2006-01-02"
	EndDate   string `json:"end_date"`   // "2006-01-02"
	IsFinal   bool   `json:"is_final"`
}

// SavePayload is the request body for POST /api/v1/schools/current-calendar.
type SavePayload struct {
	Year    int                    `json:"year"`
	Periods []SavePeriodPayload    `json:"periods"`
}

// SavePeriodPayload represents a single period in a save request.
type SavePeriodPayload struct {
	Name      string `json:"name"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	IsFinal   bool   `json:"is_final"`
}

// ErrorBody is the JSON error response body.
type ErrorBody struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}
