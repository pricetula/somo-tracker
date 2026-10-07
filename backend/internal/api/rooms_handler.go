package api

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	go_uber_zap "go.uber.org/zap"
	"somotracker/backend/internal/services"
)

type RoomsHandler struct {
	service services.RoomsService
}

func NewRoomsHandler(svc services.RoomsService) *RoomsHandler {
	return &RoomsHandler{service: svc}
}

type CreateRoomRequest struct {
	Name     string `json:"name"`
	Capacity *int   `json:"capacity"`
	RoomType string `json:"room_type"`
}

type UpdateRoomRequest struct {
	ID       string  `json:"id"`
	Name     *string `json:"name"`
	Capacity *int    `json:"capacity"`
	RoomType *string `json:"room_type"`
}

// CreateRoom creates a new room for the active school.
// @Summary Create room
// @Description Create a new room for the active school
// @Tags Rooms
// @Accept json
// @Produce json
// @Param body body CreateRoomRequest true "Room details"
// @Success 201 {object} object
// @Router /school/rooms [post]
func (h *RoomsHandler) CreateRoom(c fiber.Ctx) error {
	schoolID, ok := c.Locals("active_school_id").(string)
	if !ok || schoolID == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized: active school not found")
	}

	var req CreateRoomRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "bad_request: invalid JSON")
	}
	roomType := req.RoomType
	if roomType == "" {
		roomType = "STANDARD"
	}

	room, err := h.service.CreateRoom(c.Context(), schoolID, req.Name, req.Capacity, roomType)
	if err != nil {
		if strings.Contains(err.Error(), "bad_request:") {
			return fiber.NewError(fiber.StatusBadRequest, err.Error()[len("bad_request:"):])
		}
		go_uber_zap.L().Error("rooms: create failed", go_uber_zap.Error(err))
		return fiber.NewError(fiber.StatusInternalServerError, "internal_error: failed to create room")
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"code":    "room_created",
		"message": "Room created successfully",
		"room": fiber.Map{
			"id":         room.ID.String(),
			"school_id":  room.SchoolID.String(),
			"name":       room.Name,
			"capacity":   room.Capacity.Int32,
			"room_type":  room.RoomType,
			"created_at": room.CreatedAt,
			"updated_at": room.UpdatedAt,
		},
		"errors": fiber.Map{},
	})
}

// ListRooms lists all rooms for the active school.
// @Summary List rooms
// @Description List all rooms for the active school with pagination
// @Tags Rooms
// @Produce json
// @Param page query int false "Page number"
// @Param limit query int false "Page size"
// @Success 200 {object} object
// @Router /school/rooms [get]
func (h *RoomsHandler) ListRooms(c fiber.Ctx) error {
	schoolID, ok := c.Locals("active_school_id").(string)
	if !ok || schoolID == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized: active school not found")
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "50"))

	rooms, total, err := h.service.ListRooms(c.Context(), schoolID, page, limit)
	if err != nil {
		if strings.Contains(err.Error(), "bad_request:") {
			return fiber.NewError(fiber.StatusBadRequest, err.Error()[len("bad_request:"):])
		}
		go_uber_zap.L().Error("rooms: list failed", go_uber_zap.Error(err))
		return fiber.NewError(fiber.StatusInternalServerError, "internal_error: failed to list rooms")
	}

	items := make([]fiber.Map, 0, len(rooms))
	for _, r := range rooms {
		items = append(items, fiber.Map{
			"id":         r.ID.String(),
			"school_id":  r.SchoolID.String(),
			"name":       r.Name,
			"capacity":   r.Capacity.Int32,
			"room_type":  r.RoomType,
			"created_at": r.CreatedAt,
			"updated_at": r.UpdatedAt,
		})
	}

	return c.JSON(fiber.Map{
		"code":   "rooms_listed",
		"items":  items,
		"total":  total,
		"page":   page,
		"limit":  limit,
		"errors": fiber.Map{},
	})
}

