package api

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
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
