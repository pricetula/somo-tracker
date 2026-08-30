package timetable

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"somotracker/backend/internal/academicyears"
	"somotracker/backend/internal/middleware"
)

// Handler exposes timetable HTTP endpoints.
type Handler struct {
	svc              Service
	academicYearsSvc academicyears.AcademicYearTermResolver
}

// NewHandler creates a new timetable Handler.
func NewHandler(svc Service) *Handler {
	return &Handler{
		svc:              svc,
		academicYearsSvc: nil,
	}
}

// RegisterRoutes mounts timetable routes on the given router.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	base := router.Group("/api/v1/timetable")

	// Read-only endpoints
	base.Get("/", middleware.RequireAuth, h.GetTimetable)
	base.Get("/tracks", middleware.RequireAuth, h.ListTracks)
	base.Get("/tracks/:id", middleware.RequireAuth, h.GetTrack)

	// Track operations (ID in body)
	base.Post("/", middleware.RequireAuth, h.CreateTrackWithBlocks)
	base.Put("/", middleware.RequireAuth, h.UpdateTrack)
	base.Delete("/", middleware.RequireAuth, h.BulkDeleteTracks)

	// Block operations (track_id in body)
	base.Post("/blocks", middleware.RequireAuth, h.CreateBlocks)
	base.Put("/blocks", middleware.RequireAuth, h.UpdateBlock)
	base.Delete("/blocks", middleware.RequireAuth, h.BulkDeleteBlocks)

	// Allocation operations (block_id in body)
	base.Post("/allocations", middleware.RequireAuth, h.CreateAllocations)
	base.Put("/allocations", middleware.RequireAuth, h.UpdateAllocation)
	base.Delete("/allocations", middleware.RequireAuth, h.BulkDeleteAllocations)
}

func (h *Handler) GetTrack(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.tmMiddleware(c)
	if err != nil {
		return err
	}
	id := c.Params("id")
	track, err := h.svc.GetTrack(c.UserContext(), id, tenantID, schoolID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}
	return c.JSON(track)
}

func (h *Handler) ListTracks(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.tmMiddleware(c)
	if err != nil {
		return err
	}

	yearID := c.Query("academic_year_id", "")
	tracks, err := h.svc.ListTracks(c.UserContext(), tenantID, schoolID, yearID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}
	return c.JSON(fiber.Map{
		"items": tracks,
		"total": len(tracks),
	})
}

// CreateTrackWithBlocks handles POST /api/v1/timetable (create track + optional initial blocks)
func (h *Handler) SetAcademicYearsService(svc *academicyears.Service) {
	h.academicYearsSvc = svc
}

func (h *Handler) CreateTrackWithBlocks(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.tmMiddleware(c)
	if err != nil {
		return err
	}

	var payload CreateTrackPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "invalid request body",
		})
	}

	// Validate required fields
	if strings.TrimSpace(payload.Name) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "track name is required",
		})
	}

	// Resolve academic year server-side
	yearID, err := h.resolveCurrentYear(c, tenantID, schoolID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}
	if yearID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "NO_ACTIVE_ACADEMIC_YEAR",
			"message": "No current academic year is active.",
		})
	}

	// Resolve academic term server-side
	termID := ""
	if h.academicYearsSvc != nil {
		_, termID, err = h.academicYearsSvc.GetCurrentYearAndTermID(c.UserContext(), tenantID, schoolID)
		if err != nil {
			return middleware.HTTPError(c, err)
		}
	}
	if termID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "NO_ACTIVE_ACADEMIC_TERM",
			"message": "No current academic term is active.",
		})
	}

	// Create track first
	track, err := h.svc.CreateTrack(c.UserContext(), tenantID, schoolID, yearID, termID, payload.Name, payload.Description, payload.IsDefault)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	// If initial blocks provided, create them for this track — replicate each to all 7 days
	if len(payload.InitialBlocks) > 0 {
		for _, blockPayload := range payload.InitialBlocks {
			for day := 1; day <= 7; day++ {
				bp := blockPayload
				bp.TrackID = track.ID
				bp.DayOfWeek = day
				if bp.OrderIndex <= 0 {
					bp.OrderIndex = 1
				}
				if err := validateTimeBlockPayload(&bp); err != nil {
					_, _ = h.svc.DeleteTrack(c.UserContext(), track.ID, tenantID, schoolID)
					return middleware.HTTPError(c, err)
				}
				_, err := h.svc.CreateBlock(c.UserContext(), tenantID, schoolID, bp)
				if err != nil {
					_, _ = h.svc.DeleteTrack(c.UserContext(), track.ID, tenantID, schoolID)
					return middleware.HTTPError(c, err)
				}
			}
		}
		for i := range payload.InitialBlocks {
			payload.InitialBlocks[i].TrackID = track.ID
		}
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"track": track,
		"message": func() string {
			if len(payload.InitialBlocks) > 0 {
				return "track created with initial blocks"
			}
			return "track created successfully"
		}(),
	})
}

