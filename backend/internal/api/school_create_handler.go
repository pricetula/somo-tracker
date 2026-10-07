package api

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	go_uber_zap "go.uber.org/zap"
	"somotracker/backend/internal/services"
	"somotracker/backend/internal/session"
)

// SchoolCreateHandler handles POST /api/school (new endpoint, separate from /school/register).
// Requirements:
// - Person creating must be an admin
// - Set as admin for new school in same tenant
// - Hard-code 3 academic periods (terms) for current year
// - Education system = current school's system OR Kenyan CBE default
// - Load CBE subjects/topics/sub-topics from docs/cbc/ JSON files

type SchoolCreateHandler struct {
	service *services.SchoolService
	session *session.Store
}

func NewSchoolCreateHandler(svc *services.SchoolService) *SchoolCreateHandler {
	return &SchoolCreateHandler{service: svc}
}

// CreateSchool creates a new school with admin verification, academic periods, and CBE curriculum.
// @Summary Create school (admin-only)
// @Description Creates a new school. Requester must be admin. Auto-creates 3 terms + loads CBE curriculum.
// @Tags Schools
// @Accept json
// @Produce json
// @Param body body object{"school_name":"string"} true "School creation payload"
// @Success 201 {object} object
// @Failure 403 {object} object
// @Failure 400 {object} object
// @Router /api/school [post]
func (h *SchoolCreateHandler) CreateSchool(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "user_id not found in session")
	}

	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "tenant_id not found in session")
	}

	var body struct {
		SchoolName string `json:"school_name"`
	}
	if err := c.Bind().Body(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "bad_request: invalid JSON body")
	}
	if body.SchoolName == "" {
		return fiber.NewError(fiber.StatusBadRequest, "bad_request: school_name is required")
	}

	schoolID, err := h.service.CreateSchoolWithSetup(
		c.Context(),
		userID,
		tenantID,
		body.SchoolName,
	)
	if err != nil {
		if strings.Contains(err.Error(), "bad_request:") {
			code := err.Error()[len("bad_request:"):]
			return fiber.NewError(fiber.StatusBadRequest, code)
		}
		if strings.Contains(err.Error(), "forbidden:") {
			return fiber.NewError(fiber.StatusForbidden, err.Error()[len("forbidden:"):])
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	cookieToken := c.Cookies("session_token")
	if cookieToken != "" && h.session != nil {
		if updateErr := h.session.UpdateActiveSchool(c.Context(), cookieToken, schoolID); updateErr != nil {
			go_uber_zap.L().Warn("school_create_handler: session active_school update best-effort failed",
				go_uber_zap.String("error", updateErr.Error()),
			)
		}
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"code":        "school_created",
		"message":     "School created successfully with admin membership, academic periods, and CBE curriculum",
		"school_id":   schoolID,
		"school_name": body.SchoolName,
		"errors":      fiber.Map{},
	})
}
