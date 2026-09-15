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

// ListStreams lists all streams for the active school.
//
// @Summary List streams
// @Description Lists all streams for the active school.
// @Tags Streams
// @Produce json
// @Success 200 {object} object
// @Failure 401 {object} object
// @Router /school/streams [get]
func (h *StreamsHandler) ListStreams(c fiber.Ctx) error {
	schoolID, ok := c.Locals("active_school_id").(string)
	if !ok || schoolID == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "active_school_id not found in session")
	}

	streams, err := h.service.ListStreams(c.Context(), schoolID)
	if err != nil {
		go_uber_zap.L().Error("streams: list failed", go_uber_zap.Error(err))
		return fiber.NewError(fiber.StatusInternalServerError, "internal_error: failed to list streams")
	}

	return c.JSON(fiber.Map{
		"code":    "streams_listed",
		"message": "Streams listed successfully",
		"streams": streams,
		"errors":  fiber.Map{},
	})
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

func (h *StreamsHandler) GetStream(c fiber.Ctx) error {
	schoolID, ok := c.Locals("active_school_id").(string)
	if !ok || schoolID == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "active_school_id not found in session")
	}
	streamID := c.Params("id")
	if streamID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "bad_request: stream id required")
	}
	stream, err := h.service.GetStream(c.Context(), streamID)
	if err != nil {
		if strings.Contains(err.Error(), "bad_request:") {
			return fiber.NewError(fiber.StatusBadRequest, err.Error()[len("bad_request:"):])
		}
		go_uber_zap.L().Error("streams: get failed", go_uber_zap.Error(err))
		return fiber.NewError(fiber.StatusInternalServerError, "internal_error: failed to get stream")
	}
	return c.JSON(fiber.Map{
		"code":    "stream_fetched",
		"message": "Stream fetched successfully",
		"stream":  stream,
		"errors":  fiber.Map{},
	})
}

func (h *StreamsHandler) UpdateStream(c fiber.Ctx) error {
	schoolID, ok := c.Locals("active_school_id").(string)
	if !ok || schoolID == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "active_school_id not found in session")
	}
	streamID := c.Params("id")
	if streamID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "bad_request: stream id required")
	}
	var body map[string]interface{}
	if err := c.Bind().Body(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "bad_request: invalid JSON body")
	}
	var name *string
	var color *string
	if v, ok := body["name"].(string); ok {
		name = &v
	}
	if v, ok := body["color"].(string); ok {
		color = &v
	}
	updated, err := h.service.UpdateStream(c.Context(), streamID, name, color)
	if err != nil {
		if strings.Contains(err.Error(), "bad_request:") {
			return fiber.NewError(fiber.StatusBadRequest, err.Error()[len("bad_request:"):])
		}
		go_uber_zap.L().Error("streams: update failed", go_uber_zap.Error(err))
		return fiber.NewError(fiber.StatusInternalServerError, "internal_error: failed to update stream")
	}
	return c.JSON(fiber.Map{
		"code":    "stream_updated",
		"message": "Stream updated successfully",
		"stream":  updated,
		"errors":  fiber.Map{},
	})
}

func (h *StreamsHandler) DeleteStreams(c fiber.Ctx) error {
	schoolID, ok := c.Locals("active_school_id").(string)
	if !ok || schoolID == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "active_school_id not found in session")
	}
	var body struct {
		Ids []string `json:"ids"`
	}
	if err := c.Bind().Body(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "bad_request: invalid JSON body")
	}
	if len(body.Ids) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "bad_request: at least one stream id required")
	}
	if err := h.service.DeleteStreams(c.Context(), body.Ids); err != nil {
		go_uber_zap.L().Error("streams: multi-delete failed", go_uber_zap.Error(err))
		return fiber.NewError(fiber.StatusInternalServerError, "internal_error: failed to delete streams")
	}
	return c.JSON(fiber.Map{
		"code":    "streams_deleted",
		"message": "Streams deleted successfully",
		"errors":  fiber.Map{},
	})
}
