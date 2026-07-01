package billing

import (
	"context"
	"fmt"
	"time"

	"somotracker/backend/internal/middleware"
)

// Sentinel domain errors. Each wraps the corresponding middleware sentinel
// so that middleware.HTTPError can match them via errors.Is.
var (
	ErrNotFound      = fmt.Errorf("billing not found: %w", middleware.ErrNotFound)
	ErrAlreadyExists = fmt.Errorf("billing already exists: %w", middleware.ErrAlreadyExists)
	ErrInvalidInput  = fmt.Errorf("invalid billing input: %w", middleware.ErrInvalidInput)
	ErrUnauthorized  = fmt.Errorf("unauthorized: %w", middleware.ErrUnauthorized)
	ErrForbidden     = fmt.Errorf("forbidden: %w", middleware.ErrForbidden)
	ErrConflict      = fmt.Errorf("billing conflict: %w", middleware.ErrConflict)
)

// ─── Fee Category ──────────────────────────────────────────────────────────

// FeeCategory represents a category of fees (e.g. "Tuition", "Transport").
type FeeCategory struct {
	ID          string `db:"id"          json:"id"`
	TenantID    string `db:"tenant_id"   json:"tenant_id"`
	SchoolID    string `db:"school_id"   json:"school_id"`
	Name        string `db:"name"        json:"name"`
	IsMandatory bool   `db:"is_mandatory" json:"is_mandatory"`
}

// CreateFeeCategoryPayload is the request body for POST /api/v1/billing/fee-categories.
type CreateFeeCategoryPayload struct {
	Name        string `json:"name"`
	IsMandatory bool   `json:"is_mandatory"`
}

// UpdateFeeCategoryPayload is the request body for PUT /api/v1/billing/fee-categories/:id.
type UpdateFeeCategoryPayload struct {
	Name        *string `json:"name,omitempty"`
	IsMandatory *bool   `json:"is_mandatory,omitempty"`
}

// ListFeeCategoriesResponse wraps a list of fee categories.
type ListFeeCategoriesResponse struct {
	FeeCategories []FeeCategory `json:"fee_categories"`
	Total         int           `json:"total"`
}

// ─── Fee Template ──────────────────────────────────────────────────────────

// FeeTemplate represents a fee template that maps a fee category to a
// specific term and grade level with a fixed amount.
type FeeTemplate struct {
	ID             string    `db:"id"              json:"id"`
	TenantID       string    `db:"tenant_id"       json:"tenant_id"`
	SchoolID       string    `db:"school_id"       json:"school_id"`
	AcademicTermID string    `db:"academic_term_id" json:"academic_term_id"`
	GradeLevel     string    `db:"grade_level"     json:"grade_level"`
	FeeCategoryID  string    `db:"fee_category_id" json:"fee_category_id"`
	Amount         string    `db:"amount"          json:"amount"`
	CreatedAt      time.Time `db:"created_at"       json:"created_at"`
}

// CreateFeeTemplatePayload is the request body for POST /api/v1/billing/fee-templates.
type CreateFeeTemplatePayload struct {
	AcademicTermID string `json:"academic_term_id"`
	GradeLevel     string `json:"grade_level"`
	FeeCategoryID  string `json:"fee_category_id"`
	Amount         string `json:"amount"`
}

// UpdateFeeTemplatePayload is the request body for PUT /api/v1/billing/fee-templates/:id.
type UpdateFeeTemplatePayload struct {
	Amount *string `json:"amount,omitempty"`
}

// ListFeeTemplatesResponse wraps a list of fee templates with a total count.
type ListFeeTemplatesResponse struct {
	FeeTemplates []FeeTemplate `json:"fee_templates"`
	Total        int           `json:"total"`
}

// ─── Invoice ────────────────────────────────────────────────────────────────

// PaymentStatus represents the possible states of an invoice.
type PaymentStatus string

const (
	PaymentStatusUnpaid  PaymentStatus = "UNPAID"
	PaymentStatusPartial PaymentStatus = "PARTIAL"
	PaymentStatusPaid    PaymentStatus = "PAID"
	PaymentStatusWaived  PaymentStatus = "WAIVED"
)

// Invoice represents a per-student, per-term invoice.
type Invoice struct {
	ID             string    `json:"id"             db:"id"`
	TenantID       string    `json:"-"              db:"tenant_id"`
	StudentID      string    `json:"student_id"     db:"student_id"`
	SchoolID       string    `json:"-"              db:"school_id"`
	AcademicTermID string    `json:"academic_term_id" db:"academic_term_id"`
	ParentID       *string   `json:"parent_id,omitempty" db:"parent_id"`
	InvoiceLabel   *string   `json:"invoice_label,omitempty" db:"invoice_label"`
	PaymentStatus  string    `json:"payment_status" db:"payment_status"`
	AmountDue      string    `json:"amount_due"     db:"amount_due"`
	AmountPaid     string    `json:"amount_paid"    db:"amount_paid"`
	CreatedAt      time.Time `json:"created_at"     db:"created_at"`
}

// InvoiceItem represents a single line item on an invoice.
type InvoiceItem struct {
	ID            string `json:"id"              db:"id"`
	InvoiceID     string `json:"invoice_id"      db:"invoice_id"`
	FeeCategoryID string `json:"fee_category_id" db:"fee_category_id"`
	Description   string `json:"description,omitempty" db:"description"`
	Amount        string `json:"amount"          db:"amount"`
}

