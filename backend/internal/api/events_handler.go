package api

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"somotracker/backend/internal/services"
)

type EventsHandler struct {
	svc    services.EventsService
	logger *zap.Logger
}

func NewEventsHandler(svc services.EventsService, logger *zap.Logger) *EventsHandler {
	return &EventsHandler{svc: svc, logger: logger.With(zap.String("handler", "events"))}
}

type EventItem struct {
	ID                 string `json:"id"`
	Title              string `json:"title" validate:"required"`
	EventType          string `json:"event_type" validate:"required"`
	StartDate          string `json:"start_date" validate:"required"`
	EndDate            string `json:"end_date" validate:"required"`
	RequiresAttendance bool   `json:"requires_attendance"`
}

type CreateEventRequest struct {
	Title              string `json:"title" validate:"required"`
	EventType          string `json:"event_type" validate:"required"`
	StartDate          string `json:"start_date" validate:"required"`
	EndDate            string `json:"end_date" validate:"required"`
	RequiresAttendance bool   `json:"requires_attendance"`
}

type UpdateEventRequest struct {
	Title              string `json:"title" validate:"required"`
	EventType          string `json:"event_type" validate:"required"`
	StartDate          string `json:"start_date" validate:"required"`
	EndDate            string `json:"end_date" validate:"required"`
	RequiresAttendance bool   `json:"requires_attendance"`
}

// ListEvents returns school events for the active school within a date range.
// @Summary List events
// @Description List upcoming school events for active school
// @Tags Events
// @Produce json
// @Param from query string false "Start date YYYY-MM-DD"
// @Param to query string false "End date YYYY-MM-DD"
// @Success 200 {array} EventItem
// @Failure 401 {object} map[string]any
// @Security ApiKeyAuth
// @Router /events [get]
func (h *EventsHandler) ListEvents(c fiber.Ctx) error {
	schoolIDStr, ok := c.Locals("active_school_id").(string)
	if !ok || schoolIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"code":    "unauthorized",
			"message": "active school not found",
			"errors":  fiber.Map{},
		})
	}
	schoolID, err := uuid.Parse(schoolIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "bad_request",
			"message": "invalid school id",
			"errors":  fiber.Map{},
		})
	}

	fromStr := c.Query("from")
	toStr := c.Query("to")
	if fromStr == "" {
		fromStr = time.Now().Format("2006-01-02")
	}
	if toStr == "" {
		toDate := time.Now().AddDate(0, 0, 7)
		toStr = toDate.Format("2006-01-02")
	}

	items, err := h.svc.ListEvents(c.Context(), schoolID, fromStr, toStr)
	if err != nil {
		h.logger.Error("list events failed", zap.Error(err))
		return WriteError(c, ErrInternal("failed to list events"))
	}

	return c.Status(fiber.StatusOK).JSON(items)
}

// CreateEvent creates a new school event for the active school.
// @Summary Create event
// @Description Create a new school event
// @Tags Events
// @Accept json
// @Produce json
// @Param body body CreateEventRequest true "Event data"
// @Success 201 {object} EventItem
// @Failure 400 {object} map[string]any
// @Failure 401 {object} map[string]any
// @Security ApiKeyAuth
// @Router /events [post]
func (h *EventsHandler) CreateEvent(c fiber.Ctx) error {
	schoolIDStr, ok := c.Locals("active_school_id").(string)
	if !ok || schoolIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"code":    "unauthorized",
			"message": "active school not found",
			"errors":  fiber.Map{},
		})
	}
	schoolID, err := uuid.Parse(schoolIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "bad_request",
			"message": "invalid school id",
			"errors":  fiber.Map{},
		})
	}

	var req CreateEventRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "bad_request",
			"message": "invalid request body",
			"errors":  fiber.Map{"body": []string{"Unable to parse JSON"}},
		})
	}

	errors := make(map[string][]string)
	if req.Title == "" {
		errors["title"] = []string{"title is required"}
	}
	if req.EventType == "" {
		errors["event_type"] = []string{"event_type is required"}
	}
	if req.StartDate == "" {
		errors["start_date"] = []string{"start_date is required"}
	}
	if req.EndDate == "" {
		errors["end_date"] = []string{"end_date is required"}
	}
	if len(errors) > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "validation_error",
			"message": "validation failed",
			"errors":  errors,
		})
	}

	var id uuid.UUID
	id, err = h.svc.CreateEvent(c.Context(), schoolID, services.CreateEventRequest{
		Title:              req.Title,
		EventType:          req.EventType,
		StartDate:          req.StartDate,
		EndDate:            req.EndDate,
		RequiresAttendance: req.RequiresAttendance,
	})
	if err != nil {
		h.logger.Error("create event failed", zap.Error(err))
		return WriteError(c, ErrInternal("failed to create event"))
	}

	item := EventItem{
		ID:                 id.String(),
		Title:              req.Title,
		EventType:          req.EventType,
		StartDate:          req.StartDate,
		EndDate:            req.EndDate,
		RequiresAttendance: req.RequiresAttendance,
	}
	return c.Status(fiber.StatusCreated).JSON(item)
}

