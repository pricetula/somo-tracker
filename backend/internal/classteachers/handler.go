package classteachers

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"somotracker/backend/internal/middleware"
)

// Handler exposes class teacher HTTP endpoints.
type Handler struct {
	svc *Service
}

// NewHandler creates a new Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts class teacher routes on the given router.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	ct := router.Group("/api/v1/class-teachers")
	ct.Post("/", middleware.RequireAuth, middleware.RequireRole("SCHOOL_ADMIN"), h.Create)
	ct.Get("/by-class/:classId", middleware.RequireAuth, h.ListByClass)
	ct.Get("/by-teacher/:userId", middleware.RequireAuth, h.ListByTeacher)
	ct.Get("/:id", middleware.RequireAuth, h.GetByID)
	ct.Delete("/", middleware.RequireAuth, middleware.RequireRole("SCHOOL_ADMIN"), h.Delete)
}

// errorResponse is the canonical error JSON body.
type errorResponse struct {
	Code    string              `json:"code"`
	Message string              `json:"message"`
	Errors  map[string][]string `json:"errors,omitempty"`
}

func writeError(c *fiber.Ctx, status int, code, message string) error {
	return c.Status(status).JSON(errorResponse{
		Code:    code,
		Message: message,
	})
}

// ── CREATE ──────────────────────────────────────────────────────────────

// Create handles POST /api/v1/class-teachers.
func (h *Handler) Create(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)

	var body CreateClassTeacherPayload
	if err := c.BodyParser(&body); err != nil {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "malformed request body")
	}

	result, err := h.svc.Create(c.Context(), tenantID, body)
	if err != nil {
		if errors.Is(err, ErrInvalidInput) {
			return writeError(c, fiber.StatusBadRequest, "invalid_input", err.Error())
		}
		if errors.Is(err, ErrPrimaryAlreadyAssigned) {
			return writeError(c, fiber.StatusConflict, "primary_already_assigned",
				"This class already has a primary teacher. Remove the existing assignment first.")
		}
		if errors.Is(err, ErrAlreadyExists) {
			return writeError(c, fiber.StatusConflict, "already_exists",
				"This teacher is already assigned to this class/subject.")
		}
		return middleware.HTTPError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(result)
}

// ── GET BY ID ───────────────────────────────────────────────────────────

// GetByID handles GET /api/v1/class-teachers/:id.
func (h *Handler) GetByID(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	id := c.Params("id")
	if id == "" {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "id is required")
	}

	result, err := h.svc.GetByID(c.Context(), id, tenantID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}
	return c.JSON(result)
}

// ── LIST BY CLASS ───────────────────────────────────────────────────────

// ListByClass handles GET /api/v1/class-teachers/by-class/:classId.
func (h *Handler) ListByClass(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	classID := c.Params("classId")
	if classID == "" {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "classId is required")
	}

	result, err := h.svc.ListByClass(c.Context(), classID, tenantID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}
	return c.JSON(result)
}

// ── LIST BY TEACHER ─────────────────────────────────────────────────────

// ListByTeacher handles GET /api/v1/class-teachers/by-teacher/:userId.
func (h *Handler) ListByTeacher(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	userID := c.Params("userId")
	if userID == "" {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "userId is required")
	}

	result, err := h.svc.ListByTeacher(c.Context(), userID, tenantID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}
	return c.JSON(result)
}

// ── DELETE ──────────────────────────────────────────────────────────────

// Delete handles DELETE /api/v1/class-teachers/:id.
func (h *Handler) Delete(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)

	var payload struct {
		ID string `json:"id"`
	}
	if err := c.BodyParser(&payload); err != nil {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "malformed request body")
	}
	if payload.ID == "" {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "id is required")
	}

	if err := h.svc.Delete(c.Context(), payload.ID, tenantID); err != nil {
		return middleware.HTTPError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}
