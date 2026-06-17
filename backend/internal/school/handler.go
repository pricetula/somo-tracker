package school

import (
	"errors"
	"regexp"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"

	"somotracker/backend/internal/middleware"
)

var uuidV4Regex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

// Handler exposes school-related HTTP endpoints.
type Handler struct {
	svc *Service
}

// NewHandler creates a new Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts school routes on the given router.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	schools := router.Group("/schools")
	schools.Get("/", h.List)
	schools.Post("/", h.Create)
}

// List handles GET /schools?tenant_id=...
//
// @Summary      List schools
// @Description  Returns all active schools for the authenticated user's tenant.
// @Tags         Schools
// @Produce      json
// @Param        tenant_id  query  string  true  "Tenant ID"
// @Success      200  {array}   school.School
// @Failure      422  {object}  ErrorBody  "Invalid input"
// @Failure      500  {object}  ErrorBody  "Internal error"
// @Router       /schools [get]
func (h *Handler) List(c *fiber.Ctx) error {
	tenantID := c.Query("tenant_id")
	if tenantID == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(ErrorBody{
			Error:   "invalid_input",
			Message: "tenant_id query parameter is required",
		})
	}

	schools, err := h.svc.ListByTenant(c.Context(), tenantID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorBody{
			Error:   "internal_error",
			Message: err.Error(),
		})
	}

	return c.JSON(schools)
}

// Create handles POST /schools.
//
// @Summary      Create a school
// @Description  Creates a new school under the current tenant and assigns the current user as SCHOOL_ADMIN.
// @Tags         Schools
// @Accept       json
// @Produce      json
// @Param        body  body      CreateSchoolPayload  true  "School details"
// @Success      201   {object}  School
// @Failure      401   {object}  ErrorBody  "Unauthorized"
// @Failure      403   {object}  ErrorBody  "Forbidden"
// @Failure      422   {object}  ErrorBody  "Invalid input"
// @Failure      500   {object}  ErrorBody  "Internal error"
// @Router       /schools [post]
func (h *Handler) Create(c *fiber.Ctx) error {
	// Only authenticated users can create schools
	session, ok := c.Locals("session").(*middleware.SessionInfo)
	if !ok || session == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(ErrorBody{
			Error:   "unauthorized",
			Message: "authentication required",
		})
	}

	// Only SCHOOL_ADMIN and SYSTEM_ADMIN can create schools
	if session.Role != "SCHOOL_ADMIN" && session.Role != "SYSTEM_ADMIN" {
		return c.Status(fiber.StatusForbidden).JSON(ErrorBody{
			Error:   "forbidden",
			Message: "only school admins can create schools",
		})
	}

	var payload CreateSchoolPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(ErrorBody{
			Error:   "invalid_input",
			Message: "invalid request body",
		})
	}

	if payload.Name == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(ErrorBody{
			Error:   "invalid_input",
			Message: "name is required",
		})
	}

	if payload.EducationSystemID == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(ErrorBody{
			Error:   "invalid_input",
			Message: "education_system_id is required",
		})
	}

	if !uuidV4Regex.MatchString(payload.EducationSystemID) {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(ErrorBody{
			Error:   "invalid_input",
			Message: "education_system_id must be a valid UUID",
		})
	}

	school, err := h.svc.CreateSchool(c.Context(), session.TenantID, payload.Name, payload.EducationSystemID, session.UserID)
	if err != nil {
		if errors.Is(err, ErrNameAlreadyExists) {
			return c.Status(fiber.StatusConflict).JSON(ErrorBody{
				Error:   "already_exists",
				Message: err.Error(),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorBody{
			Error:   "internal_error",
			Message: err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(school)
}

// Module is an fx-compatible module for the school domain.
var Module = fx.Module("school",
	fx.Provide(
		NewRepository,
		NewService,
		NewHandler,
	),
)
