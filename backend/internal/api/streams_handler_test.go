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
)

// mockStreamsService stubs services.StreamsService for transport-layer tests.
type mockStreamsService struct {
	calls  []streamsCall
	result []string
	err    error
}

type streamsCall struct {
	schoolID string
	names    []string
}

func (m *mockStreamsService) CreateStreams(ctx context.Context, schoolID string, names []string) ([]string, error) {
	m.calls = append(m.calls, streamsCall{schoolID: schoolID, names: names})

	// Mimic service validation for tests that need it
	if schoolID == "" {
		return nil, fmt.Errorf("bad_request: school_id is required")
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("bad_request: at least one stream name is required")
	}

	return m.result, m.err
}

// injectSchoolMW injects active_school_id into locals for authenticated routes.
func injectSchoolMW(schoolID string) fiber.Handler {
	return func(c fiber.Ctx) error {
		c.Locals("active_school_id", schoolID)
		return c.Next()
	}
}

func newStreamsTestApp(mock *mockStreamsService, schoolID string) *fiber.App {
	app := fiber.New()
	app.Use(injectSchoolMW(schoolID))
	h := NewStreamsHandler(mock)
	app.Post("/api/school/streams", h.CreateStreams)
	return app
}

func sendStreamsRequest(t *testing.T, app *fiber.App, body []byte) *http.Response {
	t.Helper()
	req := httptest.NewRequest("POST", "/api/school/streams", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	return resp
}

func readStreamsJSONMap(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	defer func() { _ = resp.Body.Close() }()
	var got map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
	return got
}

func TestCreateStreams_HappyPath_Returns201(t *testing.T) {
	mock := &mockStreamsService{
		result: []string{"stream-1", "stream-2"},
	}
	app := newStreamsTestApp(mock, "school-42")

	body := []byte(`["Form 1", "Form 2"]`)
	resp := sendStreamsRequest(t, app, body)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)
	require.Len(t, mock.calls, 1)
	assert.Equal(t, "school-42", mock.calls[0].schoolID)
	assert.Equal(t, []string{"Form 1", "Form 2"}, mock.calls[0].names)

	respBody := readStreamsJSONMap(t, resp)
	assert.Equal(t, "streams_created", respBody["code"])
	assert.Equal(t, "Streams created successfully", respBody["message"])
	// JSON array decodes to []interface{}, convert for comparison
	streamIDs := respBody["stream_ids"].([]interface{})
	assert.Equal(t, 2, len(streamIDs))
	assert.Equal(t, "stream-1", streamIDs[0])
	assert.Equal(t, "stream-2", streamIDs[1])
}

func TestCreateStreams_MissingSchoolID_Returns401(t *testing.T) {
	mock := &mockStreamsService{}
	// No middleware to inject schoolID
	app := fiber.New()
	h := NewStreamsHandler(mock)
	app.Post("/api/school/streams", h.CreateStreams)

	body := []byte(`["Form 1"]`)
	resp := sendStreamsRequest(t, app, body)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	assert.Empty(t, mock.calls, "service must not be called when schoolID is missing")
	// fiber.NewError returns plain text by default, not JSON
}

func TestCreateStreams_EmptyBody_Returns400(t *testing.T) {
	mock := &mockStreamsService{}
	app := newStreamsTestApp(mock, "school-42")

	body := []byte(`[]`)
	resp := sendStreamsRequest(t, app, body)
	defer func() { _ = resp.Body.Close() }()

	// Empty array passes JSON parsing but mock validates and returns bad_request
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	require.Len(t, mock.calls, 1)
}

func TestCreateStreams_InvalidJSON_Returns400(t *testing.T) {
	mock := &mockStreamsService{}
	app := newStreamsTestApp(mock, "school-42")

	body := []byte(`not valid json`)
	resp := sendStreamsRequest(t, app, body)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	assert.Empty(t, mock.calls)
	// fiber.NewError returns plain text for invalid JSON
}

func TestCreateStreams_ServiceBadRequest_Returns400(t *testing.T) {
	mock := &mockStreamsService{err: streamsAssertAnError{msg: "bad_request: school_id is required"}}
	app := newStreamsTestApp(mock, "school-42")

	body := []byte(`["Form 1"]`)
	resp := sendStreamsRequest(t, app, body)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	// fiber.NewError returns plain text by default
}

func TestCreateStreams_ServiceInternalError_Returns500(t *testing.T) {
	mock := &mockStreamsService{err: streamsAssertAnError{msg: "db connection failed"}}
	app := newStreamsTestApp(mock, "school-42")

	body := []byte(`["Form 1"]`)
	resp := sendStreamsRequest(t, app, body)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	// fiber.NewError returns plain text by default
}

func TestCreateStreams_EmptyNameInArray_Skipped(t *testing.T) {
	mock := &mockStreamsService{
		result: []string{"stream-1"},
	}
	app := newStreamsTestApp(mock, "school-42")

	// One empty name, one valid - service skips empty
	body := []byte(`["", "Form 1"]`)
	resp := sendStreamsRequest(t, app, body)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)
	require.Len(t, mock.calls, 1)
	assert.Equal(t, []string{"", "Form 1"}, mock.calls[0].names)
}

func TestCreateStreams_SingleStringNotArray_Returns400(t *testing.T) {
	mock := &mockStreamsService{}
	app := newStreamsTestApp(mock, "school-42")

	// Sending a string instead of array
	body := []byte(`"Form 1"`)
	resp := sendStreamsRequest(t, app, body)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	assert.Empty(t, mock.calls)
}

// Helper error type for testing
type streamsAssertAnError struct{ msg string }

func (a streamsAssertAnError) Error() string { return a.msg }
