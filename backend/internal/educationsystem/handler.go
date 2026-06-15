package educationsystem

import (
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

// Handler exposes education-system HTTP endpoints.
type Handler struct {
	repo *Repository
}

// NewHandler creates a new Handler.
func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// RegisterRoutes mounts education-system routes on the given router.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	router.Get("/education-systems", h.List)
}

// List handles GET /education-systems.
//
// @Summary      List education systems
// @Description  Returns all available education systems (CBC, IGCSE, IB MYP, etc.)
// @Tags         Education Systems
// @Produce      json
// @Success      200  {array}   educationsystem.EducationSystem
// @Failure      500  {object}  fiber.Map
// @Router       /education-systems [get]
func (h *Handler) List(c *fiber.Ctx) error {
	systems, err := h.repo.ListAll(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "internal_error",
			"message": err.Error(),
		})
	}
	return c.JSON(systems)
}

// Module is an fx-compatible module for the education-system domain.
var Module = fx.Module("educationsystem",
	fx.Provide(
		NewRepository,
		NewHandler,
	),
)
