package timetable

import (
	"context"
	"strings"

	"github.com/gofiber/fiber/v2"

	"somotracker/backend/internal/middleware"
)

// Handler exposes timetable HTTP endpoints.
type Handler struct {
	svc              Service
	academicYearsSvc interface {
		GetCurrentYearAndTermID(ctx context.Context, tenantID, schoolID string) (yearID, termID string, err error)
	}
}

// NewHandler creates a new timetable Handler.
func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

// SetAcademicYearsService sets the academic years service reference.
func (h *Handler) SetAcademicYearsService(svc interface {
	GetCurrentYearAndTermID(ctx context.Context, tenantID, schoolID string) (yearID, termID string, err error)
}) {
	h.academicYearsSvc = svc
}

// RegisterRoutes mounts timetable routes on the given router.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	base := router.Group("/api/v1/timetable")

	// Structure mutations
	base.Post("/block", middleware.RequireAuth, h.CreateBlock)
	base.Get("/block", middleware.RequireAuth, h.ListBlocks)
	base.Get("/block/:id", middleware.RequireAuth, h.GetBlock)
	base.Put("/block/:id", middleware.RequireAuth, h.UpdateBlock)
	base.Delete("/block/:id", middleware.RequireAuth, h.DeleteBlock)

	// Slot mutations
	base.Post("/slots", middleware.RequireAuth, h.CreateSlot)
	base.Get("/slots", middleware.RequireAuth, h.ListSlots)
	base.Post("/slots/batch", middleware.RequireAuth, h.BatchCreateSlots)
	base.Put("/slots/:id", middleware.RequireAuth, h.UpdateSlot)
	base.Delete("/slots/:id", middleware.RequireAuth, h.DeleteSlot)

	// Read-only timetable view
	base.Get("/timetable", middleware.RequireAuth, h.GetTimetable)
}

// tmMiddleware extracts common tenant/school context.
func (h *Handler) tmMiddleware(c *fiber.Ctx) (tenantID, schoolID string, err error) {
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

// resolveCurrentYear resolves the current academic year ID for the school.
// Returns empty string if no current year is set.
func (h *Handler) resolveCurrentYear(c *fiber.Ctx, tenantID, schoolID string) (string, error) {
	if h.academicYearsSvc == nil {
		return "", nil
	}
	yearID, _, err := h.academicYearsSvc.GetCurrentYearAndTermID(c.Context(), tenantID, schoolID)
	return yearID, err
}

// validateTimeBlockPayload validates CreateTimeBlockPayload / UpdateTimeBlockPayload.
func validateTimeBlockPayload(p *CreateTimeBlockPayload) error {
	if p.DayOfWeek < 1 || p.DayOfWeek > 7 {
		return &middleware.FieldError{
			Err:    ErrInvalidInput,
			Fields: map[string][]string{"day_of_week": {"must be between 1 (Monday) and 7 (Sunday)"}},
		}
	}
	if strings.TrimSpace(p.PeriodName) == "" {
		return &middleware.FieldError{
			Err:    ErrInvalidInput,
			Fields: map[string][]string{"period_name": {"period_name is required"}},
		}
	}
	if p.StartTime == "" || p.EndTime == "" {
		return &middleware.FieldError{
			Err:    ErrInvalidInput,
			Fields: map[string][]string{"start_time": {"start_time and end_time are required"}},
		}
	}
	// Basic time format validation (HH:MM)
	if len(p.StartTime) != 5 || p.StartTime[2] != ':' {
		return &middleware.FieldError{
			Err:    ErrInvalidInput,
			Fields: map[string][]string{"start_time": {"must be in HH:MM format"}},
		}
	}
	if len(p.EndTime) != 5 || p.EndTime[2] != ':' {
		return &middleware.FieldError{
			Err:    ErrInvalidInput,
			Fields: map[string][]string{"end_time": {"must be in HH:MM format"}},
		}
	}
	if p.StartTime >= p.EndTime {
		return &middleware.FieldError{
			Err:    ErrInvalidInput,
			Fields: map[string][]string{"end_time": {"end_time must be after start_time"}},
		}
	}
	if p.Order < 0 {
		return &middleware.FieldError{
			Err:    ErrInvalidInput,
			Fields: map[string][]string{"order": {"must be non-negative"}},
		}
	}
	return nil
}

// ── Structure (TimeBlock) Handlers ────────────────────────────────────────

// CreateBlock handles POST /api/v1/timetable/block.
func (h *Handler) CreateBlock(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.tmMiddleware(c)
	if err != nil {
		return err
	}

	var payload CreateTimeBlockPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "invalid request body",
		})
	}

	// Resolve academic year if not provided in payload
	yearID := c.Query("academic_year_id")
	if yearID == "" {
		yearID, err = h.resolveCurrentYear(c, tenantID, schoolID)
		if err != nil {
			return middleware.HTTPError(c, err)
		}
		if yearID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"code":    "NO_ACTIVE_ACADEMIC_YEAR",
				"message": "No current academic year is active.",
			})
		}
	}
	payload.AcademicYearID = yearID

	if err := validateTimeBlockPayload(&payload); err != nil {
		return middleware.HTTPError(c, err)
	}

	block, err := h.svc.CreateBlock(c.Context(), tenantID, schoolID, payload)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(block)
}

