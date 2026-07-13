package cbctimetableslots

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"somotracker/backend/internal/middleware"
)

// Handler exposes timetable slot HTTP endpoints.
type Handler struct {
	svc *Service
}

// NewHandler creates a new Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts slot routes on the given router.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	slots := router.Group("/api/v1/timetable/slots")
	slots.Get("/", middleware.RequireAuth, h.List)
	slots.Get("/enriched", middleware.RequireAuth, h.ListEnriched)
	slots.Get("/:id", middleware.RequireAuth, h.GetByID)
	slots.Post("/", middleware.RequireAuth, h.Create)
	slots.Post("/batch", middleware.RequireAuth, h.BatchCreate)
	slots.Put("/:id", middleware.RequireAuth, h.Update)
	slots.Delete("/:id", middleware.RequireAuth, h.Delete)
}

// List handles GET /api/v1/timetable/slots.
func (h *Handler) List(c *fiber.Ctx) error {
	academicYearID := c.Query("academic_year_id")
	if academicYearID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "academic_year_id query parameter is required",
		})
	}

	filter := SlotFilter{
		AcademicYearID: academicYearID,
		StructureID:    c.Query("structure_id"),
		ClassID:        c.Query("class_id"),
		TeacherID:      c.Query("teacher_id"),
		RoomIdentifier: c.Query("room_identifier"),
	}

	result, err := h.svc.ListSlots(c.Context(), filter)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// ListEnriched handles GET /api/v1/timetable/slots/enriched.
func (h *Handler) ListEnriched(c *fiber.Ctx) error {
	academicYearID := c.Query("academic_year_id")
	if academicYearID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "academic_year_id query parameter is required",
		})
	}

	viewBy := c.Query("view_by") // class, teacher, or room
	filter := SlotFilter{
		AcademicYearID: academicYearID,
		StructureID:    c.Query("structure_id"),
	}

	switch viewBy {
	case "class":
		filter.ClassID = c.Query("class_id")
	case "teacher":
		filter.TeacherID = c.Query("teacher_id")
	case "room":
		filter.RoomIdentifier = c.Query("room_identifier")
	}

	result, err := h.svc.ListEnrichedSlots(c.Context(), filter)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// GetByID handles GET /api/v1/timetable/slots/:id.
func (h *Handler) GetByID(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "slot id is required",
		})
	}

	slot, err := h.svc.GetSlot(c.Context(), id)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(slot)
}

// Create handles POST /api/v1/timetable/slots.
func (h *Handler) Create(c *fiber.Ctx) error {
	var payload CreateSlotPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "invalid request body",
		})
	}

	fieldErrors := validateCreateSlotPayload(payload)
	if len(fieldErrors) > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "validation failed",
			"errors":  fieldErrors,
		})
	}

	slot, err := h.svc.CreateSlot(c.Context(), payload)
	if err != nil {
		if errors.Is(err, ErrClassSlotOccupied) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"code":    "slot_already_occupied",
				"message": "This class already has a scheduled lesson during this time period.",
			})
		}
		if errors.Is(err, ErrTeacherDoubleBooked) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"code":    "teacher_double_booked",
				"message": "This teacher is already assigned to another class during this period.",
			})
		}
		if errors.Is(err, ErrRoomDoubleBooked) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"code":    "room_double_booked",
				"message": "This room is already assigned to another lesson during this period.",
			})
		}
		return middleware.HTTPError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(slot)
}

// BatchCreate handles POST /api/v1/timetable/slots/batch.
func (h *Handler) BatchCreate(c *fiber.Ctx) error {
	var payload BatchCreateSlotsPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "invalid request body",
		})
	}

	if len(payload.Slots) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "at least one slot is required",
		})
	}

	result, err := h.svc.BatchCreateSlots(c.Context(), payload)
	if err != nil {
		if errors.Is(err, ErrClassSlotOccupied) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"code":    "slot_already_occupied",
				"message": "A conflict occurred: one of the slots is already occupied.",
			})
		}
		if errors.Is(err, ErrTeacherDoubleBooked) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"code":    "teacher_double_booked",
				"message": "A conflict occurred: one of the teachers is double-booked.",
			})
		}
		if errors.Is(err, ErrRoomDoubleBooked) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"code":    "room_double_booked",
				"message": "A conflict occurred: one of the rooms is double-booked.",
			})
		}
		return middleware.HTTPError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(result)
}

// Update handles PUT /api/v1/timetable/slots/:id.
func (h *Handler) Update(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "slot id is required",
		})
	}

	var payload UpdateSlotPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "invalid request body",
		})
	}

	slot, err := h.svc.UpdateSlot(c.Context(), id, payload)
	if err != nil {
		if errors.Is(err, ErrTeacherDoubleBooked) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"code":    "teacher_double_booked",
				"message": "This teacher is already assigned to another class during this period.",
			})
		}
		if errors.Is(err, ErrRoomDoubleBooked) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"code":    "room_double_booked",
				"message": "This room is already assigned to another lesson during this period.",
			})
		}
		return middleware.HTTPError(c, err)
	}

	return c.JSON(slot)
}

// Delete handles DELETE /api/v1/timetable/slots/:id.
func (h *Handler) Delete(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "slot id is required",
		})
	}

	if err := h.svc.DeleteSlot(c.Context(), id); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"code":    "ok",
		"message": "Slot removed successfully",
	})
}

// validateCreateSlotPayload performs field-level validation.
func validateCreateSlotPayload(payload CreateSlotPayload) map[string][]string {
	errors := make(map[string][]string)

	if payload.AcademicYearID == "" {
		errors["academic_year_id"] = []string{"Academic year is required"}
	}
	if payload.StructureID == "" {
		errors["structure_id"] = []string{"Structure (time block) is required"}
	}
	if payload.ClassID == "" {
		errors["class_id"] = []string{"Class is required"}
	}

	return errors
}
