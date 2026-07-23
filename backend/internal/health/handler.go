package health

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"somotracker/backend/internal/middleware"
)

// Handler exposes health HTTP endpoints.
type Handler struct {
	svc *Service
}

// NewHandler creates a new Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts health routes on the given router.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	health := router.Group("/api/v1/health")

	// Medical Incidents
	health.Post("/incidents", middleware.RequireAuth, middleware.RequireRole("NURSE", "SCHOOL_ADMIN"), h.CreateIncident)
	health.Get("/incidents/:id", middleware.RequireAuth, h.GetIncident)
	health.Get("/incidents", middleware.RequireAuth, h.ListIncidents)
	health.Put("/incidents/:id", middleware.RequireAuth, middleware.RequireRole("NURSE", "SCHOOL_ADMIN"), h.UpdateIncident)
	health.Delete("/incidents/:id", middleware.RequireAuth, middleware.RequireRole("NURSE", "SCHOOL_ADMIN"), h.DeleteIncident)

	// Student Health Profiles
	health.Put("/profiles/:studentId", middleware.RequireAuth, middleware.RequireRole("NURSE", "SCHOOL_ADMIN"), h.UpsertProfile)
	health.Get("/profiles/:studentId", middleware.RequireAuth, h.GetProfile)

	// Composite: health profile + recent incidents for a student
	health.Get("/students/:studentId", middleware.RequireAuth, h.GetStudentHealth)
}

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

// ═══════════════════════════════════════════════════════════════════════════
// MEDICAL INCIDENTS
// ═══════════════════════════════════════════════════════════════════════════

// CreateIncident handles POST /api/v1/health/incidents.
func (h *Handler) CreateIncident(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	loggedBy := c.Locals("user_id").(string)

	var body CreateMedicalIncidentPayload
	if err := c.BodyParser(&body); err != nil {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "malformed request body")
	}

	result, err := h.svc.CreateIncident(c.Context(), tenantID, loggedBy, body)
	if err != nil {
		if errors.Is(err, ErrInvalidInput) {
			return writeError(c, fiber.StatusBadRequest, "invalid_input", err.Error())
		}
		return middleware.HTTPError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(result)
}

// GetIncident handles GET /api/v1/health/incidents/:id.
func (h *Handler) GetIncident(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	id := c.Params("id")
	if id == "" {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "id is required")
	}

	result, err := h.svc.GetIncidentByID(c.Context(), id, tenantID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}
	return c.JSON(result)
}

// ListIncidents handles GET /api/v1/health/incidents.
// Query params: ?student_id=xxx (optional), ?school_id=xxx (optional), ?page=1&limit=50
func (h *Handler) ListIncidents(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	studentID := strings.TrimSpace(c.Query("student_id"))
	schoolID := strings.TrimSpace(c.Query("school_id"))
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "50"))

	if studentID != "" {
		// List by student
		result, err := h.svc.ListIncidentsByStudent(c.Context(), studentID, tenantID)
		if err != nil {
			return middleware.HTTPError(c, err)
		}
		return c.JSON(result)
	}

	// Fall back to school scope
	if schoolID == "" {
		schoolID, _ = c.Locals("active_school_id").(string)
	}

	result, err := h.svc.ListIncidentsBySchool(c.Context(), tenantID, schoolID, page, limit)
	if err != nil {
		return middleware.HTTPError(c, err)
	}
	return c.JSON(result)
}

// UpdateIncident handles PUT /api/v1/health/incidents/:id.
func (h *Handler) UpdateIncident(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	id := c.Params("id")
	if id == "" {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "id is required")
	}

	var body UpdateMedicalIncidentPayload
	if err := c.BodyParser(&body); err != nil {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "malformed request body")
	}

	if err := h.svc.UpdateIncident(c.Context(), id, tenantID, body); err != nil {
		if errors.Is(err, ErrInvalidInput) {
			return writeError(c, fiber.StatusBadRequest, "invalid_input", err.Error())
		}
		return middleware.HTTPError(c, err)
	}
	return c.SendStatus(fiber.StatusOK)
}

// DeleteIncident handles DELETE /api/v1/health/incidents/:id.
func (h *Handler) DeleteIncident(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	id := c.Params("id")
	if id == "" {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "id is required")
	}

	if err := h.svc.DeleteIncident(c.Context(), id, tenantID); err != nil {
		return middleware.HTTPError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// ═══════════════════════════════════════════════════════════════════════════
// STUDENT HEALTH PROFILES
// ═══════════════════════════════════════════════════════════════════════════

// UpsertProfile handles PUT /api/v1/health/profiles/:studentId.
func (h *Handler) UpsertProfile(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	studentID := c.Params("studentId")
	if studentID == "" {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "studentId is required")
	}

	var body UpsertHealthProfilePayload
	if err := c.BodyParser(&body); err != nil {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "malformed request body")
	}

	result, err := h.svc.UpsertProfile(c.Context(), tenantID, studentID, body)
	if err != nil {
		return middleware.HTTPError(c, err)
	}
	return c.JSON(result)
}

// GetProfile handles GET /api/v1/health/profiles/:studentId.
func (h *Handler) GetProfile(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	studentID := c.Params("studentId")
	if studentID == "" {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "studentId is required")
	}

	result, err := h.svc.GetProfileByStudent(c.Context(), studentID, tenantID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}
	return c.JSON(result)
}

// GetStudentHealth handles GET /api/v1/health/students/:studentId.
// Returns the health profile and recent incidents for a student.
func (h *Handler) GetStudentHealth(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	studentID := c.Params("studentId")
	if studentID == "" {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "studentId is required")
	}

	result, err := h.svc.GetStudentHealth(c.Context(), studentID, tenantID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}
	return c.JSON(result)
}
