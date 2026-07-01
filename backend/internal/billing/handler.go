package billing

import (
	"github.com/gofiber/fiber/v2"

	"somotracker/backend/internal/middleware"
)

// ─── Handler ───────────────────────────────────────────────────────────────

// Handler exposes billing HTTP endpoints.
type Handler struct {
	svc *Service
}

// NewHandler creates a new Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts billing routes on the given router.
// Only SCHOOL_ADMIN users can mutate (POST/PUT/DELETE);
// authenticated users can read (GET).
func (h *Handler) RegisterRoutes(router fiber.Router) {
	billing := router.Group("/api/v1/billing")

	// Fee categories
	billing.Post("/fee-categories", middleware.RequireRole("SCHOOL_ADMIN"), h.CreateFeeCategory)
	billing.Get("/fee-categories", middleware.RequireAuth, h.ListFeeCategories)
	billing.Put("/fee-categories/:id", middleware.RequireRole("SCHOOL_ADMIN"), h.UpdateFeeCategory)
	billing.Delete("/fee-categories/:id", middleware.RequireRole("SCHOOL_ADMIN"), h.DeleteFeeCategory)

	// Fee templates
	billing.Post("/fee-templates", middleware.RequireRole("SCHOOL_ADMIN"), h.CreateFeeTemplate)
	billing.Get("/fee-templates", middleware.RequireAuth, h.ListFeeTemplates)
	billing.Put("/fee-templates/:id", middleware.RequireRole("SCHOOL_ADMIN"), h.UpdateFeeTemplate)
	billing.Delete("/fee-templates/:id", middleware.RequireRole("SCHOOL_ADMIN"), h.DeleteFeeTemplate)

	// Invoices
	billing.Post("/invoices/generate", middleware.RequireRole("SCHOOL_ADMIN"), h.GenerateInvoice)
	billing.Get("/invoices", middleware.RequireAuth, h.ListInvoices)
	billing.Get("/invoices/:id", middleware.RequireAuth, h.GetInvoiceDetail)
	billing.Post("/invoices/:id/waive", middleware.RequireRole("SCHOOL_ADMIN"), h.WaiveInvoice)

	// Payments
	billing.Post("/payments", middleware.RequireRole("SCHOOL_ADMIN"), h.RecordPayment)
	billing.Get("/payments", middleware.RequireAuth, h.ListPayments)
}

// ─── Fee Category Handlers ─────────────────────────────────────────────────

// CreateFeeCategory handles POST /api/v1/billing/fee-categories.
func (h *Handler) CreateFeeCategory(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID, _ := c.Locals("active_school_id").(string)
	if schoolID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "active school not set",
		})
	}

	var payload CreateFeeCategoryPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "invalid request body",
		})
	}

	id, err := h.svc.CreateFeeCategory(c.Context(), tenantID, schoolID, payload.Name, payload.IsMandatory)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id": id,
	})
}

// ListFeeCategories handles GET /api/v1/billing/fee-categories.
func (h *Handler) ListFeeCategories(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID, _ := c.Locals("active_school_id").(string)
	if schoolID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "active school not set",
		})
	}

	categories, err := h.svc.ListFeeCategories(c.Context(), tenantID, schoolID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(ListFeeCategoriesResponse{
		FeeCategories: categories,
		Total:         len(categories),
	})
}

// UpdateFeeCategory handles PUT /api/v1/billing/fee-categories/:id.
func (h *Handler) UpdateFeeCategory(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID, _ := c.Locals("active_school_id").(string)
	if schoolID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "active school not set",
		})
	}

	categoryID := c.Params("id")
	if categoryID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "fee category id is required",
		})
	}

	var payload UpdateFeeCategoryPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "invalid request body",
		})
	}

	if err := h.svc.UpdateFeeCategory(c.Context(), categoryID, tenantID, schoolID, payload.Name, payload.IsMandatory); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.SendStatus(fiber.StatusOK)
}

// DeleteFeeCategory handles DELETE /api/v1/billing/fee-categories/:id.
func (h *Handler) DeleteFeeCategory(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID, _ := c.Locals("active_school_id").(string)
	if schoolID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "active school not set",
		})
	}

	categoryID := c.Params("id")
	if categoryID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "fee category id is required",
		})
	}

	if err := h.svc.DeleteFeeCategory(c.Context(), categoryID, tenantID, schoolID); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// ─── Fee Template Handlers ─────────────────────────────────────────────────

// CreateFeeTemplate handles POST /api/v1/billing/fee-templates.
func (h *Handler) CreateFeeTemplate(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID, _ := c.Locals("active_school_id").(string)
	if schoolID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "active school not set",
		})
	}

	var payload CreateFeeTemplatePayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "invalid request body",
		})
	}

	id, err := h.svc.CreateFeeTemplate(c.Context(), tenantID, schoolID, payload.AcademicTermID, payload.GradeLevel, payload.FeeCategoryID, payload.Amount)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id": id,
	})
}

