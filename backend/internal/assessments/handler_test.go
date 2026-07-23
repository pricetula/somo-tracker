package assessments

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// ============================================================================
// Handler Test Harness
// ============================================================================

type handlerTestHarness struct {
	app     *fiber.App
	svc     *Service
	repo    *mockRepo
	handler *Handler
}

func newHandlerTestHarness() *handlerTestHarness {
	repo := &mockRepo{}
	svc := NewService(repo)
	handler := NewHandler(svc)

	app := fiber.New()

	// Test middleware that sets auth locals (bypasses requireAuth)
	testAuth := func(c *fiber.Ctx) error {
		c.Locals("tenant_id", "tenant_001")
		c.Locals("user_id", "user_001")
		c.Locals("active_school_id", "school_001")
		return c.Next()
	}

	// Register weight-config routes with test auth
	wcfg := app.Group("/api/v1/assessments/weight-configs", testAuth)
	wcfg.Get("/", handler.ListWeightConfigs)
	wcfg.Get("/:id", handler.GetWeightConfig)
	wcfg.Post("/", handler.CreateWeightConfig)

	return &handlerTestHarness{
		app:     app,
		svc:     svc,
		repo:    repo,
		handler: handler,
	}
}

func doRequest(app *fiber.App, method, path string, body []byte) *http.Response {
	req := httptest.NewRequest(method, path, nil)
	if body != nil {
		req = httptest.NewRequest(method, path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	resp, _ := app.Test(req)
	return resp
}

// ============================================================================
// Helpers
// ============================================================================

func newValidWeightPayload() []byte {
	p := CreateWeightConfigPayload{
		GradeLevel:         "GRADE_4",
		AssessmentTypeCode: "KNEC_SBA_Project",
		TargetExam:         "KPSEA",
		WeightPercent:      20.0,
		EffectiveFrom:      2026,
		Notes:              strPtr("Test KNEC weight config"),
	}
	b, _ := json.Marshal(p)
	return b
}

// decodeError decodes the error response body into a map.
func decodeError(t *testing.T, resp *http.Response) map[string]interface{} {
	t.Helper()
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	return result
}

// ============================================================================
// Tests: POST /api/v1/assessments/weight-configs (CreateWeightConfig)
// ============================================================================

func TestHandler_CreateWeightConfig_HappyPath(t *testing.T) {
	h := newHandlerTestHarness()

	h.repo.createWeightConfigFn = func(ctx context.Context, params CreateWeightConfigParams) (string, error) {
		if params.GradeLevel != "GRADE_4" {
			t.Errorf("expected grade_level 'GRADE_4', got %q", params.GradeLevel)
		}
		if params.AssessmentTypeCode != "KNEC_SBA_Project" {
			t.Errorf("expected assessment_type_code 'KNEC_SBA_Project', got %q", params.AssessmentTypeCode)
		}
		if params.TargetExam != "KPSEA" {
			t.Errorf("expected target_exam 'KPSEA', got %q", params.TargetExam)
		}
		if params.WeightPercent != 20.0 {
			t.Errorf("expected weight_percent 20.0, got %f", params.WeightPercent)
		}
		if params.EffectiveFrom != 2026 {
			t.Errorf("expected effective_from 2026, got %d", params.EffectiveFrom)
		}
		if params.Notes == nil || *params.Notes != "Test KNEC weight config" {
			t.Errorf("expected notes 'Test KNEC weight config', got %v", params.Notes)
		}
		return "cfg_new_001", nil
	}

	resp := doRequest(h.app, "POST", "/api/v1/assessments/weight-configs", newValidWeightPayload())

	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", resp.StatusCode)
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["id"] != "cfg_new_001" {
		t.Fatalf("expected id 'cfg_new_001', got %q", body["id"])
	}
}

func TestHandler_CreateWeightConfig_InvalidJSON(t *testing.T) {
	h := newHandlerTestHarness()

	resp := doRequest(h.app, "POST", "/api/v1/assessments/weight-configs", []byte(`{invalid json`))

	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		t.Fatalf("expected 422 Unprocessable Entity, got %d", resp.StatusCode)
	}

	errBody := decodeError(t, resp)
	if errBody["code"] != "invalid_input" {
		t.Fatalf("expected code 'invalid_input', got %q", errBody["code"])
	}
}

func TestHandler_CreateWeightConfig_MissingGradeLevel(t *testing.T) {
	h := newHandlerTestHarness()

	payload := []byte(`{
		"assessment_type_code": "KNEC_SBA_Project",
		"target_exam": "KPSEA",
		"weight_percent": 20.0,
		"effective_from": 2026
	}`)

	resp := doRequest(h.app, "POST", "/api/v1/assessments/weight-configs", payload)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", resp.StatusCode)
	}

	errBody := decodeError(t, resp)
	if errBody["code"] != "invalid_input" {
		t.Fatalf("expected code 'invalid_input', got %q", errBody["code"])
	}
}

