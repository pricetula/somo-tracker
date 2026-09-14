package api

import (
	"strings"

	"somotracker/backend/internal/services"
	"somotracker/backend/internal/session"

	"github.com/gofiber/fiber/v3"
	go_uber_zap "go.uber.org/zap"
)

// SchoolHandler handles the school registration endpoint.
type SchoolHandler struct {
	service *services.SchoolService
	session *session.Store
}

func NewSchoolHandler(svc *services.SchoolService) *SchoolHandler {
	return &SchoolHandler{service: svc}
}

// RegisterSchool creates a new school for the authenticated user and assigns them as ADMIN.
// This is an atomic transaction that:
// 1. Updates the user's full_name
// 2. Creates a new school record
// 3. Creates a school membership with role=ADMIN
// @Summary Register a new school
// @Description Creates a new school for the authenticated user and assigns them as ADMIN. Updates user's full_name atomically.
// @Tags Schools
// @Accept json
// @Produce json
// @Param body body object{"school_name":"string","user_name":"string"} true "School registration payload"
// @Success 201 {object} object
// @Failure 400 {object} object
// @Router /school/register [post]
func (h *SchoolHandler) RegisterSchool(c fiber.Ctx) error {
	// Get user ID and tenant ID from session locals
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "user_id not found in session")
	}

	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "tenant_id not found in session")
	}

	// Parse JSON request body for school_name and user_name
	var body struct {
		SchoolName string `json:"school_name"`
		UserName   string `json:"user_name"`
	}
	if err := c.Bind().Body(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "bad_request: invalid JSON body")
	}
	schoolName := body.SchoolName
	userName := body.UserName
	if schoolName == "" {
		return fiber.NewError(fiber.StatusBadRequest, "bad_request: school_name is required")
	}
	if userName == "" {
		return fiber.NewError(fiber.StatusBadRequest, "bad_request: user_name is required")
	}

	// Perform the atomic transaction
	schoolID, err := h.service.RegisterSchool(
		c.Context(),
		userID,
		tenantID,
		userName,
		schoolName,
	)

	if err != nil {
		if strings.Contains(err.Error(), "bad_request:") {
			code := err.Error()[len("bad_request:"):]
			return fiber.NewError(fiber.StatusBadRequest, code)
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	// Update active school in Redis session cache so authorized clients
	// can read tenant + active_school_id without PostgreSQL round-trips.
	cookieToken := c.Cookies("session_token")
	if cookieToken != "" && h.session != nil {
		if updateErr := h.session.UpdateActiveSchool(c.Context(), cookieToken, schoolID); updateErr != nil {
			go_uber_zap.L().Warn("school_handler: session active_school update best-effort failed",
				go_uber_zap.String("error", updateErr.Error()),
			)
		}
	}

	// Return the newly created school details
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"code":        "school_registered",
		"message":     "School registered successfully",
		"school_id":   schoolID,
		"school_name": schoolName,
		"user_name":   userName,
		"errors":      fiber.Map{},
	})
}

// SetActiveSchool updates the user's active school membership.
// @Summary Set active school
// @Description Updates the user's active school in DB and session.
// @Tags Schools
// @Accept json
// @Produce json
// @Param body body object{"school_id":"string"} true "Active school payload"
// @Success 200 {object} object
// @Failure 400 {object} object
// @Router /school/set-active [post]
func (h *SchoolHandler) SetActiveSchool(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "user_id not found in session")
	}
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "tenant_id not found in session")
	}

	var body struct {
		SchoolID string `json:"school_id"`
	}
	if err := c.Bind().Body(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "bad_request: invalid JSON body")
	}
	if body.SchoolID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "bad_request: school_id is required")
	}

	currentActive := ""
	if v, ok := c.Locals("active_school_id").(string); ok {
		currentActive = v
	}
	if body.SchoolID == currentActive {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"code":      "school_set_active",
			"message":   "Active school unchanged",
			"school_id": body.SchoolID,
			"changed":   false,
			"errors":    fiber.Map{},
		})
	}

	changed, err := h.service.SetActiveSchool(c.Context(), userID, tenantID, body.SchoolID)
	if err != nil {
		if strings.Contains(err.Error(), "bad_request:") {
			code := err.Error()[len("bad_request:"):]
			return fiber.NewError(fiber.StatusBadRequest, code)
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	if changed {
		cookieToken := c.Cookies("session_token")
		if cookieToken != "" && h.session != nil {
			if updateErr := h.session.UpdateActiveSchool(c.Context(), cookieToken, body.SchoolID); updateErr != nil {
				go_uber_zap.L().Warn("school_handler: session active_school update best-effort failed",
					go_uber_zap.String("error", updateErr.Error()),
				)
			}
		}
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"code":      "school_set_active",
		"message":   "Active school updated",
		"school_id": body.SchoolID,
		"changed":   changed,
		"errors":    fiber.Map{},
	})
}

// ListSchools returns the list of schools the authenticated user belongs to.
func (h *SchoolHandler) ListSchools(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "user_id not found in session")
	}
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "tenant_id not found in session")
	}

	schools, err := h.service.ListSchools(c.Context(), userID, tenantID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"code":    "schools_listed",
		"message": "Schools listed successfully",
		"schools": schools,
		"errors":  fiber.Map{},
	})
}
