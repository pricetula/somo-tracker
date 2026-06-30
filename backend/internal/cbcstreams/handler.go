package cbcstreams

import (
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
	streams.Post("/", middleware.RequireAuth, h.Create)
	streams.Put("/:id", middleware.RequireAuth, h.Update)
	streams.Delete("/:id", middleware.RequireAuth, h.Delete)
}

// schoolIDFromContext extracts the school ID from the request context.
// Checks the query param first (explicit override), then falls back to
// c.Locals("active_school_id") set by the security middleware from the
// somo_school_id cookie.
func schoolIDFromContext(c *fiber.Ctx) string {
	if schoolID := c.Query("school_id"); schoolID != "" {
		return schoolID
	}
	if schoolID, ok := c.Locals("active_school_id").(string); ok && schoolID != "" {
		return schoolID
	}
	return ""
}

// ─── Handlers ──────────────────────────────────────────────────────────────

// List handles GET /api/v1/streams.
func (h *Handler) List(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID := schoolIDFromContext(c)
	if schoolID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "VALIDATION_ERROR",
			"message": "active school not set",
		})
	}

	streams, err := h.svc.ListStreams(c.Context(), tenantID, schoolID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(ListStreamsResponse{
		Data: streams,
	})
}

// Create handles POST /api/v1/streams.
func (h *Handler) Create(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID := schoolIDFromContext(c)
	if schoolID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "VALIDATION_ERROR",
			"message": "active school not set",
		})
	}

	var payload CreateStreamPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error":   "VALIDATION_ERROR",
			"message": "invalid request body",
		})
	}

	if payload.Name == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error":   "VALIDATION_ERROR",
			"message": "name is required",
		})
	}

	if len(payload.Name) > 100 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error":   "VALIDATION_ERROR",
			"message": "name must not exceed 100 characters",
		})
	}

	stream, err := h.svc.CreateStream(c.Context(), tenantID, schoolID, payload.Name)
	if err != nil {
		// Map DB unique violation to 409 Conflict
		return mapStreamError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(stream)
}

// Update handles PUT /api/v1/streams/:id.
func (h *Handler) Update(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID := schoolIDFromContext(c)
	if schoolID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "VALIDATION_ERROR",
			"message": "active school not set",
		})
	}

	streamID := c.Params("id")
	if streamID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "VALIDATION_ERROR",
			"message": "stream id is required",
		})
	}

	var payload UpdateStreamPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error":   "VALIDATION_ERROR",
			"message": "invalid request body",
		})
	}

	if payload.Name == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error":   "VALIDATION_ERROR",
			"message": "name is required",
		})
	}

	if len(payload.Name) > 100 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error":   "VALIDATION_ERROR",
			"message": "name must not exceed 100 characters",
		})
	}

	stream, err := h.svc.UpdateStream(c.Context(), streamID, tenantID, schoolID, payload.Name)
	if err != nil {
		return mapStreamError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(stream)
}

// Delete handles DELETE /api/v1/streams/:id.
func (h *Handler) Delete(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID := schoolIDFromContext(c)
	if schoolID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "VALIDATION_ERROR",
			"message": "active school not set",
		})
	}

	streamID := c.Params("id")
	if streamID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "VALIDATION_ERROR",
			"message": "stream id is required",
		})
	}

	err := h.svc.DeleteStream(c.Context(), streamID, tenantID, schoolID)
	if err != nil {
		return mapStreamError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// mapStreamError maps domain errors to the spec's error response shape.
func mapStreamError(c *fiber.Ctx, err error) error {
	// Check for the specific "stream in use" sentinel first
	if isErrStreamInUse(err) {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error":   "STREAM_IN_USE",
			"message": "Stream is in use by one or more classes and cannot be deleted.",
		})
	}
	// Use the standard HTTPError mapper for everything else (404, 409, etc.)
	return middleware.HTTPError(c, err)
}

// isErrStreamInUse checks if the error chain contains ErrStreamInUse.
func isErrStreamInUse(err error) bool {
	for err != nil {
		if err == ErrStreamInUse {
			return true
		}
		if unwrapper, ok := err.(interface{ Unwrap() error }); ok {
			err = unwrapper.Unwrap()
		} else {
			return false
		}
	}
	return false
}
