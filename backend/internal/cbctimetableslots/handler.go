package cbctimetableslots

import (
	"context"
	"errors"

	"github.com/gofiber/fiber/v2"

	"somotracker/backend/internal/middleware"
)

// academicYearsAdapter is the subset of academicyears.Service that the handler uses.
type academicYearsAdapter interface {
	GetCurrentAcademicYearID(ctx context.Context, tenantID, schoolID string) (string, error)
	GetCurrentAcademicTermID(ctx context.Context, academicYearID string) (string, error)
}

// Handler exposes timetable slot HTTP endpoints.
type Handler struct {
	svc              *Service
	academicYearsSvc academicYearsAdapter
}

// NewHandler creates a new Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// SetAcademicYearsService sets the academicyears service reference.
func (h *Handler) SetAcademicYearsService(aySvc academicYearsAdapter) {
	h.academicYearsSvc = aySvc
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
	slots.Delete("/", middleware.RequireAuth, h.Delete)
}

// extractTenantSchool extracts tenant_id and school_id from the request context.
// Returns an error if school_id is not set (required for write operations).
func (h *Handler) extractTenantSchool(c *fiber.Ctx) (tenantID, schoolID string, err error) {
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

// List handles GET /api/v1/timetable/slots.
func (h *Handler) List(c *fiber.Ctx) error {
	academicYearID := c.Query("academic_year_id")
	if academicYearID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "academic_year_id query parameter is required",
		})
	}

	tenantID, _ := c.Locals("tenant_id").(string)

	filter := SlotFilter{
		AcademicYearID: academicYearID,
		TenantID:       tenantID,
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

	tenantID, _ := c.Locals("tenant_id").(string)

	filter := SlotFilter{
		AcademicYearID: academicYearID,
		TenantID:       tenantID,
		StructureID:    c.Query("structure_id"),
		ClassID:        c.Query("class_id"),
		TeacherID:      c.Query("teacher_id"),
		RoomIdentifier: c.Query("room_identifier"),
		Date:           c.Query("date"),
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
// academic_year_id is resolved server-side from the current active academic year.
func (h *Handler) Create(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.extractTenantSchool(c)
	if err != nil {
		return err
	}

	var payload CreateSlotPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "invalid request body",
		})
	}

	// Resolve current academic year server-side
	academicYearID, err := h.academicYearsSvc.GetCurrentAcademicYearID(c.Context(), tenantID, schoolID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}
	if academicYearID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "NO_ACTIVE_ACADEMIC_YEAR",
			"message": "No current academic year is set for this school.",
		})
	}
	payload.AcademicYearID = academicYearID

	fieldErrors := validateCreateSlotPayload(payload)
	if len(fieldErrors) > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "validation failed",
			"errors":  fieldErrors,
		})
	}

	slot, err := h.svc.CreateSlot(c.Context(), tenantID, schoolID, payload)
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
// academic_year_id is resolved server-side from the current active academic year.
func (h *Handler) BatchCreate(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.extractTenantSchool(c)
	if err != nil {
		return err
	}

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

	// Resolve current academic year server-side
	academicYearID, err := h.academicYearsSvc.GetCurrentAcademicYearID(c.Context(), tenantID, schoolID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}
	if academicYearID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "NO_ACTIVE_ACADEMIC_YEAR",
			"message": "No current academic year is set for this school.",
		})
	}
	// Inject the resolved academic year into all slots
	for i := range payload.Slots {
		payload.Slots[i].AcademicYearID = academicYearID
	}

	result, err := h.svc.BatchCreateSlots(c.Context(), tenantID, schoolID, payload)
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
			"message": "slot id is required",
		})
	}

	if err := h.svc.DeleteSlot(c.Context(), payload.ID); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"code":    "ok",
		"message": "Slot removed successfully",
	})
}

// validateCreateSlotPayload performs field-level validation.
// academic_year_id is resolved server-side and not validated here.
func validateCreateSlotPayload(payload CreateSlotPayload) map[string][]string {
	errors := make(map[string][]string)

	if payload.StructureID == "" {
		errors["structure_id"] = []string{"Structure (time block) is required"}
	}
	if payload.ClassID == "" {
		errors["class_id"] = []string{"Class is required"}
	}
	if payload.LearningAreaID == "" {
		errors["learning_area_id"] = []string{"Learning area is required"}
	}
	if payload.TeacherID == "" {
		errors["teacher_id"] = []string{"Teacher is required"}
	}

	return errors
}
