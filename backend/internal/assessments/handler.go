package assessments

import (
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"somotracker/backend/internal/middleware"
)

// Handler exposes assessment HTTP endpoints.
type Handler struct {
	svc *Service
}

// NewHandler creates a new Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts all assessment routes on the given router.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	// Grading Scale Profiles
	profiles := router.Group("/api/v1/grading/profiles")
	profiles.Post("", middleware.RequireAuth, h.CreateScaleProfile)
	profiles.Get("", middleware.RequireAuth, h.ListScaleProfiles)
	profiles.Get("/:id", middleware.RequireAuth, h.GetScaleProfile)
	profiles.Put("/:id/toggle", middleware.RequireAuth, middleware.RequireRole("SCHOOL_ADMIN"), h.ToggleScaleProfileActive)
	profiles.Delete("", middleware.RequireAuth, middleware.RequireRole("SCHOOL_ADMIN"), h.DeleteScaleProfile)
	profiles.Get("/:id/ranges", middleware.RequireAuth, h.GetScaleRanges)
	profiles.Put("/:id/ranges", middleware.RequireAuth, middleware.RequireRole("SCHOOL_ADMIN"), h.ReplaceScaleRanges)

	// Assessment Sessions
	sessions := router.Group("/api/v1/assessments/sessions")
	sessions.Post("/", middleware.RequireAuth, h.CreateSession)
	sessions.Get("/", middleware.RequireAuth, h.ListSessions)
	sessions.Get("/:id", middleware.RequireAuth, h.GetSession)
	sessions.Delete("/", middleware.RequireAuth, h.DeleteSession)

	// Grading Data: returns session + roster + scores/grades in one call
	sessions.Get("/:id/grading-data", middleware.RequireAuth, h.GetGradingData)

	sessions.Post("/:id/submit", middleware.RequireAuth, h.SubmitSession)
	sessions.Post("/:id/approve", middleware.RequireAuth, middleware.RequireRole("SCHOOL_ADMIN"), h.ApproveSession)
	sessions.Post("/:id/reject", middleware.RequireAuth, middleware.RequireRole("SCHOOL_ADMIN"), h.RejectSession)

	// Student Scores (Quantitative)
	sessions.Post("/:id/scores", middleware.RequireAuth, h.BulkUpsertScores)
	sessions.Get("/:id/scores", middleware.RequireAuth, h.GetStudentScores)

	// Student Outcome Grades (Rubric)
	sessions.Post("/:id/grades", middleware.RequireAuth, h.BulkUpsertOutcomeGrades)
	sessions.Get("/:id/grades", middleware.RequireAuth, h.GetOutcomeGrades)

	// Parent View & Report Cards
	parent := router.Group("/api/v1/parent")
	parent.Get("/students/:studentId/assessments", middleware.RequireAuth, h.GetParentAssessments)
	parent.Get("/students/:studentId/report-card", middleware.RequireAuth, h.GetStudentTermGrades)

	// Assessment Weight Configs (system-level, read-only via API)
	wcfg := router.Group("/api/v1/assessments/weight-configs")
	wcfg.Get("/", middleware.RequireAuth, h.ListWeightConfigs)
	wcfg.Get("/:id", middleware.RequireAuth, h.GetWeightConfig)
	wcfg.Post("/", middleware.RequireAuth, middleware.RequireRole("SYSTEM_ADMIN"), h.CreateWeightConfig)
	wcfg.Delete("/", middleware.RequireAuth, middleware.RequireRole("SYSTEM_ADMIN"), h.DeleteWeightConfig)
}

// ── Helpers ──────────────────────────────────────────────────────────────

var (
	ErrTenantMissing = fmt.Errorf("tenant_id not set: %w", middleware.ErrUnauthorized)
	ErrSchoolMissing = fmt.Errorf("active_school_id not set: %w", middleware.ErrInvalidInput)
)

func getTenantAndSchool(c *fiber.Ctx) (string, string, error) {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return "", "", ErrTenantMissing
	}
	schoolID, _ := c.Locals("active_school_id").(string)
	if schoolID == "" {
		return "", "", ErrSchoolMissing
	}
	return tenantID, schoolID, nil
}

func getUserID(c *fiber.Ctx) string {
	if uid, ok := c.Locals("user_id").(string); ok {
		return uid
	}
	return ""
}

// ============================================================================
// GRADING SCALE PROFILES HANDLERS
// ============================================================================

