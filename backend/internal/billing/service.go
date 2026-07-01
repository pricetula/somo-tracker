package billing

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"somotracker/backend/internal/middleware"
)

// validGradeLevels defines the set of valid CBC grade level values.
var validGradeLevels = map[string]bool{
	"PP1": true, "PP2": true, "G1": true, "G2": true, "G3": true,
	"G4": true, "G5": true, "G6": true, "G7": true, "G8": true,
	"G9": true, "G10": true, "G11": true, "G12": true,
}

// Service contains business logic for the billing domain.
type Service struct {
	Repo Repository
}

// NewService creates a new Service.
func NewService(repo Repository) *Service {
	return &Service{Repo: repo}
}

// ─── Fee Categories ────────────────────────────────────────────────────────

// CreateFeeCategory creates a new fee category and returns its ID.
func (s *Service) CreateFeeCategory(ctx context.Context, tenantID, schoolID, name string, isMandatory bool) (string, error) {
	if strings.TrimSpace(name) == "" {
		return "", &middleware.FieldError{
			Err:    fmt.Errorf("billing.Service.CreateFeeCategory: %w", ErrInvalidInput),
			Fields: map[string][]string{"name": {"name is required"}},
		}
	}
	if len(name) > 150 {
		return "", &middleware.FieldError{
			Err:    fmt.Errorf("billing.Service.CreateFeeCategory: %w", ErrInvalidInput),
			Fields: map[string][]string{"name": {"name must not exceed 150 characters"}},
		}
	}
	return s.Repo.CreateFeeCategory(ctx, tenantID, schoolID, name, isMandatory)
}

// ListFeeCategories returns all fee categories for a tenant and school.
func (s *Service) ListFeeCategories(ctx context.Context, tenantID, schoolID string) ([]FeeCategory, error) {
	if tenantID == "" || schoolID == "" {
		return nil, fmt.Errorf("billing.Service.ListFeeCategories: %w", ErrInvalidInput)
	}
	return s.Repo.ListFeeCategories(ctx, tenantID, schoolID)
}

// UpdateFeeCategory applies partial updates to a fee category.
func (s *Service) UpdateFeeCategory(ctx context.Context, id, tenantID, schoolID string, name *string, isMandatory *bool) error {
	if id == "" {
		return fmt.Errorf("billing.Service.UpdateFeeCategory: %w", ErrInvalidInput)
	}
	if name == nil && isMandatory == nil {
		return fmt.Errorf("billing.Service.UpdateFeeCategory: %w", ErrInvalidInput)
	}
	if name != nil && strings.TrimSpace(*name) == "" {
		return &middleware.FieldError{
			Err:    fmt.Errorf("billing.Service.UpdateFeeCategory: %w", ErrInvalidInput),
			Fields: map[string][]string{"name": {"name is required"}},
		}
	}
	return s.Repo.UpdateFeeCategory(ctx, id, tenantID, schoolID, name, isMandatory)
}

// DeleteFeeCategory removes a fee category by ID.
func (s *Service) DeleteFeeCategory(ctx context.Context, id, tenantID, schoolID string) error {
	if id == "" {
		return fmt.Errorf("billing.Service.DeleteFeeCategory: %w", ErrInvalidInput)
	}
	return s.Repo.DeleteFeeCategory(ctx, id, tenantID, schoolID)
}

// ─── Fee Templates ─────────────────────────────────────────────────────────

// validateAmount checks that the amount string is a valid non-negative decimal.
func validateAmount(amount string) error {
	if strings.TrimSpace(amount) == "" {
		return &middleware.FieldError{
			Err:    fmt.Errorf("billing.Service: %w", ErrInvalidInput),
			Fields: map[string][]string{"amount": {"amount is required"}},
		}
	}
	a, ok := new(big.Rat).SetString(amount)
	if !ok || a == nil {
		return &middleware.FieldError{
			Err:    fmt.Errorf("billing.Service: %w", ErrInvalidInput),
			Fields: map[string][]string{"amount": {"amount must be a valid decimal number"}},
		}
	}
	if a.Sign() < 0 {
		return &middleware.FieldError{
			Err:    fmt.Errorf("billing.Service: %w", ErrInvalidInput),
			Fields: map[string][]string{"amount": {"amount must be >= 0"}},
		}
	}
	return nil
}