func TestHandler_CreateWeightConfig_MissingAssessmentTypeCode(t *testing.T) {
	h := newHandlerTestHarness()

	payload := []byte(`{
		"grade_level": "GRADE_4",
		"target_exam": "KPSEA",
		"weight_percent": 20.0,
		"effective_from": 2026
	}`)

	resp := doRequest(h.app, "POST", "/api/v1/assessments/weight-configs", payload)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", resp.StatusCode)
	}

	errBody := decodeError(t, resp)
	if errBody["code"] != "invalid_input" {
		t.Fatalf("expected code 'invalid_input', got %q", errBody["code"])
	}
}

func TestHandler_CreateWeightConfig_WeightPercentTooHigh(t *testing.T) {
	h := newHandlerTestHarness()

	payload := []byte(`{
		"grade_level": "GRADE_4",
		"assessment_type_code": "KNEC_SBA_Project",
		"target_exam": "KPSEA",
		"weight_percent": 150,
		"effective_from": 2026
	}`)

	resp := doRequest(h.app, "POST", "/api/v1/assessments/weight-configs", payload)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", resp.StatusCode)
	}

	errBody := decodeError(t, resp)
	if errBody["code"] != "invalid_input" {
		t.Fatalf("expected code 'invalid_input', got %q", errBody["code"])
	}
}

func TestHandler_CreateWeightConfig_Duplicate(t *testing.T) {
	h := newHandlerTestHarness()

	h.repo.createWeightConfigFn = func(ctx context.Context, params CreateWeightConfigParams) (string, error) {
		return "", ErrAlreadyExists
	}

	resp := doRequest(h.app, "POST", "/api/v1/assessments/weight-configs", newValidWeightPayload())

	if resp.StatusCode != fiber.StatusConflict {
		t.Fatalf("expected 409 Conflict, got %d", resp.StatusCode)
	}

	errBody := decodeError(t, resp)
	if errBody["code"] != "already_exists" {
		t.Fatalf("expected code 'already_exists', got %q", errBody["code"])
	}
}

func TestHandler_CreateWeightConfig_RepoError(t *testing.T) {
	h := newHandlerTestHarness()

	h.repo.createWeightConfigFn = func(ctx context.Context, params CreateWeightConfigParams) (string, error) {
		return "", fiber.ErrInternalServerError
	}

	resp := doRequest(h.app, "POST", "/api/v1/assessments/weight-configs", newValidWeightPayload())

	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("expected 500 Internal Server Error, got %d", resp.StatusCode)
	}
}

// ============================================================================
// Tests: GET /api/v1/assessments/weight-configs (ListWeightConfigs)
// ============================================================================

func TestHandler_ListWeightConfigs_HappyPath(t *testing.T) {
	h := newHandlerTestHarness()

	expectedConfigs := []AssessmentWeightConfig{
		{ID: "cfg_001", GradeLevel: "GRADE_4", AssessmentTypeCode: "KNEC_SBA_Project", TargetExam: "KPSEA", WeightPercent: 20.0, EffectiveFrom: 2026},
		{ID: "cfg_002", GradeLevel: "GRADE_4", AssessmentTypeCode: "National_KPSEA", TargetExam: "KPSEA", WeightPercent: 40.0, EffectiveFrom: 2026},
	}

	h.repo.listWeightConfigsFn = func(ctx context.Context, filter AssessmentWeightConfigFilter) ([]AssessmentWeightConfig, error) {
		return expectedConfigs, nil
	}

	resp := doRequest(h.app, "GET", "/api/v1/assessments/weight-configs", nil)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var result ListWeightConfigsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result.Items))
	}
	if result.Items[0].AssessmentTypeCode != "KNEC_SBA_Project" {
		t.Fatalf("expected first item type 'KNEC_SBA_Project', got %q", result.Items[0].AssessmentTypeCode)
	}
}