// ---------------------------------------------------------------------------
// CreateScaleProfile — POST /api/v1/grading/profiles
//
// Creates a new grading scale profile together with its percentage-to-level
// ranges in a single atomic transaction. A scale profile is a named collection
// of rules that map percentages to CBC rubric levels (EE/ME/AE/BE).
//
// Ranges are required — every profile must have at least EE, ME, and AE
// ranges defined at creation time. This eliminates the two-step workflow of
// creating a profile and then separately adding ranges.
//
// Once created, the profile name is immutable. To change a scale, create a
// new profile and mark the old one as inactive via ToggleScaleProfileActive.
//
// Request body (SCHOOL_ADMIN only):
//
//	{
//	  "name": "Grade 4 Standard Conversion",
//	  "ranges": [
//	    { "performance_level": "EE", "min_percentage": 80, "max_percentage": 100 },
//	    { "performance_level": "ME", "min_percentage": 60, "max_percentage": 79 },
//	    { "performance_level": "AE", "min_percentage": 40, "max_percentage": 59 },
//	    { "performance_level": "BE", "min_percentage": 0,  "max_percentage": 39 }
//	  ]
//	}
//
// Each range can optionally include "default_percentage_mapping" (e.g. 90
// for EE), which serves as the midpoint used during conversion. If omitted,
// the system calculates from the range bounds.
//
// At minimum EE, ME, and AE ranges are required. BE is recommended.
//
// Response (201):
//
//	{ "id": "uuid-of-new-profile", "range_ids": ["uuid-1", "uuid-2", ...] }
//
// Errors:
//   - 400: name is required, exceeds 255 characters, missing ranges, or ranges fail validation
//   - 401: authentication required
//   - 403: not a SCHOOL_ADMIN
//   - 409: overlapping ranges or duplicate performance levels
//
// ---------------------------------------------------------------------------
func (h *Handler) CreateScaleProfile(c *fiber.Ctx) error {
	tenantID, schoolID, err := getTenantAndSchool(c)
	if err != nil {
		return err
	}

	var payload CreateScaleProfilePayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "invalid request body",
		})
	}

	ranges := make([]CreateScaleRangeParams, len(payload.Ranges))
	for i, r := range payload.Ranges {
		ranges[i] = CreateScaleRangeParams{
			ProfileID:                "", // filled by repo
			PerformanceLevel:         r.PerformanceLevel,
			MinPercentage:            r.MinPercentage,
			MaxPercentage:            r.MaxPercentage,
			DefaultPercentageMapping: r.DefaultPercentageMapping,
		}
	}

	id, rangeIDs, err := h.svc.CreateScaleProfile(c.Context(), CreateScaleProfileParams{
		TenantID: tenantID,
		SchoolID: schoolID,
		Name:     payload.Name,
		Ranges:   ranges,
	})
	if err != nil {
		return middleware.HTTPError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id":        id,
		"range_ids": rangeIDs,
	})
}

// ---------------------------------------------------------------------------
// ListScaleProfiles — GET /api/v1/grading/profiles
//
// Lists all grading scale profiles for the current school. By default returns
// both active and inactive profiles. Use ?active_only=true to filter.
//
// Query parameters:
//   - active_only (bool, optional): when "true", only returns profiles where
//     is_active = true
//
// Response (200):
//
//	{
//	  "items": [
//	    { "id": "...", "name": "Grade 4 Standard Conversion",
//	      "is_active": true, "created_at": "...", "updated_at": "..." }
//	  ]
//	}
//
// Errors:
//   - 401: authentication required
//
// ---------------------------------------------------------------------------
func (h *Handler) ListScaleProfiles(c *fiber.Ctx) error {
	tenantID, schoolID, err := getTenantAndSchool(c)
	if err != nil {
		return err
	}

	activeOnly := c.Query("active_only") == "true"

	profiles, err := h.svc.ListScaleProfiles(c.Context(), tenantID, schoolID, activeOnly)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(ListScaleProfilesResponse{
		Items: profiles,
		Total: len(profiles),
		Page:  1,
		Limit: len(profiles),
	})
}

// ---------------------------------------------------------------------------
// GetScaleProfile — GET /api/v1/grading/profiles/:id
//
// Retrieves a single grading scale profile by ID, including its percentage-to-
// level ranges nested inside the profile object.
//
// Response (200):
//
//	{
//	  "id": "...", "name": "Grade 4 Standard Conversion", "is_active": true,
//	  "ranges": [
//	    { "id": "...", "performance_level": "EE", "min_percentage": 80,
//	      "max_percentage": 100, "default_percentage_mapping": null }
//	  ]
//	}
//
// Errors:
//   - 401: authentication required
//   - 404: profile not found
//
// ---------------------------------------------------------------------------
func (h *Handler) GetScaleProfile(c *fiber.Ctx) error {
	tenantID, schoolID, err := getTenantAndSchool(c)
	if err != nil {
		return err
	}

	id := c.Params("id")
	profile, err := h.svc.GetScaleProfile(c.Context(), id, tenantID, schoolID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}
	return c.JSON(profile)
}

// ---------------------------------------------------------------------------
// GetScaleRanges — GET /api/v1/grading/profiles/:id/ranges
//
// Returns all percentage-to-level ranges for a grading scale profile.
//
// Response (200):
//
//	{
//	  "items": [
//	    { "id": "...", "profile_id": "...", "performance_level": "EE",
//	      "min_percentage": 80, "max_percentage": 100 }
//	  ]
//	}
//
// Errors:
//   - 401: authentication required
//   - 404: profile not found
//
// ---------------------------------------------------------------------------
func (h *Handler) GetScaleRanges(c *fiber.Ctx) error {
	profileID := c.Params("id")
	if profileID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "profile id is required",
		})
	}

	ranges, err := h.svc.GetScaleRanges(c.Context(), profileID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}
	return c.JSON(fiber.Map{"items": ranges})
}

