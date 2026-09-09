package api

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	go_uber_zap "go.uber.org/zap"
	"somotracker/backend/internal/services"
)

type StreamsHandler struct {
	service services.StreamsService
}

func NewStreamsHandler(svc services.StreamsService) *StreamsHandler {
	return &StreamsHandler{service: svc}
}

// CreateStreams creates one or more streams for the active school.
//
// @Summary Create streams
// @Description Creates one or more streams for the active school.
// @Tags Streams
// @Accept json
// @Produce json
// @Param body body []string true "Stream names"
// @Success 201 {object} object
// @Failure 400 {object} object
// @Failure 401 {object} object
// @Router /school/streams [post]
func (h *StreamsHandler) CreateStreams(c fiber.Ctx) error {
	schoolID, ok := c.Locals("active_school_id").(string)
	if !ok || schoolID == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "active_school_id not found in session")
	}

	var body []string
	if err := c.Bind().Body(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "bad_request: invalid JSON body")
	}

	ids, err := h.service.CreateStreams(c.Context(), schoolID, body)
	if err != nil {
		if strings.Contains(err.Error(), "bad_request:") {
			return fiber.NewError(fiber.StatusBadRequest, err.Error()[len("bad_request:"):])
		}
		go_uber_zap.L().Error("streams: create failed", go_uber_zap.Error(err))
		return fiber.NewError(fiber.StatusInternalServerError, "internal_error: failed to create streams")
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"code":       "streams_created",
		"message":    "Streams created successfully",
		"stream_ids": ids,
		"errors":     fiber.Map{},
	})
}