// ListFeeTemplates handles GET /api/v1/billing/fee-templates.
func (h *Handler) ListFeeTemplates(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID, _ := c.Locals("active_school_id").(string)
	if schoolID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "active school not set",
		})
	}

	var academicTermID, gradeLevel *string
	if q := c.Query("academic_term_id"); q != "" {
		academicTermID = &q
	}
	if q := c.Query("grade_level"); q != "" {
		gradeLevel = &q
	}

	templates, err := h.svc.ListFeeTemplates(c.Context(), tenantID, schoolID, academicTermID, gradeLevel)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(ListFeeTemplatesResponse{
		FeeTemplates: templates,
		Total:        len(templates),
	})
}

// UpdateFeeTemplate handles PUT /api/v1/billing/fee-templates/:id.
func (h *Handler) UpdateFeeTemplate(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID, _ := c.Locals("active_school_id").(string)
	if schoolID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "active school not set",
		})
	}

	templateID := c.Params("id")
	if templateID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "fee template id is required",
		})
	}

	var payload UpdateFeeTemplatePayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "invalid request body",
		})
	}

	if err := h.svc.UpdateFeeTemplate(c.Context(), templateID, tenantID, schoolID, payload.Amount); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.SendStatus(fiber.StatusOK)
}

// DeleteFeeTemplate handles DELETE /api/v1/billing/fee-templates/:id.
func (h *Handler) DeleteFeeTemplate(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID, _ := c.Locals("active_school_id").(string)
	if schoolID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "active school not set",
		})
	}

	templateID := c.Params("id")
	if templateID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "fee template id is required",
		})
	}

	if err := h.svc.DeleteFeeTemplate(c.Context(), templateID, tenantID, schoolID); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// ─── Invoice Handlers ───────────────────────────────────────────────────────

// GenerateInvoice handles POST /api/v1/billing/invoices/generate.
func (h *Handler) GenerateInvoice(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID, _ := c.Locals("active_school_id").(string)
	if schoolID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "active school not set",
		})
	}

	var payload GenerateInvoicePayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "invalid request body",
		})
	}

	result, err := h.svc.GenerateInvoice(c.Context(), tenantID, schoolID, payload)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(result)
}

// GetInvoiceDetail handles GET /api/v1/billing/invoices/:id.
func (h *Handler) GetInvoiceDetail(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID, _ := c.Locals("active_school_id").(string)
	if schoolID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "active school not set",
		})
	}

	invoiceID := c.Params("id")
	if invoiceID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "invoice id is required",
		})
	}

	result, err := h.svc.GetInvoiceDetail(c.Context(), invoiceID, tenantID, schoolID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// ListInvoices handles GET /api/v1/billing/invoices.
func (h *Handler) ListInvoices(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID, _ := c.Locals("active_school_id").(string)
	if schoolID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "active school not set",
		})
	}

	filter := InvoiceFilter{}
	if q := c.Query("student_id"); q != "" {
		filter.StudentID = &q
	}
	if q := c.Query("academic_term_id"); q != "" {
		filter.AcademicTermID = &q
	}
	if q := c.Query("payment_status"); q != "" {
		filter.PaymentStatus = &q
	}

	result, err := h.svc.ListInvoices(c.Context(), tenantID, schoolID, filter)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// WaiveInvoice handles POST /api/v1/billing/invoices/:id/waive.
func (h *Handler) WaiveInvoice(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID, _ := c.Locals("active_school_id").(string)
	if schoolID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "active school not set",
		})
	}

	invoiceID := c.Params("id")
	if invoiceID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "invoice id is required",
		})
	}

	if err := h.svc.WaiveInvoice(c.Context(), invoiceID, tenantID, schoolID); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(fiber.Map{
		"payment_status": "WAIVED",
	})
}

// ─── Payment Handlers ───────────────────────────────────────────────────────

// RecordPayment handles POST /api/v1/billing/payments.
func (h *Handler) RecordPayment(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	userID := c.Locals("user_id").(string)
	schoolID, _ := c.Locals("active_school_id").(string)

	var payload RecordPaymentPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "invalid request body",
		})
	}

	payment, err := h.svc.RecordPayment(c.Context(), tenantID, schoolID, userID, payload)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id": payment.ID,
	})
}

// ListPayments handles GET /api/v1/billing/payments?invoice_id=X.
func (h *Handler) ListPayments(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)

	invoiceID := c.Query("invoice_id")
	if invoiceID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "invoice_id query parameter is required",
		})
	}

	result, err := h.svc.ListPayments(c.Context(), tenantID, invoiceID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}
