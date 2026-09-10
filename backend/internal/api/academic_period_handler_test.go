package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"somotracker/backend/internal/services"
)

// mockAcademicPeriodService stubs services.AcademicPeriodService for transport-layer tests.
type mockAcademicPeriodService struct {
	calls  []academicPeriodCall
	result string
	err    error
}

type academicPeriodCall struct {
	schoolID string
	req      services.AcademicPeriodRequest
}

func (m *mockAcademicPeriodService) CreateAcademicPeriod(ctx context.Context, schoolID string, req services.AcademicPeriodRequest) (string, error) {
	m.calls = append(m.calls, academicPeriodCall{schoolID: schoolID, req: req})

	// Mimic service validation for tests that need it
	if schoolID == "" {
		return "", fmt.Errorf("bad_request: school_id is required")
	}
	if req.Year == 0 {
		return "", fmt.Errorf("bad_request: year is required")
	}
	if len(req.Terms) == 0 {
		return "", fmt.Errorf("bad_request: at least one term is required")
	}

	return m.result, m.err
}

// injectAcademicPeriodSchoolMW injects active_school_id into locals for authenticated routes.
func injectAcademicPeriodSchoolMW(schoolID string) fiber.Handler {
	return func(c fiber.Ctx) error {
		c.Locals("active_school_id", schoolID)
		return c.Next()
	}
}

func newAcademicPeriodTestApp(mock *mockAcademicPeriodService, schoolID string) *fiber.App {
	app := fiber.New()
	app.Use(injectAcademicPeriodSchoolMW(schoolID))
	h := NewAcademicPeriodHandler(mock)
	app.Post("/api/school/academic-period", h.CreateAcademicPeriod)
	return app
}

func sendAcademicPeriodRequest(t *testing.T, app *fiber.App, body []byte) *http.Response {
	t.Helper()
	req := httptest.NewRequest("POST", "/api/school/academic-period", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	return resp
}

func readAcademicPeriodJSONMap(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	defer func() { _ = resp.Body.Close() }()
	var got map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
	return got
}

func TestCreateAcademicPeriod_HappyPath_Returns201(t *testing.T) {
	mock := &mockAcademicPeriodService{
		result: "academic-year-123",
	}
	app := newAcademicPeriodTestApp(mock, "school-42")

	body := []byte(`{
		"year": 2026,
		"terms": [
			{"name": "Term 1", "start_date": "2026-01-15", "end_date": "2026-04-15"},
			{"name": "Term 2", "start_date": "2026-05-01", "end_date": "2026-08-15"},
			{"name": "Term 3", "start_date": "2026-09-01", "end_date": "2026-12-15"}
		]
	}`)
	resp := sendAcademicPeriodRequest(t, app, body)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)
	require.Len(t, mock.calls, 1)
	assert.Equal(t, "school-42", mock.calls[0].schoolID)
	assert.Equal(t, 2026, mock.calls[0].req.Year)
	assert.Len(t, mock.calls[0].req.Terms, 3)

	respBody := readAcademicPeriodJSONMap(t, resp)
	assert.Equal(t, "academic_period_created", respBody["code"])
	assert.Equal(t, "Academic period created successfully", respBody["message"])
	assert.Equal(t, "academic-year-123", respBody["academic_year_id"])
}

func TestCreateAcademicPeriod_MissingSchoolID_Returns401(t *testing.T) {
	mock := &mockAcademicPeriodService{}
	// No middleware to inject schoolID
	app := fiber.New()
	h := NewAcademicPeriodHandler(mock)
	app.Post("/api/school/academic-period", h.CreateAcademicPeriod)

	body := []byte(`{"year": 2026, "terms": [{"name": "Term 1", "start_date": "2026-01-15", "end_date": "2026-04-15"}]}`)
	resp := sendAcademicPeriodRequest(t, app, body)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	assert.Empty(t, mock.calls, "service must not be called when schoolID is missing")
	// fiber.NewError returns plain text by default, not JSON
}