// validateGradeLevel checks that the grade level is a valid CBC grade level.
func validateGradeLevel(gradeLevel string) error {
	if !validGradeLevels[gradeLevel] {
		return &middleware.FieldError{
			Err:    fmt.Errorf("billing.Service: %w", ErrInvalidInput),
			Fields: map[string][]string{"grade_level": {fmt.Sprintf("invalid grade level %q; must be one of PP1, PP2, G1..G12", gradeLevel)}},
		}
	}
	return nil
}

// CreateFeeTemplate creates a new fee template and returns its ID.
func (s *Service) CreateFeeTemplate(ctx context.Context, tenantID, schoolID, academicTermID, gradeLevel, feeCategoryID, amount string) (string, error) {
	if strings.TrimSpace(academicTermID) == "" {
		return "", &middleware.FieldError{
			Err:    fmt.Errorf("billing.Service.CreateFeeTemplate: %w", ErrInvalidInput),
			Fields: map[string][]string{"academic_term_id": {"academic_term_id is required"}},
		}
	}
	if strings.TrimSpace(feeCategoryID) == "" {
		return "", &middleware.FieldError{
			Err:    fmt.Errorf("billing.Service.CreateFeeTemplate: %w", ErrInvalidInput),
			Fields: map[string][]string{"fee_category_id": {"fee_category_id is required"}},
		}
	}

	if err := validateGradeLevel(gradeLevel); err != nil {
		return "", err
	}
	if err := validateAmount(amount); err != nil {
		return "", err
	}

	return s.Repo.CreateFeeTemplate(ctx, tenantID, schoolID, academicTermID, gradeLevel, feeCategoryID, amount)
}

// ListFeeTemplates returns fee templates for a tenant and school,
// optionally filtered by academicTermID and/or gradeLevel.
func (s *Service) ListFeeTemplates(ctx context.Context, tenantID, schoolID string, academicTermID, gradeLevel *string) ([]FeeTemplate, error) {
	if tenantID == "" || schoolID == "" {
		return nil, fmt.Errorf("billing.Service.ListFeeTemplates: %w", ErrInvalidInput)
	}

	// Validate grade level filter if provided
	if gradeLevel != nil {
		if err := validateGradeLevel(*gradeLevel); err != nil {
			return nil, err
		}
	}

	return s.Repo.ListFeeTemplates(ctx, tenantID, schoolID, academicTermID, gradeLevel)
}

// UpdateFeeTemplate applies partial updates to a fee template.
func (s *Service) UpdateFeeTemplate(ctx context.Context, id, tenantID, schoolID string, amount *string) error {
	if id == "" {
		return fmt.Errorf("billing.Service.UpdateFeeTemplate: %w", ErrInvalidInput)
	}
	if amount == nil {
		return fmt.Errorf("billing.Service.UpdateFeeTemplate: %w", ErrInvalidInput)
	}
	if err := validateAmount(*amount); err != nil {
		return err
	}

	return s.Repo.UpdateFeeTemplate(ctx, id, tenantID, schoolID, amount)
}

// DeleteFeeTemplate removes a fee template by ID.
func (s *Service) DeleteFeeTemplate(ctx context.Context, id, tenantID, schoolID string) error {
	if id == "" {
		return fmt.Errorf("billing.Service.DeleteFeeTemplate: %w", ErrInvalidInput)
	}
	return s.Repo.DeleteFeeTemplate(ctx, id, tenantID, schoolID)
}

// ─── Invoices ───────────────────────────────────────────────────────────────

// validatePositiveAmount checks that the amount is a valid positive decimal.
func validatePositiveAmount(amount string) error {
	if strings.TrimSpace(amount) == "" {
		return &middleware.FieldError{
			Err:    fmt.Errorf("billing.Service: %w", ErrInvalidInput),
			Fields: map[string][]string{"amount": {"amount is required"}},
		}
	}
	a, ok := new(big.Rat).SetString(amount)
	if !ok || a == nil {
		return &middleware.FieldError{
			Err:    fmt.Errorf("billing.Service: %w", ErrInvalidInput),
			Fields: map[string][]string{"amount": {"amount must be a valid decimal number"}},
		}
	}
	if a.Sign() <= 0 {
		return &middleware.FieldError{
			Err:    fmt.Errorf("billing.Service: %w", ErrInvalidInput),
			Fields: map[string][]string{"amount": {"amount must be greater than 0"}},
		}
	}
	return nil
}

