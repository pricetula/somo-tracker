package billing

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"somotracker/backend/internal/database"
)

// PgRepository handles billing database operations.
type PgRepository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new PgRepository.
func NewRepository(pools *database.Pools) *PgRepository {
	return &PgRepository{pool: pools.PG}
}

// ─── Fee Categories ────────────────────────────────────────────────────────

// CreateFeeCategory inserts a new fee category and returns its ID.
func (r *PgRepository) CreateFeeCategory(ctx context.Context, tenantID, schoolID, name string, isMandatory bool) (string, error) {
	const query = `
		INSERT INTO fee_categories (tenant_id, school_id, name, is_mandatory)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`
	var id string
	err := r.pool.QueryRow(ctx, query, tenantID, schoolID, name, isMandatory).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("billing.Repository.CreateFeeCategory: %w", err)
	}
	return id, nil
}

// ListFeeCategories retrieves all fee categories for a tenant and school.
func (r *PgRepository) ListFeeCategories(ctx context.Context, tenantID, schoolID string) ([]FeeCategory, error) {
	const query = `
		SELECT id, tenant_id, school_id, name, is_mandatory
		FROM fee_categories
		WHERE tenant_id = $1 AND school_id = $2
		ORDER BY name ASC
	`
	rows, err := r.pool.Query(ctx, query, tenantID, schoolID)
	if err != nil {
		return nil, fmt.Errorf("billing.Repository.ListFeeCategories: %w", err)
	}
	defer rows.Close()

	var categories []FeeCategory
	for rows.Next() {
		var c FeeCategory
		if err := rows.Scan(&c.ID, &c.TenantID, &c.SchoolID, &c.Name, &c.IsMandatory); err != nil {
			return nil, fmt.Errorf("billing.Repository.ListFeeCategories: scan: %w", err)
		}
		categories = append(categories, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("billing.Repository.ListFeeCategories: rows: %w", err)
	}

	if categories == nil {
		categories = []FeeCategory{}
	}

	return categories, nil
}

// GetFeeCategoryByID retrieves a fee category by its ID, scoped to tenant and school.
func (r *PgRepository) GetFeeCategoryByID(ctx context.Context, id, tenantID, schoolID string) (*FeeCategory, error) {
	const query = `
		SELECT id, tenant_id, school_id, name, is_mandatory
		FROM fee_categories
		WHERE id = $1 AND tenant_id = $2 AND school_id = $3
	`
	var c FeeCategory
	err := r.pool.QueryRow(ctx, query, id, tenantID, schoolID).Scan(
		&c.ID, &c.TenantID, &c.SchoolID, &c.Name, &c.IsMandatory,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("billing.Repository.GetFeeCategoryByID: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("billing.Repository.GetFeeCategoryByID: %w", err)
	}
	return &c, nil
}

// UpdateFeeCategory modifies fee category fields. Only non-nil fields are applied.
func (r *PgRepository) UpdateFeeCategory(ctx context.Context, id, tenantID, schoolID string, name *string, isMandatory *bool) error {
	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	if name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *name)
		argIdx++
	}
	if isMandatory != nil {
		setClauses = append(setClauses, fmt.Sprintf("is_mandatory = $%d", argIdx))
		args = append(args, *isMandatory)
		argIdx++
	}

	if len(setClauses) == 0 {
		return fmt.Errorf("billing.Repository.UpdateFeeCategory: %w", ErrInvalidInput)
	}

	args = append(args, id, tenantID, schoolID)
	query := fmt.Sprintf(`
		UPDATE fee_categories
		SET %s
		WHERE id = $%d AND tenant_id = $%d AND school_id = $%d
	`, joinClauses(setClauses, ", "), argIdx, argIdx+1, argIdx+2)

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("billing.Repository.UpdateFeeCategory: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("billing.Repository.UpdateFeeCategory: %w", ErrNotFound)
	}

	return nil
}

