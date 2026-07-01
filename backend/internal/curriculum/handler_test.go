package curriculum

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
// Test Harness
// ============================================================================

type handlerTestHarness struct {
	app     *fiber.App
	svc     *Service
	repo    *MockRepository
	handler *Handler
}

func newHandlerTestHarness(t *testing.T) *handlerTestHarness {
	t.Helper()

	repo := &MockRepository{}
	svc := NewService(repo)
	handler := NewHandler(svc)

	app := fiber.New()

	// Test middleware that sets tenant_id, user_id, and active_school_id
	// (bypasses RequireAuth for unit testing)
	testAuth := func(c *fiber.Ctx) error {
		c.Locals("tenant_id", "tenant_001")
		c.Locals("user_id", "user_001")
		c.Locals("active_school_id", "school_001")
		return c.Next()
	}

	// Register routes manually with test auth (bypasses RequireAuth for unit testing)
	areas := app.Group("/api/v1/curriculum/learning-areas", testAuth)
	areas.Post("/", handler.CreateLearningArea)
	areas.Get("/", handler.ListLearningAreas)
	areas.Get("/:id", handler.GetLearningAreaByID)
	areas.Get("/:id/tree", handler.GetTree)
	areas.Put("/:id", handler.UpdateLearningArea)
	areas.Delete("/:id", handler.DeleteLearningArea)

	strands := app.Group("/api/v1/curriculum/strands", testAuth)
	strands.Post("/", handler.CreateStrand)
	strands.Get("/", handler.ListStrands)
	strands.Put("/:id", handler.UpdateStrand)
	strands.Delete("/:id", handler.DeleteStrand)

	subStrands := app.Group("/api/v1/curriculum/sub-strands", testAuth)
	subStrands.Post("/", handler.CreateSubStrand)
	subStrands.Get("/", handler.ListSubStrands)
	subStrands.Put("/:id", handler.UpdateSubStrand)
	subStrands.Delete("/:id", handler.DeleteSubStrand)

	indicators := app.Group("/api/v1/curriculum/performance-indicators", testAuth)
	indicators.Post("/", handler.CreatePerformanceIndicator)
	indicators.Get("/", handler.ListPerformanceIndicators)
	indicators.Put("/:id", handler.UpdatePerformanceIndicator)
	indicators.Delete("/:id", handler.DeletePerformanceIndicator)

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
// Tests: Learning Areas (existing, adapted to new method names)
// ============================================================================

func TestHandler_CreateLearningArea_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.createLearningAreaFn = func(ctx context.Context, params CreateLearningAreaParams) (string, error) {
		if params.TenantID != "tenant_001" {
			t.Errorf("expected tenantID 'tenant_001', got %q", params.TenantID)
		}
		if params.SchoolID != "school_001" {
			t.Errorf("expected schoolID 'school_001', got %q", params.SchoolID)
		}
		if params.Name != "Mathematics" {
			t.Errorf("expected Name 'Mathematics', got %q", params.Name)
		}
		if params.Code != "MATH" {
			t.Errorf("expected Code 'MATH', got %q", params.Code)
		}
		if params.EducationLevel != "Junior_Secondary" {
			t.Errorf("expected EducationLevel 'Junior_Secondary', got %q", params.EducationLevel)
		}
		return "area_001", nil
	}

	body, _ := json.Marshal(CreateLearningAreaPayload{Name: "Mathematics", Code: "MATH", EducationLevel: "Junior_Secondary"})
	resp := doRequest(h.app, "POST", "/api/v1/curriculum/learning-areas", body)

	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", resp.StatusCode)
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if result.ID != "area_001" {
		t.Fatalf("expected id 'area_001', got %q", result.ID)
	}
}

func TestHandler_CreateLearningArea_MissingName(t *testing.T) {
	h := newHandlerTestHarness(t)
	body, _ := json.Marshal(CreateLearningAreaPayload{Name: "", Code: "MATH", EducationLevel: "Junior_Secondary"})
	resp := doRequest(h.app, "POST", "/api/v1/curriculum/learning-areas", body)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", resp.StatusCode)
	}
}

func TestHandler_CreateLearningArea_InvalidBody(t *testing.T) {
	h := newHandlerTestHarness(t)
	resp := doRequest(h.app, "POST", "/api/v1/curriculum/learning-areas", []byte("not json"))
	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		t.Fatalf("expected 422 Unprocessable Entity, got %d", resp.StatusCode)
	}
}

