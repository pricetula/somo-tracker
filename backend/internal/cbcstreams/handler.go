package cbcstreams

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"somotracker/backend/internal/middleware"
)

// Handler exposes stream HTTP endpoints.
type Handler struct {
	svc *Service
}

// NewHandler creates a new Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts stream routes on the given router.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	streams := router.Group("/api/v1/streams")
	streams.Get("/", middleware.RequireAuth, h.List)
	streams.Get("/:id", middleware.RequireAuth, h.GetByID)
	streams.Post("/", middleware.RequireAuth, h.Create)
	streams.Put("/:id", middleware.RequireAuth, h.Update)
	streams.Delete("/", middleware.RequireAuth, h.Delete)
}

// List handles GET /api/v1/streams.
func (h *Handler) List(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID, _ := c.Locals("active_school_id").(string)
	if schoolID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "active school not set",
		})
	}

	result, err := h.svc.ListStreams(c.Context(), tenantID, schoolID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// GetByID handles GET /api/v1/streams/:id.
func (h *Handler) GetByID(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.streamMiddleware(c)
	if err != nil {
		return err
	}

	streamID := c.Params("id")
	if streamID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "stream id is required",
		})
	}

	stream, err := h.svc.GetStreamByID(c.Context(), streamID, tenantID, schoolID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(stream)
}

// streamMiddleware extracts common tenant/school from context and validates active school.
func (h *Handler) streamMiddleware(c *fiber.Ctx) (tenantID, schoolID string, err error) {
	tenantID = c.Locals("tenant_id").(string)
	schoolID, _ = c.Locals("active_school_id").(string)
	if schoolID == "" {
		return "", "", c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "active school not set",
		})
	}
	return tenantID, schoolID, nil
}

// Create handles POST /api/v1/streams.
func (h *Handler) Create(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.streamMiddleware(c)
	if err != nil {
		return err
	}

	var payload CreateStreamPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "invalid request body",
		})
	}

	if payload.Name == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "name is required",
			"errors":  map[string][]string{"name": {"Stream name is required"}},
		})
	}

	stream, err := h.svc.CreateStream(c.Context(), tenantID, schoolID, payload.Name, payload.Color)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(stream)
}

// Update handles PUT /api/v1/streams/:id.
func (h *Handler) Update(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.streamMiddleware(c)
	if err != nil {
		return err
	}

	streamID := c.Params("id")
	if streamID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "stream id is required",
		})
	}

	var payload UpdateStreamPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "invalid request body",
		})
	}

	if payload.Name == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "name is required",
			"errors":  map[string][]string{"name": {"Stream name is required"}},
		})
	}

	stream, err := h.svc.UpdateStream(c.Context(), streamID, tenantID, schoolID, payload.Name, payload.Color)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(stream)
}

// Delete handles DELETE /api/v1/streams/:id.
func (h *Handler) Delete(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.streamMiddleware(c)
	if err != nil {
		return err
	}

	var payload struct {
		ID string `json:"id"`
	}
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "invalid request body",
		})
	}
	if payload.ID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "stream id is required",
		})
	}

	if err := h.svc.DeleteStream(c.Context(), payload.ID, tenantID, schoolID); err != nil {
		// Stream has active enrollments — return a human-readable 409.
		if errors.Is(err, ErrStreamHasActiveEnrollments) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"code":    "stream_has_active_enrollments",
				"message": err.Error(),
			})
		}
		return middleware.HTTPError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}