// ---------------------------------------------------------------------------
// ReplaceScaleRanges — PUT /api/v1/grading/profiles/:id/ranges
//
// Replaces all ranges for a grading scale profile (deletes existing ranges,
// then inserts new ones). This is an atomic operation — if any range fails
// validation, none are applied.
//
// Request body (SCHOOL_ADMIN only):
//
//	{
//	  "ranges": [
//	    { "performance_level": "EE", "min_percentage": 80, "max_percentage": 100 },
//	    { "performance_level": "ME", "min_percentage": 60, "max_percentage": 79 },
//	    { "performance_level": "AE", "min_percentage": 40, "max_percentage": 59 },
//	    { "performance_level": "BE", "min_percentage": 0,  "max_percentage": 39 }
//	  ]
//	}
//
// At minimum EE, ME, and AE ranges are required.
//
// Response (200):
//
//	{ "ids": ["uuid-1", "uuid-2", ...] }
//
// Errors:
//   - 400: validation failure (missing ranges, invalid values, overlaps)
//   - 401: authentication required
//   - 403: not a SCHOOL_ADMIN
//   - 404: profile not found
//   - 409: overlapping ranges
//
// ---------------------------------------------------------------------------
func (h *Handler) ReplaceScaleRanges(c *fiber.Ctx) error {
	profileID := c.Params("id")
	if profileID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "profile id is required",
		})
	}

	var payload struct {
		Ranges []ScaleRangePayload `json:"ranges"`
	}
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "invalid request body",
		})
	}

	ranges := make([]CreateScaleRangeParams, len(payload.Ranges))
	for i, r := range payload.Ranges {
		ranges[i] = CreateScaleRangeParams{
			ProfileID:                profileID,
			PerformanceLevel:         r.PerformanceLevel,
			MinPercentage:            r.MinPercentage,
			MaxPercentage:            r.MaxPercentage,
			DefaultPercentageMapping: r.DefaultPercentageMapping,
		}
	}

	ids, err := h.svc.ReplaceScaleRanges(c.Context(), profileID, ranges)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(fiber.Map{"ids": ids})
}

// ---------------------------------------------------------------------------
// ToggleScaleProfileActive — PUT /api/v1/grading/profiles/:id/toggle
//
// Toggles the is_active flag on a scale profile. Use this to deprecate an old
// scale without deleting it (so historical snapshots remain interpretable).
//
// Query parameters:
//   - is_active (bool, optional): defaults to "true". Set to "false" to
//     deactivate.
//
// Example: PUT /api/v1/grading/profiles/uuid-123/toggle?is_active=false
//
// Response (200):
//
//	{ "message": "profile updated" }
//
// Errors:
//   - 401: authentication required
//   - 403: not a SCHOOL_ADMIN
//   - 404: profile not found
//
// ---------------------------------------------------------------------------
func (h *Handler) ToggleScaleProfileActive(c *fiber.Ctx) error {
	tenantID, schoolID, err := getTenantAndSchool(c)
	if err != nil {
		return err
	}

	id := c.Params("id")
	isActive := c.Query("is_active") != "false"

	if err := h.svc.ToggleScaleProfileActive(c.Context(), id, tenantID, schoolID, isActive); err != nil {
		return middleware.HTTPError(c, err)
	}
	return c.JSON(fiber.Map{"message": "profile updated"})
}

// ---------------------------------------------------------------------------
// DeleteScaleProfile — DELETE /api/v1/grading/profiles/:id
//
// Permanently deletes a grading scale profile and all its ranges. This is
// blocked (409 Conflict) if any assessment session references this profile,
// to preserve data integrity. For in-use profiles, use ToggleScaleProfileActive
// to deprecate instead.
//
// Response (200):
//
//	{ "message": "profile deleted" }
//
// Errors:
//   - 401: authentication required
//   - 403: not a SCHOOL_ADMIN
//   - 404: profile not found
//   - 409: profile is referenced by existing sessions (use toggle is_active instead)
//
// ---------------------------------------------------------------------------
func (h *Handler) DeleteScaleProfile(c *fiber.Ctx) error {
	tenantID, schoolID, err := getTenantAndSchool(c)
	if err != nil {
		return err
	}

	var payload struct {
		ID string `json:"id"`
	}
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "invalid request body",
		})
	}
	if payload.ID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "id is required",
		})
	}

	if err := h.svc.DeleteScaleProfile(c.Context(), payload.ID, tenantID, schoolID); err != nil {
		return middleware.HTTPError(c, err)
	}
	return c.JSON(fiber.Map{"message": "profile deleted"})
}

// ============================================================================
// ASSESSMENT SESSION HANDLERS
// ============================================================================