// UpdateTrack handles PUT /api/v1/timetable/:id
func (h *Handler) UpdateTrack(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.tmMiddleware(c)
	if err != nil {
		return err
	}

	var payload UpdateTrackPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "invalid request body",
		})
	}

	track, err := h.svc.UpdateTrack(c.UserContext(), payload.ID, tenantID, schoolID, payload)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(fiber.Map{"updated": true, "track": track})
}

// DeleteTrack handles DELETE /api/v1/timetable/:id
func (h *Handler) DeleteTrack(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.tmMiddleware(c)
	if err != nil {
		return err
	}

	id := c.Params("id")
	result, err := h.svc.DeleteTrack(c.UserContext(), id, tenantID, schoolID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// BulkDeleteTracks handles DELETE /api/v1/timetable
func (h *Handler) BulkDeleteTracks(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.tmMiddleware(c)
	if err != nil {
		return err
	}

	var req struct {
		IDs []string `json:"ids"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "request body",
		})
	}

	if len(req.IDs) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "at least one ID is required",
		})
	}

	deleted := 0
	for _, id := range req.IDs {
		_, err := h.svc.DeleteTrack(c.UserContext(), id, tenantID, schoolID)
		if err == nil {
			deleted++
		}
	}

	return c.JSON(fiber.Map{
		"deleted": deleted,
		"total":   len(req.IDs),
	})
}

// CreateBlocks handles POST /api/v1/timetable/blocks (create blocks for a track)
// Each block in the payload is replicated for all 7 days of the week (Monday-Sunday).
func (h *Handler) CreateBlocks(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.tmMiddleware(c)
	if err != nil {
		return err
	}

	var payloads []CreateTimeBlockPayload
	if err := c.BodyParser(&payloads); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "invalid request body",
		})
	}

	if len(payloads) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "at least one block is required",
		})
	}

	for _, blockPayload := range payloads {
		if blockPayload.TrackID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"code":    "VALIDATION_ERROR",
				"message": "track_id is required",
			})
		}
		// Replicate each block for all 7 days of the week
		for day := 1; day <= 7; day++ {
			bp := blockPayload
			bp.DayOfWeek = day
			if bp.OrderIndex <= 0 {
				bp.OrderIndex = 1
			}
			if err := validateTimeBlockPayload(&bp); err != nil {
				return middleware.HTTPError(c, err)
			}
			if _, err := h.svc.CreateBlock(c.UserContext(), tenantID, schoolID, bp); err != nil {
				return middleware.HTTPError(c, err)
			}
		}
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"blocks":          payloads,
		"replicated_days": 7,
	})
}

// UpdateBlock handles PUT /api/v1/timetable/blocks/:id
func (h *Handler) UpdateBlock(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.tmMiddleware(c)
	if err != nil {
		return err
	}

	var payload UpdateTimeBlockPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "invalid request body",
		})
	}

	if payload.ID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "id is required",
		})
	}

	_, err = h.svc.UpdateBlock(c.UserContext(), payload.ID, tenantID, schoolID, payload)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(fiber.Map{"updated": true})
}

// BulkDeleteBlocks handles DELETE /api/v1/timetable/blocks
func (h *Handler) BulkDeleteBlocks(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.tmMiddleware(c)
	if err != nil {
		return err
	}

	var req struct {
		IDs []string `json:"ids"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "request body",
		})
	}

	if len(req.IDs) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "at least one ID is required",
		})
	}

	deleted := 0
	for _, id := range req.IDs {
		_, err := h.svc.DeleteBlock(c.UserContext(), id, tenantID, schoolID)
		if err == nil {
			deleted++
		}
	}

	return c.JSON(fiber.Map{
		"deleted": deleted,
		"total":   len(req.IDs),
	})
}