func TestHandler_ListLearningAreas_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	expectedAreas := []LearningArea{
		{ID: "area_001", Name: "English", Code: "ENG", EducationLevel: "Early_Years"},
		{ID: "area_002", Name: "Mathematics", Code: "MATH", EducationLevel: "Early_Years"},
	}

	h.repo.listLearningAreasFn = func(ctx context.Context, tenantID, schoolID string, educationLevel *string) ([]LearningArea, error) {
		return expectedAreas, nil
	}

	resp := doRequest(h.app, "GET", "/api/v1/curriculum/learning-areas", nil)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var result ListLearningAreasResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if result.Total != 2 {
		t.Fatalf("expected total 2, got %d", result.Total)
	}
}

func TestHandler_GetLearningAreaByID_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.getLearningAreaByIDFn = func(ctx context.Context, id, tenantID, schoolID string) (*LearningArea, error) {
		return &LearningArea{ID: id, Name: "Mathematics", Code: "MATH", EducationLevel: "Junior_Secondary"}, nil
	}

	resp := doRequest(h.app, "GET", "/api/v1/curriculum/learning-areas/area_001", nil)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var result LearningArea
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if result.Name != "Mathematics" {
		t.Fatalf("expected Name 'Mathematics', got %q", result.Name)
	}
}

func TestHandler_GetLearningAreaByID_NotFound(t *testing.T) {
	h := newHandlerTestHarness(t)
	h.repo.getLearningAreaByIDFn = func(ctx context.Context, id, tenantID, schoolID string) (*LearningArea, error) {
		return nil, ErrNotFound
	}
	resp := doRequest(h.app, "GET", "/api/v1/curriculum/learning-areas/area_999", nil)
	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestHandler_UpdateLearningArea_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	newName := "Advanced Mathematics"

	h.repo.getLearningAreaByIDFn = func(ctx context.Context, id, tenantID, schoolID string) (*LearningArea, error) {
		return &LearningArea{ID: id, TenantID: tenantID, SchoolID: schoolID, Name: "Mathematics", Code: "MATH", EducationLevel: "Junior_Secondary"}, nil
	}
	h.repo.updateLearningAreaFn = func(ctx context.Context, params UpdateLearningAreaParams) error {
		if params.Name == nil || *params.Name != "Advanced Mathematics" {
			t.Errorf("unexpected name")
		}
		return nil
	}

	body, _ := json.Marshal(UpdateLearningAreaPayload{Name: &newName})
	resp := doRequest(h.app, "PUT", "/api/v1/curriculum/learning-areas/area_001", body)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}
}

func TestHandler_DeleteLearningArea_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)
	h.repo.deleteLearningAreaFn = func(ctx context.Context, id, tenantID, schoolID string) error { return nil }
	resp := doRequest(h.app, "DELETE", "/api/v1/curriculum/learning-areas/area_001", nil)
	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("expected 204 No Content, got %d", resp.StatusCode)
	}
}

// ============================================================================
// Tests: Tree
// ============================================================================

func TestHandler_GetTree_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.getTreeFn = func(ctx context.Context, learningAreaID string) (*LearningAreaTree, error) {
		return &LearningAreaTree{
			LearningArea: LearningArea{ID: "area_001", Name: "Mathematics", Code: "MATH", EducationLevel: "Junior_Secondary"},
			Strands: []StrandTree{
				{
					Strand: Strand{ID: "strand_001", LearningAreaID: "area_001", Name: "Numbers"},
					SubStrands: []SubStrandTree{
						{
							SubStrand: SubStrand{ID: "sub_001", StrandID: "strand_001", Name: "Addition"},
							PerformanceIndicators: []PerformanceIndicator{
								{ID: "pi_001", SubStrandID: "sub_001", Description: "1+1", SequenceOrder: 1},
							},
						},
					},
				},
			},
		}, nil
	}

	resp := doRequest(h.app, "GET", "/api/v1/curriculum/learning-areas/area_001/tree", nil)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var result LearningAreaTree
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(result.Strands) != 1 {
		t.Fatalf("expected 1 strand, got %d", len(result.Strands))
	}
	if len(result.Strands[0].SubStrands) != 1 {
		t.Fatalf("expected 1 sub-strand, got %d", len(result.Strands[0].SubStrands))
	}
	if len(result.Strands[0].SubStrands[0].PerformanceIndicators) != 1 {
		t.Fatalf("expected 1 indicator, got %d", len(result.Strands[0].SubStrands[0].PerformanceIndicators))
	}
}

