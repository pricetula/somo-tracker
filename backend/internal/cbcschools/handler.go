package cbcschools

import (
	"github.com/gofiber/fiber/v2"

	"somotracker/backend/internal/middleware"
)

// ─── Request payloads ──────────────────────────────────────────────────────

// CreateSchoolPayload is the request body for POST /api/v1/schools.
type CreateSchoolPayload struct {
	Name string `json:"name"`
}

// UpdateSchoolPayload is the request body for PUT /api/v1/schools/:id.
type UpdateSchoolPayload struct {
	Name           *string `json:"name,omitempty"`
	County         *string `json:"county,omitempty"`
	SubCounty      *string `json:"sub_county,omitempty"`
	Ward           *string `json:"ward,omitempty"`
	KnecSchoolCode *string `json:"knec_school_code,omitempty"`
	NemisCode      *string `json:"nemis_code,omitempty"`
	SchoolType     *string `json:"school_type,omitempty"`
	IsActive       *bool   `json:"is_active,omitempty"`
}

// OnboardingStatusResponse is the response payload for GET /api/v1/school/status.
type OnboardingStatusResponse struct {
	TenantID             string `json:"tenant_id"`
	IsOnboardingComplete bool   `json:"is_onboarding_complete"`
	Steps                struct {
		AcademicCalendarConfigured bool `json:"academic_calendar_configured"`
		CurriculumInitialized      bool `json:"curriculum_initialized"`
		ClassStreamsCreated        bool `json:"class_streams_created"`
		StaffInvited               bool `json:"staff_invited"`
		StudentsEnrolled           bool `json:"students_enrolled"`
	} `json:"steps"`
}

// ─── Handler ───────────────────────────────────────────────────────────────

// Handler exposes school HTTP endpoints.
type Handler struct {
	svc *Service
}

// NewHandler creates a new Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts school routes on the given router.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	schools := router.Group("/api/v1/schools")
	schools.Post("/", middleware.RequireAuth, h.Create)
	schools.Get("/", middleware.RequireAuth, h.List)
	schools.Put("/:id", middleware.RequireAuth, h.Update)
	schools.Delete("/", middleware.RequireAuth, h.Delete)
	schools.Post("/:id/activate", middleware.RequireAuth, h.SetActive)
	schools.Post("/seed-curriculum", middleware.RequireAuth, h.SeedCurriculum)

	// Singular school endpoints (e.g., status)
	school := router.Group("/api/v1/school")
	school.Get("/status", middleware.RequireAuth, h.OnboardingStatus)
}

// ─── Handlers ──────────────────────────────────────────────────────────────

// Create handles POST /api/v1/schools.
// Creates the school, enrolls the creator as SCHOOL_ADMIN, sets it as their
// active school, and updates the somo_school_id cookie.
func (h *Handler) Create(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	userID := c.Locals("user_id").(string)

	var payload CreateSchoolPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "invalid request body",
		})
	}

	if payload.Name == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "name is required",
		})
	}

	schoolID, err := h.svc.CreateSchool(c.Context(), tenantID, payload.Name, "SCHOOL_ADMIN", userID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	// Update the somo_school_id cookie so the frontend immediately reflects
	// the new school as the active school without needing a page refresh.
	c.Cookie(&fiber.Cookie{
		Name:     "somo_school_id",
		Value:    schoolID,
		HTTPOnly: false,
		Secure:   c.Secure(),
		SameSite: "Lax",
		Path:     "/",
		MaxAge:   2592000, // 30 days
	})

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id": schoolID,
	})
}

