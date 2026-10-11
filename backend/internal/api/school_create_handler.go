package api

import (
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
		return WriteError(c, ErrUnauthorized("user_id not found in session"))
	}

	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return WriteError(c, ErrUnauthorized("tenant_id not found in session"))
	}

	var body struct {
		SchoolName string `json:"school_name"`
	}
	if err := c.Bind().Body(&body); err != nil {
		return WriteError(c, ErrBadRequest("invalid JSON body", nil))
	}
	if body.SchoolName == "" {
		return WriteError(c, ErrBadRequest("school_name is required", nil))
	}

	schoolID, err := h.service.CreateSchoolWithSetup(
		c.Context(),
		userID,
		tenantID,
		body.SchoolName,
	)
	if err != nil {
		msg := err.Error()
		if len(msg) > 11 && msg[:11] == "bad_request:" {
			return WriteError(c, ErrBadRequest(msg[11:], nil))
		}
		if len(msg) > 10 && msg[:10] == "forbidden:" {
			return WriteError(c, ErrForbidden("forbidden", msg[10:]))
		}
		go_uber_zap.L().Error("school_create: create failed", go_uber_zap.Error(err))
		return WriteError(c, ErrInternal("failed to create school"))
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
