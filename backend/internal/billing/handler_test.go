package billing

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

	// Test middleware that sets tenant_id, user_id, active_school_id and role (bypasses requireAuth/requireRole)
	testAuth := func(c *fiber.Ctx) error {
		c.Locals("tenant_id", "tenant_001")
		c.Locals("user_id", "user_001")
		c.Locals("active_school_id", "school_001")
		c.Locals("role", "SCHOOL_ADMIN")
		return c.Next()
	}

	// Register routes manually with test auth (bypassing real middleware)
	billing := app.Group("/api/v1/billing")
	billing.Post("/fee-categories", testAuth, handler.CreateFeeCategory)
	billing.Get("/fee-categories", testAuth, handler.ListFeeCategories)
	billing.Put("/fee-categories/:id", testAuth, handler.UpdateFeeCategory)
	billing.Delete("/fee-categories/:id", testAuth, handler.DeleteFeeCategory)

	billing.Post("/fee-templates", testAuth, handler.CreateFeeTemplate)
	billing.Get("/fee-templates", testAuth, handler.ListFeeTemplates)
	billing.Put("/fee-templates/:id", testAuth, handler.UpdateFeeTemplate)
	billing.Delete("/fee-templates/:id", testAuth, handler.DeleteFeeTemplate)

	// Invoices
	billing.Post("/invoices/generate", testAuth, handler.GenerateInvoice)
	billing.Get("/invoices", testAuth, handler.ListInvoices)
	billing.Get("/invoices/:id", testAuth, handler.GetInvoiceDetail)
	billing.Post("/invoices/:id/waive", testAuth, handler.WaiveInvoice)

	// Payments
	billing.Post("/payments", testAuth, handler.RecordPayment)
	billing.Get("/payments", testAuth, handler.ListPayments)

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

// testHandlerWithoutAuth creates a handler harness that does NOT inject
// session auth — the real requireAuth middleware will reject.
func testHandlerWithoutAuth(t *testing.T) *handlerTestHarness {
	t.Helper()

	repo := &MockRepository{}
	svc := NewService(repo)
	handler := NewHandler(svc)

	app := fiber.New()

	// Register routes WITHOUT testAuth middleware — RequireAuth will reject
	handler.RegisterRoutes(app) // has real middleware

	return &handlerTestHarness{
		app:     app,
		svc:     svc,
		repo:    repo,
		handler: handler,
	}
}

// ============================================================================
// Fee Categories: Create (POST)
// ============================================================================

func TestHandler_CreateFeeCategory_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.createFeeCategoryFn = func(ctx context.Context, tenantID, schoolID, name string, isMandatory bool) (string, error) {
		if tenantID != "tenant_001" {
			t.Errorf("expected tenantID 'tenant_001', got %q", tenantID)
		}
		if schoolID != "school_001" {
			t.Errorf("expected schoolID 'school_001', got %q", schoolID)
		}
		if name != "Tuition" {
			t.Errorf("expected name 'Tuition', got %q", name)
		}
		if !isMandatory {
			t.Error("expected isMandatory true")
		}
		return "cat_001", nil
	}

	body, _ := json.Marshal(CreateFeeCategoryPayload{Name: "Tuition", IsMandatory: true})
	resp := doRequest(h.app, "POST", "/api/v1/billing/fee-categories", body)

	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", resp.StatusCode)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result["id"] != "cat_001" {
		t.Fatalf("expected id 'cat_001', got %q", result["id"])
	}
}

