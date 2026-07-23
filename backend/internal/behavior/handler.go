package behavior

import (
	"github.com/gofiber/fiber/v2"

	"somotracker/backend/internal/middleware"
)

// Handler exposes behavior HTTP endpoints.
type Handler struct {
	svc *Service
}

// NewHandler creates a new behavior Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts behavior routes on the given router.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	// Behavior categories (admin-managed)
	cat := router.Group("/api/v1/behavior/categories")
	cat.Get("/", middleware.RequireAuth, h.ListCategories)
	cat.Post("/", middleware.RequireAuth, h.CreateCategory)
	cat.Get("/:id", middleware.RequireAuth, h.GetCategory)
	cat.Put("/:id", middleware.RequireAuth, h.UpdateCategory)

	// Behavior notes
	notes := router.Group("/api/v1/behavior/notes")
	notes.Post("/", middleware.RequireAuth, h.CreateNote)
	notes.Get("/", middleware.RequireAuth, h.ListNotes)
	notes.Get("/queue", middleware.RequireAuth, h.PendingQueue)
	notes.Get("/:id", middleware.RequireAuth, h.GetNote)
	notes.Put("/:id", middleware.RequireAuth, h.UpdateNote)
	notes.Post("/:id/review", middleware.RequireAuth, h.ReviewNote)
}

// behMiddleware extracts common tenant/school from context.
func (h *Handler) behMiddleware(c *fiber.Ctx) (tenantID, schoolID string, err error) {
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

// ── Categories ────────────────────────────────────────────────────────────

// ListCategories handles GET /api/v1/behavior/categories.
func (h *Handler) ListCategories(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.behMiddleware(c)
	if err != nil {
		return err
	}

	activeOnly := c.Query("active_only") == "true"
	categories, err := h.svc.ListCategories(c.Context(), tenantID, schoolID, activeOnly)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(fiber.Map{
		"items": categories,
		"total": len(categories),
	})
}

// CreateCategory handles POST /api/v1/behavior/categories.
func (h *Handler) CreateCategory(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.behMiddleware(c)
	if err != nil {
		return err
	}

	var payload CreateCategoryPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "invalid request body",
		})
	}

	category, err := h.svc.CreateCategory(c.Context(), tenantID, schoolID, payload)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(category)
}

// GetCategory handles GET /api/v1/behavior/categories/:id.
func (h *Handler) GetCategory(c *fiber.Ctx) error {
	tenantID, _, err := h.behMiddleware(c)
	if err != nil {
		return err
	}

	id := c.Params("id")
	category, err := h.svc.GetCategory(c.Context(), id, tenantID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(category)
}

// UpdateCategory handles PUT /api/v1/behavior/categories/:id.
func (h *Handler) UpdateCategory(c *fiber.Ctx) error {
	tenantID, _, err := h.behMiddleware(c)
	if err != nil {
		return err
	}

	id := c.Params("id")
	var payload UpdateCategoryPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "invalid request body",
		})
	}

	category, err := h.svc.UpdateCategory(c.Context(), id, tenantID, payload)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(category)
}

// ── Notes ─────────────────────────────────────────────────────────────────

// CreateNote handles POST /api/v1/behavior/notes.
func (h *Handler) CreateNote(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.behMiddleware(c)
	if err != nil {
		return err
	}

	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"code":    "UNAUTHORIZED",
			"message": "user not authenticated",
		})
	}

	var payload CreateNotePayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "invalid request body",
		})
	}

	note, err := h.svc.CreateNote(c.Context(), tenantID, schoolID, payload, userID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(note)
}

// ListNotes handles GET /api/v1/behavior/notes — returns notes authored
// by the authenticated user (teacher view).
func (h *Handler) ListNotes(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.behMiddleware(c)
	if err != nil {
		return err
	}

	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"code":    "UNAUTHORIZED",
			"message": "user not authenticated",
		})
	}

	result, err := h.svc.ListNotesByAuthor(c.Context(), tenantID, schoolID, userID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// PendingQueue handles GET /api/v1/behavior/notes/queue.
func (h *Handler) PendingQueue(c *fiber.Ctx) error {
	tenantID, schoolID, err := h.behMiddleware(c)
	if err != nil {
		return err
	}

	result, err := h.svc.GetPendingQueue(c.Context(), tenantID, schoolID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(result)
}

// GetNote handles GET /api/v1/behavior/notes/:id.
func (h *Handler) GetNote(c *fiber.Ctx) error {
	tenantID, _, err := h.behMiddleware(c)
	if err != nil {
		return err
	}

	id := c.Params("id")
	note, err := h.svc.GetNote(c.Context(), id, tenantID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(note)
}

// UpdateNote handles PUT /api/v1/behavior/notes/:id.
func (h *Handler) UpdateNote(c *fiber.Ctx) error {
	tenantID, _, err := h.behMiddleware(c)
	if err != nil {
		return err
	}

	noteID := c.Params("id")
	if noteID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "note id is required",
		})
	}

	var payload struct {
		Description string `json:"description"`
	}
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "invalid request body",
		})
	}

	if err := h.svc.UpdateNote(c.Context(), noteID, tenantID, payload.Description); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(fiber.Map{
		"message": "Behavior note updated",
	})
}

// ReviewNote handles POST /api/v1/behavior/notes/:id/review.
func (h *Handler) ReviewNote(c *fiber.Ctx) error {
	tenantID, _, err := h.behMiddleware(c)
	if err != nil {
		return err
	}

	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"code":    "UNAUTHORIZED",
			"message": "user not authenticated",
		})
	}

	noteID := c.Params("id")
	var payload ReviewDecisionPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "invalid request body",
		})
	}

	if err := h.svc.ReviewNote(c.Context(), noteID, tenantID, userID, payload); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(fiber.Map{
		"message":  "Behavior note reviewed",
		"decision": payload.Decision,
	})
}