// GetRoom gets a single room by ID.
// @Summary Get room
// @Description Get a room by ID for the active school
// @Tags Rooms
// @Produce json
// @Param id path string true "Room ID"
// @Success 200 {object} object
// @Router /school/rooms/{id} [get]
func (h *RoomsHandler) GetRoom(c fiber.Ctx) error {
	schoolID, ok := c.Locals("active_school_id").(string)
	if !ok || schoolID == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized: active school not found")
	}
	roomID := c.Params("id")
	if roomID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "bad_request: room id required")
	}

	room, err := h.service.GetRoom(c.Context(), schoolID, roomID)
	if err != nil {
		if strings.Contains(err.Error(), "bad_request:") {
			return fiber.NewError(fiber.StatusBadRequest, err.Error()[len("bad_request:"):])
		}
		go_uber_zap.L().Error("rooms: get failed", go_uber_zap.Error(err))
		return fiber.NewError(fiber.StatusInternalServerError, "internal_error: failed to get room")
	}

	return c.JSON(fiber.Map{
		"code": "room_retrieved",
		"room": fiber.Map{
			"id":         room.ID.String(),
			"school_id":  room.SchoolID.String(),
			"name":       room.Name,
			"capacity":   room.Capacity.Int32,
			"room_type":  room.RoomType,
			"created_at": room.CreatedAt,
			"updated_at": room.UpdatedAt,
		},
		"errors": fiber.Map{},
	})
}

// UpdateRoom updates an existing room.
// @Summary Update room
// @Description Update room details for the active school
// @Tags Rooms
// @Accept json
// @Produce json
// @Param body body UpdateRoomRequest true "Room update"
// @Success 200 {object} object
// @Router /school/rooms [patch]
func (h *RoomsHandler) UpdateRoom(c fiber.Ctx) error {
	schoolID, ok := c.Locals("active_school_id").(string)
	if !ok || schoolID == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized: active school not found")
	}

	var req UpdateRoomRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "bad_request: invalid JSON")
	}
	if req.ID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "bad_request: id is required")
	}

	room, err := h.service.UpdateRoom(c.Context(), schoolID, req.ID, req.Name, req.Capacity, req.RoomType)
	if err != nil {
		if strings.Contains(err.Error(), "bad_request:") {
			return fiber.NewError(fiber.StatusBadRequest, err.Error()[len("bad_request:"):])
		}
		go_uber_zap.L().Error("rooms: update failed", go_uber_zap.Error(err))
		return fiber.NewError(fiber.StatusInternalServerError, "internal_error: failed to update room")
	}

	return c.JSON(fiber.Map{
		"code":    "room_updated",
		"message": "Room updated successfully",
		"room": fiber.Map{
			"id":         room.ID.String(),
			"school_id":  room.SchoolID.String(),
			"name":       room.Name,
			"capacity":   room.Capacity.Int32,
			"room_type":  room.RoomType,
			"created_at": room.CreatedAt,
			"updated_at": room.UpdatedAt,
		},
		"errors": fiber.Map{},
	})
}

// DeleteRoom deletes a room by id.
// @Summary Delete room
// @Description Delete a room for the active school
// @Tags Rooms
// @Produce json
// @Param id path string true "Room ID"
// @Success 200 {object} object
// @Router /school/rooms/{id} [delete]
func (h *RoomsHandler) DeleteRoom(c fiber.Ctx) error {
	schoolID, ok := c.Locals("active_school_id").(string)
	if !ok || schoolID == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized: active school not found")
	}
	roomID := c.Params("id")
	if roomID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "bad_request: room id required")
	}

	if err := h.service.DeleteRoom(c.Context(), schoolID, roomID); err != nil {
		if strings.Contains(err.Error(), "bad_request:") {
			return fiber.NewError(fiber.StatusBadRequest, err.Error()[len("bad_request:"):])
		}
		go_uber_zap.L().Error("rooms: delete failed", go_uber_zap.Error(err))
		return fiber.NewError(fiber.StatusInternalServerError, "internal_error: failed to delete room")
	}

	return c.JSON(fiber.Map{
		"code":    "room_deleted",
		"message": "Room deleted successfully",
		"errors":  fiber.Map{},
	})
}