func TestHandler_CreateFeeCategory_MissingName(t *testing.T) {
	h := newHandlerTestHarness(t)

	body, _ := json.Marshal(CreateFeeCategoryPayload{Name: ""})
	resp := doRequest(h.app, "POST", "/api/v1/billing/fee-categories", body)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", resp.StatusCode)
	}

	var result struct {
		Code    string              `json:"code"`
		Message string              `json:"message"`
		Errors  map[string][]string `json:"errors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result.Code != "invalid_input" {
		t.Fatalf("expected code 'invalid_input', got %q", result.Code)
	}
	if result.Errors["name"] == nil {
		t.Fatalf("expected field error for 'name'")
	}
}

func TestHandler_CreateFeeCategory_InvalidBody(t *testing.T) {
	h := newHandlerTestHarness(t)

	resp := doRequest(h.app, "POST", "/api/v1/billing/fee-categories", []byte("not json"))

	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		t.Fatalf("expected 422 Unprocessable Entity, got %d", resp.StatusCode)
	}
}

func TestHandler_CreateFeeCategory_Duplicate(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.createFeeCategoryFn = func(ctx context.Context, tenantID, schoolID, name string, isMandatory bool) (string, error) {
		return "", ErrAlreadyExists
	}

	body, _ := json.Marshal(CreateFeeCategoryPayload{Name: "Tuition", IsMandatory: true})
	resp := doRequest(h.app, "POST", "/api/v1/billing/fee-categories", body)

	if resp.StatusCode != fiber.StatusConflict {
		t.Fatalf("expected 409 Conflict, got %d", resp.StatusCode)
	}
}

// ============================================================================
// Fee Categories: List (GET)
// ============================================================================

func TestHandler_ListFeeCategories_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	expected := []FeeCategory{
		{ID: "cat_001", Name: "Tuition", IsMandatory: true},
		{ID: "cat_002", Name: "Transport", IsMandatory: false},
	}

	h.repo.listFeeCategoriesFn = func(ctx context.Context, tenantID, schoolID string) ([]FeeCategory, error) {
		if tenantID != "tenant_001" {
			t.Errorf("expected tenantID 'tenant_001', got %q", tenantID)
		}
		if schoolID != "school_001" {
			t.Errorf("expected schoolID 'school_001', got %q", schoolID)
		}
		return expected, nil
	}

	resp := doRequest(h.app, "GET", "/api/v1/billing/fee-categories", nil)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var result ListFeeCategoriesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result.Total != 2 {
		t.Fatalf("expected total 2, got %d", result.Total)
	}
	if len(result.FeeCategories) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(result.FeeCategories))
	}
}

func TestHandler_ListFeeCategories_Empty(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.listFeeCategoriesFn = func(ctx context.Context, tenantID, schoolID string) ([]FeeCategory, error) {
		return []FeeCategory{}, nil
	}

	resp := doRequest(h.app, "GET", "/api/v1/billing/fee-categories", nil)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var result ListFeeCategoriesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result.Total != 0 {
		t.Fatalf("expected total 0, got %d", result.Total)
	}
}

// ============================================================================
// Fee Categories: Update (PUT)
// ============================================================================

func TestHandler_UpdateFeeCategory_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.updateFeeCategoryFn = func(ctx context.Context, id, tenantID, schoolID string, name *string, isMandatory *bool) error {
		if id != "cat_001" {
			t.Errorf("expected id 'cat_001', got %q", id)
		}
		if name == nil || *name != "Updated Tuition" {
			t.Errorf("expected name 'Updated Tuition', got %v", name)
		}
		return nil
	}

	body, _ := json.Marshal(UpdateFeeCategoryPayload{Name: strPtr("Updated Tuition")})
	resp := doRequest(h.app, "PUT", "/api/v1/billing/fee-categories/cat_001", body)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}
}

func TestHandler_UpdateFeeCategory_InvalidBody(t *testing.T) {
	h := newHandlerTestHarness(t)

	resp := doRequest(h.app, "PUT", "/api/v1/billing/fee-categories/cat_001", []byte("not json"))

	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		t.Fatalf("expected 422 Unprocessable Entity, got %d", resp.StatusCode)
	}
}

func TestHandler_UpdateFeeCategory_NotFound(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.updateFeeCategoryFn = func(ctx context.Context, id, tenantID, schoolID string, name *string, isMandatory *bool) error {
		return ErrNotFound
	}

	body, _ := json.Marshal(UpdateFeeCategoryPayload{Name: strPtr("Updated Tuition")})
	resp := doRequest(h.app, "PUT", "/api/v1/billing/fee-categories/cat_999", body)

	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d", resp.StatusCode)
	}
}

// ============================================================================
// Fee Categories: Delete (DELETE)
// ============================================================================

func TestHandler_DeleteFeeCategory_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.deleteFeeCategoryFn = func(ctx context.Context, id, tenantID, schoolID string) error {
		if id != "cat_001" {
			t.Errorf("expected id 'cat_001', got %q", id)
		}
		return nil
	}

	resp := doRequest(h.app, "DELETE", "/api/v1/billing/fee-categories/cat_001", nil)

	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("expected 204 No Content, got %d", resp.StatusCode)
	}
}

func TestHandler_DeleteFeeCategory_NotFound(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.deleteFeeCategoryFn = func(ctx context.Context, id, tenantID, schoolID string) error {
		return ErrNotFound
	}

	resp := doRequest(h.app, "DELETE", "/api/v1/billing/fee-categories/cat_999", nil)

	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d", resp.StatusCode)
	}
}

// ============================================================================
// Fee Templates: Create (POST)
// ============================================================================

func TestHandler_CreateFeeTemplate_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.createFeeTemplateFn = func(ctx context.Context, tenantID, schoolID, academicTermID, gradeLevel, feeCategoryID, amount string) (string, error) {
		if tenantID != "tenant_001" {
			t.Errorf("expected tenantID 'tenant_001', got %q", tenantID)
		}
		if schoolID != "school_001" {
			t.Errorf("expected schoolID 'school_001', got %q", schoolID)
		}
		if academicTermID != "term_001" {
			t.Errorf("expected academicTermID 'term_001', got %q", academicTermID)
		}
		if gradeLevel != "G1" {
			t.Errorf("expected gradeLevel 'G1', got %q", gradeLevel)
		}
		if feeCategoryID != "cat_001" {
			t.Errorf("expected feeCategoryID 'cat_001', got %q", feeCategoryID)
		}
		if amount != "5000.00" {
			t.Errorf("expected amount '5000.00', got %q", amount)
		}
		return "tmp_001", nil
	}

	body, _ := json.Marshal(CreateFeeTemplatePayload{
		AcademicTermID: "term_001",
		GradeLevel:     "G1",
		FeeCategoryID:  "cat_001",
		Amount:         "5000.00",
	})
	resp := doRequest(h.app, "POST", "/api/v1/billing/fee-templates", body)

	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", resp.StatusCode)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result["id"] != "tmp_001" {
		t.Fatalf("expected id 'tmp_001', got %q", result["id"])
	}
}

func TestHandler_CreateFeeTemplate_NegativeAmount(t *testing.T) {
	h := newHandlerTestHarness(t)

	body, _ := json.Marshal(CreateFeeTemplatePayload{
		AcademicTermID: "term_001",
		GradeLevel:     "G1",
		FeeCategoryID:  "cat_001",
		Amount:         "-100.00",
	})
	resp := doRequest(h.app, "POST", "/api/v1/billing/fee-templates", body)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", resp.StatusCode)
	}

	var result struct {
		Code    string              `json:"code"`
		Message string              `json:"message"`
		Errors  map[string][]string `json:"errors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result.Code != "invalid_input" {
		t.Fatalf("expected code 'invalid_input', got %q", result.Code)
	}
	if result.Errors["amount"] == nil {
		t.Fatalf("expected field error for 'amount'")
	}
}