func TestHandler_GetTree_NotFound(t *testing.T) {
	h := newHandlerTestHarness(t)
	h.repo.verifyLearningAreaBelongsToTenantFn = func(ctx context.Context, id, tenantID, schoolID string) error { return ErrNotFound }
	resp := doRequest(h.app, "GET", "/api/v1/curriculum/learning-areas/area_999/tree", nil)
	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

// ============================================================================
// Tests: Strands
// ============================================================================

func TestHandler_CreateStrand_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.createStrandFn = func(ctx context.Context, params CreateStrandParams) (string, error) {
		if params.LearningAreaID != "area_001" {
			t.Errorf("expected LearningAreaID 'area_001', got %q", params.LearningAreaID)
		}
		if params.Name != "Numbers" {
			t.Errorf("expected Name 'Numbers', got %q", params.Name)
		}
		return "strand_001", nil
	}

	body, _ := json.Marshal(CreateStrandPayload{LearningAreaID: "area_001", Name: "Numbers"})
	resp := doRequest(h.app, "POST", "/api/v1/curriculum/strands", body)

	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", resp.StatusCode)
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if result.ID != "strand_001" {
		t.Fatalf("expected id 'strand_001', got %q", result.ID)
	}
}

func TestHandler_CreateStrand_InvalidBody(t *testing.T) {
	h := newHandlerTestHarness(t)
	resp := doRequest(h.app, "POST", "/api/v1/curriculum/strands", []byte("not json"))
	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", resp.StatusCode)
	}
}

func TestHandler_ListStrands_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.listStrandsFn = func(ctx context.Context, learningAreaID string) ([]Strand, error) {
		return []Strand{
			{ID: "strand_001", LearningAreaID: learningAreaID, Name: "Numbers"},
		}, nil
	}

	resp := doRequest(h.app, "GET", "/api/v1/curriculum/strands?learning_area_id=area_001", nil)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result ListStrandsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("expected total 1, got %d", result.Total)
	}
}

func TestHandler_ListStrands_MissingParam(t *testing.T) {
	h := newHandlerTestHarness(t)
	resp := doRequest(h.app, "GET", "/api/v1/curriculum/strands", nil)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestHandler_UpdateStrand_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	newName := "Advanced Numbers"
	h.repo.updateStrandFn = func(ctx context.Context, params UpdateStrandParams) error {
		if params.Name == nil || *params.Name != "Advanced Numbers" {
			t.Errorf("unexpected name")
		}
		return nil
	}

	body, _ := json.Marshal(UpdateStrandPayload{Name: &newName})
	resp := doRequest(h.app, "PUT", "/api/v1/curriculum/strands/strand_001", body)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestHandler_DeleteStrand_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)
	h.repo.deleteStrandFn = func(ctx context.Context, id string) error { return nil }
	resp := doRequest(h.app, "DELETE", "/api/v1/curriculum/strands/strand_001", nil)
	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
}

func TestHandler_DeleteStrand_NotFound(t *testing.T) {
	h := newHandlerTestHarness(t)
	h.repo.deleteStrandFn = func(ctx context.Context, id string) error { return ErrNotFound }
	resp := doRequest(h.app, "DELETE", "/api/v1/curriculum/strands/strand_999", nil)
	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

// ============================================================================
// Tests: Sub-Strands
// ============================================================================

func TestHandler_CreateSubStrand_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.createSubStrandFn = func(ctx context.Context, params CreateSubStrandParams) (string, error) {
		if params.StrandID != "strand_001" {
			t.Errorf("expected StrandID 'strand_001', got %q", params.StrandID)
		}
		if params.Name != "Addition" {
			t.Errorf("expected Name 'Addition', got %q", params.Name)
		}
		return "sub_001", nil
	}

	body, _ := json.Marshal(CreateSubStrandPayload{StrandID: "strand_001", Name: "Addition"})
	resp := doRequest(h.app, "POST", "/api/v1/curriculum/sub-strands", body)

	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if result.ID != "sub_001" {
		t.Fatalf("expected id 'sub_001', got %q", result.ID)
	}
}

func TestHandler_ListSubStrands_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.listSubStrandsFn = func(ctx context.Context, strandID string) ([]SubStrand, error) {
		return []SubStrand{
			{ID: "sub_001", StrandID: strandID, Name: "Addition"},
		}, nil
	}

	resp := doRequest(h.app, "GET", "/api/v1/curriculum/sub-strands?strand_id=strand_001", nil)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result ListSubStrandsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("expected total 1, got %d", result.Total)
	}
}

