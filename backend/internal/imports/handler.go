package imports

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"somotracker/backend/internal/middleware"
)

// ============================================================================
// Handler — Import Job Status + Failures
// ============================================================================

type Handler struct {
	svc *Service
}

// NewHandler creates a new Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts all import endpoints.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	router.Get("/api/v1/imports/:job_id", middleware.RequireAuth, h.GetJobAPI)
	router.Get("/api/v1/imports/:job_id/failures", middleware.RequireAuth, h.GetFailures)
	router.Post("/api/v1/imports/:job_id/cancel", middleware.RequireAuth, h.CancelJobAPI)
	// Resolves active school from the authenticated session — no school_id in URL needed
	router.Get("/api/v1/imports/active", middleware.RequireAuth, h.GetSessionActiveJobAPI)
	// Legacy: explicit school_id param (for cross-school access or future multi-school users)
	router.Get("/api/v1/schools/:school_id/imports/active", middleware.RequireAuth, h.GetActiveJobAPI)
}

// ============================================================================
// Job status (for polling fallback)
// ============================================================================

func (h *Handler) GetJobAPI(c *fiber.Ctx) error {
	jobID, err := uuid.Parse(c.Params("job_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": "invalid_input", "message": "bad job_id"})
	}

	tenantID, _ := uuid.Parse(c.Locals("tenant_id").(string))

	job, err := h.svc.GetJob(c.Context(), jobID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}
	if job.TenantID != tenantID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"code": "forbidden", "message": "access denied"})
	}
	return c.JSON(job)
}

// ============================================================================
// Cancel a job
// ============================================================================

func (h *Handler) CancelJobAPI(c *fiber.Ctx) error {
	jobID, err := uuid.Parse(c.Params("job_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": "invalid_input", "message": "bad job_id"})
	}

	tenantID, _ := uuid.Parse(c.Locals("tenant_id").(string))
	schoolID, _ := uuid.Parse(c.Locals("school_id").(string))

	// Verify the job exists and belongs to the caller's tenant/school
	job, err := h.svc.GetJob(c.Context(), jobID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}
	if job.TenantID != tenantID || job.SchoolID != schoolID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"code": "forbidden", "message": "access denied"})
	}

	// Attempt cancellation
	updated, err := h.svc.CancelJob(c.Context(), jobID)
	if err != nil {
		if errors.Is(err, ErrNotCancellable) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"code":    "job_not_cancellable",
				"message": "the job is not in a cancellable state (it may already be completed, failed, or cancelled)",
			})
		}
		return middleware.HTTPError(c, err)
	}

	return c.JSON(updated)
}

// ============================================================================
// Get active job from session (proactive check — no school_id in URL)
// Resolves the active school from the authenticated session context.
// ============================================================================

func (h *Handler) GetSessionActiveJobAPI(c *fiber.Ctx) error {
	schoolIDStr := c.Locals("active_school_id").(string)
	if schoolIDStr == "" {
		schoolIDStr = c.Locals("school_id").(string)
	}
	if schoolIDStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": "invalid_input", "message": "active school not set"})
	}

	schoolID, err := uuid.Parse(schoolIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": "invalid_input", "message": "invalid school id"})
	}

	tenantID, _ := uuid.Parse(c.Locals("tenant_id").(string))

	job, err := h.svc.GetActiveJobBySchool(c.Context(), schoolID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return c.JSON(fiber.Map{"active": false, "job": nil})
		}
		return middleware.HTTPError(c, err)
	}

	// Verify tenant ownership
	if job.TenantID != tenantID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"code": "forbidden", "message": "access denied"})
	}

	return c.JSON(fiber.Map{"active": true, "job": job})
}

// ============================================================================
// Get active job for school (proactive check before showing import form)
// ============================================================================

func (h *Handler) GetActiveJobAPI(c *fiber.Ctx) error {
	schoolID, err := uuid.Parse(c.Params("school_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": "invalid_input", "message": "bad school_id"})
	}

	tenantID, _ := uuid.Parse(c.Locals("tenant_id").(string))

	// Verify the school belongs to the caller's tenant
	schoolTenantIDStr := c.Locals("school_tenant_id").(string)
	if schoolTenantIDStr != "" {
		schoolTenantID, _ := uuid.Parse(schoolTenantIDStr)
		if schoolTenantID != uuid.Nil && schoolTenantID != tenantID {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"code": "forbidden", "message": "access denied"})
		}
	}

	job, err := h.svc.GetActiveJobBySchool(c.Context(), schoolID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return c.JSON(fiber.Map{"active": false, "job": nil})
		}
		return middleware.HTTPError(c, err)
	}

	// Verify tenant ownership
	if job.TenantID != tenantID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"code": "forbidden", "message": "access denied"})
	}

	return c.JSON(fiber.Map{"active": true, "job": job})
}

// ============================================================================
// Failures
// ============================================================================

func (h *Handler) GetFailures(c *fiber.Ctx) error {
	jobID, err := uuid.Parse(c.Params("job_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": "invalid_input", "message": "bad job_id"})
	}

	tenantUUID, _ := uuid.Parse(c.Locals("tenant_id").(string))

	job, err := h.svc.GetJob(c.Context(), jobID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}
	if job.TenantID != tenantUUID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"code": "forbidden", "message": "access denied"})
	}

	limit := 50
	if l := c.Query("limit"); l != "" {
		if v, e := strconv.Atoi(l); e == nil && v > 0 && v <= 5000 {
			limit = v
		}
	}
	page := 1
	if p := c.Query("page"); p != "" {
		if v, e := strconv.Atoi(p); e == nil && v > 0 {
			page = v
		}
	}

	failures, total, err := h.svc.GetFailures(c.Context(), jobID, limit, (page-1)*limit)
	if err != nil {
		return middleware.HTTPError(c, err)
	}
	return c.JSON(fiber.Map{"failures": failures, "total": total})
}