// Payment represents a payment recorded against an invoice.
type Payment struct {
	ID            string  `json:"id"              db:"id"`
	InvoiceID     string  `json:"invoice_id"      db:"invoice_id"`
	Amount        string  `json:"amount"          db:"amount"`
	ParentID      *string `json:"parent_id,omitempty" db:"parent_id"`
	PaymentMethod *string `json:"payment_method,omitempty" db:"payment_method"`
	ReferenceCode *string `json:"reference_code,omitempty" db:"reference_code"`
	RecordedBy    string  `json:"recorded_by"     db:"recorded_by"`
	CreatedAt     string  `json:"created_at"      db:"created_at"`
}

// ─── Payloads ───────────────────────────────────────────────────────────────

// InvoiceItemInput is used in GenerateInvoicePayload when providing explicit items.
type InvoiceItemInput struct {
	FeeCategoryID string  `json:"fee_category_id"`
	Description   *string `json:"description,omitempty"`
	Amount        string  `json:"amount"`
}

// GenerateInvoicePayload is the request body for POST /api/v1/billing/invoices/generate.
type GenerateInvoicePayload struct {
	StudentID      string             `json:"student_id"`
	AcademicTermID string             `json:"academic_term_id"`
	InvoiceLabel   *string            `json:"invoice_label,omitempty"`
	Items          []InvoiceItemInput `json:"items,omitempty"`
}

// RecordPaymentPayload is the request body for POST /api/v1/billing/payments.
type RecordPaymentPayload struct {
	InvoiceID     string  `json:"invoice_id"`
	Amount        string  `json:"amount"`
	ParentID      *string `json:"parent_id,omitempty"`
	PaymentMethod *string `json:"payment_method,omitempty"`
	ReferenceCode *string `json:"reference_code,omitempty"`
}

// ─── Responses ──────────────────────────────────────────────────────────────

// InvoiceDetailResponse is the full invoice view with nested items and payments.
type InvoiceDetailResponse struct {
	Invoice  Invoice       `json:"invoice"`
	Items    []InvoiceItem `json:"items"`
	Payments []Payment     `json:"payments"`
}

// ListInvoicesResponse wraps a list of invoices with a total count.
type ListInvoicesResponse struct {
	Invoices []Invoice `json:"invoices"`
	Total    int       `json:"total"`
}

// ListPaymentsResponse wraps a list of payments with a total count.
type ListPaymentsResponse struct {
	Payments []Payment `json:"payments"`
	Total    int       `json:"total"`
}

// InvoiceFilter holds query parameters for listing invoices.
type InvoiceFilter struct {
	StudentID      *string
	AcademicTermID *string
	PaymentStatus  *string
}

// ─── Repository Interface ──────────────────────────────────────────────────

// Repository defines the contract for billing persistence.
type Repository interface {
	// Fee categories
	CreateFeeCategory(ctx context.Context, tenantID, schoolID, name string, isMandatory bool) (string, error)
	ListFeeCategories(ctx context.Context, tenantID, schoolID string) ([]FeeCategory, error)
	GetFeeCategoryByID(ctx context.Context, id, tenantID, schoolID string) (*FeeCategory, error)
	UpdateFeeCategory(ctx context.Context, id, tenantID, schoolID string, name *string, isMandatory *bool) error
	DeleteFeeCategory(ctx context.Context, id, tenantID, schoolID string) error

	// Fee templates
	CreateFeeTemplate(ctx context.Context, tenantID, schoolID, academicTermID, gradeLevel, feeCategoryID, amount string) (string, error)
	ListFeeTemplates(ctx context.Context, tenantID, schoolID string, academicTermID, gradeLevel *string) ([]FeeTemplate, error)
	GetFeeTemplateByID(ctx context.Context, id, tenantID, schoolID string) (*FeeTemplate, error)
	UpdateFeeTemplate(ctx context.Context, id, tenantID, schoolID string, amount *string) error
	DeleteFeeTemplate(ctx context.Context, id, tenantID, schoolID string) error

	// Invoices
	CreateInvoice(ctx context.Context, tenantID, schoolID, studentID, academicTermID string, parentID *string, invoiceLabel *string, amountDue string) (string, error)
	CreateInvoiceItem(ctx context.Context, tenantID, invoiceID, feeCategoryID, description, amount string) error
	GetInvoiceByID(ctx context.Context, id, tenantID, schoolID string) (*Invoice, error)
	ListInvoices(ctx context.Context, tenantID, schoolID string, filter InvoiceFilter) ([]Invoice, int, error)
	GetInvoiceItems(ctx context.Context, invoiceID, tenantID string) ([]InvoiceItem, error)
	GetInvoiceDetail(ctx context.Context, id, tenantID, schoolID string) (*InvoiceDetailResponse, error)
	WaiveInvoice(ctx context.Context, id, tenantID, schoolID string) error
	// ResolveGradeLevel returns the student's grade_level for a given term.
	ResolveGradeLevel(ctx context.Context, tenantID, studentID, academicTermID string) (string, error)
	// ListFeeTemplatesByTermAndGrade fetches fee templates for a term + grade within a school.
	ListFeeTemplatesByTermAndGrade(ctx context.Context, tenantID, schoolID, academicTermID, gradeLevel string) ([]FeeTemplate, error)

	// Payments
	RecordPayment(ctx context.Context, tenantID, invoiceID, amount, recordedBy string, parentID, paymentMethod, referenceCode *string) (string, error)
	ListPayments(ctx context.Context, tenantID, invoiceID string) ([]Payment, error)
	GetPaymentByID(ctx context.Context, id, tenantID string) (*Payment, error)
}
