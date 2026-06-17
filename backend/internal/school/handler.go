package school

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"

	"somotracker/backend/internal/database"
	"somotracker/backend/internal/middleware"
)

var uuidV4Regex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

// Handler exposes school-related HTTP endpoints.
type Handler struct {
	svc  *Service
	pool *pgxpool.Pool
}

// loadSchoolSession is the session info extracted from the cookie manually.
type loadSchoolSession struct {
	UserID   string
	TenantID string
	Role     string
}

// NewHandler creates a new Handler.
func NewHandler(svc *Service, pool *database.Pools) *Handler {
	return &Handler{svc: svc, pool: pool.PG}
}

// RegisterRoutes mounts school routes on the given router.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	schools := router.Group("/schools")
	schools.Get("/", h.List)
	schools.Post("/", h.Create)

	// Activate school — switch the user's current active school.
	// Manually extracts session from cookie since school routes aren't under /api/.
	schools.Post("/:id/activate", h.Activate)
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

// Activate handles POST /schools/:id/activate.
//
// @Summary      Activate school
// @Description  Switches the authenticated user's active school by deactivating all
//
//	memberships and activating the target school membership.
//
// @Tags         Schools
// @Produce      json
// @Success      200  {object}  School
// @Failure      401  {object}  ErrorBody  "Unauthorized"
// @Failure      404  {object}  ErrorBody  "Not found"
// @Failure      500  {object}  ErrorBody  "Internal error"
// @Router       /schools/{id}/activate [post]
func (h *Handler) Activate(c *fiber.Ctx) error {
	schoolID := c.Params("id")
	if schoolID == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(ErrorBody{
			Error:   "invalid_input",
			Message: "school id is required",
		})
	}

	// Manually extract session from the somo_sid cookie
	// (school routes aren't under /api/ so the session middleware doesn't load)
	session, err := h.loadSessionFromCookie(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(ErrorBody{
			Error:   "unauthorized",
			Message: "authentication required",
		})
	}

	if err := h.svc.ActivateSchool(c.Context(), session.UserID, schoolID, session.TenantID); err != nil {
		if err.Error() == "school not found" {
			return c.Status(fiber.StatusNotFound).JSON(ErrorBody{
				Error:   "not_found",
				Message: "school not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorBody{
			Error:   "internal_error",
			Message: err.Error(),
		})
	}

	// Return the updated school
	school, err := h.svc.GetByID(c.Context(), schoolID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorBody{
			Error:   "internal_error",
			Message: err.Error(),
		})
	}

	return c.JSON(school)
}

// loadSessionFromCookie reads the somo_sid cookie and queries the session + active
// membership from Postgres. This is needed because the session middleware only runs
// for /api/ routes, but school routes are mounted at /schools/.
func (h *Handler) loadSessionFromCookie(c *fiber.Ctx) (*loadSchoolSession, error) {
	token := c.Cookies("somo_sid")
	if token == "" {
		return nil, fmt.Errorf("no session cookie")
	}

	const query = `
		SELECT s.user_id, s.tenant_id,
		       COALESCE(
		         (SELECT role::text FROM memberships
		           WHERE user_id = s.user_id AND is_active = true
		           ORDER BY
		             CASE role
		               WHEN 'SYSTEM_ADMIN' THEN 1
		               WHEN 'SCHOOL_ADMIN' THEN 2
		               WHEN 'TEACHER' THEN 3
		               WHEN 'SUPPORT_STAFF' THEN 4
		             END
		           LIMIT 1),
		         'TEACHER'
		       ) as role
		FROM sessions s
		WHERE s.token = $1 AND s.expires_at > NOW()
	`

	var s loadSchoolSession
	err := h.pool.QueryRow(c.Context(), query, token).Scan(&s.UserID, &s.TenantID, &s.Role)
	if err != nil {
		return nil, fmt.Errorf("load session from cookie: %w", err)
	}

	return &s, nil
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