func TestHandler_CreateFeeTemplate_Duplicate(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.createFeeTemplateFn = func(ctx context.Context, tenantID, schoolID, academicTermID, gradeLevel, feeCategoryID, amount string) (string, error) {
		return "", ErrAlreadyExists
	}

	body, _ := json.Marshal(CreateFeeTemplatePayload{
		AcademicTermID: "term_001",
		GradeLevel:     "G1",
		FeeCategoryID:  "cat_001",
		Amount:         "5000.00",
	})
	resp := doRequest(h.app, "POST", "/api/v1/billing/fee-templates", body)

	if resp.StatusCode != fiber.StatusConflict {
		t.Fatalf("expected 409 Conflict, got %d", resp.StatusCode)
	}
}

func TestHandler_CreateFeeTemplate_InvalidBody(t *testing.T) {
	h := newHandlerTestHarness(t)

	resp := doRequest(h.app, "POST", "/api/v1/billing/fee-templates", []byte("not json"))

	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		t.Fatalf("expected 422 Unprocessable Entity, got %d", resp.StatusCode)
	}
}

// ============================================================================
// Fee Templates: List (GET)
// ============================================================================

func TestHandler_ListFeeTemplates_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	expected := []FeeTemplate{
		{ID: "tmp_001", GradeLevel: "G1", Amount: "5000.00"},
		{ID: "tmp_002", GradeLevel: "G2", Amount: "5500.00"},
	}

	h.repo.listFeeTemplatesFn = func(ctx context.Context, tenantID, schoolID string, academicTermID, gradeLevel *string) ([]FeeTemplate, error) {
		if tenantID != "tenant_001" {
			t.Errorf("expected tenantID 'tenant_001', got %q", tenantID)
		}
		if schoolID != "school_001" {
			t.Errorf("expected schoolID 'school_001', got %q", schoolID)
		}
		return expected, nil
	}

	resp := doRequest(h.app, "GET", "/api/v1/billing/fee-templates", nil)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var result ListFeeTemplatesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result.Total != 2 {
		t.Fatalf("expected total 2, got %d", result.Total)
	}
}

