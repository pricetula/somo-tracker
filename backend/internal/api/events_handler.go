package api

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type EventsHandler struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

func NewEventsHandler(pool *pgxpool.Pool, logger *zap.Logger) *EventsHandler {
	return &EventsHandler{pool: pool, logger: logger.With(zap.String("handler", "events"))}
}

type EventItem struct {
	ID                 string `json:"id"`
	Title              string `json:"title"`
	EventType          string `json:"event_type"`
	StartDate          string `json:"start_date"`
	EndDate            string `json:"end_date"`
	RequiresAttendance bool   `json:"requires_attendance"`
}

type CreateEventRequest struct {
	Title              string `json:"title"`
	EventType          string `json:"event_type"`
	StartDate          string `json:"start_date"`
	EndDate            string `json:"end_date"`
	RequiresAttendance bool   `json:"requires_attendance"`
}

type UpdateEventRequest struct {
	Title              string `json:"title"`
	EventType          string `json:"event_type"`
	StartDate          string `json:"start_date"`
	EndDate            string `json:"end_date"`
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

	rows, err := h.pool.Query(c.Context(), `
		SELECT id, title, event_type, start_date, end_date, requires_attendance
		FROM school_events
		WHERE school_id = $1 AND start_date >= $2 AND start_date <= $3
		ORDER BY start_date ASC, start_date DESC
	`, schoolID, fromStr, toStr)
	if err != nil {
		h.logger.Error("list events failed", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    "internal_error",
			"message": "failed to list events",
			"errors":  fiber.Map{},
		})
	}
	defer rows.Close()

	var items []EventItem
	for rows.Next() {
		var id uuid.UUID
		var title, eventType string
		var startDate, endDate time.Time
		var requiresAttendance bool
		if err := rows.Scan(&id, &title, &eventType, &startDate, &endDate, &requiresAttendance); err != nil {
			h.logger.Error("scan event row failed", zap.Error(err))
			continue
		}
		items = append(items, EventItem{
			ID:                 id.String(),
			Title:              title,
			EventType:          eventType,
			StartDate:          startDate.Format("2006-01-02"),
			EndDate:            endDate.Format("2006-01-02"),
			RequiresAttendance: requiresAttendance,
		})
	}
	if err := rows.Err(); err != nil {
		h.logger.Error("rows error", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    "internal_error",
			"message": "failed to list events",
			"errors":  fiber.Map{},
		})
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
	err = h.pool.QueryRow(c.Context(), `
		INSERT INTO school_events (school_id, title, event_type, start_date, end_date, requires_attendance)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, schoolID, req.Title, req.EventType, req.StartDate, req.EndDate, req.RequiresAttendance).Scan(&id)
	if err != nil {
		h.logger.Error("create event failed", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    "internal_error",
			"message": "failed to create event",
			"errors":  fiber.Map{},
		})
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

	var id uuid.UUID
	var title, eventType string
	var startDate, endDate time.Time
	var requiresAttendance bool
	err = h.pool.QueryRow(c.Context(), `
		UPDATE school_events
		SET title = $2, event_type = $3, start_date = $4, end_date = $5, requires_attendance = $6
		WHERE id = $7 AND school_id = $1
		RETURNING id, title, event_type, start_date, end_date, requires_attendance
	`, schoolID, req.Title, req.EventType, req.StartDate, req.EndDate, req.RequiresAttendance, eventID).Scan(
		&id, &title, &eventType, &startDate, &endDate, &requiresAttendance,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"code":    "not_found",
				"message": "event not found",
				"errors":  fiber.Map{},
			})
		}
		h.logger.Error("update event failed", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    "internal_error",
			"message": "failed to update event",
			"errors":  fiber.Map{},
		})
	}

	updated := EventItem{
		ID:                 id.String(),
		Title:              title,
		EventType:          eventType,
		StartDate:          startDate.Format("2006-01-02"),
		EndDate:            endDate.Format("2006-01-02"),
		RequiresAttendance: requiresAttendance,
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

	_, err = h.pool.Exec(c.Context(), "DELETE FROM school_events WHERE id = $1 AND school_id = $2", eventID, schoolID)
	if err != nil {
		h.logger.Error("delete event failed", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    "internal_error",
			"message": "failed to delete event",
			"errors":  fiber.Map{},
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"code":    "event_deleted",
		"message": "Event deleted successfully",
		"errors":  fiber.Map{},
	})
}