// ---------------------------------------------------------------------------
// CreateSession — POST /api/v1/assessments/sessions
//
// Creates a new assessment session in DRAFT status. Two evaluation methods
// are supported:
//
//   - QUANTITATIVE: total-marks grading (e.g. "Mathematics CAT 1" scored 41/50).
//     Requires max_points and grading_scale_profile_id (the profile whose ranges
//     will convert percentages to CBC levels at approval time).
//
//   - RUBRIC: indicator-level grading (e.g. "Practical Skill Assessment" where
//     the teacher assigns EE/ME/AE/BE per KICD performance indicator directly).
//     Must NOT include max_points or grading_scale_profile_id.
//
// Request body (Teacher):
//
//	{
//	  "class_id": "uuid-of-class",
//	  "learning_area_id": "uuid-of-maths",
//	  "academic_term_id": "uuid-of-term-1",
//	  "academic_year_id": "uuid-of-year",
//	  "name": "Mathematics CAT 1",
//	  "evaluation_method": "QUANTITATIVE",
//	  "max_points": 50,
//	  "grading_scale_profile_id": "uuid-of-grade4-profile",
//	  "scheduled_date": "2026-01-15"
//	}
//
// For RUBRIC sessions, omit max_points and grading_scale_profile_id:
//
//	{
//	  "class_id": "...",
//	  "learning_area_id": "...",
//	  "academic_term_id": "...",
//	  "academic_year_id": "...",
//	  "name": "Practical Skills Assessment",
//	  "evaluation_method": "RUBRIC"
//	}
//
// Response (201):
//
//	{ "id": "uuid-of-new-session" }
//
// Errors:
//   - 400: validation failure (missing fields, invalid evaluation_method)
//   - 401: authentication required
//   - 403: term is finalised (is_final = true)
//
// ---------------------------------------------------------------------------
func (h *Handler) CreateSession(c *fiber.Ctx) error {
	tenantID, schoolID, err := getTenantAndSchool(c)
	if err != nil {
		return err
	}

	var payload CreateSessionPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "invalid request body",
		})
	}

	id, err := h.svc.CreateSession(c.Context(), CreateSessionParams{
		TenantID:              tenantID,
		SchoolID:              schoolID,
		ClassID:               payload.ClassID,
		LearningAreaID:        payload.LearningAreaID,
		AcademicTermID:        payload.AcademicTermID,
		AcademicYearID:        payload.AcademicYearID,
		Name:                  payload.Name,
		EvaluationMethod:      payload.EvaluationMethod,
		MaxPoints:             payload.MaxPoints,
		GradingScaleProfileID: payload.GradingScaleProfileID,
		CreatedBy:             getUserID(c),
		ScheduledDate:         payload.ScheduledDate,
	})
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id": id,
	})
}

// ---------------------------------------------------------------------------
// GetGradingData — GET /api/v1/assessments/sessions/:id/grading-data
//
// Returns session details, class roster (resolved from the session's class_id
// and academic_term_id), and existing scores/grades merged into a single
// response. The frontend no longer needs to call the class roster endpoint
// separately.
//
// Response (200):
//
//	{
//	  "session": { "id": "...", "class_id": "...", "academic_term_id": "...", ... },
//	  "students": [
//	    {
//	      "student_id": "uuid",
//	      "student_name": "Mary Wanjiku",
//	      "admission_number": "CBC/2026/0142",
//	      "gender": "F",
//	      "enrollment_status": "ACTIVE",
//	      "score": { "raw_score": 41, "calculated_percentage": 82.0, ... }
//	    },
//	    {
//	      "student_id": "uuid",
//	      "student_name": "John Kamau",
//	      ...
//	      "score": null  // not yet scored
//	    }
//	  ]
//	}
//
// For RUBRIC sessions, "grades" replaces "score":
//
//	"grades": [{ "performance_indicator_id": "...", "awarded_level": "AE" }]
//
// Errors:
//   - 401: authentication required
//   - 404: session not found
func (h *Handler) GetGradingData(c *fiber.Ctx) error {
	tenantID, schoolID, err := getTenantAndSchool(c)
	if err != nil {
		return err
	}

	id := c.Params("id")
	result, err := h.svc.GetGradingData(c.Context(), id, tenantID, schoolID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}
	return c.JSON(result)
}

// GetSession — GET /api/v1/assessments/sessions/:id
//
// Retrieves a single assessment session by ID, including its current status,
// evaluation method, linked learning area, and audit trail (submitted_by,
// approved_by, rejection_comment).
//
// Example response:
//
//	{
//	  "id": "uuid", "class_id": "uuid", "learning_area_id": "uuid",
//	  "academic_term_id": "uuid", "name": "Mathematics CAT 1",
//	  "evaluation_method": "QUANTITATIVE", "max_points": 50,
//	  "grading_scale_profile_id": "uuid",
//	  "status": "DRAFT", "rejection_comment": null,
//	  "submitted_by": null, "approved_by": null,
//	  "scheduled_date": "2026-01-15",
//	  "created_at": "...", "updated_at": "..."
//	}
//
// Errors:
//   - 401: authentication required
//   - 404: session not found
//
// ---------------------------------------------------------------------------
func (h *Handler) GetSession(c *fiber.Ctx) error {
	tenantID, schoolID, err := getTenantAndSchool(c)
	if err != nil {
		return err
	}

	id := c.Params("id")
	session, err := h.svc.GetSession(c.Context(), id, tenantID, schoolID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}
	return c.JSON(session)
}

