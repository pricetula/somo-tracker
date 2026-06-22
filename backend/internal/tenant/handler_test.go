package tenant

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// ============================================================================
// MockRepository for handler test
// ============================================================================

type handlerMockRepo struct {
	existsByNameFn func(ctx context.Context, name string) (bool, error)
	existsBySlugFn func(ctx context.Context, slug string) (bool, error)
	createFn       func(ctx context.Context, name, slug string) (*Tenant, error)
}

func (m *handlerMockRepo) ExistsByName(ctx context.Context, name string) (bool, error) {
	if m.existsByNameFn != nil {
		return m.existsByNameFn(ctx, name)
	}
	return false, nil
}

func (m *handlerMockRepo) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	if m.existsBySlugFn != nil {
		return m.existsBySlugFn(ctx, slug)
	}
	return false, nil
}

func (m *handlerMockRepo) Create(ctx context.Context, name, slug string) (*Tenant, error) {
	if m.createFn != nil {
		return m.createFn(ctx, name, slug)
	}
	return &Tenant{ID: "tenant_001", Name: name, Slug: slug}, nil
}

func (m *handlerMockRepo) GetByID(ctx context.Context, id string) (*Tenant, error) {
	return nil, nil
}

// ============================================================================
// Test Harness
// ============================================================================

type handlerTestHarness struct {
	app     *fiber.App
	svc     *Service
	handler *Handler
}

func newHandlerTestHarness() *handlerTestHarness {
	repo := &handlerMockRepo{}
	svc := &Service{repo: repo}
	handler := &Handler{svc: svc}

	app := fiber.New()
	handler.RegisterRoutes(app)

	return &handlerTestHarness{
		app:     app,
		svc:     svc,
		handler: handler,
	}
}

// ============================================================================
// Tests: Create Tenant Handler (POST /tenants)
// ============================================================================

func TestHandler_CreateTenant_HappyPath(t *testing.T) {
	h := newHandlerTestHarness()

	body := CreateTenantPayload{
		Name: "New School",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/tenants", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := h.app.Test(req)

	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", resp.StatusCode)
	}

	var result Tenant
	_ = json.NewDecoder(resp.Body).Decode(&result)
	if result.Name != "New School" {
		t.Fatalf("expected name 'New School', got %q", result.Name)
	}
}

func TestHandler_CreateTenant_EmptyName(t *testing.T) {
	h := newHandlerTestHarness()

	body := CreateTenantPayload{
		Name: "",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/tenants", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := h.app.Test(req)

	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		t.Fatalf("expected 422 Unprocessable Entity, got %d", resp.StatusCode)
	}

	var errBody struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&errBody)
	if errBody.Code != "invalid_input" {
		t.Fatalf("expected code 'invalid_input', got %q", errBody.Code)
	}
}

func TestHandler_CreateTenant_InvalidJSON(t *testing.T) {
	h := newHandlerTestHarness()

	req := httptest.NewRequest("POST", "/tenants", bytes.NewReader([]byte("{bad json}")))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := h.app.Test(req)

	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		t.Fatalf("expected 422 Unprocessable Entity, got %d", resp.StatusCode)
	}
}

func TestHandler_CreateTenant_ServiceError(t *testing.T) {
	h := newHandlerTestHarness()

	// Replace the svc with one that has a failing repo
	h.handler.svc = &Service{
		repo: &handlerMockRepo{
			createFn: func(ctx context.Context, name, slug string) (*Tenant, error) {
				return nil, errors.New("tenant with name 'Fail School' already exists")
			},
		},
	}

	body := CreateTenantPayload{
		Name: "Fail School",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/tenants", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := h.app.Test(req)

	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("expected 500 Internal Server Error, got %d", resp.StatusCode)
	}

	var errBody struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&errBody)
	if errBody.Code != "internal_error" {
		t.Fatalf("expected code 'internal_error', got %q", errBody.Code)
	}
}
