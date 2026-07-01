package parents

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"

	"somotracker/backend/internal/middleware"
)

// ============================================================================
// Handler
// ============================================================================

// Handler exposes parent HTTP endpoints.
type Handler struct {
	svc *Service
}

// NewHandler creates a new Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts parent routes on the given router.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	parents := router.Group("/api/v1/parents")
	parents.Post("/", middleware.RequireAuth, h.Create)
	parents.Get("/", middleware.RequireAuth, h.List)
	parents.Get("/:id", middleware.RequireAuth, h.GetDetail)
	parents.Put("/:id", middleware.RequireAuth, h.Update)
	parents.Delete("/:id", middleware.RequireAuth, h.Delete)
	parents.Post("/:parent_id/students", middleware.RequireAuth, h.LinkStudent)
	parents.Delete("/:parent_id/students/:student_id", middleware.RequireAuth, h.UnlinkStudent)
}

// ============================================================================
// Error response helper
// ============================================================================

type errorResponse struct {
	Code    string              `json:"code"`
	Message string              `json:"message"`
	Errors  map[string][]string `json:"errors,omitempty"`
}

func writeError(c *fiber.Ctx, status int, code, message string, fieldErrors map[string][]string) error {
	return c.Status(status).JSON(errorResponse{
		Code:    code,
		Message: message,
		Errors:  fieldErrors,
	})
}

// ============================================================================
// CREATE
// ============================================================================

// Create handles POST /api/v1/parents.
func (h *Handler) Create(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)

	var body CreateParentPayload
	if err := c.BodyParser(&body); err != nil {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "malformed request body", nil)
	}

	if body.Email == "" {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "email is required", nil)
	}
	if body.FullName == "" {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "full_name is required", nil)
	}
	if body.PhoneNumber == "" {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "phone_number is required", nil)
	}

	parent, err := h.svc.Create(c.Context(), tenantID, body)
	if err != nil {
		if errors.Is(err, ErrAlreadyExists) {
			return writeError(c, fiber.StatusConflict, "already_exists",
				"A parent profile for this email already exists in this school.", nil)
		}
		return middleware.HTTPError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(CreateParentResponse{ID: parent.ID})
}

// ============================================================================
// LIST
// ============================================================================

// List handles GET /api/v1/parents.
func (h *Handler) List(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)

	search := c.Query("search")
	studentID := c.Query("student_id")

	parents, err := h.svc.List(c.Context(), tenantID, search, studentID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(ListParentsResponse{Data: parents})
}

// ============================================================================
// GET DETAIL
// ============================================================================

// GetDetail handles GET /api/v1/parents/:id.
func (h *Handler) GetDetail(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	id := c.Params("id")

	if id == "" {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "parent id is required", nil)
	}

	detail, err := h.svc.GetDetail(c.Context(), id, tenantID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(ParentDetailResponse{Data: *detail})
}

// ============================================================================
// UPDATE
// ============================================================================

// Update handles PUT /api/v1/parents/:id.
func (h *Handler) Update(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	id := c.Params("id")

	if id == "" {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "parent id is required", nil)
	}

	var body UpdateParentPayload
	if err := c.BodyParser(&body); err != nil {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "malformed request body", nil)
	}

	if body.PhoneNumber == nil && body.IsActive == nil {
		return writeError(c, fiber.StatusBadRequest, "invalid_input",
			"at least one of phone_number or is_active must be provided", nil)
	}

	if err := h.svc.Update(c.Context(), id, tenantID, body); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.SendStatus(fiber.StatusOK)
}

// ============================================================================
// DELETE
// ============================================================================

// Delete handles DELETE /api/v1/parents/:id.
func (h *Handler) Delete(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	id := c.Params("id")

	if id == "" {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "parent id is required", nil)
	}

	if err := h.svc.Delete(c.Context(), id, tenantID); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// ============================================================================
// LINK STUDENT
// ============================================================================

// LinkStudent handles POST /api/v1/parents/:parent_id/students.
func (h *Handler) LinkStudent(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	parentID := c.Params("parent_id")

	if parentID == "" {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "parent_id is required", nil)
	}

	var body LinkStudentPayload
	if err := c.BodyParser(&body); err != nil {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "malformed request body", nil)
	}

	if body.StudentID == "" {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "student_id is required", nil)
	}

	if err := h.svc.LinkStudent(c.Context(), parentID, tenantID, body); err != nil {
		if errors.Is(err, ErrStudentNotFound) {
			return writeError(c, fiber.StatusNotFound, "student_not_found",
				"The specified student was not found in this school.", nil)
		}
		if errors.Is(err, ErrDuplicateLink) {
			return writeError(c, fiber.StatusConflict, "duplicate_link",
				"This student is already linked to this parent.", nil)
		}
		return middleware.HTTPError(c, err)
	}

	return c.SendStatus(fiber.StatusOK)
}

// ============================================================================
// UNLINK STUDENT
// ============================================================================

// UnlinkStudent handles DELETE /api/v1/parents/:parent_id/students/:student_id.
func (h *Handler) UnlinkStudent(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	parentID := c.Params("parent_id")
	studentID := c.Params("student_id")

	if parentID == "" || studentID == "" {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "parent_id and student_id are required", nil)
	}

	if err := h.svc.UnlinkStudent(c.Context(), parentID, studentID, tenantID); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// ============================================================================
// fx Module
// ============================================================================

// Module is an fx-compatible module for the parents domain.
var Module = fx.Module("parents",
	fx.Provide(
		fx.Annotate(
			NewRepository,
			fx.As(new(Repository)),
			fx.As(new(StudentResolver)),
		),
		NewService,
		NewHandler,
	),
)