func TestHandler_ListFeeTemplates_FilterByTerm(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.listFeeTemplatesFn = func(ctx context.Context, tenantID, schoolID string, academicTermID, gradeLevel *string) ([]FeeTemplate, error) {
		if academicTermID == nil || *academicTermID != "term_001" {
			t.Errorf("expected academicTermID 'term_001', got %v", academicTermID)
		}
		return []FeeTemplate{
			{ID: "tmp_001", AcademicTermID: "term_001", Amount: "5000.00"},
		}, nil
	}

	resp := doRequest(h.app, "GET", "/api/v1/billing/fee-templates?academic_term_id=term_001", nil)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var result ListFeeTemplatesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("expected total 1, got %d", result.Total)
	}
}

// ============================================================================
// Fee Templates: Update (PUT)
// ============================================================================

func TestHandler_UpdateFeeTemplate_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.updateFeeTemplateFn = func(ctx context.Context, id, tenantID, schoolID string, amount *string) error {
		if id != "tmp_001" {
			t.Errorf("expected id 'tmp_001', got %q", id)
		}
		if amount == nil || *amount != "6000.00" {
			t.Errorf("expected amount '6000.00', got %v", amount)
		}
		return nil
	}

	body, _ := json.Marshal(UpdateFeeTemplatePayload{Amount: strPtr("6000.00")})
	resp := doRequest(h.app, "PUT", "/api/v1/billing/fee-templates/tmp_001", body)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}
}

func TestHandler_UpdateFeeTemplate_NotFound(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.updateFeeTemplateFn = func(ctx context.Context, id, tenantID, schoolID string, amount *string) error {
		return ErrNotFound
	}

	body, _ := json.Marshal(UpdateFeeTemplatePayload{Amount: strPtr("6000.00")})
	resp := doRequest(h.app, "PUT", "/api/v1/billing/fee-templates/tmp_999", body)

	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d", resp.StatusCode)
	}
}

// ============================================================================
// Fee Templates: Delete (DELETE)
// ============================================================================

func TestHandler_DeleteFeeTemplate_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.deleteFeeTemplateFn = func(ctx context.Context, id, tenantID, schoolID string) error {
		if id != "tmp_001" {
			t.Errorf("expected id 'tmp_001', got %q", id)
		}
		return nil
	}

	resp := doRequest(h.app, "DELETE", "/api/v1/billing/fee-templates/tmp_001", nil)

	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("expected 204 No Content, got %d", resp.StatusCode)
	}
}

