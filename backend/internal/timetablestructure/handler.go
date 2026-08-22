package timetablestructure

import (
	"context"
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"somotracker/backend/internal/middleware"
)

// AcademicYearTermResolver defines the interface for resolving current academic year and term.
type AcademicYearTermResolver interface {
	GetCurrentYearAndTermID(ctx context.Context, tenantID, schoolID string) (yearID, termID string, err error)
}

// Handler exposes time block HTTP endpoints.
type Handler struct {
	svc              *Service
	academicYearsSvc AcademicYearTermResolver
}

// NewHandler creates a new Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// SetAcademicYearsService sets the academicyears service reference.
func (h *Handler) SetAcademicYearsService(aySvc AcademicYearTermResolver) {
	h.academicYearsSvc = aySvc
}

// RegisterRoutes mounts time block routes on the given router.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	blocks := router.Group("/api/v1/timetable/structure")
	blocks.Get("/", middleware.RequireAuth, h.ListAll)
	blocks.Get("/day/:day", middleware.RequireAuth, h.ListByDay)
	blocks.Post("/", middleware.RequireAuth, h.Create)
	blocks.Post("/batch", middleware.RequireAuth, h.BatchCreate)
	blocks.Post("/replicate", middleware.RequireAuth, h.ReplicateDay)
	blocks.Delete("/by-name", middleware.RequireAuth, h.DeleteByName)
	blocks.Put("/:id", middleware.RequireAuth, h.Update)
	blocks.Delete("/", middleware.RequireAuth, h.Delete)
	blocks.Delete("/day", middleware.RequireAuth, h.DeleteDay)
}

// extractTenantSchool extracts tenant_id and school_id from the request context.
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

// ListAll handles GET /api/v1/timetable/structure.
func (h *Handler) ListAll(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.extractTenantSchool(c)
	if err != nil {
		return err
	}

	academicYearID := c.Query("academic_year_id")
	if academicYearID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "academic_year_id query parameter is required",
		})
	}

	result, err := h.svc.ListAllBlocks(c.Context(), tenantID, schoolID, academicYearID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// ListByDay handles GET /api/v1/timetable/structure/day/:day.
func (h *Handler) ListByDay(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.extractTenantSchool(c)
	if err != nil {
		return err
	}

	academicYearID := c.Query("academic_year_id")
	if academicYearID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "academic_year_id query parameter is required",
		})
	}

	dayStr := c.Params("day")
	day, err := strconv.Atoi(dayStr)
	if err != nil || day < 1 || day > 7 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "day must be an integer between 1 (Monday) and 7 (Sunday)",
		})
	}

	result, err := h.svc.ListBlocksByDay(c.Context(), tenantID, schoolID, academicYearID, day)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// Create handles POST /api/v1/timetable/structure.
func (h *Handler) Create(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.extractTenantSchool(c)
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

	// Resolve current academic year server-side
	academicYearID, _, err := h.academicYearsSvc.GetCurrentYearAndTermID(c.Context(), tenantID, schoolID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}
	if academicYearID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "NO_ACTIVE_ACADEMIC_YEAR",
			"message": "No current academic year is set for this school.",
		})
	}
	// Inject the resolved academic year into the payload for validation/service
	payload.AcademicYearID = academicYearID

	fieldErrors := validateCreatePayload(payload)
	if len(fieldErrors) > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "validation failed",
			"errors":  fieldErrors,
		})
	}

	block, err := h.svc.CreateBlock(c.Context(), tenantID, schoolID, payload)
	if err != nil {
		if errors.Is(err, ErrBlockOverlap) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"code":    "time_block_overlap",
				"message": err.Error(),
			})
		}
		return middleware.HTTPError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(block)
}

// BatchCreate handles POST /api/v1/timetable/structure/batch.
func (h *Handler) BatchCreate(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.extractTenantSchool(c)
	if err != nil {
		return err
	}

	var payload BatchCreateTimeBlockPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "invalid request body",
		})
	}

	if len(payload.Blocks) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "at least one block is required",
		})
	}

	// Resolve current academic year server-side
	academicYearID, _, err := h.academicYearsSvc.GetCurrentYearAndTermID(c.Context(), tenantID, schoolID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}
	if academicYearID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "NO_ACTIVE_ACADEMIC_YEAR",
			"message": "No current academic year is set for this school.",
		})
	}
	// Inject the resolved academic year into all blocks
	for i := range payload.Blocks {
		payload.Blocks[i].AcademicYearID = academicYearID
	}

	result, err := h.svc.BatchCreateBlocks(c.Context(), tenantID, schoolID, payload)
	if err != nil {
		if errors.Is(err, ErrBlockOverlap) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"code":    "time_block_overlap",
				"message": err.Error(),
			})
		}
		return middleware.HTTPError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(result)
}

