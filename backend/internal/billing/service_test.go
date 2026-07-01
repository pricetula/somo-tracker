package billing

import (
	"context"
	"errors"
	"testing"
)

// ============================================================================
// MockRepository
// ============================================================================

type MockRepository struct {
	createFeeCategoryFn  func(ctx context.Context, tenantID, schoolID, name string, isMandatory bool) (string, error)
	listFeeCategoriesFn  func(ctx context.Context, tenantID, schoolID string) ([]FeeCategory, error)
	getFeeCategoryByIDFn func(ctx context.Context, id, tenantID, schoolID string) (*FeeCategory, error)
	updateFeeCategoryFn  func(ctx context.Context, id, tenantID, schoolID string, name *string, isMandatory *bool) error
	deleteFeeCategoryFn  func(ctx context.Context, id, tenantID, schoolID string) error

	createFeeTemplateFn  func(ctx context.Context, tenantID, schoolID, academicTermID, gradeLevel, feeCategoryID, amount string) (string, error)
	listFeeTemplatesFn   func(ctx context.Context, tenantID, schoolID string, academicTermID, gradeLevel *string) ([]FeeTemplate, error)
	getFeeTemplateByIDFn func(ctx context.Context, id, tenantID, schoolID string) (*FeeTemplate, error)
	updateFeeTemplateFn  func(ctx context.Context, id, tenantID, schoolID string, amount *string) error
	deleteFeeTemplateFn  func(ctx context.Context, id, tenantID, schoolID string) error

	// Invoices
	createInvoiceFn                  func(ctx context.Context, tenantID, schoolID, studentID, academicTermID string, parentID *string, invoiceLabel *string, amountDue string) (string, error)
	createInvoiceItemFn              func(ctx context.Context, tenantID, invoiceID, feeCategoryID, description, amount string) error
	getInvoiceByIDFn                 func(ctx context.Context, id, tenantID, schoolID string) (*Invoice, error)
	listInvoicesFn                   func(ctx context.Context, tenantID, schoolID string, filter InvoiceFilter) ([]Invoice, int, error)
	getInvoiceItemsFn                func(ctx context.Context, invoiceID, tenantID string) ([]InvoiceItem, error)
	getInvoiceDetailFn               func(ctx context.Context, id, tenantID, schoolID string) (*InvoiceDetailResponse, error)
	waiveInvoiceFn                   func(ctx context.Context, id, tenantID, schoolID string) error
	resolveGradeLevelFn              func(ctx context.Context, tenantID, studentID, academicTermID string) (string, error)
	listFeeTemplatesByTermAndGradeFn func(ctx context.Context, tenantID, schoolID, academicTermID, gradeLevel string) ([]FeeTemplate, error)

	// Payments
	recordPaymentFn  func(ctx context.Context, tenantID, invoiceID, amount, recordedBy string, parentID, paymentMethod, referenceCode *string) (string, error)
	listPaymentsFn   func(ctx context.Context, tenantID, invoiceID string) ([]Payment, error)
	getPaymentByIDFn func(ctx context.Context, id, tenantID string) (*Payment, error)
}

func (m *MockRepository) CreateFeeCategory(ctx context.Context, tenantID, schoolID, name string, isMandatory bool) (string, error) {
	if m.createFeeCategoryFn != nil {
		return m.createFeeCategoryFn(ctx, tenantID, schoolID, name, isMandatory)
	}
	return "cat_001", nil
}

func (m *MockRepository) ListFeeCategories(ctx context.Context, tenantID, schoolID string) ([]FeeCategory, error) {
	if m.listFeeCategoriesFn != nil {
		return m.listFeeCategoriesFn(ctx, tenantID, schoolID)
	}
	return []FeeCategory{}, nil
}

func (m *MockRepository) GetFeeCategoryByID(ctx context.Context, id, tenantID, schoolID string) (*FeeCategory, error) {
	if m.getFeeCategoryByIDFn != nil {
		return m.getFeeCategoryByIDFn(ctx, id, tenantID, schoolID)
	}
	return &FeeCategory{ID: id, Name: "Test Category", IsMandatory: true}, nil
}

func (m *MockRepository) UpdateFeeCategory(ctx context.Context, id, tenantID, schoolID string, name *string, isMandatory *bool) error {
	if m.updateFeeCategoryFn != nil {
		return m.updateFeeCategoryFn(ctx, id, tenantID, schoolID, name, isMandatory)
	}
	return nil
}

func (m *MockRepository) DeleteFeeCategory(ctx context.Context, id, tenantID, schoolID string) error {
	if m.deleteFeeCategoryFn != nil {
		return m.deleteFeeCategoryFn(ctx, id, tenantID, schoolID)
	}
	return nil
}

func (m *MockRepository) CreateFeeTemplate(ctx context.Context, tenantID, schoolID, academicTermID, gradeLevel, feeCategoryID, amount string) (string, error) {
	if m.createFeeTemplateFn != nil {
		return m.createFeeTemplateFn(ctx, tenantID, schoolID, academicTermID, gradeLevel, feeCategoryID, amount)
	}
	return "tmp_001", nil
}

func (m *MockRepository) ListFeeTemplates(ctx context.Context, tenantID, schoolID string, academicTermID, gradeLevel *string) ([]FeeTemplate, error) {
	if m.listFeeTemplatesFn != nil {
		return m.listFeeTemplatesFn(ctx, tenantID, schoolID, academicTermID, gradeLevel)
	}
	return []FeeTemplate{}, nil
}