func TestHandler_DeleteFeeTemplate_NotFound(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.deleteFeeTemplateFn = func(ctx context.Context, id, tenantID, schoolID string) error {
		return ErrNotFound
	}

	resp := doRequest(h.app, "DELETE", "/api/v1/billing/fee-templates/tmp_999", nil)

	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d", resp.StatusCode)
	}
}

// ============================================================================
// Auth rejection (unauthenticated requests)
// ============================================================================

func TestHandler_CreateFeeCategory_Unauthenticated(t *testing.T) {
	h := testHandlerWithoutAuth(t)

	body, _ := json.Marshal(CreateFeeCategoryPayload{Name: "Tuition"})
	resp := doRequest(h.app, "POST", "/api/v1/billing/fee-categories", body)

	// POST uses RequireRole("SCHOOL_ADMIN") which falls through to 403 when
	// RequireAuth writes the 401 response body without returning a non-nil error.
	if resp.StatusCode != fiber.StatusForbidden {
		t.Fatalf("expected 403 Forbidden, got %d", resp.StatusCode)
	}
}

func TestHandler_ListFeeCategories_Unauthenticated(t *testing.T) {
	h := testHandlerWithoutAuth(t)

	resp := doRequest(h.app, "GET", "/api/v1/billing/fee-categories", nil)

	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", resp.StatusCode)
	}
}

func TestHandler_CreateFeeTemplate_Unauthenticated(t *testing.T) {
	h := testHandlerWithoutAuth(t)

	body, _ := json.Marshal(CreateFeeTemplatePayload{
		AcademicTermID: "term_001",
		GradeLevel:     "G1",
		FeeCategoryID:  "cat_001",
		Amount:         "5000.00",
	})
	resp := doRequest(h.app, "POST", "/api/v1/billing/fee-templates", body)

	// POST uses RequireRole("SCHOOL_ADMIN") which falls through to 403 when
	// RequireAuth writes the 401 response body without returning a non-nil error.
	if resp.StatusCode != fiber.StatusForbidden {
		t.Fatalf("expected 403 Forbidden, got %d", resp.StatusCode)
	}
}

func TestHandler_ListFeeTemplates_Unauthenticated(t *testing.T) {
	h := testHandlerWithoutAuth(t)

	resp := doRequest(h.app, "GET", "/api/v1/billing/fee-templates", nil)

	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", resp.StatusCode)
	}
}

// ============================================================================
// Invoices: Generate (POST)
// ============================================================================

func TestHandler_GenerateInvoice_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.resolveGradeLevelFn = func(ctx context.Context, tenantID, studentID, academicTermID string) (string, error) {
		return "G4", nil
	}

	h.repo.listFeeTemplatesByTermAndGradeFn = func(ctx context.Context, tenantID, schoolID, academicTermID, gradeLevel string) ([]FeeTemplate, error) {
		return []FeeTemplate{
			{ID: "tmp_001", FeeCategoryID: "cat_001", Amount: "5000.00"},
			{ID: "tmp_002", FeeCategoryID: "cat_002", Amount: "3000.00"},
		}, nil
	}

	h.repo.createInvoiceFn = func(ctx context.Context, tenantID, schoolID, studentID, academicTermID string, parentID *string, invoiceLabel *string, amountDue string) (string, error) {
		return "inv_001", nil
	}

	h.repo.createInvoiceItemFn = func(ctx context.Context, tenantID, invoiceID, feeCategoryID, description, amount string) error {
		return nil
	}

	body, _ := json.Marshal(GenerateInvoicePayload{
		StudentID:      "stu_001",
		AcademicTermID: "term_001",
		InvoiceLabel:   strPtr("Term 1 Fees"),
	})
	resp := doRequest(h.app, "POST", "/api/v1/billing/invoices/generate", body)

	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", resp.StatusCode)
	}

	var result InvoiceDetailResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
}

