package imports

import (
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