// GenerateInvoice creates a new invoice for a student+term. If no explicit items
// are provided, items are auto-generated from fee_templates matching the student's
// grade level and the given term.
func (s *Service) GenerateInvoice(ctx context.Context, tenantID, schoolID string, payload GenerateInvoicePayload) (*InvoiceDetailResponse, error) {
	// Validate required fields
	if strings.TrimSpace(payload.StudentID) == "" {
		return nil, &middleware.FieldError{
			Err:    fmt.Errorf("billing.Service.GenerateInvoice: %w", ErrInvalidInput),
			Fields: map[string][]string{"student_id": {"student_id is required"}},
		}
	}
	if strings.TrimSpace(payload.AcademicTermID) == "" {
		return nil, &middleware.FieldError{
			Err:    fmt.Errorf("billing.Service.GenerateInvoice: %w", ErrInvalidInput),
			Fields: map[string][]string{"academic_term_id": {"academic_term_id is required"}},
		}
	}

	// Resolve items
	var items []InvoiceItemInput
	if len(payload.Items) > 0 {
		// Use explicit items
		items = payload.Items
		for i, item := range items {
			if strings.TrimSpace(item.FeeCategoryID) == "" {
				return nil, &middleware.FieldError{
					Err:    fmt.Errorf("billing.Service.GenerateInvoice: %w", ErrInvalidInput),
					Fields: map[string][]string{fmt.Sprintf("items[%d].fee_category_id", i): {"fee_category_id is required"}},
				}
			}
			if err := validateAmount(item.Amount); err != nil {
				return nil, err
			}
		}
	} else {
		// Auto-generate from fee_templates
		gradeLevel, err := s.Repo.ResolveGradeLevel(ctx, tenantID, payload.StudentID, payload.AcademicTermID)
		if err != nil {
			return nil, fmt.Errorf("billing.Service.GenerateInvoice: %w", err)
		}

		templates, err := s.Repo.ListFeeTemplatesByTermAndGrade(ctx, tenantID, schoolID, payload.AcademicTermID, gradeLevel)
		if err != nil {
			return nil, fmt.Errorf("billing.Service.GenerateInvoice: %w", err)
		}
		if len(templates) == 0 {
			return nil, fmt.Errorf("billing.Service.GenerateInvoice: %w", ErrNotFound)
		}

		for _, t := range templates {
			items = append(items, InvoiceItemInput{
				FeeCategoryID: t.FeeCategoryID,
				Amount:        t.Amount,
			})
		}
	}

	// Calculate amount_due as sum of item amounts
	amountDue := new(big.Rat)
	for _, item := range items {
		amt, ok := new(big.Rat).SetString(item.Amount)
		if !ok {
			return nil, fmt.Errorf("billing.Service.GenerateInvoice: %w", ErrInvalidInput)
		}
		amountDue = amountDue.Add(amountDue, amt)
	}

	amountDueStr := amountDue.FloatString(2)

	// Create invoice (app-level; no transaction since we're using individual calls)
	// If the DB has unique_invoice_per_student_term, a duplicate will return an error.
	invoiceID, err := s.Repo.CreateInvoice(ctx, tenantID, schoolID, payload.StudentID, payload.AcademicTermID, nil, payload.InvoiceLabel, amountDueStr)
	if err != nil {
		return nil, fmt.Errorf("billing.Service.GenerateInvoice: %w", err)
	}

	// Create invoice items
	for _, item := range items {
		description := ""
		if item.Description != nil {
			description = *item.Description
		}
		if err := s.Repo.CreateInvoiceItem(ctx, tenantID, invoiceID, item.FeeCategoryID, description, item.Amount); err != nil {
			return nil, fmt.Errorf("billing.Service.GenerateInvoice: %w", err)
		}
	}

	// Return full invoice detail
	return s.Repo.GetInvoiceDetail(ctx, invoiceID, tenantID, schoolID)
}

// GetInvoiceByID returns an invoice by ID, scoped to tenant and school.
func (s *Service) GetInvoiceByID(ctx context.Context, id, tenantID, schoolID string) (*Invoice, error) {
	if id == "" {
		return nil, fmt.Errorf("billing.Service.GetInvoiceByID: %w", ErrInvalidInput)
	}
	return s.Repo.GetInvoiceByID(ctx, id, tenantID, schoolID)
}

// GetInvoiceDetail returns an invoice with its items and payments.
func (s *Service) GetInvoiceDetail(ctx context.Context, id, tenantID, schoolID string) (*InvoiceDetailResponse, error) {
	if id == "" {
		return nil, fmt.Errorf("billing.Service.GetInvoiceDetail: %w", ErrInvalidInput)
	}
	return s.Repo.GetInvoiceDetail(ctx, id, tenantID, schoolID)
}