func TestHandler_GenerateInvoice_MissingStudentID(t *testing.T) {
	h := newHandlerTestHarness(t)

	body, _ := json.Marshal(GenerateInvoicePayload{
		AcademicTermID: "term_001",
	})
	resp := doRequest(h.app, "POST", "/api/v1/billing/invoices/generate", body)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", resp.StatusCode)
	}
}

func TestHandler_GenerateInvoice_InvalidBody(t *testing.T) {
	h := newHandlerTestHarness(t)

	resp := doRequest(h.app, "POST", "/api/v1/billing/invoices/generate", []byte("not json"))

	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		t.Fatalf("expected 422 Unprocessable Entity, got %d", resp.StatusCode)
	}
}

func TestHandler_GenerateInvoice_Duplicate(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.resolveGradeLevelFn = func(ctx context.Context, tenantID, studentID, academicTermID string) (string, error) {
		return "G1", nil
	}

	h.repo.createInvoiceFn = func(ctx context.Context, tenantID, schoolID, studentID, academicTermID string, parentID *string, invoiceLabel *string, amountDue string) (string, error) {
		return "", ErrAlreadyExists
	}

	body, _ := json.Marshal(GenerateInvoicePayload{
		StudentID:      "stu_001",
		AcademicTermID: "term_001",
	})
	resp := doRequest(h.app, "POST", "/api/v1/billing/invoices/generate", body)

	if resp.StatusCode != fiber.StatusConflict {
		t.Fatalf("expected 409 Conflict, got %d", resp.StatusCode)
	}
}

// ============================================================================
// Invoices: Get Detail (GET /:id)
// ============================================================================

func TestHandler_GetInvoiceDetail_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.getInvoiceDetailFn = func(ctx context.Context, id, tenantID, schoolID string) (*InvoiceDetailResponse, error) {
		return &InvoiceDetailResponse{
			Invoice: Invoice{ID: id, StudentID: "stu_001", PaymentStatus: "UNPAID", AmountDue: "8000.00"},
			Items: []InvoiceItem{
				{ID: "item_001", FeeCategoryID: "cat_001", Amount: "5000.00"},
			},
			Payments: []Payment{},
		}, nil
	}

	resp := doRequest(h.app, "GET", "/api/v1/billing/invoices/inv_001", nil)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var result InvoiceDetailResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result.Invoice.ID != "inv_001" {
		t.Fatalf("expected invoice ID 'inv_001', got %q", result.Invoice.ID)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}
}

func TestHandler_GetInvoiceDetail_NotFound(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.getInvoiceDetailFn = func(ctx context.Context, id, tenantID, schoolID string) (*InvoiceDetailResponse, error) {
		return nil, ErrNotFound
	}

	resp := doRequest(h.app, "GET", "/api/v1/billing/invoices/inv_999", nil)

	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d", resp.StatusCode)
	}
}

// ============================================================================
// Invoices: List (GET)
// ============================================================================

func TestHandler_ListInvoices_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.listInvoicesFn = func(ctx context.Context, tenantID, schoolID string, filter InvoiceFilter) ([]Invoice, int, error) {
		return []Invoice{
			{ID: "inv_001", StudentID: "stu_001", PaymentStatus: "UNPAID"},
			{ID: "inv_002", StudentID: "stu_002", PaymentStatus: "PAID"},
		}, 2, nil
	}

	resp := doRequest(h.app, "GET", "/api/v1/billing/invoices", nil)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var result ListInvoicesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result.Total != 2 {
		t.Fatalf("expected total 2, got %d", result.Total)
	}
}

func TestHandler_ListInvoices_Filtered(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.listInvoicesFn = func(ctx context.Context, tenantID, schoolID string, filter InvoiceFilter) ([]Invoice, int, error) {
		if filter.StudentID == nil || *filter.StudentID != "stu_001" {
			t.Errorf("expected StudentID 'stu_001', got %v", filter.StudentID)
		}
		return []Invoice{{ID: "inv_001", StudentID: "stu_001"}}, 1, nil
	}

	resp := doRequest(h.app, "GET", "/api/v1/billing/invoices?student_id=stu_001", nil)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var result ListInvoicesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("expected total 1, got %d", result.Total)
	}
}

