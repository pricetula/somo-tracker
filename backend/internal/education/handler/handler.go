package handler

import (
	"net/http"

	"github.com/gofiber/fiber/v3"

	"somotracker/internal/platform/apperr"
	"somotracker/internal/platform/config"
	"somotracker/internal/education/service"
	"somotracker/internal/platform/logger"
)

// Handler is the HTTP handler for the education domain.
type Handler struct {
	svc  *service.Service
	cfg  *config.Config
	log  *logger.Logger
}

// New creates an education handler.
func New(svc *service.Service, cfg *config.Config, log *logger.Logger) *Handler {
	return &Handler{
		svc: svc,
		cfg: cfg,
		log: log,
	}
}

// ListCountries handles GET /api/v1/education/countries.
func (h *Handler) ListCountries(c fiber.Ctx) error {
	resp, err := h.svc.ListCountries(c.Context())
	if err != nil {
		return err
	}
	return c.Status(http.StatusOK).JSON(resp)
}

// ListEducationSystems handles GET /api/v1/education/systems.
func (h *Handler) ListEducationSystems(c fiber.Ctx) error {
	resp, err := h.svc.ListEducationSystems(c.Context())
	if err != nil {
		return err
	}
	return c.Status(http.StatusOK).JSON(resp)
}

// GetEducationSystem handles GET /api/v1/education/systems/:id.
func (h *Handler) GetEducationSystem(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return apperr.New(apperr.INVALID_REQUEST, "education system ID required")
	}

	resp, err := h.svc.GetEducationSystem(c.Context(), id)
	if err != nil {
		return err
	}
	return c.Status(http.StatusOK).JSON(resp)
}

// RegisterRoutes registers education routes under /api/v1/education.
func (h *Handler) RegisterRoutes(edu fiber.Router) {
	edu.Get("/countries", h.ListCountries)
	edu.Get("/systems", h.ListEducationSystems)
	edu.Get("/systems/:id", h.GetEducationSystem)
}