// ---------------------------------------------------------------------------
// ListSessions — GET /api/v1/assessments/sessions
//
// Provides paginated listing of assessment sessions with optional filters.
//
// Query parameters (all optional):
//   - class_id:          filter by class
//   - learning_area_id:  filter by subject
//   - academic_term_id:  filter by term
//   - status:            filter by status (DRAFT, PENDING_APPROVAL, PUBLISHED)
//   - evaluation_method: filter by method (QUANTITATIVE, RUBRIC)
//   - search:            fuzzy match on session name
//   - page:              page number (default: 1)
//   - limit:             items per page (default: 50, max: 100)
//
// Example:
//
//	GET /api/v1/assessments/sessions?status=PENDING_APPROVAL&learning_area_id=uuid&page=1&limit=20
//
// Response (200):
//
//	{
//	  "items": [ { session fields... } ],
//	  "total": 42,
//	  "page": 1,
//	  "limit": 20
//	}
//
// Errors:
//   - 401: authentication required
//
// ---------------------------------------------------------------------------
func (h *Handler) ListSessions(c *fiber.Ctx) error {
	tenantID, schoolID, err := getTenantAndSchool(c)
	if err != nil {
		return err
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "50"))

	var classID, learningAreaID, academicTermID, status, evalMethod *string
	if v := c.Query("class_id"); v != "" {
		classID = &v
	}
	if v := c.Query("learning_area_id"); v != "" {
		learningAreaID = &v
	}
	if v := c.Query("academic_term_id"); v != "" {
		academicTermID = &v
	}
	if v := c.Query("status"); v != "" {
		status = &v
	}
	if v := c.Query("evaluation_method"); v != "" {
		evalMethod = &v
	}

	filters := SessionFilters{
		ClassID:          classID,
		LearningAreaID:   learningAreaID,
		AcademicTermID:   academicTermID,
		Status:           status,
		EvaluationMethod: evalMethod,
		Search:           c.Query("search"),
		Page:             page,
		Limit:            limit,
	}

	sessions, total, err := h.svc.ListSessions(c.Context(), tenantID, schoolID, filters)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(ListSessionsResponse{
		Items: sessions,
		Total: total,
		Page:  page,
		Limit: limit,
	})
}

// ---------------------------------------------------------------------------
// DeleteSession — DELETE /api/v1/assessments/sessions/:id
//
// Permanently deletes a DRAFT assessment session and all its scores/grades.
// Sessions in PENDING_APPROVAL or PUBLISHED status cannot be deleted.
//
// Response (204 No Content).
//
// Errors:
//   - 401: authentication required
//   - 404: session not found
//   - 409: session is not in DRAFT status
//
// ---------------------------------------------------------------------------
func (h *Handler) DeleteSession(c *fiber.Ctx) error {
	tenantID, schoolID, err := getTenantAndSchool(c)
	if err != nil {
		return err
	}

	var payload struct {
		ID string `json:"id"`
	}
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "invalid request body",
		})
	}
	if payload.ID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "id is required",
		})
	}

	if err := h.svc.DeleteSession(c.Context(), payload.ID, tenantID, schoolID); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// ---------------------------------------------------------------------------
// SubmitSession — POST /api/v1/assessments/sessions/:id/submit
//
// Transitions a session from DRAFT → PENDING_APPROVAL. This locks the session
// and its scores from further teacher edits. The admin review queue picks it
// up for approval or rejection.
//
// The calling user is recorded as submitted_by. Any existing rejection_comment
// is cleared.
//
// Response (200):
//
//	{ "message": "session submitted for approval" }
//
// Errors:
//   - 400: invalid input
//   - 401: authentication required
//   - 403: term is finalised
//   - 404: session not found
//   - 409: invalid state transition (session is not in DRAFT)
//
// ---------------------------------------------------------------------------
func (h *Handler) SubmitSession(c *fiber.Ctx) error {
	tenantID, schoolID, err := getTenantAndSchool(c)
	if err != nil {
		return err
	}

	id := c.Params("id")
	userID := getUserID(c)

	if err := h.svc.SubmitSession(c.Context(), id, tenantID, schoolID, userID); err != nil {
		return middleware.HTTPError(c, err)
	}
	return c.JSON(fiber.Map{"message": "session submitted for approval"})
}

// ---------------------------------------------------------------------------
// ApproveSession — POST /api/v1/assessments/sessions/:id/approve
//
// Transitions a session from PENDING_APPROVAL → PUBLISHED (School Admin only).
//
// Side effects for QUANTITATIVE sessions:
//   - Reads the linked grading_scale_profile ranges
//   - Computes each student's calculated_percentage → final_performance_level
//   - Writes (snapshots) the CBC level (EE/ME/AE/BE) to the score row
//
// The calling admin is recorded as approved_by. Once published, the session
// is visible to parents in their portal.
//
// Example flow:
//
//	Student Kamau scored 41/50 → 82% → resolves to "EE" via scale profile
//	→ final_performance_level = "EE" is snapshotted permanently
//
// Response (200):
//
//	{ "message": "session approved and published" }
//
// Errors:
//   - 401: authentication required
//   - 403: not a SCHOOL_ADMIN, or term is finalised
//   - 404: session not found
//   - 409: invalid state transition (session is not PENDING_APPROVAL)
//
// ---------------------------------------------------------------------------
func (h *Handler) ApproveSession(c *fiber.Ctx) error {
	tenantID, schoolID, err := getTenantAndSchool(c)
	if err != nil {
		return err
	}

	id := c.Params("id")
	userID := getUserID(c)

	if err := h.svc.ApproveSession(c.Context(), id, tenantID, schoolID, userID); err != nil {
		return middleware.HTTPError(c, err)
	}
	return c.JSON(fiber.Map{"message": "session approved and published"})
}

