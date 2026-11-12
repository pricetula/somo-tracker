package handler

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"somotracker/internal/platform/apperr"
	"somotracker/internal/platform/authz"
	"somotracker/internal/students/service"
)

// Handler is HTTP handler for students domain (thin, BE-006).
type Handler struct {
	svc *service.Service
}

// New creates handler.
func New(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

// DTOs are defined per endpoint, not domain structs (BE-007).

// ListStudentsRequest represents query parameters for listing students.
type ListStudentsRequest struct {
	Limit     int32  `query:"limit"`
	Cursor    string `query:"cursor"`
	Sort      string `query:"sort"`
	GuardianID string `query:"guardianId"`
}

// ListStudentsResponse is the paginated list response.
type ListStudentsResponse struct {
	Items      []service.StudentDTO `json:"items"`
	NextCursor *string              `json:"nextCursor,omitempty"`
}

// CreateStudentRequest is the request body for creating a student.
type CreateStudentRequest struct {
	AdmissionNumber string         `json:"admission_number" validate:"required"`
	NationalNumber  *string        `json:"national_number,omitempty"`
	FullName        string         `json:"full_name" validate:"required"`
	DateOfBirth     string         `json:"date_of_birth" validate:"required"`
	Gender          string         `json:"gender" validate:"required"`
	Metadata        json.RawMessage `json:"metadata"`
}

// UpdateStudentRequest is the request body for updating a student (full body).
type UpdateStudentRequest struct {
	AdmissionNumber string         `json:"admission_number" validate:"required"`
	NationalNumber  *string        `json:"national_number,omitempty"`
	FullName        string         `json:"full_name" validate:"required"`
	DateOfBirth     string         `json:"date_of_birth" validate:"required"`
	Gender          string         `json:"gender" validate:"required"`
	Metadata        json.RawMessage `json:"metadata"`
}

// GuardianLinkRequest is the request body for linking a guardian.
type GuardianLinkRequest struct {
	RelationshipType   string `json:"relationship_type" validate:"required"`
	IsEmergencyContact bool   `json:"is_emergency_contact"`
}

// RegisterRoutes registers student routes under the given fiber router.
// Routes are grouped under /students with authz permission checks (BE-020).
func (h *Handler) RegisterRoutes(api fiber.Router) {
	students := api.Group("/students")

	// List and get - require student:read
	students.Get("/", authz.Require(authz.PermStudentRead), h.ListStudents)
	students.Get("/:id", authz.Require(authz.PermStudentRead), h.GetStudent)

	// Create, update, delete - require student:write
	students.Post("/", authz.Require(authz.PermStudentWrite), h.CreateStudent)
	students.Put("/:id", authz.Require(authz.PermStudentWrite), h.UpdateStudent)
	students.Delete("/:id", authz.Require(authz.PermStudentWrite), h.DeleteStudent)

	// Guardian links - require student:read for list, student:write for create/delete
	students.Get("/:id/guardians", authz.Require(authz.PermStudentRead), h.ListGuardianLinks)
	students.Post("/:id/guardians", authz.Require(authz.PermStudentWrite), h.CreateGuardianLink)
	students.Delete("/:id/guardians/:guardian_id", authz.Require(authz.PermStudentWrite), h.DeleteGuardianLink)
}

// ListStudents handles GET /api/v1/students
func (h *Handler) ListStudents(c fiber.Ctx) error {
	limit := int32(50)
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			limit = int32(n)
		}
	}

	var guardianID *uuid.UUID
	if gid := c.Query("guardianId"); gid != "" {
		if id, err := uuid.Parse(gid); err == nil {
			guardianID = &id
		}
	}

	var cursorFullName string
	var cursorStudentID uuid.UUID
	if cursor := c.Query("cursor"); cursor != "" {
		parts := strings.SplitN(cursor, "|", 2)
		if len(parts) == 2 {
			cursorFullName = parts[0]
			if id, err := uuid.Parse(parts[1]); err == nil {
				cursorStudentID = id
			}
		}
	}

	resp, err := h.svc.ListStudents(c.Context(), guardianID, limit, cursorFullName, cursorStudentID)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(ListStudentsResponse{
		Items:      resp.Items,
		NextCursor: resp.NextCursor,
	})
}

// GetStudent handles GET /api/v1/students/{id}
func (h *Handler) GetStudent(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperr.New(apperr.INVALID_REQUEST, "invalid student ID")
	}

	dto, err := h.svc.GetStudent(c.Context(), id)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(dto)
}

