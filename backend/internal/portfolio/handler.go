package portfolio

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"

	"somotracker/backend/internal/middleware"
)

// ============================================================================
// Handler
// ============================================================================

// Handler exposes portfolio entry HTTP endpoints.
type Handler struct {
	svc *Service
}

// NewHandler creates a new Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts portfolio routes on the given router.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	entries := router.Group("/api/v1/portfolio/entries")
	entries.Post("/", middleware.RequireAuth, h.CreateEntry)
	entries.Get("/", middleware.RequireAuth, h.ListEntries)
	entries.Get("/:id", middleware.RequireAuth, h.GetEntry)
	entries.Put("/:id", middleware.RequireAuth, h.UpdateEntry)
	entries.Delete("/:id", middleware.RequireAuth, h.DeleteEntry)
}

// ============================================================================
// Error response helper — matches the canonical { code, message, errors } shape
// ============================================================================

type errorResponse struct {
	Code    string              `json:"code"`
	Message string              `json:"message"`
	Errors  map[string][]string `json:"errors,omitempty"`
}

func writeError(c *fiber.Ctx, status int, code, message string, fieldErrors map[string][]string) error {
	return c.Status(status).JSON(errorResponse{
		Code:    code,
		Message: message,
		Errors:  fieldErrors,
	})
}

// ============================================================================
// CREATE
// ============================================================================

// CreateEntry handles POST /api/v1/portfolio/entries.
func (h *Handler) CreateEntry(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)

	var body CreatePortfolioEntryPayload
	if err := c.BodyParser(&body); err != nil {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "malformed request body", nil)
	}

	entry, err := h.svc.CreateEntry(c.Context(), tenantID, body)
	if err != nil {
		if errors.Is(err, ErrDuplicateAdvised) {
			return writeError(c, fiber.StatusConflict, "duplicate_entry",
				"An entry for this student, sub-strand, and evidence type already exists.", nil)
		}
		if errors.Is(err, ErrStudentNotFound) {
			return writeError(c, fiber.StatusNotFound, "student_not_found",
				"The specified student was not found.", nil)
		}
		if errors.Is(err, ErrSubStrandNotFound) {
			return writeError(c, fiber.StatusNotFound, "sub_strand_not_found",
				"The specified sub-strand was not found.", nil)
		}
		if errors.Is(err, ErrInvalidEvidenceType) {
			return writeError(c, fiber.StatusBadRequest, "invalid_evidence_type",
				"Invalid evidence type. Allowed: Physical_File_Reference, Digital_Artifact_URL, Video_Recording, Audio_Log, Observation_Checklist", nil)
		}
		return middleware.HTTPError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(CreateEntryResponse{ID: entry.ID})
}

// ============================================================================
// LIST
// ============================================================================

// ListEntries handles GET /api/v1/portfolio/entries.
func (h *Handler) ListEntries(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)

	query := ListEntriesQuery{
		StudentID:   c.Query("student_id"),
		SubStrandID: c.Query("sub_strand_id"),
	}

	entries, err := h.svc.ListEntries(c.Context(), tenantID, query)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(ListEntriesResponse{Data: entries})
}

// ============================================================================
// GET
// ============================================================================

// GetEntry handles GET /api/v1/portfolio/entries/:id.
func (h *Handler) GetEntry(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	id := c.Params("id")

	entry, err := h.svc.GetEntry(c.Context(), id, tenantID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(EntryResponse{Data: *entry})
}

// ============================================================================
// UPDATE
// ============================================================================

// UpdateEntry handles PUT /api/v1/portfolio/entries/:id.
func (h *Handler) UpdateEntry(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	id := c.Params("id")

	var body UpdatePortfolioEntryPayload
	if err := c.BodyParser(&body); err != nil {
		return writeError(c, fiber.StatusBadRequest, "invalid_input", "malformed request body", nil)
	}

	// At least one field must be provided
	if body.StoragePointer == nil && body.DateCollected == nil && body.LinkedResultID == nil {
		return writeError(c, fiber.StatusBadRequest, "invalid_input",
			"at least one of storage_pointer, date_collected, or linked_result_id must be provided", nil)
	}

	if err := h.svc.UpdateEntry(c.Context(), id, tenantID, body); err != nil {
		if errors.Is(err, ErrNotFound) {
			return writeError(c, fiber.StatusNotFound, "entry_not_found",
				"The portfolio entry was not found.", nil)
		}
		return middleware.HTTPError(c, err)
	}

	return c.SendStatus(fiber.StatusOK)
}

// ============================================================================
// DELETE
// ============================================================================

// DeleteEntry handles DELETE /api/v1/portfolio/entries/:id.
func (h *Handler) DeleteEntry(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	id := c.Params("id")

	if err := h.svc.DeleteEntry(c.Context(), id, tenantID); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// ============================================================================
// fx Module
// ============================================================================

// Module is an fx-compatible module for the portfolio domain.
var Module = fx.Module("portfolio",
	fx.Provide(
		fx.Annotate(
			NewRepository,
			fx.As(new(Repository)),
			fx.As(new(StudentResolver)),
			fx.As(new(SubStrandResolver)),
		),
		NewService,
		NewHandler,
	),
)