func (m *MockRepository) GetFeeTemplateByID(ctx context.Context, id, tenantID, schoolID string) (*FeeTemplate, error) {
	if m.getFeeTemplateByIDFn != nil {
		return m.getFeeTemplateByIDFn(ctx, id, tenantID, schoolID)
	}
	return &FeeTemplate{ID: id, Amount: "1000.00"}, nil
}

func (m *MockRepository) UpdateFeeTemplate(ctx context.Context, id, tenantID, schoolID string, amount *string) error {
	if m.updateFeeTemplateFn != nil {
		return m.updateFeeTemplateFn(ctx, id, tenantID, schoolID, amount)
	}
	return nil
}

func (m *MockRepository) DeleteFeeTemplate(ctx context.Context, id, tenantID, schoolID string) error {
	if m.deleteFeeTemplateFn != nil {
		return m.deleteFeeTemplateFn(ctx, id, tenantID, schoolID)
	}
	return nil
}

// ─── Invoice Mock Methods ───────────────────────────────────────────────────

func (m *MockRepository) CreateInvoice(ctx context.Context, tenantID, schoolID, studentID, academicTermID string, parentID *string, invoiceLabel *string, amountDue string) (string, error) {
	if m.createInvoiceFn != nil {
		return m.createInvoiceFn(ctx, tenantID, schoolID, studentID, academicTermID, parentID, invoiceLabel, amountDue)
	}
	return "inv_001", nil
}

func (m *MockRepository) CreateInvoiceItem(ctx context.Context, tenantID, invoiceID, feeCategoryID, description, amount string) error {
	if m.createInvoiceItemFn != nil {
		return m.createInvoiceItemFn(ctx, tenantID, invoiceID, feeCategoryID, description, amount)
	}
	return nil
}

func (m *MockRepository) GetInvoiceByID(ctx context.Context, id, tenantID, schoolID string) (*Invoice, error) {
	if m.getInvoiceByIDFn != nil {
		return m.getInvoiceByIDFn(ctx, id, tenantID, schoolID)
	}
	return &Invoice{ID: id, PaymentStatus: "UNPAID"}, nil
}

func (m *MockRepository) ListInvoices(ctx context.Context, tenantID, schoolID string, filter InvoiceFilter) ([]Invoice, int, error) {
	if m.listInvoicesFn != nil {
		return m.listInvoicesFn(ctx, tenantID, schoolID, filter)
	}
	return []Invoice{}, 0, nil
}

func (m *MockRepository) GetInvoiceItems(ctx context.Context, invoiceID, tenantID string) ([]InvoiceItem, error) {
	if m.getInvoiceItemsFn != nil {
		return m.getInvoiceItemsFn(ctx, invoiceID, tenantID)
	}
	return []InvoiceItem{}, nil
}

func (m *MockRepository) GetInvoiceDetail(ctx context.Context, id, tenantID, schoolID string) (*InvoiceDetailResponse, error) {
	if m.getInvoiceDetailFn != nil {
		return m.getInvoiceDetailFn(ctx, id, tenantID, schoolID)
	}
	return &InvoiceDetailResponse{
		Invoice:  Invoice{ID: id, PaymentStatus: "UNPAID"},
		Items:    []InvoiceItem{},
		Payments: []Payment{},
	}, nil
}

func (m *MockRepository) WaiveInvoice(ctx context.Context, id, tenantID, schoolID string) error {
	if m.waiveInvoiceFn != nil {
		return m.waiveInvoiceFn(ctx, id, tenantID, schoolID)
	}
	return nil
}

func (m *MockRepository) ResolveGradeLevel(ctx context.Context, tenantID, studentID, academicTermID string) (string, error) {
	if m.resolveGradeLevelFn != nil {
		return m.resolveGradeLevelFn(ctx, tenantID, studentID, academicTermID)
	}
	return "G1", nil
}

func (m *MockRepository) ListFeeTemplatesByTermAndGrade(ctx context.Context, tenantID, schoolID, academicTermID, gradeLevel string) ([]FeeTemplate, error) {
	if m.listFeeTemplatesByTermAndGradeFn != nil {
		return m.listFeeTemplatesByTermAndGradeFn(ctx, tenantID, schoolID, academicTermID, gradeLevel)
	}
	return []FeeTemplate{
		{ID: "tmp_001", FeeCategoryID: "cat_001", Amount: "5000.00"},
		{ID: "tmp_002", FeeCategoryID: "cat_002", Amount: "3000.00"},
	}, nil
}

// ─── Payment Mock Methods ───────────────────────────────────────────────────

func (m *MockRepository) RecordPayment(ctx context.Context, tenantID, invoiceID, amount, recordedBy string, parentID, paymentMethod, referenceCode *string) (string, error) {
	if m.recordPaymentFn != nil {
		return m.recordPaymentFn(ctx, tenantID, invoiceID, amount, recordedBy, parentID, paymentMethod, referenceCode)
	}
	return "pay_001", nil
}

func (m *MockRepository) ListPayments(ctx context.Context, tenantID, invoiceID string) ([]Payment, error) {
	if m.listPaymentsFn != nil {
		return m.listPaymentsFn(ctx, tenantID, invoiceID)
	}
	return []Payment{}, nil
}