// ListBlocks handles GET /api/v1/timetable/block.
func (h *Handler) ListBlocks(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.tmMiddleware(c)
	if err != nil {
		return err
	}

	yearID := c.Query("academic_year_id")
	if yearID == "" {
		yearID, err = h.resolveCurrentYear(c, tenantID, schoolID)
		if err != nil {
			return middleware.HTTPError(c, err)
		}
		if yearID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"code":    "NO_ACTIVE_ACADEMIC_YEAR",
				"message": "No current academic year is active.",
			})
		}
	}

	blocks, err := h.svc.ListBlocks(c.Context(), tenantID, schoolID, yearID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(blocks)
}

// GetBlock handles GET /api/v1/timetable/block/:id.
func (h *Handler) GetBlock(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.tmMiddleware(c)
	if err != nil {
		return err
	}

	id := c.Params("id")
	block, err := h.svc.GetBlock(c.Context(), id, tenantID, schoolID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(block)
}

// UpdateBlock handles PUT /api/v1/timetable/block/:id.
func (h *Handler) UpdateBlock(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.tmMiddleware(c)
	if err != nil {
		return err
	}

	id := c.Params("id")
	var payload UpdateTimeBlockPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "invalid request body",
		})
	}

	// For update, academic_year_id is optional in query but can be in body
	yearID := c.Query("academic_year_id")
	if yearID != "" {
		payload.AcademicYearID = yearID
	}

	if err := validateTimeBlockPayload((*CreateTimeBlockPayload)(&payload)); err != nil {
		return middleware.HTTPError(c, err)
	}

	block, err := h.svc.UpdateBlock(c.Context(), id, tenantID, schoolID, payload)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(block)
}

// DeleteBlock handles DELETE /api/v1/timetable/block/:id.
func (h *Handler) DeleteBlock(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.tmMiddleware(c)
	if err != nil {
		return err
	}

	id := c.Params("id")
	result, err := h.svc.DeleteBlock(c.Context(), id, tenantID, schoolID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// ── Slot Handlers ─────────────────────────────────────────────────────────

// CreateSlot handles POST /api/v1/timetable/slots.
func (h *Handler) CreateSlot(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.tmMiddleware(c)
	if err != nil {
		return err
	}

	var payload SlotPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "invalid request body",
		})
	}

	// Validate required fields
	if payload.BlockID == "" || payload.ClassID == "" || payload.LearningAreaID == "" || payload.TeacherID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "block_id, class_id, learning_area_id, and teacher_id are required",
		})
	}

	// Resolve academic year
	yearID := c.Query("academic_year_id")
	if yearID == "" {
		yearID, err = h.resolveCurrentYear(c, tenantID, schoolID)
		if err != nil {
			return middleware.HTTPError(c, err)
		}
		if yearID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"code":    "NO_ACTIVE_ACADEMIC_YEAR",
				"message": "No current academic year is active.",
			})
		}
	}

	slot, err := h.svc.CreateSlot(c.Context(), tenantID, schoolID, yearID, payload)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(slot)
}

