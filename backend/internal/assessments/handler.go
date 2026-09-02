package assessments

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"somotracker/backend/internal/academicyears"
	"somotracker/backend/internal/middleware"
)

// Handler exposes assessment session HTTP endpoints.
type Handler struct {
	svc              *Service
	academicYearsSvc academicyears.AcademicYearTermResolver
}

// NewHandler creates a new Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// SetAcademicYearsService sets the academicyears service reference.
func (h *Handler) SetAcademicYearsService(aySvc academicyears.AcademicYearTermResolver) {
	h.academicYearsSvc = aySvc
}

// RegisterRoutes mounts assessment session routes.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	sessions := router.Group("/api/v1/assessments/sessions")
	sessions.Get("/", middleware.RequireAuth, h.List)
	sessions.Get("/:id", middleware.RequireAuth, h.Get)
	sessions.Post("/", middleware.RequireAuth, h.Create)
	sessions.Put("/:id", middleware.RequireAuth, h.Update)
	sessions.Delete("/:id", middleware.RequireAuth, h.Delete)
	sessions.Post("/:id/submit", middleware.RequireAuth, h.Submit)
	sessions.Post("/:id/approve", middleware.RequireAuth, h.Approve)
	sessions.Post("/:id/reject", middleware.RequireAuth, h.Reject)
	sessions.Post("/:id/scores", middleware.RequireAuth, h.UpsertScores)
	sessions.Get("/:id/scores", middleware.RequireAuth, h.ListScores)
	sessions.Post("/:id/rubric-outcomes", middleware.RequireAuth, h.UpsertRubricOutcomes)
	sessions.Get("/:id/rubric-outcomes", middleware.RequireAuth, h.ListRubricOutcomes)
	profiles := router.Group("/api/v1/assessments/grading-scale-profiles")
	profiles.Get("/", middleware.RequireAuth, h.ListGradingScaleProfiles)
}

func (h *Handler) List(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID, _ := c.Locals("active_school_id").(string)
	if schoolID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "active school not set",
		})
	}
	filter := SessionListFilter{
		TenantID: tenantID, SchoolID: schoolID,
		Page: 1, Limit: 50,
	}
	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			filter.Page = parsed
		}
	}
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 200 {
			filter.Limit = parsed
		}
	}
	if s := c.Query("status"); s != "" {
		filter.Status = s
	}
	if m := c.Query("evaluation_method"); m != "" {
		filter.EvaluationMethod = m
	}
	if classID := c.Query("class_id"); classID != "" {
		filter.ClassID = classID
	}
	result, err := h.svc.List(c.Context(), filter)
	if err != nil {
		return middleware.HTTPError(c, err)
	}
	return c.JSON(result)
}

func (h *Handler) Get(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID, _ := c.Locals("active_school_id").(string)
	if schoolID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "active school not set",
		})
	}
	session, err := h.svc.GetByID(c.Context(), c.Params("id"), tenantID, schoolID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}
	return c.JSON(session)
}

func (h *Handler) Create(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID, _ := c.Locals("active_school_id").(string)
	if schoolID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "active school not set",
		})
	}
	var payload CreateSessionPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "invalid request body",
		})
	}
	userID, _ := c.Locals("user_id").(string)
	session, err := h.svc.Create(c.Context(), payload, tenantID, schoolID, userID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(session)
}

func (h *Handler) Update(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID, _ := c.Locals("active_school_id").(string)
	if schoolID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "active school not set",
		})
	}
	var payload UpdateSessionPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "invalid request body",
		})
	}
	session, err := h.svc.Update(c.Context(), payload, tenantID, schoolID, c.Params("id"))
	if err != nil {
		return middleware.HTTPError(c, err)
	}
	return c.JSON(session)
}

func (h *Handler) Delete(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID, _ := c.Locals("active_school_id").(string)
	if err := h.svc.Delete(c.Context(), c.Params("id"), tenantID, schoolID); err != nil {
		return middleware.HTTPError(c, err)
	}
	return c.JSON(fiber.Map{"code": "success", "message": "session deleted"})
}

func (h *Handler) Submit(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID, _ := c.Locals("active_school_id").(string)
	userID, _ := c.Locals("user_id").(string)
	if err := h.svc.Submit(c.Context(), c.Params("id"), tenantID, schoolID, userID); err != nil {
		return middleware.HTTPError(c, err)
	}
	return c.JSON(fiber.Map{"code": "success", "message": "session submitted for approval"})
}

func (h *Handler) Approve(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID, _ := c.Locals("active_school_id").(string)
	userID, _ := c.Locals("user_id").(string)
	if err := h.svc.Approve(c.Context(), c.Params("id"), tenantID, schoolID, userID); err != nil {
		return middleware.HTTPError(c, err)
	}
	return c.JSON(fiber.Map{"code": "success", "message": "session approved and published"})
}

func (h *Handler) Reject(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID, _ := c.Locals("active_school_id").(string)
	userID, _ := c.Locals("user_id").(string)
	var payload RejectSessionPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "comment is required",
		})
	}
	if err := h.svc.Reject(c.Context(), c.Params("id"), tenantID, schoolID, userID, payload.Comment); err != nil {
		return middleware.HTTPError(c, err)
	}
	return c.JSON(fiber.Map{"code": "success", "message": "session rejected"})
}

func (h *Handler) UpsertScores(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	var payload BatchScorePayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "invalid request body",
		})
	}
	if len(payload.Scores) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "scores array is required",
		})
	}
	count, err := h.svc.UpsertScores(c.Context(), c.Params("id"), tenantID, payload.Scores)
	if err != nil {
		return middleware.HTTPError(c, err)
	}
	return c.JSON(fiber.Map{"code": "success", "message": "scores saved", "count": count})
}

func (h *Handler) ListScores(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	page, limit := 1, 50
	if p := c.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 200 {
			limit = v
		}
	}
	result, err := h.svc.ListScores(c.Context(), c.Params("id"), tenantID, page, limit)
	if err != nil {
		return middleware.HTTPError(c, err)
	}
	return c.JSON(result)
}

// ListGradingScaleProfiles handles GET /api/v1/assessments/grading-scale-profiles.
func (h *Handler) ListGradingScaleProfiles(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID, _ := c.Locals("active_school_id").(string)
	if schoolID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "active school not set",
		})
	}
	profiles, err := h.svc.ListGradingScaleProfiles(c.Context(), tenantID, schoolID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}
	return c.JSON(fiber.Map{"items": profiles})
}

// UpsertRubricOutcomes handles POST /api/v1/assessments/sessions/:id/rubric-outcomes.
func (h *Handler) UpsertRubricOutcomes(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	var payload RubricBatchPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "invalid request body",
		})
	}
	if len(payload.Grading) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "VALIDATION_ERROR",
			"message": "grading array is required",
		})
	}
	count, err := h.svc.UpsertRubricOutcomes(c.Context(), c.Params("id"), tenantID, payload.Grading)
	if err != nil {
		return middleware.HTTPError(c, err)
	}
	return c.JSON(fiber.Map{"code": "success", "message": "rubric grades saved", "count": count})
}

// ListRubricOutcomes handles GET /api/v1/assessments/sessions/:id/rubric-outcomes.
func (h *Handler) ListRubricOutcomes(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	result, err := h.svc.ListRubricOutcomes(c.Context(), c.Params("id"), tenantID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}
	return c.JSON(fiber.Map{"items": result})
}