// UpdateEvent updates an existing school event for the active school.
// @Summary Update event
// @Description Update a school event
// @Tags Events
// @Accept json
// @Produce json
// @Param id path string true "Event ID"
// @Param body body UpdateEventRequest true "Event data"
// @Success 200 {object} EventItem
// @Failure 400 {object} map[string]any
// @Failure 401 {object} map[string]any
// @Security ApiKeyAuth
// @Router /events/{id} [patch]
func (h *EventsHandler) UpdateEvent(c fiber.Ctx) error {
	schoolIDStr, ok := c.Locals("active_school_id").(string)
	if !ok || schoolIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"code":    "unauthorized",
			"message": "active school not found",
			"errors":  fiber.Map{},
		})
	}
	schoolID, err := uuid.Parse(schoolIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "bad_request",
			"message": "invalid school id",
			"errors":  fiber.Map{},
		})
	}

	eventIDStr := c.Params("id")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "bad_request",
			"message": "invalid event id",
			"errors":  fiber.Map{},
		})
	}

	var req UpdateEventRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "bad_request",
			"message": "invalid request body",
			"errors":  fiber.Map{"body": []string{"Unable to parse JSON"}},
		})
	}

	errors := make(map[string][]string)
	if req.Title == "" {
		errors["title"] = []string{"title is required"}
	}
	if req.EventType == "" {
		errors["event_type"] = []string{"event_type is required"}
	}
	if req.StartDate == "" {
		errors["start_date"] = []string{"start_date is required"}
	}
	if req.EndDate == "" {
		errors["end_date"] = []string{"end_date is required"}
	}
	if len(errors) > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "validation_error",
			"message": "validation failed",
			"errors":  errors,
		})
	}

	updated, err := h.svc.UpdateEvent(c.Context(), schoolID, eventID, services.UpdateEventRequest{
		Title:              req.Title,
		EventType:          req.EventType,
		StartDate:          req.StartDate,
		EndDate:            req.EndDate,
		RequiresAttendance: req.RequiresAttendance,
	})
	if err != nil {
		if err.Error() == "not_found" {
			return WriteError(c, ErrNotFound("event not found"))
		}
		h.logger.Error("update event failed", zap.Error(err))
		return WriteError(c, ErrInternal("failed to update event"))
	}
	return c.Status(fiber.StatusOK).JSON(updated)
}

// DeleteEvent deletes an existing school event for the active school.
// @Summary Delete event
// @Description Delete a school event
// @Tags Events
// @Produce json
// @Param id path string true "Event ID"
// @Success 200 {object} map[string]any
// @Failure 401 {object} map[string]any
// @Security ApiKeyAuth
// @Router /events/{id} [delete]
func (h *EventsHandler) DeleteEvent(c fiber.Ctx) error {
	schoolIDStr, ok := c.Locals("active_school_id").(string)
	if !ok || schoolIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"code":    "unauthorized",
			"message": "active school not found",
			"errors":  fiber.Map{},
		})
	}
	schoolID, err := uuid.Parse(schoolIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "bad_request",
			"message": "invalid school id",
			"errors":  fiber.Map{},
		})
	}

	eventIDStr := c.Params("id")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "bad_request",
			"message": "invalid event id",
			"errors":  fiber.Map{},
		})
	}

	if err := h.svc.DeleteEvent(c.Context(), schoolID, eventID); err != nil {
		h.logger.Error("delete event failed", zap.Error(err))
		return WriteError(c, ErrInternal("failed to delete event"))
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Event deleted successfully"})
}