// ListInvoices returns invoices filtered by optional student_id, academic_term_id, payment_status.
func (s *Service) ListInvoices(ctx context.Context, tenantID, schoolID string, filter InvoiceFilter) (*ListInvoicesResponse, error) {
	if tenantID == "" || schoolID == "" {
		return nil, fmt.Errorf("billing.Service.ListInvoices: %w", ErrInvalidInput)
	}

	invoices, total, err := s.Repo.ListInvoices(ctx, tenantID, schoolID, filter)
	if err != nil {
		return nil, fmt.Errorf("billing.Service.ListInvoices: %w", err)
	}

	return &ListInvoicesResponse{
		Invoices: invoices,
		Total:    total,
	}, nil
}

// WaiveInvoice sets an invoice's payment_status to WAIVED.
func (s *Service) WaiveInvoice(ctx context.Context, id, tenantID, schoolID string) error {
	if id == "" {
		return fmt.Errorf("billing.Service.WaiveInvoice: %w", ErrInvalidInput)
	}

	// Check invoice exists and is not already PAID
	inv, err := s.Repo.GetInvoiceByID(ctx, id, tenantID, schoolID)
	if err != nil {
		return fmt.Errorf("billing.Service.WaiveInvoice: %w", err)
	}
	if inv.PaymentStatus == string(PaymentStatusPaid) {
		return fmt.Errorf("billing.Service.WaiveInvoice: cannot waive a PAID invoice: %w", ErrConflict)
	}
	if inv.PaymentStatus == string(PaymentStatusWaived) {
		return fmt.Errorf("billing.Service.WaiveInvoice: %w", ErrConflict)
	}

	return s.Repo.WaiveInvoice(ctx, id, tenantID, schoolID)
}

// ─── Payments ───────────────────────────────────────────────────────────────

// RecordPayment records a payment against an invoice and returns the payment ID.
func (s *Service) RecordPayment(ctx context.Context, tenantID, schoolID, recordedBy string, payload RecordPaymentPayload) (*Payment, error) {
	// Validate amount
	if err := validatePositiveAmount(payload.Amount); err != nil {
		return nil, err
	}

	if strings.TrimSpace(payload.InvoiceID) == "" {
		return nil, &middleware.FieldError{
			Err:    fmt.Errorf("billing.Service.RecordPayment: %w", ErrInvalidInput),
			Fields: map[string][]string{"invoice_id": {"invoice_id is required"}},
		}
	}

	// Reference code optional but validate length if provided
	if payload.ReferenceCode != nil && len(*payload.ReferenceCode) > 100 {
		return nil, &middleware.FieldError{
			Err:    fmt.Errorf("billing.Service.RecordPayment: %w", ErrInvalidInput),
			Fields: map[string][]string{"reference_code": {"reference_code must not exceed 100 characters"}},
		}
	}

	// Verify the invoice exists and is not WAIVED (prevent payments on waived invoices)
	inv, err := s.Repo.GetInvoiceByID(ctx, payload.InvoiceID, tenantID, schoolID)
	if err != nil {
		return nil, fmt.Errorf("billing.Service.RecordPayment: %w", err)
	}
	if inv.PaymentStatus == string(PaymentStatusWaived) {
		return nil, fmt.Errorf("billing.Service.RecordPayment: cannot record payment on a WAIVED invoice: %w", ErrConflict)
	}

	paymentID, err := s.Repo.RecordPayment(ctx, tenantID, payload.InvoiceID, payload.Amount, recordedBy, payload.ParentID, payload.PaymentMethod, payload.ReferenceCode)
	if err != nil {
		return nil, fmt.Errorf("billing.Service.RecordPayment: %w", err)
	}

	return s.Repo.GetPaymentByID(ctx, paymentID, tenantID)
}

// ListPayments returns payments. If invoiceID is provided, filtered to that invoice.
func (s *Service) ListPayments(ctx context.Context, tenantID, invoiceID string) (*ListPaymentsResponse, error) {
	if tenantID == "" {
		return nil, fmt.Errorf("billing.Service.ListPayments: %w", ErrInvalidInput)
	}
	if strings.TrimSpace(invoiceID) == "" {
		return nil, &middleware.FieldError{
			Err:    fmt.Errorf("billing.Service.ListPayments: %w", ErrInvalidInput),
			Fields: map[string][]string{"invoice_id": {"invoice_id is required"}},
		}
	}

	payments, err := s.Repo.ListPayments(ctx, tenantID, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("billing.Service.ListPayments: %w", err)
	}

	return &ListPaymentsResponse{
		Payments: payments,
		Total:    len(payments),
	}, nil
}