// DeleteFeeCategory removes a fee category by ID, scoped to tenant and school.
func (r *PgRepository) DeleteFeeCategory(ctx context.Context, id, tenantID, schoolID string) error {
	const query = `DELETE FROM fee_categories WHERE id = $1 AND tenant_id = $2 AND school_id = $3`
	result, err := r.pool.Exec(ctx, query, id, tenantID, schoolID)
	if err != nil {
		return fmt.Errorf("billing.Repository.DeleteFeeCategory: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("billing.Repository.DeleteFeeCategory: %w", ErrNotFound)
	}
	return nil
}

// ─── Fee Templates ─────────────────────────────────────────────────────────

// CreateFeeTemplate inserts a new fee template and returns its ID.
func (r *PgRepository) CreateFeeTemplate(ctx context.Context, tenantID, schoolID, academicTermID, gradeLevel, feeCategoryID, amount string) (string, error) {
	const query = `
		INSERT INTO fee_templates (tenant_id, school_id, academic_term_id, grade_level, fee_category_id, amount)
		VALUES ($1, $2, $3, $4::cbc_grade_level, $5, $6::NUMERIC(12,2))
		RETURNING id
	`
	var id string
	err := r.pool.QueryRow(ctx, query, tenantID, schoolID, academicTermID, gradeLevel, feeCategoryID, amount).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("billing.Repository.CreateFeeTemplate: %w", err)
	}
	return id, nil
}