// ---------------------------------------------------------------------------
// RejectSession — POST /api/v1/assessments/sessions/:id/reject
//
// Transitions a session from PENDING_APPROVAL back to DRAFT (School Admin
// only). Requires a rejection_comment explaining why. The session is unlocked
// and the teacher can edit scores and re-submit.
//
// Request body (SCHOOL_ADMIN only):
//
//	{ "rejection_comment": "Please review Kamau's score — it looks like a typo." }
//
// Response (200):
//
//	{ "message": "session rejected and returned to draft" }
//
// Errors:
//   - 400: rejection_comment is required
//   - 401: authentication required
//   - 403: not a SCHOOL_ADMIN, or term is finalised
//   - 404: session not found
//   - 409: invalid state transition (session is not PENDING_APPROVAL)
//
// ---------------------------------------------------------------------------
func (h *Handler) RejectSession(c *fiber.Ctx) error {
	tenantID, schoolID, err := getTenantAndSchool(c)
	if err != nil {
		return err
	}

	id := c.Params("id")

	var payload RejectSessionPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "invalid request body",
		})
	}

	if err := h.svc.RejectSession(c.Context(), id, tenantID, schoolID, payload.RejectionComment); err != nil {
		return middleware.HTTPError(c, err)
	}
	return c.JSON(fiber.Map{"message": "session rejected and returned to draft"})
}

// ============================================================================
// STUDENT SCORES HANDLERS (Quantitative)
// ============================================================================

// ---------------------------------------------------------------------------
// BulkUpsertScores — POST /api/v1/assessments/sessions/:id/scores
//
// Bulk-upserts quantitative raw scores for all students in a session. Only
// applicable for QUANTITATIVE sessions. Each score is computed as:
//
//	calculated_percentage = (raw_score / max_points) * 100
//
// The final_performance_level is NOT computed here — it's snapshotted later
// at approval time by ApproveSession.
//
// Scores can only be upserted while the session is in DRAFT status. Once
// submitted for approval, scores are locked. Students with enrollment_status
// of "ABSENT" or "EXEMPT" cannot receive scores (No Grade Ghosting).
//
// Request body (Teacher):
//
//	{
//	  "scores": [
//	    { "student_id": "uuid-kamau",  "raw_score": 41 },
//	    { "student_id": "uuid-achesa", "raw_score": 38 },
//	    { "student_id": "uuid-juma",   "raw_score": 25 }
//	  ]
//	}
//
// Omit raw_score (or set to null) for students who were not graded. The
// upsert is idempotent — re-sending the same data overwrites previous values.
//
// Response (200):
//
//	{ "message": "scores saved" }
//
// Errors:
//   - 400: missing student_id, negative raw_score, or student is ABSENT/EXEMPT
//   - 401: authentication required
//   - 403: term is finalised
//   - 404: session not found
//   - 409: session is not in DRAFT status
//
// ---------------------------------------------------------------------------
func (h *Handler) BulkUpsertScores(c *fiber.Ctx) error {
	tenantID, _, err := getTenantAndSchool(c)
	if err != nil {
		return err
	}

	sessionID := c.Params("id")

	var payload BulkUpsertScoresPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "invalid request body",
		})
	}

	params := make([]UpsertScoreParams, len(payload.Scores))
	for i, s := range payload.Scores {
		params[i] = UpsertScoreParams{
			TenantID:         tenantID,
			SessionID:        sessionID,
			StudentID:        s.StudentID,
			RawScore:         s.RawScore,
			EnrollmentStatus: "ACTIVE",
		}
	}

	if err := h.svc.BulkUpsertScores(c.Context(), params); err != nil {
		return middleware.HTTPError(c, err)
	}
	return c.JSON(fiber.Map{"message": "scores saved"})
}