// CreateStudent handles POST /api/v1/students
func (h *Handler) CreateStudent(c fiber.Ctx) error {
	var req CreateStudentRequest
	if err := c.Bind().Body(&req); err != nil {
		return apperr.New(apperr.INVALID_REQUEST, "invalid request body")
	}

	// Parse date of birth
	dob, err := time.Parse("2006-01-02", req.DateOfBirth)
	if err != nil {
		return apperr.New(apperr.VALIDATION_FAILED, "invalid date_of_birth, expected yyyy-MM-dd")
	}

	var metadata json.RawMessage
	if req.Metadata == nil {
		metadata = []byte("{}")
	} else {
		metadata = req.Metadata
	}

	// Get school ID from session context (frontend never sends school ID, FE-007)
	schoolID := getPrimarySchoolID(c)
	if schoolID == uuid.Nil {
		return apperr.New(apperr.FORBIDDEN, "no active school in session")
	}

	dto, err := h.svc.CreateStudent(c.Context(), service.CreateStudentRequest{
		SchoolID:        schoolID,
		AdmissionNumber: req.AdmissionNumber,
		NationalNumber:  req.NationalNumber,
		FullName:        req.FullName,
		DateOfBirth:     dob,
		Gender:          req.Gender,
		Metadata:        metadata,
	})
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(dto)
}

// UpdateStudent handles PUT /api/v1/students/{id}
func (h *Handler) UpdateStudent(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperr.New(apperr.INVALID_REQUEST, "invalid student ID")
	}

	var req UpdateStudentRequest
	if err := c.Bind().Body(&req); err != nil {
		return apperr.New(apperr.INVALID_REQUEST, "invalid request body")
	}

	dob, err := time.Parse("2006-01-02", req.DateOfBirth)
	if err != nil {
		return apperr.New(apperr.VALIDATION_FAILED, "invalid date_of_birth, expected yyyy-MM-dd")
	}

	var metadata json.RawMessage
	if req.Metadata == nil {
		metadata = []byte("{}")
	} else {
		metadata = req.Metadata
	}

	dto, err := h.svc.UpdateStudent(c.Context(), service.UpdateStudentRequest{
		StudentID:       id,
		AdmissionNumber: req.AdmissionNumber,
		NationalNumber:  req.NationalNumber,
		FullName:        req.FullName,
		DateOfBirth:     dob,
		Gender:          req.Gender,
		Metadata:        metadata,
	})
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(dto)
}

// DeleteStudent handles DELETE /api/v1/students/{id}
func (h *Handler) DeleteStudent(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperr.New(apperr.INVALID_REQUEST, "invalid student ID")
	}

	if err := h.svc.DeleteStudent(c.Context(), id); err != nil {
		return err
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// ListGuardianLinks handles GET /api/v1/students/{id}/guardians
func (h *Handler) ListGuardianLinks(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperr.New(apperr.INVALID_REQUEST, "invalid student ID")
	}

	links, err := h.svc.ListGuardianLinks(c.Context(), id)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(links)
}

// CreateGuardianLink handles POST /api/v1/students/{id}/guardians
func (h *Handler) CreateGuardianLink(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperr.New(apperr.INVALID_REQUEST, "invalid student ID")
	}

	var req GuardianLinkRequest
	if err := c.Bind().Body(&req); err != nil {
		return apperr.New(apperr.INVALID_REQUEST, "invalid request body")
	}

	// Guardian ID comes from session context, not request body
	guardianID, ok := c.Locals("guardian_id").(uuid.UUID)
	if !ok {
		return apperr.New(apperr.FORBIDDEN, "guardian membership not found in session")
	}

	dto, err := h.svc.CreateGuardianLink(c.Context(), service.CreateGuardianLinkRequest{
		GuardianID:         guardianID,
		StudentID:          id,
		RelationshipType:   req.RelationshipType,
		IsEmergencyContact: req.IsEmergencyContact,
	})
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(dto)
}

// DeleteGuardianLink handles DELETE /api/v1/students/{id}/guardians/{guardian_id}
func (h *Handler) DeleteGuardianLink(c fiber.Ctx) error {
	studentID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperr.New(apperr.INVALID_REQUEST, "invalid student ID")
	}

	guardianID, err := uuid.Parse(c.Params("guardian_id"))
	if err != nil {
		return apperr.New(apperr.INVALID_REQUEST, "invalid guardian ID")
	}

	if err := h.svc.DeleteGuardianLink(c.Context(), studentID, guardianID); err != nil {
		return err
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// getPrimarySchoolID returns the first active school ID from session context.
func getPrimarySchoolID(c fiber.Ctx) uuid.UUID {
	if schoolIDs, ok := c.Locals("school_ids").([]uuid.UUID); ok && len(schoolIDs) > 0 {
		return schoolIDs[0]
	}
	return uuid.Nil
}
