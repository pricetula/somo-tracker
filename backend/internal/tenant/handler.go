package tenant

import (
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"

	"somotracker/backend/internal/middleware"
)

// CreateTenantPayload is the request body for POST /tenants.
type CreateTenantPayload struct {
	Name string `json:"name"`
	Slug string `json:"slug,omitempty"`
}

// Handler exposes tenant-related HTTP endpoints.
type Handler struct {
	svc *Service
}

// NewHandler creates a new Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts tenant routes on the given router.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	tenants := router.Group("/tenants")
	tenants.Post("/", h.Create)
}

// Create handles POST /tenants.
func (h *Handler) Create(c *fiber.Ctx) error {
	var payload CreateTenantPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "invalid request body",
		})
	}
	if payload.Name == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "name is required",
		})
	}

	tenant, err := h.svc.CreateTenant(c.Context(), payload.Name, payload.Slug)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(tenant)
}

// Module is an fx-compatible module for the tenant domain.
var Module = fx.Module("tenant",
	fx.Provide(
		fx.Annotate(NewRepository, fx.As(new(Repository))),
		NewService,
		NewHandler,
	),
)