func TestHandler_ListSubStrands_MissingParam(t *testing.T) {
	h := newHandlerTestHarness(t)
	resp := doRequest(h.app, "GET", "/api/v1/curriculum/sub-strands", nil)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestHandler_UpdateSubStrand_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)
	newName := "Advanced Addition"
	h.repo.updateSubStrandFn = func(ctx context.Context, params UpdateSubStrandParams) error { return nil }
	body, _ := json.Marshal(UpdateSubStrandPayload{Name: &newName})
	resp := doRequest(h.app, "PUT", "/api/v1/curriculum/sub-strands/sub_001", body)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestHandler_DeleteSubStrand_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)
	h.repo.deleteSubStrandFn = func(ctx context.Context, id string) error { return nil }
	resp := doRequest(h.app, "DELETE", "/api/v1/curriculum/sub-strands/sub_001", nil)
	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
}

// ============================================================================
// Tests: Performance Indicators
// ============================================================================

func TestHandler_CreatePerformanceIndicator_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.createPIFn = func(ctx context.Context, params CreatePerformanceIndicatorParams) (string, error) {
		if params.SubStrandID != "sub_001" {
			t.Errorf("expected SubStrandID 'sub_001', got %q", params.SubStrandID)
		}
		if params.Description != "Solve 1+1" {
			t.Errorf("expected Description 'Solve 1+1', got %q", params.Description)
		}
		return "pi_001", nil
	}

	body, _ := json.Marshal(CreatePerformanceIndicatorPayload{
		SubStrandID: "sub_001",
		Description: "Solve 1+1",
	})
	resp := doRequest(h.app, "POST", "/api/v1/curriculum/performance-indicators", body)

	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if result.ID != "pi_001" {
		t.Fatalf("expected id 'pi_001', got %q", result.ID)
	}
}

func TestHandler_CreatePerformanceIndicator_WithExplicitOrder(t *testing.T) {
	h := newHandlerTestHarness(t)

	order := 5
	h.repo.createPIFn = func(ctx context.Context, params CreatePerformanceIndicatorParams) (string, error) {
		if params.SequenceOrder == nil || *params.SequenceOrder != 5 {
			t.Errorf("expected SequenceOrder 5, got %v", params.SequenceOrder)
		}
		return "pi_005", nil
	}

	body, _ := json.Marshal(CreatePerformanceIndicatorPayload{
		SubStrandID:   "sub_001",
		Description:   "Advanced",
		SequenceOrder: &order,
	})
	resp := doRequest(h.app, "POST", "/api/v1/curriculum/performance-indicators", body)
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
}

func TestHandler_ListPerformanceIndicators_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.listPIFn = func(ctx context.Context, subStrandID string) ([]PerformanceIndicator, error) {
		return []PerformanceIndicator{
			{ID: "pi_001", SubStrandID: subStrandID, Description: "First", SequenceOrder: 1},
		}, nil
	}

	resp := doRequest(h.app, "GET", "/api/v1/curriculum/performance-indicators?sub_strand_id=sub_001", nil)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result ListPerformanceIndicatorsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("expected total 1, got %d", result.Total)
	}
}

func TestHandler_ListPerformanceIndicators_MissingParam(t *testing.T) {
	h := newHandlerTestHarness(t)
	resp := doRequest(h.app, "GET", "/api/v1/curriculum/performance-indicators", nil)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestHandler_UpdatePerformanceIndicator_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)
	newDesc := "Updated description"
	h.repo.updatePIFn = func(ctx context.Context, params UpdatePerformanceIndicatorParams) error { return nil }
	body, _ := json.Marshal(UpdatePerformanceIndicatorPayload{Description: &newDesc})
	resp := doRequest(h.app, "PUT", "/api/v1/curriculum/performance-indicators/pi_001", body)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestHandler_DeletePerformanceIndicator_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)
	h.repo.deletePIFn = func(ctx context.Context, id string) error { return nil }
	resp := doRequest(h.app, "DELETE", "/api/v1/curriculum/performance-indicators/pi_001", nil)
	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
}

// ============================================================================
// Tests: No active school
// ============================================================================

func TestHandler_NoActiveSchool(t *testing.T) {
	repo := &MockRepository{}
	svc := NewService(repo)
	handler := NewHandler(svc)

	app := fiber.New()

	testAuth := func(c *fiber.Ctx) error {
		c.Locals("tenant_id", "tenant_001")
		c.Locals("user_id", "user_001")
		return c.Next()
	}

	app.Group("/api/v1/curriculum/learning-areas", testAuth).Post("/", handler.CreateLearningArea)

	body, _ := json.Marshal(CreateLearningAreaPayload{Name: "Test", Code: "TEST", EducationLevel: "Early_Years"})
	resp := doRequest(app, "POST", "/api/v1/curriculum/learning-areas", body)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for missing active school, got %d", resp.StatusCode)
	}
}