// CreateAllocations handles POST /api/v1/timetable/allocations (create allocations for a block)
func (h *Handler) CreateAllocations(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.tmMiddleware(c)
	if err != nil {
		return err
	}

	var payloads []CreateAllocationPayload
	if err := c.BodyParser(&payloads); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "invalid request body",
		})
	}

	if len(payloads) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "at least one allocation is required",
		})
	}

	for _, allocationPayload := range payloads {
		if allocationPayload.BlockID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"code":    "VALIDATION_ERROR",
				"message": "block_id is required",
			})
		}
		if _, err := h.svc.CreateAllocation(c.UserContext(), tenantID, schoolID, allocationPayload); err != nil {
			return middleware.HTTPError(c, err)
		}
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"allocations": payloads,
	})
}

// UpdateAllocation handles PUT /api/v1/timetable/allocations/:id
func (h *Handler) UpdateAllocation(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.tmMiddleware(c)
	if err != nil {
		return err
	}

	var payload UpdateAllocationPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "invalid request body",
		})
	}

	if payload.ID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "id is required",
		})
	}

	if _, err := h.svc.UpdateAllocation(c.UserContext(), payload.ID, tenantID, schoolID, payload); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(fiber.Map{"updated": true})
}

// BulkDeleteAllocations handles DELETE /api/v1/timetable/allocations
func (h *Handler) BulkDeleteAllocations(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.tmMiddleware(c)
	if err != nil {
		return err
	}

	var req struct {
		IDs []string `json:"ids"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "request body",
		})
	}

	if len(req.IDs) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "at least one ID is required",
		})
	}

	deleted := 0
	for _, id := range req.IDs {
		err := h.svc.DeleteAllocation(c.UserContext(), id, tenantID, schoolID)
		if err == nil {
			deleted++
		}
	}

	return c.JSON(fiber.Map{
		"deleted": deleted,
		"total":   len(req.IDs),
	})
}

// GetTimetable handles GET /api/v1/timetable (full timetable view)
func (h *Handler) GetTimetable(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.tmMiddleware(c)
	if err != nil {
		return err
	}

	trackID := c.Query("track_id", "")
	classID := c.Query("class_id", "")
	teacherID := c.Query("teacher_id", "")

	blocks, err := h.svc.ListBlocks(c.UserContext(), tenantID, schoolID, "")
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	// Filter blocks by track if provided
	if trackID != "" {
		filtered := make([]TimeBlock, 0, len(blocks))
		for _, b := range blocks {
			if b.TrackID == trackID {
				filtered = append(filtered, b)
			}
		}
		blocks = filtered
	}

	allocations, err := h.svc.ListAllocations(c.UserContext(), AllocationFilter{
		TenantID:  tenantID,
		SchoolID:  schoolID,
		TrackID:   trackID,
		ClassID:   classID,
		TeacherID: teacherID,
	})
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(fiber.Map{
		"blocks":      blocks,
		"allocations": allocations,
	})
}

// tmMiddleware extracts common tenant/school context.
func (h *Handler) tmMiddleware(c *fiber.Ctx) (tenantID, schoolID string, err error) {
	var ok bool
	tenantID, ok = c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return "", "", middleware.ErrUnauthorized
	}
	schoolID, _ = c.Locals("active_school_id").(string)
	if schoolID == "" {
		return "", "", &middleware.FieldError{
			Err:    middleware.ErrInvalidInput,
			Fields: map[string][]string{"active_school_id": {"active school not set"}},
		}
	}
	return tenantID, schoolID, nil
}

// resolveCurrentYear resolves the current academic year ID for the school.
func (h *Handler) resolveCurrentYear(c *fiber.Ctx, tenantID, schoolID string) (string, error) {
	if h.academicYearsSvc == nil {
		return "", nil
	}
	return h.academicYearsSvc.GetCurrentAcademicYearID(c.UserContext(), tenantID, schoolID)
}

// validateTimeBlockPayload validates CreateTimeBlockPayload / UpdateTimeBlockPayload.
func validateTimeBlockPayload(p *CreateTimeBlockPayload) error {
	if p.DayOfWeek < 1 || p.DayOfWeek > 7 {
		return &middleware.FieldError{
			Err:    middleware.ErrInvalidInput,
			Fields: map[string][]string{"day_of_week": {"must be between 1 (Monday) and 7 (Sunday)"}},
		}
	}
	if p.StartTime >= p.EndTime {
		return &middleware.FieldError{
			Err:    middleware.ErrInvalidInput,
			Fields: map[string][]string{"end_time": {"end_time must be after start_time"}},
		}
	}
	return nil
}
