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

// @Summary      List countries
// @Description  Get all countries ordered by name
// @Tags         education
// @Produce      json
// @Success      200 {object} service.CountryListResponse
// @Router       /api/v1/education/countries [get]
func (h *Handler) ListCountries(c fiber.Ctx) error {
	resp, err := h.svc.ListCountries(c.Context())
	if err != nil {
		return err
	}
	return c.Status(http.StatusOK).JSON(resp)
}

// @Summary      List education systems
// @Description  Get all education systems with country info
// @Tags         education
// @Produce      json
// @Success      200 {object} service.EducationSystemListResponse
// @Router       /api/v1/education/systems [get]
func (h *Handler) ListEducationSystems(c fiber.Ctx) error {
	resp, err := h.svc.ListEducationSystems(c.Context())
	if err != nil {
		return err
	}
	return c.Status(http.StatusOK).JSON(resp)
}

// @Summary      Get education system
// @Description  Get a single education system with levels, bands, subjects, and terms
// @Tags         education
// @Produce      json
// @Param        id path string true "Education system ID"
// @Success      200 {object} service.EducationSystemResponse
// @Failure      400 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Router       /api/v1/education/systems/{id} [get]
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