// ListFeeTemplates retrieves fee templates for a tenant and school,
// optionally filtered by academic_term_id and/or grade_level.
func (r *PgRepository) ListFeeTemplates(ctx context.Context, tenantID, schoolID string, academicTermID, gradeLevel *string) ([]FeeTemplate, error) {
	baseQuery := `
		SELECT id, tenant_id, school_id, academic_term_id, grade_level, fee_category_id, amount, created_at
		FROM fee_templates
		WHERE tenant_id = $1 AND school_id = $2
	`
	args := []interface{}{tenantID, schoolID}
	argIdx := 3

	if academicTermID != nil {
		baseQuery += fmt.Sprintf(" AND academic_term_id = $%d", argIdx)
		args = append(args, *academicTermID)
		argIdx++
	}
	if gradeLevel != nil {
		baseQuery += fmt.Sprintf(" AND grade_level = $%d::cbc_grade_level", argIdx)
		args = append(args, *gradeLevel)
		argIdx++
	}

	_ = argIdx

	baseQuery += " ORDER BY grade_level ASC, fee_category_id ASC"

	rows, err := r.pool.Query(ctx, baseQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("billing.Repository.ListFeeTemplates: %w", err)
	}
	defer rows.Close()

	var templates []FeeTemplate
	for rows.Next() {
		var t FeeTemplate
		if err := rows.Scan(
			&t.ID, &t.TenantID, &t.SchoolID, &t.AcademicTermID,
			&t.GradeLevel, &t.FeeCategoryID, &t.Amount, &t.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("billing.Repository.ListFeeTemplates: scan: %w", err)
		}
		templates = append(templates, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("billing.Repository.ListFeeTemplates: rows: %w", err)
	}

	if templates == nil {
		templates = []FeeTemplate{}
	}

	return templates, nil
}

// GetFeeTemplateByID retrieves a fee template by its ID, scoped to tenant and school.
func (r *PgRepository) GetFeeTemplateByID(ctx context.Context, id, tenantID, schoolID string) (*FeeTemplate, error) {
	const query = `
		SELECT id, tenant_id, school_id, academic_term_id, grade_level, fee_category_id, amount, created_at
		FROM fee_templates
		WHERE id = $1 AND tenant_id = $2 AND school_id = $3
	`
	var t FeeTemplate
	err := r.pool.QueryRow(ctx, query, id, tenantID, schoolID).Scan(
		&t.ID, &t.TenantID, &t.SchoolID, &t.AcademicTermID,
		&t.GradeLevel, &t.FeeCategoryID, &t.Amount, &t.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("billing.Repository.GetFeeTemplateByID: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("billing.Repository.GetFeeTemplateByID: %w", err)
	}
	return &t, nil
}

// UpdateFeeTemplate modifies fee template fields. Only non-nil fields are applied.
func (r *PgRepository) UpdateFeeTemplate(ctx context.Context, id, tenantID, schoolID string, amount *string) error {
	if amount == nil {
		return fmt.Errorf("billing.Repository.UpdateFeeTemplate: %w", ErrInvalidInput)
	}

	const query = `
		UPDATE fee_templates
		SET amount = $1::NUMERIC(12,2)
		WHERE id = $2 AND tenant_id = $3 AND school_id = $4
	`
	result, err := r.pool.Exec(ctx, query, *amount, id, tenantID, schoolID)
	if err != nil {
		return fmt.Errorf("billing.Repository.UpdateFeeTemplate: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("billing.Repository.UpdateFeeTemplate: %w", ErrNotFound)
	}

	return nil
}

// DeleteFeeTemplate removes a fee template by ID, scoped to tenant and school.
func (r *PgRepository) DeleteFeeTemplate(ctx context.Context, id, tenantID, schoolID string) error {
	const query = `DELETE FROM fee_templates WHERE id = $1 AND tenant_id = $2 AND school_id = $3`
	result, err := r.pool.Exec(ctx, query, id, tenantID, schoolID)
	if err != nil {
		return fmt.Errorf("billing.Repository.DeleteFeeTemplate: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("billing.Repository.DeleteFeeTemplate: %w", ErrNotFound)
	}
	return nil
}

// ─── Helpers ───────────────────────────────────────────────────────────────

// ─── Invoices ──────────────────────────────────────────────────────────────

// CreateInvoice inserts a new invoice and returns its ID.
func (r *PgRepository) CreateInvoice(ctx context.Context, tenantID, schoolID, studentID, academicTermID string, parentID *string, invoiceLabel *string, amountDue string) (string, error) {
	const query = `
		INSERT INTO invoices (tenant_id, school_id, student_id, academic_term_id, parent_id, invoice_label, amount_due)
		VALUES ($1, $2, $3, $4, $5, $6, $7::NUMERIC(12,2))
		RETURNING id
	`
	var id string
	err := r.pool.QueryRow(ctx, query, tenantID, schoolID, studentID, academicTermID, parentID, invoiceLabel, amountDue).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("billing.Repository.CreateInvoice: %w", err)
	}
	return id, nil
}

// CreateInvoiceItem inserts a single invoice item.
func (r *PgRepository) CreateInvoiceItem(ctx context.Context, tenantID, invoiceID, feeCategoryID, description, amount string) error {
	const query = `
		INSERT INTO invoice_items (tenant_id, invoice_id, fee_category_id, description, amount)
		VALUES ($1, $2, $3, $4, $5::NUMERIC(12,2))
	`
	_, err := r.pool.Exec(ctx, query, tenantID, invoiceID, feeCategoryID, description, amount)
	if err != nil {
		return fmt.Errorf("billing.Repository.CreateInvoiceItem: %w", err)
	}
	return nil
}

// GetInvoiceByID retrieves an invoice scoped to tenant and school.
func (r *PgRepository) GetInvoiceByID(ctx context.Context, id, tenantID, schoolID string) (*Invoice, error) {
	const query = `
		SELECT id, tenant_id, student_id, school_id, academic_term_id,
		       parent_id, invoice_label, payment_status,
		       amount_due::TEXT, amount_paid::TEXT, created_at
		FROM invoices
		WHERE id = $1 AND tenant_id = $2 AND school_id = $3
	`
	var inv Invoice
	err := r.pool.QueryRow(ctx, query, id, tenantID, schoolID).Scan(
		&inv.ID, &inv.TenantID, &inv.StudentID, &inv.SchoolID, &inv.AcademicTermID,
		&inv.ParentID, &inv.InvoiceLabel, &inv.PaymentStatus,
		&inv.AmountDue, &inv.AmountPaid, &inv.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("billing.Repository.GetInvoiceByID: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("billing.Repository.GetInvoiceByID: %w", err)
	}
	return &inv, nil
}

// ListInvoices retrieves invoices for a tenant and school, optionally filtered.
func (r *PgRepository) ListInvoices(ctx context.Context, tenantID, schoolID string, filter InvoiceFilter) ([]Invoice, int, error) {
	countQuery := `
		SELECT COUNT(*)
		FROM invoices
		WHERE tenant_id = $1 AND school_id = $2
	`
	dataQuery := `
		SELECT id, tenant_id, student_id, school_id, academic_term_id,
		       parent_id, invoice_label, payment_status,
		       amount_due::TEXT, amount_paid::TEXT, created_at
		FROM invoices
		WHERE tenant_id = $1 AND school_id = $2
	`
	args := []interface{}{tenantID, schoolID}
	argIdx := 3

	if filter.StudentID != nil {
		clause := fmt.Sprintf(" AND student_id = $%d", argIdx)
		countQuery += clause
		dataQuery += clause
		args = append(args, *filter.StudentID)
		argIdx++
	}
	if filter.AcademicTermID != nil {
		clause := fmt.Sprintf(" AND academic_term_id = $%d", argIdx)
		countQuery += clause
		dataQuery += clause
		args = append(args, *filter.AcademicTermID)
		argIdx++
	}
	if filter.PaymentStatus != nil {
		clause := fmt.Sprintf(" AND payment_status = $%d::invoice_payment_status", argIdx)
		countQuery += clause
		dataQuery += clause
		args = append(args, *filter.PaymentStatus)
		argIdx++
	}

	_ = argIdx

	dataQuery += " ORDER BY created_at DESC"

	var total int
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("billing.Repository.ListInvoices: count: %w", err)
	}
	if total == 0 {
		return []Invoice{}, 0, nil
	}

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("billing.Repository.ListInvoices: query: %w", err)
	}
	defer rows.Close()

	var invoices []Invoice
	for rows.Next() {
		var inv Invoice
		if err := rows.Scan(
			&inv.ID, &inv.TenantID, &inv.StudentID, &inv.SchoolID, &inv.AcademicTermID,
			&inv.ParentID, &inv.InvoiceLabel, &inv.PaymentStatus,
			&inv.AmountDue, &inv.AmountPaid, &inv.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("billing.Repository.ListInvoices: scan: %w", err)
		}
		invoices = append(invoices, inv)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("billing.Repository.ListInvoices: rows: %w", err)
	}

	if invoices == nil {
		invoices = []Invoice{}
	}

	return invoices, total, nil
}

// GetInvoiceItems retrieves all items for a given invoice.
func (r *PgRepository) GetInvoiceItems(ctx context.Context, invoiceID, tenantID string) ([]InvoiceItem, error) {
	const query = `
		SELECT id, invoice_id, fee_category_id, COALESCE(description, ''), amount::TEXT
		FROM invoice_items
		WHERE invoice_id = $1 AND tenant_id = $2
		ORDER BY fee_category_id ASC
	`
	rows, err := r.pool.Query(ctx, query, invoiceID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("billing.Repository.GetInvoiceItems: %w", err)
	}
	defer rows.Close()

	var items []InvoiceItem
	for rows.Next() {
		var it InvoiceItem
		if err := rows.Scan(&it.ID, &it.InvoiceID, &it.FeeCategoryID, &it.Description, &it.Amount); err != nil {
			return nil, fmt.Errorf("billing.Repository.GetInvoiceItems: scan: %w", err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("billing.Repository.GetInvoiceItems: rows: %w", err)
	}

	if items == nil {
		items = []InvoiceItem{}
	}
	return items, nil
}

// GetInvoiceDetail returns an invoice with its items and payments.
func (r *PgRepository) GetInvoiceDetail(ctx context.Context, id, tenantID, schoolID string) (*InvoiceDetailResponse, error) {
	inv, err := r.GetInvoiceByID(ctx, id, tenantID, schoolID)
	if err != nil {
		return nil, err
	}

	items, err := r.GetInvoiceItems(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	payments, err := r.ListPayments(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	return &InvoiceDetailResponse{
		Invoice:  *inv,
		Items:    items,
		Payments: payments,
	}, nil
}

// WaiveInvoice sets payment_status to WAIVED for an invoice.
func (r *PgRepository) WaiveInvoice(ctx context.Context, id, tenantID, schoolID string) error {
	const query = `
		UPDATE invoices
		SET payment_status = 'WAIVED'
		WHERE id = $1 AND tenant_id = $2 AND school_id = $3
	`
	result, err := r.pool.Exec(ctx, query, id, tenantID, schoolID)
	if err != nil {
		return fmt.Errorf("billing.Repository.WaiveInvoice: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("billing.Repository.WaiveInvoice: %w", ErrNotFound)
	}
	return nil
}

// ResolveGradeLevel returns the student's grade_level for a given term by
// joining through student_enrollments → cbc_classes.
func (r *PgRepository) ResolveGradeLevel(ctx context.Context, tenantID, studentID, academicTermID string) (string, error) {
	const query = `
		SELECT c.grade_level::TEXT
		FROM cbc_student_enrollments e
		JOIN cbc_classes c ON c.id = e.class_id AND c.tenant_id = e.tenant_id
		WHERE e.student_id = $1
		  AND e.academic_term_id = $2
		  AND e.tenant_id = $3
		  AND e.status = 'ACTIVE'
	`
	var gradeLevel string
	err := r.pool.QueryRow(ctx, query, studentID, academicTermID, tenantID).Scan(&gradeLevel)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", fmt.Errorf("billing.Repository.ResolveGradeLevel: %w", ErrNotFound)
		}
		return "", fmt.Errorf("billing.Repository.ResolveGradeLevel: %w", err)
	}
	return gradeLevel, nil
}

// ListFeeTemplatesByTermAndGrade fetches fee templates matching a term and grade level.
func (r *PgRepository) ListFeeTemplatesByTermAndGrade(ctx context.Context, tenantID, schoolID, academicTermID, gradeLevel string) ([]FeeTemplate, error) {
	const query = `
		SELECT id, tenant_id, school_id, academic_term_id, grade_level::TEXT,
		       fee_category_id, amount::TEXT, created_at
		FROM fee_templates
		WHERE tenant_id = $1
		  AND school_id = $2
		  AND academic_term_id = $3
		  AND grade_level = $4::cbc_grade_level
		ORDER BY fee_category_id ASC
	`
	rows, err := r.pool.Query(ctx, query, tenantID, schoolID, academicTermID, gradeLevel)
	if err != nil {
		return nil, fmt.Errorf("billing.Repository.ListFeeTemplatesByTermAndGrade: %w", err)
	}
	defer rows.Close()

	var templates []FeeTemplate
	for rows.Next() {
		var t FeeTemplate
		if err := rows.Scan(
			&t.ID, &t.TenantID, &t.SchoolID, &t.AcademicTermID,
			&t.GradeLevel, &t.FeeCategoryID, &t.Amount, &t.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("billing.Repository.ListFeeTemplatesByTermAndGrade: scan: %w", err)
		}
		templates = append(templates, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("billing.Repository.ListFeeTemplatesByTermAndGrade: rows: %w", err)
	}

	if templates == nil {
		templates = []FeeTemplate{}
	}
	return templates, nil
}

// ─── Payments ───────────────────────────────────────────────────────────────

// RecordPayment inserts a payment record. The DB trigger trg_sync_invoice_payment_status
// automatically updates the invoice's amount_paid and payment_status.
func (r *PgRepository) RecordPayment(ctx context.Context, tenantID, invoiceID, amount, recordedBy string, parentID, paymentMethod, referenceCode *string) (string, error) {
	const query = `
		INSERT INTO payments (tenant_id, invoice_id, amount, recorded_by, parent_id, payment_method, reference_code)
		VALUES ($1, $2, $3::NUMERIC(12,2), $4, $5, $6, $7)
		RETURNING id
	`
	var id string
	err := r.pool.QueryRow(ctx, query, tenantID, invoiceID, amount, recordedBy, parentID, paymentMethod, referenceCode).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("billing.Repository.RecordPayment: %w", err)
	}
	return id, nil
}

// ListPayments retrieves payments for a given invoice.
func (r *PgRepository) ListPayments(ctx context.Context, tenantID, invoiceID string) ([]Payment, error) {
	const query = `
		SELECT id, invoice_id, amount::TEXT, parent_id, payment_method,
		       reference_code, recorded_by, created_at::TEXT
		FROM payments
		WHERE tenant_id = $1 AND invoice_id = $2
		ORDER BY created_at ASC
	`
	rows, err := r.pool.Query(ctx, query, tenantID, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("billing.Repository.ListPayments: %w", err)
	}
	defer rows.Close()

	var payments []Payment
	for rows.Next() {
		var p Payment
		if err := rows.Scan(
			&p.ID, &p.InvoiceID, &p.Amount, &p.ParentID,
			&p.PaymentMethod, &p.ReferenceCode, &p.RecordedBy, &p.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("billing.Repository.ListPayments: scan: %w", err)
		}
		payments = append(payments, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("billing.Repository.ListPayments: rows: %w", err)
	}

	if payments == nil {
		payments = []Payment{}
	}
	return payments, nil
}

// GetPaymentByID retrieves a single payment by ID.
func (r *PgRepository) GetPaymentByID(ctx context.Context, id, tenantID string) (*Payment, error) {
	const query = `
		SELECT id, invoice_id, amount::TEXT, parent_id, payment_method,
		       reference_code, recorded_by, created_at::TEXT
		FROM payments
		WHERE id = $1 AND tenant_id = $2
	`
	var p Payment
	err := r.pool.QueryRow(ctx, query, id, tenantID).Scan(
		&p.ID, &p.InvoiceID, &p.Amount, &p.ParentID,
		&p.PaymentMethod, &p.ReferenceCode, &p.RecordedBy, &p.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("billing.Repository.GetPaymentByID: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("billing.Repository.GetPaymentByID: %w", err)
	}
	return &p, nil
}

// ─── Helpers ───────────────────────────────────────────────────────────────

// joinClauses joins strings with a separator. Helper for dynamic SET clauses.
func joinClauses(clauses []string, sep string) string {
	if len(clauses) == 0 {
		return ""
	}
	result := clauses[0]
	for _, c := range clauses[1:] {
		result += sep + c
	}
	return result
}