// ============================================================================
// Invoices: Waive (POST /:id/waive)
// ============================================================================

func TestHandler_WaiveInvoice_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.getInvoiceByIDFn = func(ctx context.Context, id, tenantID, schoolID string) (*Invoice, error) {
		return &Invoice{ID: id, PaymentStatus: "UNPAID"}, nil
	}

	h.repo.waiveInvoiceFn = func(ctx context.Context, id, tenantID, schoolID string) error {
		return nil
	}

	resp := doRequest(h.app, "POST", "/api/v1/billing/invoices/inv_001/waive", nil)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result["payment_status"] != "WAIVED" {
		t.Fatalf("expected payment_status 'WAIVED', got %q", result["payment_status"])
	}
}

func TestHandler_WaiveInvoice_PaidInvoice(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.getInvoiceByIDFn = func(ctx context.Context, id, tenantID, schoolID string) (*Invoice, error) {
		return &Invoice{ID: id, PaymentStatus: "PAID"}, nil
	}

	resp := doRequest(h.app, "POST", "/api/v1/billing/invoices/inv_001/waive", nil)

	if resp.StatusCode != fiber.StatusConflict {
		t.Fatalf("expected 409 Conflict, got %d", resp.StatusCode)
	}
}

func TestHandler_WaiveInvoice_NotFound(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.getInvoiceByIDFn = func(ctx context.Context, id, tenantID, schoolID string) (*Invoice, error) {
		return nil, ErrNotFound
	}

	resp := doRequest(h.app, "POST", "/api/v1/billing/invoices/inv_999/waive", nil)

	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d", resp.StatusCode)
	}
}

// ============================================================================
// Payments: Record (POST)
// ============================================================================

func TestHandler_RecordPayment_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.getInvoiceByIDFn = func(ctx context.Context, id, tenantID, schoolID string) (*Invoice, error) {
		return &Invoice{ID: id, PaymentStatus: "UNPAID"}, nil
	}

	h.repo.recordPaymentFn = func(ctx context.Context, tenantID, invoiceID, amount, recordedBy string, parentID, paymentMethod, referenceCode *string) (string, error) {
		return "pay_001", nil
	}

	h.repo.getPaymentByIDFn = func(ctx context.Context, id, tenantID string) (*Payment, error) {
		return &Payment{ID: "pay_001", Amount: "5000.00", InvoiceID: "inv_001"}, nil
	}

	body, _ := json.Marshal(RecordPaymentPayload{
		InvoiceID: "inv_001",
		Amount:    "5000.00",
	})
	resp := doRequest(h.app, "POST", "/api/v1/billing/payments", body)

	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", resp.StatusCode)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result["id"] != "pay_001" {
		t.Fatalf("expected id 'pay_001', got %q", result["id"])
	}
}

func TestHandler_RecordPayment_WaivedInvoice(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.getInvoiceByIDFn = func(ctx context.Context, id, tenantID, schoolID string) (*Invoice, error) {
		return &Invoice{ID: id, PaymentStatus: "WAIVED"}, nil
	}

	body, _ := json.Marshal(RecordPaymentPayload{
		InvoiceID: "inv_001",
		Amount:    "5000.00",
	})
	resp := doRequest(h.app, "POST", "/api/v1/billing/payments", body)

	if resp.StatusCode != fiber.StatusConflict {
		t.Fatalf("expected 409 Conflict, got %d", resp.StatusCode)
	}

	var result struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result.Code != "conflict" {
		t.Fatalf("expected code 'conflict', got %q", result.Code)
	}
}

func TestHandler_RecordPayment_InvalidBody(t *testing.T) {
	h := newHandlerTestHarness(t)

	resp := doRequest(h.app, "POST", "/api/v1/billing/payments", []byte("not json"))

	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		t.Fatalf("expected 422 Unprocessable Entity, got %d", resp.StatusCode)
	}
}

