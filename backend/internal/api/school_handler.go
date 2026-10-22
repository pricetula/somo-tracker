package api

import (
	"somotracker/backend/internal/services"

	"github.com/gofiber/fiber/v3"
	go_uber_zap "go.uber.org/zap"
)

// SchoolHandler handles the school registration endpoint.
type SchoolHandler struct {
	service *services.SchoolService
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
		return WriteError(c, ErrUnauthorized("user_id not found in session"))
	}

	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return WriteError(c, ErrUnauthorized("tenant_id not found in session"))
	}

	// Parse JSON request body for school_name and user_name
	var body struct {
		SchoolName string `json:"school_name"`
		UserName   string `json:"user_name"`
	}
	if err := c.Bind().Body(&body); err != nil {
		return WriteError(c, ErrBadRequest("invalid JSON body", nil))
	}
	schoolName := body.SchoolName
	userName := body.UserName
	if schoolName == "" {
		return WriteError(c, ErrBadRequest("school_name is required", nil))
	}
	if userName == "" {
		return WriteError(c, ErrBadRequest("user_name is required", nil))
	}

	// Perform the atomic transaction with best-effort side effects handled in service
	cookieToken := c.Cookies("session_token")
	schoolID, err := h.service.RegisterSchool(
		c.Context(),
		userID,
		tenantID,
		userName,
		schoolName,
		cookieToken,
	)

	if err != nil {
		msg := err.Error()
		if len(msg) > 11 && msg[:11] == "bad_request:" {
			return WriteError(c, ErrBadRequest(msg[11:], nil))
		}
		go_uber_zap.L().Error("school_register: failed", go_uber_zap.Error(err))
		return WriteError(c, ErrInternal("failed to register school"))
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
		return WriteError(c, ErrUnauthorized("user_id not found in session"))
	}
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return WriteError(c, ErrUnauthorized("tenant_id not found in session"))
	}

	var body struct {
		SchoolID string `json:"school_id"`
	}
	if err := c.Bind().Body(&body); err != nil {
		return WriteError(c, ErrBadRequest("invalid JSON body", nil))
	}
	if body.SchoolID == "" {
		return WriteError(c, ErrBadRequest("school_id is required", nil))
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

	cookieToken := c.Cookies("session_token")
	changed, err := h.service.SetActiveSchool(c.Context(), userID, tenantID, body.SchoolID, cookieToken)
	if err != nil {
		msg := err.Error()
		if len(msg) > 11 && msg[:11] == "bad_request:" {
			return WriteError(c, ErrBadRequest(msg[11:], nil))
		}
		go_uber_zap.L().Error("school_set_active: failed", go_uber_zap.Error(err))
		return WriteError(c, ErrInternal("failed to set active school"))
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
		return WriteError(c, ErrUnauthorized("user_id not found in session"))
	}
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return WriteError(c, ErrUnauthorized("tenant_id not found in session"))
	}

	schools, err := h.service.ListSchools(c.Context(), userID, tenantID)
	if err != nil {
		return WriteError(c, ErrInternal(err.Error()))
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"code":    "schools_listed",
		"message": "Schools listed successfully",
		"schools": schools,
		"errors":  fiber.Map{},
	})
}