func TestCreateAcademicPeriod_InvalidJSON_Returns400(t *testing.T) {
	mock := &mockAcademicPeriodService{}
	app := newAcademicPeriodTestApp(mock, "school-42")

	body := []byte(`not valid json`)
	resp := sendAcademicPeriodRequest(t, app, body)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	assert.Empty(t, mock.calls)
	// fiber.NewError returns plain text for invalid JSON
}

func TestCreateAcademicPeriod_MissingYear_Returns400(t *testing.T) {
	mock := &mockAcademicPeriodService{}
	app := newAcademicPeriodTestApp(mock, "school-42")

	// Year is 0 (zero value) - mock validates and returns bad_request
	body := []byte(`{"year": 0, "terms": [{"name": "Term 1", "start_date": "2026-01-15", "end_date": "2026-04-15"}]}`)
	resp := sendAcademicPeriodRequest(t, app, body)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	require.Len(t, mock.calls, 1)
}

func TestCreateAcademicPeriod_MissingTerms_Returns400(t *testing.T) {
	mock := &mockAcademicPeriodService{}
	app := newAcademicPeriodTestApp(mock, "school-42")

	// Empty terms array - mock validates and returns bad_request
	body := []byte(`{"year": 2026, "terms": []}`)
	resp := sendAcademicPeriodRequest(t, app, body)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	require.Len(t, mock.calls, 1)
}

func TestCreateAcademicPeriod_ServiceBadRequest_Returns400(t *testing.T) {
	mock := &mockAcademicPeriodService{err: academicPeriodAssertAnError{msg: "bad_request: invalid school_id"}}
	app := newAcademicPeriodTestApp(mock, "school-42")

	body := []byte(`{"year": 2026, "terms": [{"name": "Term 1", "start_date": "2026-01-15", "end_date": "2026-04-15"}]}`)
	resp := sendAcademicPeriodRequest(t, app, body)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	// fiber.NewError returns plain text by default
}

func TestCreateAcademicPeriod_ServiceInternalError_Returns500(t *testing.T) {
	mock := &mockAcademicPeriodService{err: academicPeriodAssertAnError{msg: "db connection failed"}}
	app := newAcademicPeriodTestApp(mock, "school-42")

	body := []byte(`{"year": 2026, "terms": [{"name": "Term 1", "start_date": "2026-01-15", "end_date": "2026-04-15"}]}`)
	resp := sendAcademicPeriodRequest(t, app, body)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	// fiber.NewError returns plain text by default
}

func TestCreateAcademicPeriod_InvalidTermDates_Returns400(t *testing.T) {
	mock := &mockAcademicPeriodService{err: academicPeriodAssertAnError{msg: "bad_request: invalid term start_date 2026-13-01: parsing time"}}
	app := newAcademicPeriodTestApp(mock, "school-42")

	body := []byte(`{"year": 2026, "terms": [{"name": "Term 1", "start_date": "2026-13-01", "end_date": "2026-04-15"}]}`)
	resp := sendAcademicPeriodRequest(t, app, body)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	// fiber.NewError returns plain text by default
}

func TestCreateAcademicPeriod_HandlerPassesRequestToService(t *testing.T) {
	// Verify the handler passes the request to the service as-is
	// The service's internal sorting is an implementation detail
	mock := &mockAcademicPeriodService{result: "year-1"}
	app := newAcademicPeriodTestApp(mock, "school-42")

	// Terms provided out of order
	body := []byte(`{
		"year": 2026,
		"terms": [
			{"name": "Term 3", "start_date": "2026-09-01", "end_date": "2026-12-15"},
			{"name": "Term 1", "start_date": "2026-01-15", "end_date": "2026-04-15"},
			{"name": "Term 2", "start_date": "2026-05-01", "end_date": "2026-08-15"}
		]
	}`)
	resp := sendAcademicPeriodRequest(t, app, body)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)
	require.Len(t, mock.calls, 1)
	// Handler passes request as-is to service (service sorts internally)
	terms := mock.calls[0].req.Terms
	assert.Equal(t, "Term 3", terms[0].Name)
	assert.Equal(t, "Term 1", terms[1].Name)
	assert.Equal(t, "Term 2", terms[2].Name)
}

// Helper error type for testing
type academicPeriodAssertAnError struct{ msg string }

func (a academicPeriodAssertAnError) Error() string { return a.msg }