func TestHandler_ListWeightConfigs_WithFilters(t *testing.T) {
	h := newHandlerTestHarness()

	h.repo.listWeightConfigsFn = func(ctx context.Context, filter AssessmentWeightConfigFilter) ([]AssessmentWeightConfig, error) {
		if filter.GradeLevel == nil || *filter.GradeLevel != "GRADE_7" {
			t.Errorf("expected grade_level filter 'GRADE_7', got %v", filter.GradeLevel)
		}
		if filter.TargetExam == nil || *filter.TargetExam != "KJSEA" {
			t.Errorf("expected target_exam filter 'KJSEA', got %v", filter.TargetExam)
		}
		return []AssessmentWeightConfig{
			{ID: "cfg_010", GradeLevel: "GRADE_7", AssessmentTypeCode: "KNEC_SBA_Project", TargetExam: "KJSEA", WeightPercent: 20.0, EffectiveFrom: 2024},
		}, nil
	}

	resp := doRequest(h.app, "GET", "/api/v1/assessments/weight-configs?grade_level=GRADE_7&target_exam=KJSEA", nil)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var result ListWeightConfigsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}
	if result.Items[0].ID != "cfg_010" {
		t.Fatalf("expected id 'cfg_010', got %q", result.Items[0].ID)
	}
}

func TestHandler_ListWeightConfigs_Empty(t *testing.T) {
	h := newHandlerTestHarness()

	h.repo.listWeightConfigsFn = func(ctx context.Context, filter AssessmentWeightConfigFilter) ([]AssessmentWeightConfig, error) {
		return []AssessmentWeightConfig{}, nil
	}

	resp := doRequest(h.app, "GET", "/api/v1/assessments/weight-configs", nil)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var result ListWeightConfigsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(result.Items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(result.Items))
	}
}

// ============================================================================
// Tests: GET /api/v1/assessments/weight-configs/:id (GetWeightConfig)
// ============================================================================

func TestHandler_GetWeightConfig_HappyPath(t *testing.T) {
	h := newHandlerTestHarness()

	expected := &AssessmentWeightConfig{
		ID: "cfg_001", GradeLevel: "GRADE_4", AssessmentTypeCode: "KNEC_SBA_Project",
		TargetExam: "KPSEA", WeightPercent: 20.0, EffectiveFrom: 2026,
	}

	h.repo.getWeightConfigByIDFn = func(ctx context.Context, id string) (*AssessmentWeightConfig, error) {
		if id != "cfg_001" {
			t.Errorf("expected id 'cfg_001', got %q", id)
		}
		return expected, nil
	}

	resp := doRequest(h.app, "GET", "/api/v1/assessments/weight-configs/cfg_001", nil)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var result AssessmentWeightConfig
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result.ID != "cfg_001" {
		t.Fatalf("expected id 'cfg_001', got %q", result.ID)
	}
	if result.WeightPercent != 20.0 {
		t.Fatalf("expected weight_percent 20.0, got %f", result.WeightPercent)
	}
}

func TestHandler_GetWeightConfig_NotFound(t *testing.T) {
	h := newHandlerTestHarness()

	h.repo.getWeightConfigByIDFn = func(ctx context.Context, id string) (*AssessmentWeightConfig, error) {
		return nil, ErrNotFound
	}

	resp := doRequest(h.app, "GET", "/api/v1/assessments/weight-configs/cfg_999", nil)

	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d", resp.StatusCode)
	}

	errBody := decodeError(t, resp)
	if errBody["code"] != "not_found" {
		t.Fatalf("expected code 'not_found', got %q", errBody["code"])
	}
}

func TestHandler_GetWeightConfig_InvalidID(t *testing.T) {
	h := newHandlerTestHarness()

	h.repo.getWeightConfigByIDFn = func(ctx context.Context, id string) (*AssessmentWeightConfig, error) {
		return nil, ErrNotFound
	}

	// Use a non-existent but well-formed UUID to trigger a 404 from the handler
	resp := doRequest(h.app, "GET", "/api/v1/assessments/weight-configs/nonexistent-id", nil)

	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d", resp.StatusCode)
	}

	errBody := decodeError(t, resp)
	if errBody["code"] != "not_found" {
		t.Fatalf("expected code 'not_found', got %q", errBody["code"])
	}
}