// List handles GET /api/v1/schools.
func (h *Handler) List(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	userID := c.Locals("user_id").(string)

	schools, err := h.svc.ListSchoolsByTenantID(c.Context(), tenantID, userID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(ListSchoolsResponse{
		Items: schools,
		Total: len(schools),
		Page:  1,
		Limit: len(schools),
	})
}

// Update handles PUT /api/v1/schools/:id.
func (h *Handler) Update(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	schoolID := c.Params("id")
	if schoolID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "school id is required",
		})
	}

	var payload UpdateSchoolPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "invalid request body",
		})
	}

	// Verify the school exists and belongs to this tenant
	if _, err := h.svc.GetSchool(c.Context(), schoolID, tenantID); err != nil {
		return middleware.HTTPError(c, err)
	}

	fields := SchoolUpdateFields{
		ID: schoolID,
	}
	if payload.Name != nil {
		fields.Name = payload.Name
	}
	if payload.County != nil {
		fields.County = payload.County
	}
	if payload.SubCounty != nil {
		fields.SubCounty = payload.SubCounty
	}
	if payload.Ward != nil {
		fields.Ward = payload.Ward
	}
	if payload.KnecSchoolCode != nil {
		fields.KnecSchoolCode = payload.KnecSchoolCode
	}
	if payload.NemisCode != nil {
		fields.NemisCode = payload.NemisCode
	}
	if payload.SchoolType != nil {
		fields.SchoolType = payload.SchoolType
	}
	if payload.IsActive != nil {
		fields.IsActive = payload.IsActive
	}

	if err := h.svc.UpdateSchool(c.Context(), fields); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.SendStatus(fiber.StatusOK)
}

// SetActive handles POST /api/v1/schools/:id/activate.
// Sets the school as the user's active school and updates the somo_school_id cookie.
func (h *Handler) SetActive(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	userID := c.Locals("user_id").(string)
	schoolID := c.Params("id")

	if schoolID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "school id is required",
		})
	}

	if err := h.svc.SetActiveSchool(c.Context(), userID, tenantID, schoolID); err != nil {
		return middleware.HTTPError(c, err)
	}

	// Update the somo_school_id cookie so the frontend immediately reflects
	// the new active school without needing a page refresh.
	c.Cookie(&fiber.Cookie{
		Name:     "somo_school_id",
		Value:    schoolID,
		HTTPOnly: false,
		Secure:   c.Secure(),
		SameSite: "Lax",
		Path:     "/",
		MaxAge:   2592000, // 30 days
	})

	return c.SendStatus(fiber.StatusNoContent)
}

// Sets the school as the user's active school and updates the somo_school_id cookie.
func (h *Handler) SeedCurriculum(c *fiber.Ctx) error {
	tenantID, schoolID, err := getTenantAndSchool(c)
	if err != nil {
		return err
	}

	if err := h.svc.SeedCurriculum(c.Context(), tenantID, schoolID); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.SendStatus(fiber.StatusOK)
}

// OnboardingStatus handles GET /api/v1/school/status.
// Returns the onboarding status for the current tenant.
func (h *Handler) OnboardingStatus(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)

	status, err := h.svc.OnboardingStatus(c.Context(), tenantID)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	response := OnboardingStatusResponse{
		TenantID:             status.TenantID,
		IsOnboardingComplete: status.IsOnboardingComplete,
		Steps: struct {
			AcademicCalendarConfigured bool `json:"academic_calendar_configured"`
			CurriculumInitialized      bool `json:"curriculum_initialized"`
			ClassStreamsCreated        bool `json:"class_streams_created"`
			StaffInvited               bool `json:"staff_invited"`
			StudentsEnrolled           bool `json:"students_enrolled"`
		}{
			AcademicCalendarConfigured: status.AcademicCalendarConfigured,
			CurriculumInitialized:      status.CurriculumInitialized,
			ClassStreamsCreated:        status.ClassStreamsCreated,
			StaffInvited:               status.StaffInvited,
			StudentsEnrolled:           status.StudentsEnrolled,
		},
	}

	return c.JSON(response)
}

// Delete handles DELETE /api/v1/schools/:id.
func (h *Handler) Delete(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)

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
			"message": "school id is required",
		})
	}

	// Verify the school exists and belongs to this tenant
	if _, err := h.svc.GetSchool(c.Context(), payload.ID, tenantID); err != nil {
		return middleware.HTTPError(c, err)
	}

	if err := h.svc.DeleteSchool(c.Context(), payload.ID); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func getTenantAndSchool(c *fiber.Ctx) (string, string, error) {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return "", "", c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"code":    "unauthorized",
			"message": "authentication required",
		})
	}
	schoolID, _ := c.Locals("active_school_id").(string)
	if schoolID == "" {
		return "", "", c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "active school not set",
		})
	}
	return tenantID, schoolID, nil
}