func TestHandler_RecordPayment_NegativeAmount(t *testing.T) {
	h := newHandlerTestHarness(t)

	body, _ := json.Marshal(RecordPaymentPayload{
		InvoiceID: "inv_001",
		Amount:    "-100.00",
	})
	resp := doRequest(h.app, "POST", "/api/v1/billing/payments", body)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", resp.StatusCode)
	}

	var result struct {
		Code    string              `json:"code"`
		Message string              `json:"message"`
		Errors  map[string][]string `json:"errors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result.Code != "invalid_input" {
		t.Fatalf("expected code 'invalid_input', got %q", result.Code)
	}
}

// ============================================================================
// Payments: List (GET)
// ============================================================================

func TestHandler_ListPayments_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.listPaymentsFn = func(ctx context.Context, tenantID, invoiceID string) ([]Payment, error) {
		return []Payment{
			{ID: "pay_001", Amount: "3000.00"},
			{ID: "pay_002", Amount: "2000.00"},
		}, nil
	}

	resp := doRequest(h.app, "GET", "/api/v1/billing/payments?invoice_id=inv_001", nil)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var result ListPaymentsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result.Total != 2 {
		t.Fatalf("expected total 2, got %d", result.Total)
	}
}

func TestHandler_ListPayments_MissingInvoiceID(t *testing.T) {
	h := newHandlerTestHarness(t)

	resp := doRequest(h.app, "GET", "/api/v1/billing/payments", nil)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", resp.StatusCode)
	}

	var result struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result.Code != "invalid_input" {
		t.Fatalf("expected code 'invalid_input', got %q", result.Code)
	}
}

func TestHandler_GenerateInvoice_Unauthenticated(t *testing.T) {
	h := testHandlerWithoutAuth(t)

	body, _ := json.Marshal(GenerateInvoicePayload{
		StudentID:      "stu_001",
		AcademicTermID: "term_001",
	})
	resp := doRequest(h.app, "POST", "/api/v1/billing/invoices/generate", body)

	if resp.StatusCode != fiber.StatusForbidden {
		t.Fatalf("expected 403 Forbidden, got %d", resp.StatusCode)
	}
}

func TestHandler_ListInvoices_Unauthenticated(t *testing.T) {
	h := testHandlerWithoutAuth(t)

	resp := doRequest(h.app, "GET", "/api/v1/billing/invoices", nil)

	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", resp.StatusCode)
	}
}

func TestHandler_GetInvoiceDetail_Unauthenticated(t *testing.T) {
	h := testHandlerWithoutAuth(t)

	resp := doRequest(h.app, "GET", "/api/v1/billing/invoices/inv_001", nil)

	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", resp.StatusCode)
	}
}

func TestHandler_WaiveInvoice_Unauthenticated(t *testing.T) {
	h := testHandlerWithoutAuth(t)

	resp := doRequest(h.app, "POST", "/api/v1/billing/invoices/inv_001/waive", nil)

	if resp.StatusCode != fiber.StatusForbidden {
		t.Fatalf("expected 403 Forbidden, got %d", resp.StatusCode)
	}
}

func TestHandler_RecordPayment_Unauthenticated(t *testing.T) {
	h := testHandlerWithoutAuth(t)

	body, _ := json.Marshal(RecordPaymentPayload{InvoiceID: "inv_001", Amount: "5000.00"})
	resp := doRequest(h.app, "POST", "/api/v1/billing/payments", body)

	if resp.StatusCode != fiber.StatusForbidden {
		t.Fatalf("expected 403 Forbidden, got %d", resp.StatusCode)
	}
}

func TestHandler_ListPayments_Unauthenticated(t *testing.T) {
	h := testHandlerWithoutAuth(t)

	resp := doRequest(h.app, "GET", "/api/v1/billing/payments?invoice_id=inv_001", nil)

	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", resp.StatusCode)
	}
}

// ============================================================================
// Helpers
// ============================================================================

func strPtr(s string) *string {
	return &s
}