// ReplicateDay handles POST /api/v1/timetable/structure/replicate.
// Mass-replicates one day's schedule to selected target days.
func (h *Handler) ReplicateDay(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.extractTenantSchool(c)
	if err != nil {
		return err
	}

	var payload ReplicateDayPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "invalid request body",
		})
	}

	if payload.SourceDay < 1 || payload.SourceDay > 7 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "source_day must be between 1 and 7",
		})
	}

	if len(payload.TargetDays) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "at least one target_day is required",
		})
	}

	result, err := h.svc.ReplicateDayBlocks(c.Context(), tenantID, schoolID, payload)
	if err != nil {
		if errors.Is(err, ErrBlockOverlap) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"code":    "time_block_overlap",
				"message": err.Error(),
			})
		}
		return middleware.HTTPError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(result)
}

// Update handles PUT /api/v1/timetable/structure/:id.
// Accepts UpdateTimeBlockPayload which extends the base with propagate
// ("all_days" to cascade) and shift_following options.
// academic_year_id is resolved server-side from the current active academic year.
func (h *Handler) Update(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.extractTenantSchool(c)
	if err != nil {
		return err
	}

	blockID := c.Params("id")
	if blockID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "block id is required",
		})
	}

	var payload UpdateTimeBlockPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "invalid request body",
		})
	}

	// Resolve current academic year server-side
	academicYearID, _, err := h.academicYearsSvc.GetCurrentYearAndTermID(c.Context(), tenantID, schoolID)
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

	// Validate propagate mode
	if payload.Propagate != "" && payload.Propagate != "all_days" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "propagate must be 'all_days' or empty",
		})
	}

	result, err := h.svc.UpdateBlock(c.Context(), blockID, tenantID, schoolID, payload)
	if err != nil {
		if errors.Is(err, ErrBlockOverlap) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"code":    "time_block_overlap",
				"message": err.Error(),
			})
		}
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// Delete handles DELETE /api/v1/timetable/structure/:id.
func (h *Handler) Delete(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.extractTenantSchool(c)
	if err != nil {
		return err
	}

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
			"message": "block id is required",
		})
	}

	result, err := h.svc.DeleteBlock(c.Context(), payload.ID, tenantID, schoolID)
	if err != nil {
		if errors.Is(err, ErrBlockHasLessons) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"code":    "time_block_has_linked_lessons",
				"message": result.Message,
				"details": fiber.Map{
					"linked_lessons": result.LinkedLessons,
				},
			})
		}
		return middleware.HTTPError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(result)
}

// DeleteDay handles DELETE /api/v1/timetable/structure/day.
// academic_year_id is resolved server-side from the current active academic year.
// Day is provided in the request body.
func (h *Handler) DeleteDay(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.extractTenantSchool(c)
	if err != nil {
		return err
	}

	var payload struct {
		Day int `json:"day"`
	}
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "invalid request body",
		})
	}

	// Resolve current academic year server-side
	academicYearID, _, err := h.academicYearsSvc.GetCurrentYearAndTermID(c.Context(), tenantID, schoolID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}
	if academicYearID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "NO_ACTIVE_ACADEMIC_YEAR",
			"message": "No current academic year is set for this school.",
		})
	}

	if payload.Day < 1 || payload.Day > 7 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "day must be an integer between 1 (Monday) and 7 (Sunday)",
		})
	}

	if err := h.svc.DeleteDayBlocks(c.Context(), tenantID, schoolID, academicYearID, payload.Day); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"code":    "ok",
		"message": "Day blocks removed successfully",
	})
}

// DeleteByName handles DELETE /api/v1/timetable/structure/by-name.
// Deletes all blocks with the given period name across all days.
// academic_year_id is resolved server-side from the current active academic year.
func (h *Handler) DeleteByName(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.extractTenantSchool(c)
	if err != nil {
		return err
	}

	var payload DeleteByNamePayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "invalid request body",
		})
	}

	// Resolve current academic year server-side
	academicYearID, _, err := h.academicYearsSvc.GetCurrentYearAndTermID(c.Context(), tenantID, schoolID)
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

	if payload.PeriodName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "period_name is required",
		})
	}

	result, err := h.svc.DeleteBlocksByName(c.Context(), tenantID, schoolID, payload)
	if err != nil {
		if errors.Is(err, ErrBlockHasLessons) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"code":    "time_block_has_linked_lessons",
				"message": result.Message,
				"details": fiber.Map{
					"linked_lessons": result.LinkedLessons,
				},
			})
		}
		return middleware.HTTPError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(result)
}

// validateCreatePayload performs field-level validation for create/update payloads.
// academic_year_id is resolved server-side and not validated here.
func validateCreatePayload(payload CreateTimeBlockPayload) map[string][]string {
	errors := make(map[string][]string)

	if payload.DayOfWeek < 1 || payload.DayOfWeek > 7 {
		errors["day_of_week"] = []string{"Day must be between 1 (Monday) and 7 (Sunday)"}
	}
	if payload.PeriodName == "" {
		errors["period_name"] = []string{"Period name is required"}
	}
	if payload.StartTime == "" {
		errors["start_time"] = []string{"Start time is required"}
	}
	if payload.EndTime == "" {
		errors["end_time"] = []string{"End time is required"}
	}
	if payload.StartTime != "" && payload.EndTime != "" && payload.StartTime >= payload.EndTime {
		if _, exists := errors["end_time"]; !exists {
			errors["end_time"] = []string{"End time must be after start time"}
		}
	}

	return errors
}