func (m *MockRepository) GetPaymentByID(ctx context.Context, id, tenantID string) (*Payment, error) {
	if m.getPaymentByIDFn != nil {
		return m.getPaymentByIDFn(ctx, id, tenantID)
	}
	return &Payment{ID: id, Amount: "500.00"}, nil
}

// ============================================================================
// Test Harness
// ============================================================================

type testHarness struct {
	svc  *Service
	repo *MockRepository
}

func newTestHarness() *testHarness {
	repo := &MockRepository{}
	svc := NewService(repo)
	return &testHarness{
		svc:  svc,
		repo: repo,
	}
}

// ============================================================================
// Tests: CreateFeeCategory
// ============================================================================

func TestCreateFeeCategory_HappyPath(t *testing.T) {
	h := newTestHarness()

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

	id, err := h.svc.CreateFeeCategory(context.Background(), "tenant_001", "school_001", "Tuition", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "cat_001" {
		t.Fatalf("expected id 'cat_001', got %q", id)
	}
}

func TestCreateFeeCategory_EmptyName(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.CreateFeeCategory(context.Background(), "tenant_001", "school_001", "", true)
	if err == nil {
		t.Fatal("expected error for empty name, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}

	// Check field error
	var fe interface{ FieldErrors() map[string][]string }
	if errors.As(err, &fe) {
		fields := fe.FieldErrors()
		if fields["name"] == nil {
			t.Fatal("expected field error for 'name'")
		}
	}
}

func TestCreateFeeCategory_WhitespaceName(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.CreateFeeCategory(context.Background(), "tenant_001", "school_001", "   ", true)
	if err == nil {
		t.Fatal("expected error for whitespace-only name, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateFeeCategory_NameTooLong(t *testing.T) {
	h := newTestHarness()

	longName := ""
	for i := 0; i < 151; i++ {
		longName += "a"
	}

	_, err := h.svc.CreateFeeCategory(context.Background(), "tenant_001", "school_001", longName, true)
	if err == nil {
		t.Fatal("expected error for name > 150 chars, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateFeeCategory_NonMandatory(t *testing.T) {
	h := newTestHarness()

	h.repo.createFeeCategoryFn = func(ctx context.Context, tenantID, schoolID, name string, isMandatory bool) (string, error) {
		if isMandatory {
			t.Error("expected isMandatory false")
		}
		return "cat_002", nil
	}

	id, err := h.svc.CreateFeeCategory(context.Background(), "tenant_001", "school_001", "Library Fee", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "cat_002" {
		t.Fatalf("expected id 'cat_002', got %q", id)
	}
}

// ============================================================================
// Tests: ListFeeCategories
// ============================================================================

func TestListFeeCategories_HappyPath(t *testing.T) {
	h := newTestHarness()

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

	categories, err := h.svc.ListFeeCategories(context.Background(), "tenant_001", "school_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(categories) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(categories))
	}
	if categories[0].Name != "Tuition" {
		t.Fatalf("expected name 'Tuition', got %q", categories[0].Name)
	}
}

func TestListFeeCategories_EmptyTenantID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.ListFeeCategories(context.Background(), "", "school_001")
	if err == nil {
		t.Fatal("expected error for empty tenantID, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestListFeeCategories_EmptySchoolID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.ListFeeCategories(context.Background(), "tenant_001", "")
	if err == nil {
		t.Fatal("expected error for empty schoolID, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestListFeeCategories_EmptyResults(t *testing.T) {
	h := newTestHarness()

	h.repo.listFeeCategoriesFn = func(ctx context.Context, tenantID, schoolID string) ([]FeeCategory, error) {
		return []FeeCategory{}, nil
	}

	categories, err := h.svc.ListFeeCategories(context.Background(), "tenant_001", "school_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(categories) != 0 {
		t.Fatalf("expected 0 categories, got %d", len(categories))
	}
}

// ============================================================================
// Tests: UpdateFeeCategory
// ============================================================================

func TestUpdateFeeCategory_HappyPath_Name(t *testing.T) {
	h := newTestHarness()

	newName := "Updated Tuition"

	h.repo.updateFeeCategoryFn = func(ctx context.Context, id, tenantID, schoolID string, name *string, isMandatory *bool) error {
		if id != "cat_001" {
			t.Errorf("expected id 'cat_001', got %q", id)
		}
		if name == nil || *name != "Updated Tuition" {
			t.Errorf("expected name 'Updated Tuition', got %v", name)
		}
		if isMandatory != nil {
			t.Error("expected isMandatory nil")
		}
		return nil
	}

	err := h.svc.UpdateFeeCategory(context.Background(), "cat_001", "tenant_001", "school_001", &newName, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateFeeCategory_EmptyID(t *testing.T) {
	h := newTestHarness()

	newName := "Tuition"
	err := h.svc.UpdateFeeCategory(context.Background(), "", "tenant_001", "school_001", &newName, nil)
	if err == nil {
		t.Fatal("expected error for empty id, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestUpdateFeeCategory_NoFields(t *testing.T) {
	h := newTestHarness()

	err := h.svc.UpdateFeeCategory(context.Background(), "cat_001", "tenant_001", "school_001", nil, nil)
	if err == nil {
		t.Fatal("expected error for no fields, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestUpdateFeeCategory_EmptyName(t *testing.T) {
	h := newTestHarness()

	emptyName := ""
	err := h.svc.UpdateFeeCategory(context.Background(), "cat_001", "tenant_001", "school_001", &emptyName, nil)
	if err == nil {
		t.Fatal("expected error for empty name, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestUpdateFeeCategory_NotFound(t *testing.T) {
	h := newTestHarness()

	h.repo.updateFeeCategoryFn = func(ctx context.Context, id, tenantID, schoolID string, name *string, isMandatory *bool) error {
		return ErrNotFound
	}

	newName := "NonExistent"
	err := h.svc.UpdateFeeCategory(context.Background(), "cat_999", "tenant_001", "school_001", &newName, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// ============================================================================
// Tests: DeleteFeeCategory
// ============================================================================

func TestDeleteFeeCategory_HappyPath(t *testing.T) {
	h := newTestHarness()

	h.repo.deleteFeeCategoryFn = func(ctx context.Context, id, tenantID, schoolID string) error {
		if id != "cat_001" {
			t.Errorf("expected id 'cat_001', got %q", id)
		}
		return nil
	}

	err := h.svc.DeleteFeeCategory(context.Background(), "cat_001", "tenant_001", "school_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteFeeCategory_EmptyID(t *testing.T) {
	h := newTestHarness()

	err := h.svc.DeleteFeeCategory(context.Background(), "", "tenant_001", "school_001")
	if err == nil {
		t.Fatal("expected error for empty id, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestDeleteFeeCategory_NotFound(t *testing.T) {
	h := newTestHarness()

	h.repo.deleteFeeCategoryFn = func(ctx context.Context, id, tenantID, schoolID string) error {
		return ErrNotFound
	}

	err := h.svc.DeleteFeeCategory(context.Background(), "cat_999", "tenant_001", "school_001")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// ============================================================================
// Tests: CreateFeeTemplate
// ============================================================================

func TestCreateFeeTemplate_HappyPath(t *testing.T) {
	h := newTestHarness()

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

	id, err := h.svc.CreateFeeTemplate(context.Background(), "tenant_001", "school_001", "term_001", "G1", "cat_001", "5000.00")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "tmp_001" {
		t.Fatalf("expected id 'tmp_001', got %q", id)
	}
}

func TestCreateFeeTemplate_NegativeAmount(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.CreateFeeTemplate(context.Background(), "tenant_001", "school_001", "term_001", "G1", "cat_001", "-100.00")
	if err == nil {
		t.Fatal("expected error for negative amount, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}

	var fe interface{ FieldErrors() map[string][]string }
	if errors.As(err, &fe) {
		fields := fe.FieldErrors()
		if fields["amount"] == nil {
			t.Fatal("expected field error for 'amount'")
		}
	}
}

func TestCreateFeeTemplate_ZeroAmount(t *testing.T) {
	h := newTestHarness()

	h.repo.createFeeTemplateFn = func(ctx context.Context, tenantID, schoolID, academicTermID, gradeLevel, feeCategoryID, amount string) (string, error) {
		if amount != "0.00" {
			t.Errorf("expected amount '0.00', got %q", amount)
		}
		return "tmp_002", nil
	}

	id, err := h.svc.CreateFeeTemplate(context.Background(), "tenant_001", "school_001", "term_001", "G1", "cat_001", "0.00")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "tmp_002" {
		t.Fatalf("expected id 'tmp_002', got %q", id)
	}
}

func TestCreateFeeTemplate_EmptyAmount(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.CreateFeeTemplate(context.Background(), "tenant_001", "school_001", "term_001", "G1", "cat_001", "")
	if err == nil {
		t.Fatal("expected error for empty amount, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateFeeTemplate_InvalidAmount(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.CreateFeeTemplate(context.Background(), "tenant_001", "school_001", "term_001", "G1", "cat_001", "not-a-number")
	if err == nil {
		t.Fatal("expected error for invalid amount, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateFeeTemplate_InvalidGradeLevel(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.CreateFeeTemplate(context.Background(), "tenant_001", "school_001", "term_001", "INVALID", "cat_001", "1000.00")
	if err == nil {
		t.Fatal("expected error for invalid grade level, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateFeeTemplate_EmptyAcademicTermID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.CreateFeeTemplate(context.Background(), "tenant_001", "school_001", "", "G1", "cat_001", "1000.00")
	if err == nil {
		t.Fatal("expected error for empty academicTermID, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateFeeTemplate_EmptyFeeCategoryID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.CreateFeeTemplate(context.Background(), "tenant_001", "school_001", "term_001", "G1", "", "1000.00")
	if err == nil {
		t.Fatal("expected error for empty feeCategoryID, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

// ============================================================================
// Tests: ListFeeTemplates
// ============================================================================

func TestListFeeTemplates_HappyPath(t *testing.T) {
	h := newTestHarness()

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
		if academicTermID != nil {
			t.Errorf("expected nil academicTermID, got %q", *academicTermID)
		}
		if gradeLevel != nil {
			t.Errorf("expected nil gradeLevel, got %q", *gradeLevel)
		}
		return expected, nil
	}

	templates, err := h.svc.ListFeeTemplates(context.Background(), "tenant_001", "school_001", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(templates) != 2 {
		t.Fatalf("expected 2 templates, got %d", len(templates))
	}
}

func TestListFeeTemplates_FilterByTerm(t *testing.T) {
	h := newTestHarness()

	termID := "term_001"

	h.repo.listFeeTemplatesFn = func(ctx context.Context, tenantID, schoolID string, academicTermID, gradeLevel *string) ([]FeeTemplate, error) {
		if academicTermID == nil || *academicTermID != "term_001" {
			t.Errorf("expected academicTermID 'term_001', got %v", academicTermID)
		}
		return []FeeTemplate{
			{ID: "tmp_001", AcademicTermID: "term_001", Amount: "5000.00"},
		}, nil
	}

	templates, err := h.svc.ListFeeTemplates(context.Background(), "tenant_001", "school_001", &termID, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(templates) != 1 {
		t.Fatalf("expected 1 template, got %d", len(templates))
	}
}

func TestListFeeTemplates_FilterByGradeLevel(t *testing.T) {
	h := newTestHarness()

	grade := "G1"

	h.repo.listFeeTemplatesFn = func(ctx context.Context, tenantID, schoolID string, academicTermID, gradeLevel *string) ([]FeeTemplate, error) {
		if gradeLevel == nil || *gradeLevel != "G1" {
			t.Errorf("expected gradeLevel 'G1', got %v", gradeLevel)
		}
		return []FeeTemplate{
			{ID: "tmp_001", GradeLevel: "G1", Amount: "5000.00"},
		}, nil
	}

	templates, err := h.svc.ListFeeTemplates(context.Background(), "tenant_001", "school_001", nil, &grade)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(templates) != 1 {
		t.Fatalf("expected 1 template, got %d", len(templates))
	}
}

func TestListFeeTemplates_InvalidGradeLevelFilter(t *testing.T) {
	h := newTestHarness()

	invalidGrade := "INVALID"
	_, err := h.svc.ListFeeTemplates(context.Background(), "tenant_001", "school_001", nil, &invalidGrade)
	if err == nil {
		t.Fatal("expected error for invalid grade level filter, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

// ============================================================================
// Tests: UpdateFeeTemplate
// ============================================================================

func TestUpdateFeeTemplate_HappyPath(t *testing.T) {
	h := newTestHarness()

	newAmount := "6000.00"

	h.repo.updateFeeTemplateFn = func(ctx context.Context, id, tenantID, schoolID string, amount *string) error {
		if id != "tmp_001" {
			t.Errorf("expected id 'tmp_001', got %q", id)
		}
		if amount == nil || *amount != "6000.00" {
			t.Errorf("expected amount '6000.00', got %v", amount)
		}
		return nil
	}

	err := h.svc.UpdateFeeTemplate(context.Background(), "tmp_001", "tenant_001", "school_001", &newAmount)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateFeeTemplate_EmptyID(t *testing.T) {
	h := newTestHarness()

	amount := "6000.00"
	err := h.svc.UpdateFeeTemplate(context.Background(), "", "tenant_001", "school_001", &amount)
	if err == nil {
		t.Fatal("expected error for empty id, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestUpdateFeeTemplate_NilAmount(t *testing.T) {
	h := newTestHarness()

	err := h.svc.UpdateFeeTemplate(context.Background(), "tmp_001", "tenant_001", "school_001", nil)
	if err == nil {
		t.Fatal("expected error for nil amount, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestUpdateFeeTemplate_NegativeAmount(t *testing.T) {
	h := newTestHarness()

	negAmount := "-100.00"
	err := h.svc.UpdateFeeTemplate(context.Background(), "tmp_001", "tenant_001", "school_001", &negAmount)
	if err == nil {
		t.Fatal("expected error for negative amount, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestUpdateFeeTemplate_NotFound(t *testing.T) {
	h := newTestHarness()

	amount := "6000.00"
	h.repo.updateFeeTemplateFn = func(ctx context.Context, id, tenantID, schoolID string, amount *string) error {
		return ErrNotFound
	}

	err := h.svc.UpdateFeeTemplate(context.Background(), "tmp_999", "tenant_001", "school_001", &amount)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// ============================================================================
// Tests: DeleteFeeTemplate
// ============================================================================

func TestDeleteFeeTemplate_HappyPath(t *testing.T) {
	h := newTestHarness()

	h.repo.deleteFeeTemplateFn = func(ctx context.Context, id, tenantID, schoolID string) error {
		if id != "tmp_001" {
			t.Errorf("expected id 'tmp_001', got %q", id)
		}
		return nil
	}

	err := h.svc.DeleteFeeTemplate(context.Background(), "tmp_001", "tenant_001", "school_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteFeeTemplate_EmptyID(t *testing.T) {
	h := newTestHarness()

	err := h.svc.DeleteFeeTemplate(context.Background(), "", "tenant_001", "school_001")
	if err == nil {
		t.Fatal("expected error for empty id, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestDeleteFeeTemplate_NotFound(t *testing.T) {
	h := newTestHarness()

	h.repo.deleteFeeTemplateFn = func(ctx context.Context, id, tenantID, schoolID string) error {
		return ErrNotFound
	}

	err := h.svc.DeleteFeeTemplate(context.Background(), "tmp_999", "tenant_001", "school_001")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// ============================================================================
// Tests: GenerateInvoice
// ============================================================================

func TestGenerateInvoice_HappyPath_AutoItems(t *testing.T) {
	h := newTestHarness()

	h.repo.resolveGradeLevelFn = func(ctx context.Context, tenantID, studentID, academicTermID string) (string, error) {
		if studentID != "stu_001" {
			t.Errorf("expected studentID 'stu_001', got %q", studentID)
		}
		return "G4", nil
	}

	h.repo.listFeeTemplatesByTermAndGradeFn = func(ctx context.Context, tenantID, schoolID, academicTermID, gradeLevel string) ([]FeeTemplate, error) {
		if gradeLevel != "G4" {
			t.Errorf("expected gradeLevel 'G4', got %q", gradeLevel)
		}
		return []FeeTemplate{
			{ID: "tmp_001", FeeCategoryID: "cat_001", Amount: "5000.00"},
			{ID: "tmp_002", FeeCategoryID: "cat_002", Amount: "3000.00"},
		}, nil
	}

	h.repo.createInvoiceFn = func(ctx context.Context, tenantID, schoolID, studentID, academicTermID string, parentID *string, invoiceLabel *string, amountDue string) (string, error) {
		if amountDue != "8000.00" {
			t.Errorf("expected amountDue '8000.00', got %q", amountDue)
		}
		return "inv_001", nil
	}

	itemCount := 0
	h.repo.createInvoiceItemFn = func(ctx context.Context, tenantID, invoiceID, feeCategoryID, description, amount string) error {
		itemCount++
		if invoiceID != "inv_001" {
			t.Errorf("expected invoiceID 'inv_001', got %q", invoiceID)
		}
		return nil
	}

	result, err := h.svc.GenerateInvoice(context.Background(), "tenant_001", "school_001", GenerateInvoicePayload{
		StudentID:      "stu_001",
		AcademicTermID: "term_001",
		InvoiceLabel:   strPtr("Term 1 Fees"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if itemCount != 2 {
		t.Fatalf("expected 2 invoice items, got %d", itemCount)
	}
}

func TestGenerateInvoice_HappyPath_ExplicitItems(t *testing.T) {
	h := newTestHarness()

	h.repo.createInvoiceFn = func(ctx context.Context, tenantID, schoolID, studentID, academicTermID string, parentID *string, invoiceLabel *string, amountDue string) (string, error) {
		if amountDue != "8500.00" {
			t.Errorf("expected amountDue '8500.00', got %q", amountDue)
		}
		return "inv_002", nil
	}

	itemCount := 0
	h.repo.createInvoiceItemFn = func(ctx context.Context, tenantID, invoiceID, feeCategoryID, description, amount string) error {
		itemCount++
		return nil
	}

	result, err := h.svc.GenerateInvoice(context.Background(), "tenant_001", "school_001", GenerateInvoicePayload{
		StudentID:      "stu_001",
		AcademicTermID: "term_001",
		Items: []InvoiceItemInput{
			{FeeCategoryID: "cat_001", Amount: "5000.00", Description: strPtr("Tuition")},
			{FeeCategoryID: "cat_002", Amount: "3500.00"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if itemCount != 2 {
		t.Fatalf("expected 2 invoice items, got %d", itemCount)
	}
}

func TestGenerateInvoice_MissingStudentID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.GenerateInvoice(context.Background(), "tenant_001", "school_001", GenerateInvoicePayload{
		AcademicTermID: "term_001",
	})
	if err == nil {
		t.Fatal("expected error for missing student_id, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestGenerateInvoice_MissingAcademicTermID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.GenerateInvoice(context.Background(), "tenant_001", "school_001", GenerateInvoicePayload{
		StudentID: "stu_001",
	})
	if err == nil {
		t.Fatal("expected error for missing academic_term_id, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestGenerateInvoice_GradeLevelNotFound(t *testing.T) {
	h := newTestHarness()

	h.repo.resolveGradeLevelFn = func(ctx context.Context, tenantID, studentID, academicTermID string) (string, error) {
		return "", ErrNotFound
	}

	_, err := h.svc.GenerateInvoice(context.Background(), "tenant_001", "school_001", GenerateInvoicePayload{
		StudentID:      "stu_001",
		AcademicTermID: "term_001",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGenerateInvoice_NoTemplates(t *testing.T) {
	h := newTestHarness()

	h.repo.resolveGradeLevelFn = func(ctx context.Context, tenantID, studentID, academicTermID string) (string, error) {
		return "G4", nil
	}

	h.repo.listFeeTemplatesByTermAndGradeFn = func(ctx context.Context, tenantID, schoolID, academicTermID, gradeLevel string) ([]FeeTemplate, error) {
		return []FeeTemplate{}, nil
	}

	_, err := h.svc.GenerateInvoice(context.Background(), "tenant_001", "school_001", GenerateInvoicePayload{
		StudentID:      "stu_001",
		AcademicTermID: "term_001",
	})
	if err == nil {
		t.Fatal("expected error for no fee templates, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGenerateInvoice_Duplicate(t *testing.T) {
	h := newTestHarness()

	h.repo.resolveGradeLevelFn = func(ctx context.Context, tenantID, studentID, academicTermID string) (string, error) {
		return "G1", nil
	}

	h.repo.createInvoiceFn = func(ctx context.Context, tenantID, schoolID, studentID, academicTermID string, parentID *string, invoiceLabel *string, amountDue string) (string, error) {
		return "", ErrAlreadyExists
	}

	_, err := h.svc.GenerateInvoice(context.Background(), "tenant_001", "school_001", GenerateInvoicePayload{
		StudentID:      "stu_001",
		AcademicTermID: "term_001",
	})
	if err == nil {
		t.Fatal("expected error for duplicate, got nil")
	}
	if !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}
}

// ============================================================================
// Tests: GetInvoiceDetail
// ============================================================================

func TestGetInvoiceDetail_HappyPath(t *testing.T) {
	h := newTestHarness()

	expected := &InvoiceDetailResponse{
		Invoice: Invoice{ID: "inv_001", PaymentStatus: "UNPAID", AmountDue: "8000.00"},
		Items: []InvoiceItem{
			{ID: "item_001", FeeCategoryID: "cat_001", Amount: "5000.00"},
		},
		Payments: []Payment{
			{ID: "pay_001", Amount: "3000.00"},
		},
	}

	h.repo.getInvoiceDetailFn = func(ctx context.Context, id, tenantID, schoolID string) (*InvoiceDetailResponse, error) {
		if id != "inv_001" {
			t.Errorf("expected id 'inv_001', got %q", id)
		}
		return expected, nil
	}

	result, err := h.svc.GetInvoiceDetail(context.Background(), "inv_001", "tenant_001", "school_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Invoice.ID != "inv_001" {
		t.Fatalf("expected invoice ID 'inv_001', got %q", result.Invoice.ID)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}
	if len(result.Payments) != 1 {
		t.Fatalf("expected 1 payment, got %d", len(result.Payments))
	}
}

func TestGetInvoiceDetail_EmptyID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.GetInvoiceDetail(context.Background(), "", "tenant_001", "school_001")
	if err == nil {
		t.Fatal("expected error for empty id, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestGetInvoiceDetail_NotFound(t *testing.T) {
	h := newTestHarness()

	h.repo.getInvoiceDetailFn = func(ctx context.Context, id, tenantID, schoolID string) (*InvoiceDetailResponse, error) {
		return nil, ErrNotFound
	}

	_, err := h.svc.GetInvoiceDetail(context.Background(), "inv_999", "tenant_001", "school_001")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// ============================================================================
// Tests: ListInvoices
// ============================================================================

func TestListInvoices_HappyPath(t *testing.T) {
	h := newTestHarness()

	expectedInvoices := []Invoice{
		{ID: "inv_001", StudentID: "stu_001", PaymentStatus: "UNPAID"},
		{ID: "inv_002", StudentID: "stu_002", PaymentStatus: "PAID"},
	}

	h.repo.listInvoicesFn = func(ctx context.Context, tenantID, schoolID string, filter InvoiceFilter) ([]Invoice, int, error) {
		if filter.StudentID != nil {
			t.Errorf("expected nil StudentID filter, got %q", *filter.StudentID)
		}
		return expectedInvoices, 2, nil
	}

	result, err := h.svc.ListInvoices(context.Background(), "tenant_001", "school_001", InvoiceFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 2 {
		t.Fatalf("expected total 2, got %d", result.Total)
	}
	if len(result.Invoices) != 2 {
		t.Fatalf("expected 2 invoices, got %d", len(result.Invoices))
	}
}

func TestListInvoices_Filtered(t *testing.T) {
	h := newTestHarness()

	studentID := "stu_001"
	termID := "term_001"
	status := "PARTIAL"

	h.repo.listInvoicesFn = func(ctx context.Context, tenantID, schoolID string, filter InvoiceFilter) ([]Invoice, int, error) {
		if filter.StudentID == nil || *filter.StudentID != "stu_001" {
			t.Errorf("expected StudentID 'stu_001', got %v", filter.StudentID)
		}
		if filter.AcademicTermID == nil || *filter.AcademicTermID != "term_001" {
			t.Errorf("expected AcademicTermID 'term_001', got %v", filter.AcademicTermID)
		}
		if filter.PaymentStatus == nil || *filter.PaymentStatus != "PARTIAL" {
			t.Errorf("expected PaymentStatus 'PARTIAL', got %v", filter.PaymentStatus)
		}
		return []Invoice{{ID: "inv_001", StudentID: "stu_001", PaymentStatus: "PARTIAL"}}, 1, nil
	}

	result, err := h.svc.ListInvoices(context.Background(), "tenant_001", "school_001", InvoiceFilter{
		StudentID:      &studentID,
		AcademicTermID: &termID,
		PaymentStatus:  &status,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("expected total 1, got %d", result.Total)
	}
}

func TestListInvoices_EmptyTenantID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.ListInvoices(context.Background(), "", "school_001", InvoiceFilter{})
	if err == nil {
		t.Fatal("expected error for empty tenantID, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

// ============================================================================
// Tests: WaiveInvoice
// ============================================================================

func TestWaiveInvoice_HappyPath(t *testing.T) {
	h := newTestHarness()

	h.repo.getInvoiceByIDFn = func(ctx context.Context, id, tenantID, schoolID string) (*Invoice, error) {
		return &Invoice{ID: id, PaymentStatus: "UNPAID"}, nil
	}

	called := false
	h.repo.waiveInvoiceFn = func(ctx context.Context, id, tenantID, schoolID string) error {
		called = true
		if id != "inv_001" {
			t.Errorf("expected id 'inv_001', got %q", id)
		}
		return nil
	}

	err := h.svc.WaiveInvoice(context.Background(), "inv_001", "tenant_001", "school_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("expected waiveInvoiceFn to be called")
	}
}

func TestWaiveInvoice_PaidInvoice(t *testing.T) {
	h := newTestHarness()

	h.repo.getInvoiceByIDFn = func(ctx context.Context, id, tenantID, schoolID string) (*Invoice, error) {
		return &Invoice{ID: id, PaymentStatus: "PAID"}, nil
	}

	err := h.svc.WaiveInvoice(context.Background(), "inv_001", "tenant_001", "school_001")
	if err == nil {
		t.Fatal("expected error for PAID invoice, got nil")
	}
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}

func TestWaiveInvoice_AlreadyWaived(t *testing.T) {
	h := newTestHarness()

	h.repo.getInvoiceByIDFn = func(ctx context.Context, id, tenantID, schoolID string) (*Invoice, error) {
		return &Invoice{ID: id, PaymentStatus: "WAIVED"}, nil
	}

	err := h.svc.WaiveInvoice(context.Background(), "inv_001", "tenant_001", "school_001")
	if err == nil {
		t.Fatal("expected error for already WAIVED invoice, got nil")
	}
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}

func TestWaiveInvoice_EmptyID(t *testing.T) {
	h := newTestHarness()

	err := h.svc.WaiveInvoice(context.Background(), "", "tenant_001", "school_001")
	if err == nil {
		t.Fatal("expected error for empty id, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestWaiveInvoice_NotFound(t *testing.T) {
	h := newTestHarness()

	h.repo.getInvoiceByIDFn = func(ctx context.Context, id, tenantID, schoolID string) (*Invoice, error) {
		return nil, ErrNotFound
	}

	err := h.svc.WaiveInvoice(context.Background(), "inv_999", "tenant_001", "school_001")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// ============================================================================
// Tests: RecordPayment
// ============================================================================

func TestRecordPayment_HappyPath(t *testing.T) {
	h := newTestHarness()

	h.repo.recordPaymentFn = func(ctx context.Context, tenantID, invoiceID, amount, recordedBy string, parentID, paymentMethod, referenceCode *string) (string, error) {
		if invoiceID != "inv_001" {
			t.Errorf("expected invoiceID 'inv_001', got %q", invoiceID)
		}
		if amount != "5000.00" {
			t.Errorf("expected amount '5000.00', got %q", amount)
		}
		if recordedBy != "user_001" {
			t.Errorf("expected recordedBy 'user_001', got %q", recordedBy)
		}
		return "pay_001", nil
	}

	h.repo.getPaymentByIDFn = func(ctx context.Context, id, tenantID string) (*Payment, error) {
		return &Payment{ID: "pay_001", Amount: "5000.00", InvoiceID: "inv_001"}, nil
	}

	h.repo.getInvoiceByIDFn = func(ctx context.Context, id, tenantID, schoolID string) (*Invoice, error) {
		return &Invoice{ID: id, PaymentStatus: "UNPAID"}, nil
	}

	payment, err := h.svc.RecordPayment(context.Background(), "tenant_001", "school_001", "user_001", RecordPaymentPayload{
		InvoiceID: "inv_001",
		Amount:    "5000.00",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if payment.ID != "pay_001" {
		t.Fatalf("expected payment ID 'pay_001', got %q", payment.ID)
	}
}

func TestRecordPayment_NegativeAmount(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.RecordPayment(context.Background(), "tenant_001", "school_001", "user_001", RecordPaymentPayload{
		InvoiceID: "inv_001",
		Amount:    "-100.00",
	})
	if err == nil {
		t.Fatal("expected error for negative amount, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestRecordPayment_ZeroAmount(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.RecordPayment(context.Background(), "tenant_001", "school_001", "user_001", RecordPaymentPayload{
		InvoiceID: "inv_001",
		Amount:    "0.00",
	})
	if err == nil {
		t.Fatal("expected error for zero amount, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestRecordPayment_MissingInvoiceID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.RecordPayment(context.Background(), "tenant_001", "school_001", "user_001", RecordPaymentPayload{
		Amount: "5000.00",
	})
	if err == nil {
		t.Fatal("expected error for missing invoice_id, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestRecordPayment_WaivedInvoice(t *testing.T) {
	h := newTestHarness()

	h.repo.getInvoiceByIDFn = func(ctx context.Context, id, tenantID, schoolID string) (*Invoice, error) {
		return &Invoice{ID: id, PaymentStatus: "WAIVED"}, nil
	}

	_, err := h.svc.RecordPayment(context.Background(), "tenant_001", "school_001", "user_001", RecordPaymentPayload{
		InvoiceID: "inv_001",
		Amount:    "5000.00",
	})
	if err == nil {
		t.Fatal("expected error for WAIVED invoice, got nil")
	}
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}

// ============================================================================
// Tests: ListPayments
// ============================================================================

func TestListPayments_HappyPath(t *testing.T) {
	h := newTestHarness()

	expected := []Payment{
		{ID: "pay_001", Amount: "3000.00"},
		{ID: "pay_002", Amount: "2000.00"},
	}

	h.repo.listPaymentsFn = func(ctx context.Context, tenantID, invoiceID string) ([]Payment, error) {
		if invoiceID != "inv_001" {
			t.Errorf("expected invoiceID 'inv_001', got %q", invoiceID)
		}
		return expected, nil
	}

	result, err := h.svc.ListPayments(context.Background(), "tenant_001", "inv_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 2 {
		t.Fatalf("expected total 2, got %d", result.Total)
	}
	if len(result.Payments) != 2 {
		t.Fatalf("expected 2 payments, got %d", len(result.Payments))
	}
}

func TestListPayments_EmptyInvoiceID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.ListPayments(context.Background(), "tenant_001", "")
	if err == nil {
		t.Fatal("expected error for empty invoiceID, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestListPayments_EmptyTenantID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.ListPayments(context.Background(), "", "inv_001")
	if err == nil {
		t.Fatal("expected error for empty tenantID, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}