// ---------------------------------------------------------------------------
// GetStudentScores — GET /api/v1/assessments/sessions/:id/scores
//
// Returns all quantitative scores for a session, including the calculated
// percentage and final snapshotted performance level (if published).
//
// Response (200):
//
//	{
//	  "items": [
//	    { "id": "...", "session_id": "...", "student_id": "...",
//	      "raw_score": 41, "calculated_percentage": 82.0,
//	      "final_performance_level": "EE", "enrollment_status": "ACTIVE" }
//	  ]
//	}
//
// Before approval, final_performance_level will be null.
//
// Errors:
//   - 401: authentication required
//   - 404: session not found
//
// ---------------------------------------------------------------------------
func (h *Handler) GetStudentScores(c *fiber.Ctx) error {
	tenantID, schoolID, err := getTenantAndSchool(c)
	if err != nil {
		return err
	}

	sessionID := c.Params("id")
	scores, err := h.svc.GetStudentScores(c.Context(), sessionID, tenantID, schoolID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	if scores == nil {
		scores = []StudentScore{}
	}
	return c.JSON(StudentScoresResponse{Items: scores})
}

// ============================================================================
// STUDENT OUTCOME GRADES HANDLERS (Rubric)
// ============================================================================

// ---------------------------------------------------------------------------
// BulkUpsertOutcomeGrades — POST /api/v1/assessments/sessions/:id/grades
//
// Bulk-upserts rubric outcome grades for a RUBRIC session. Each grade maps a
// student to a KICD performance indicator with a directly assigned CBC level
// (EE, ME, AE, BE). No raw scores or percentages are involved — the teacher
// evaluates each indicator holistically.
//
// Only applicable for RUBRIC sessions. Grades can only be modified while the
// session is in DRAFT status.
//
// Request body (Teacher):
//
//	{
//	  "grades": [
//	    { "student_id": "uuid-kamau",  "performance_indicator_id": "ind-001",
//	      "awarded_level": "ME" },
//	    { "student_id": "uuid-achesa", "performance_indicator_id": "ind-001",
//	      "awarded_level": "AE" },
//	    { "student_id": "uuid-kamau",  "performance_indicator_id": "ind-002",
//	      "awarded_level": "EE" }
//	  ]
//	}
//
// Each combination of (session_id, student_id, performance_indicator_id) is
// unique — re-sending the same combination overwrites the awarded_level.
//
// Response (200):
//
//	{ "message": "grades saved" }
//
// Errors:
//   - 400: missing required fields or invalid performance level
//   - 401: authentication required
//   - 403: term is finalised
//   - 404: session not found
//   - 409: session is not in DRAFT status
//
// ---------------------------------------------------------------------------
func (h *Handler) BulkUpsertOutcomeGrades(c *fiber.Ctx) error {
	tenantID, _, err := getTenantAndSchool(c)
	if err != nil {
		return err
	}

	sessionID := c.Params("id")

	var payload BulkUpsertOutcomeGradesPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "invalid request body",
		})
	}

	params := make([]UpsertOutcomeGradeParams, len(payload.Grades))
	for i, g := range payload.Grades {
		params[i] = UpsertOutcomeGradeParams{
			TenantID:               tenantID,
			SessionID:              sessionID,
			StudentID:              g.StudentID,
			PerformanceIndicatorID: g.PerformanceIndicatorID,
			AwardedLevel:           g.AwardedLevel,
		}
	}

	if err := h.svc.BulkUpsertOutcomeGrades(c.Context(), params); err != nil {
		return middleware.HTTPError(c, err)
	}
	return c.JSON(fiber.Map{"message": "grades saved"})
}

// ---------------------------------------------------------------------------
// GetOutcomeGrades — GET /api/v1/assessments/sessions/:id/grades
//
// Returns all rubric outcome grades for a RUBRIC session, showing each
// student's awarded level per performance indicator.
//
// Response (200):
//
//	{
//	  "items": [
//	    { "id": "...", "session_id": "...", "student_id": "...",
//	      "performance_indicator_id": "ind-001", "awarded_level": "ME" }
//	  ]
//	}
//
// Errors:
//   - 401: authentication required
//   - 404: session not found
//
// ---------------------------------------------------------------------------
func (h *Handler) GetOutcomeGrades(c *fiber.Ctx) error {
	tenantID, schoolID, err := getTenantAndSchool(c)
	if err != nil {
		return err
	}

	sessionID := c.Params("id")
	grades, err := h.svc.GetOutcomeGrades(c.Context(), sessionID, tenantID, schoolID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	if grades == nil {
		grades = []OutcomeGrade{}
	}
	return c.JSON(OutcomeGradesResponse{Items: grades})
}

// ============================================================================
// PARENT VIEW HANDLERS
// ============================================================================

// ---------------------------------------------------------------------------
// GetParentAssessments — GET /api/v1/parent/students/:studentId/assessments
//
// Returns all PUBLISHED assessment results for a specific student within a
// given academic term. This is the real-time parent portal stream — as soon
// as any session transitions to PUBLISHED, its results appear here.
//
// For QUANTITATIVE sessions, includes raw_score, max_points, and the
// snapshotted performance_level.
// For RUBRIC sessions, includes the per-indicator outcome_grades array.
//
// Query parameters:
//   - academic_term_id (required): the term to retrieve results for
//
// Example:
//
//	GET /api/v1/parent/students/student-uuid/assessments?academic_term_id=term-uuid
//
// Response (200):
//
//	{
//	  "items": [
//	    {
//	      "session_id": "...", "session_name": "Mathematics CAT 1",
//	      "evaluation_method": "QUANTITATIVE",
//	      "scheduled_date": "2026-01-15",
//	      "raw_score": 41, "max_points": 50,
//	      "performance_level": "EE"
//	    },
//	    {
//	      "session_id": "...", "session_name": "Practical Skills",
//	      "evaluation_method": "RUBRIC",
//	      "scheduled_date": "2026-02-01",
//	      "outcome_grades": [
//	        { "performance_indicator_id": "ind-001", "awarded_level": "ME" },
//	        { "performance_indicator_id": "ind-002", "awarded_level": "AE" }
//	      ]
//	    }
//	  ]
//	}
//
// Errors:
//   - 400: academic_term_id is required
//   - 401: authentication required
//
// ---------------------------------------------------------------------------
func (h *Handler) GetParentAssessments(c *fiber.Ctx) error {
	tenantID, schoolID, err := getTenantAndSchool(c)
	if err != nil {
		return err
	}

	studentID := c.Params("studentId")
	termID := c.Query("academic_term_id")

	if termID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "academic_term_id query parameter is required",
		})
	}

	assessments, err := h.svc.GetParentAssessments(c.Context(), tenantID, schoolID, studentID, termID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(ParentTermAssessmentsResponse{Items: assessments})
}