// ListSlots handles GET /api/v1/timetable/slots.
func (h *Handler) ListSlots(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.tmMiddleware(c)
	if err != nil {
		return err
	}

	filter := SlotFilter{
		TenantID:       tenantID,
		SchoolID:       schoolID,
		AcademicYearID: c.Query("academic_year_id"),
		BlockID:        c.Query("block_id"),
		ClassID:        c.Query("class_id"),
		TeacherID:      c.Query("teacher_id"),
		LearningAreaID: c.Query("learning_area_id"),
	}

	slots, err := h.svc.ListSlots(c.Context(), filter)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(slots)
}

// BatchCreateSlots handles POST /api/v1/timetable/slots/batch.
func (h *Handler) BatchCreateSlots(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.tmMiddleware(c)
	if err != nil {
		return err
	}

	var payloads []SlotPayload
	if err := c.BodyParser(&payloads); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "invalid request body",
		})
	}

	if len(payloads) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "at least one slot payload is required",
		})
	}

	// Validate each payload
	for _, p := range payloads {
		if p.BlockID == "" || p.ClassID == "" || p.LearningAreaID == "" || p.TeacherID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"code":    "VALIDATION_ERROR",
				"message": "block_id, class_id, learning_area_id, and teacher_id are required for all payloads",
			})
		}
	}

	// Resolve academic year
	yearID := c.Query("academic_year_id")
	if yearID == "" {
		yearID, err = h.resolveCurrentYear(c, tenantID, schoolID)
		if err != nil {
			return middleware.HTTPError(c, err)
		}
		if yearID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"code":    "NO_ACTIVE_ACADEMIC_YEAR",
				"message": "No current academic year is active.",
			})
		}
	}

	slots, err := h.svc.BatchCreateSlots(c.Context(), tenantID, schoolID, yearID, payloads)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(slots)
}

// UpdateSlot handles PUT /api/v1/timetable/slots/:id.
func (h *Handler) UpdateSlot(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.tmMiddleware(c)
	if err != nil {
		return err
	}

	id := c.Params("id")
	var payload UpdateSlotPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "invalid request body",
		})
	}

	slot, err := h.svc.UpdateSlot(c.Context(), id, tenantID, schoolID, payload)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(slot)
}

// DeleteSlot handles DELETE /api/v1/timetable/slots/:id.
func (h *Handler) DeleteSlot(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.tmMiddleware(c)
	if err != nil {
		return err
	}

	id := c.Params("id")
	err = h.svc.DeleteSlot(c.Context(), id, tenantID, schoolID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(fiber.Map{"deleted": true})
}

// GetTimetable handles GET /api/v1/timetable/timetable.
// Returns a combined view of structures + slots for a given academic year.
func (h *Handler) GetTimetable(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.tmMiddleware(c)
	if err != nil {
		return err
	}

	yearID := c.Query("academic_year_id")
	if yearID == "" {
		yearID, err = h.resolveCurrentYear(c, tenantID, schoolID)
		if err != nil {
			return middleware.HTTPError(c, err)
		}
		if yearID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"code":    "NO_ACTIVE_ACADEMIC_YEAR",
				"message": "No current academic year is active.",
			})
		}
	}

	// Fetch structures and slots
	blocks, err := h.svc.ListBlocks(c.Context(), tenantID, schoolID, yearID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	filter := SlotFilter{
		TenantID:       tenantID,
		SchoolID:       schoolID,
		AcademicYearID: yearID,
	}
	slots, err := h.svc.ListSlots(c.Context(), filter)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(fiber.Map{
		"structures": blocks,
		"slots":      slots,
	})
}