// ---------------------------------------------------------------------------
// GetStudentTermGrades — GET /api/v1/parent/students/:studentId/report-card
//
// Compiles the end-of-term report card using the "Last One" Chronological
// Mode aggregator. For each learning area (subject), the system:
//
//  1. Collects: All PUBLISHED performance levels from both quantitative
//     assessments (snapshotted) and rubric outcome grades.
//
//  2. Counts: Groups by performance level and finds the mode (most frequent).
//
//  3. Tie-Breaks: If two+ levels have equal frequency, picks the one from
//     the chronologically latest assessment (by session created_at).
//
// Example: Kamau's Mathematics term results:
//
//   - CAT 1 (Jan 15): EE
//
//   - Practical 1 (Feb 1): ME
//
//   - Practical 2 (Feb 1): AE
//
//   - CAT 2 (Feb 20): ME
//
//   - Project (Mar 10): EE
//
//     Counts: EE=2, ME=2, AE=1 → Tie between EE & ME
//     Latest among tied: EE (Mar 10) vs ME (Feb 20) → EE wins
//     Final grade: EE
//
// Query parameters:
//   - academic_term_id (required): the term to compile for
//
// Example:
//
//	GET /api/v1/parent/students/student-uuid/report-card?academic_term_id=term-uuid
//
// Response (200):
//
//	{
//	  "items": [
//	    { "learning_area_id": "...", "learning_area_name": "Mathematics",
//	      "learning_area_code": "MATH", "final_level": "EE",
//	      "assessment_count": 5 }
//	  ]
//	}
//
// Errors:
//   - 400: academic_term_id is required
//   - 401: authentication required
//
// ---------------------------------------------------------------------------
func (h *Handler) GetStudentTermGrades(c *fiber.Ctx) error {
	tenantID, schoolID, err := getTenantAndSchool(c)
	if err != nil {
		return err
	}

	studentID := c.Params("studentId")
	termID := c.Query("academic_term_id")

	if termID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "academic_term_id query parameter is required",
		})
	}

	grades, err := h.svc.GetStudentTermGrades(c.Context(), tenantID, schoolID, studentID, termID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(StudentTermGradesResponse{Items: grades})
}

// ═══════════════════════════════════════════════════════════════════════════
// ASSESSMENT WEIGHT CONFIGS
// ═══════════════════════════════════════════════════════════════════════════

// CreateWeightConfig handles POST /api/v1/assessments/weight-configs.
//
// Creates a new KNEC weight configuration entry. Weight configs are
// system-level (not tenant-scoped) and define the national weighting
// formulas that specify how different assessment types contribute to
// the final target exam placement score.
//
// Only SYSTEM_ADMIN users can create weight configs.
//
// Request body (SYSTEM_ADMIN only):
//
//	{
//	  "grade_level": "GRADE_4",
//	  "assessment_type_code": "KNEC_SBA_Project",
//	  "target_exam": "KPSEA",
//	  "weight_percent": 20.0,
//	  "effective_from": 2026,
//	  "notes": "Grade 4 project component contributes 20% to KPSEA placement"
//	}
//
// The combination (grade_level, assessment_type_code, target_exam,
// effective_from) must be unique.
//
// Response (201):
//
//	{ "id": "uuid-of-new-weight-config" }
//
// Errors:
//   - 400: validation failure (missing fields, invalid percentages, etc.)
//   - 401: authentication required
//   - 403: not a SYSTEM_ADMIN
//   - 409: duplicate config (UNIQUE constraint violation)
//
// --------------------------------------------------------------------------
func (h *Handler) CreateWeightConfig(c *fiber.Ctx) error {
	var payload CreateWeightConfigPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "invalid request body",
		})
	}

	id, err := h.svc.CreateWeightConfig(c.Context(), CreateWeightConfigParams(payload))
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id": id,
	})
}

// ListWeightConfigs handles GET /api/v1/assessments/weight-configs.
func (h *Handler) ListWeightConfigs(c *fiber.Ctx) error {
	var filter AssessmentWeightConfigFilter

	if gl := c.Query("grade_level"); gl != "" {
		filter.GradeLevel = &gl
	}
	if te := c.Query("target_exam"); te != "" {
		filter.TargetExam = &te
	}
	if ef := c.Query("effective_from"); ef != "" {
		if v, err := strconv.Atoi(ef); err == nil {
			filter.EffectiveFrom = &v
		}
	}

	result, err := h.svc.ListWeightConfigs(c.Context(), filter)
	if err != nil {
		return middleware.HTTPError(c, err)
	}
	return c.JSON(result)
}

// DeleteWeightConfig handles DELETE /api/v1/assessments/weight-configs/:id.
// SYSTEM_ADMIN only.
func (h *Handler) DeleteWeightConfig(c *fiber.Ctx) error {
	var payload struct {
		ID string `json:"id"`
	}
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "invalid request body",
		})
	}
	if payload.ID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "id is required",
		})
	}

	if err := h.svc.DeleteWeightConfig(c.Context(), payload.ID); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// GetWeightConfig handles GET /api/v1/assessments/weight-configs/:id.
func (h *Handler) GetWeightConfig(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "id is required",
		})
	}

	result, err := h.svc.GetWeightConfigByID(c.Context(), id)
	if err != nil {
		return middleware.HTTPError(c, err)
	}
	return c.JSON(result)
}
