package handler

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"somotracker/internal/platform/apperr"
	"somotracker/internal/platform/authz"
	schoolsSvc "somotracker/internal/schools/service"
)

// Service is the interface the handler depends on.
// The real schools service satisfies this interface.
type Service interface {
	ListSchools(ctx context.Context, limit int32, cursorName string, cursorID uuid.UUID) (*schoolsSvc.ListResponse, error)
	GetSchool(ctx context.Context, id uuid.UUID) (*schoolsSvc.SchoolDTO, error)
	CreateSchool(ctx context.Context, req schoolsSvc.CreateSchoolRequest) (*schoolsSvc.SchoolDTO, error)
	UpdateSchool(ctx context.Context, req schoolsSvc.UpdateSchoolRequest) (*schoolsSvc.SchoolDTO, error)
	DeactivateSchools(ctx context.Context, ids []uuid.UUID) error
}

// Handler is HTTP handler for schools domain (thin, BE-006).
type Handler struct {
	svc Service
}

// New creates handler.
func New(svc Service) *Handler {
	return &Handler{svc: svc}
}

// Ensure the real service satisfies the Service interface.
var _ Service = (*schoolsSvc.Service)(nil)

// DTOs are defined per endpoint, not domain structs (BE-007).

// ListSchoolsResponse is the paginated list response.
type ListSchoolsResponse struct {
	Items      []schoolsSvc.SchoolDTO `json:"items"`
	NextCursor *string                `json:"nextCursor,omitempty"`
}

// CreateSchoolRequest is the request body for creating a school.
type CreateSchoolRequest struct {
	Name              string     `json:"name" validate:"required"`
	Code              string     `json:"code" validate:"required"`
	CountryID         uuid.UUID  `json:"country_id" validate:"required"`
	EducationSystemID uuid.UUID  `json:"education_system_id" validate:"required"`
}

// UpdateSchoolRequest is the request body for updating a school (full body).
type UpdateSchoolRequest struct {
	Name              string     `json:"name" validate:"required"`
	Code              string     `json:"code" validate:"required"`
	CountryID         uuid.UUID  `json:"country_id" validate:"required"`
	EducationSystemID uuid.UUID  `json:"education_system_id" validate:"required"`
}

// SchoolDTO is the response DTO for school operations.
type SchoolDTO struct {
	ID                uuid.UUID    `json:"id"`
	TenantID          uuid.UUID    `json:"tenant_id"`
	Name              string       `json:"name"`
	Code              string       `json:"code"`
	CountryID         uuid.UUID    `json:"country_id"`
	EducationSystemID uuid.UUID    `json:"education_system_id"`
	CreatedAt         time.Time    `json:"created_at"`
	UpdatedAt         time.Time    `json:"updated_at"`
	DeletedAt         *time.Time   `json:"deleted_at,omitempty"`
}

// DeactivateSchoolsRequest is the request body for deactivating schools.
type DeactivateSchoolsRequest struct {
	IDs []uuid.UUID `json:"ids" validate:"required"`
}

// DeactivateSchoolsResponse is the response body for deactivating schools.
type DeactivateSchoolsResponse struct {
	Success bool `json:"success"`
}

// RegisterRoutes registers school routes under the given fiber router.
// Routes are grouped under /schools with authz permission checks (BE-020).
func (h *Handler) RegisterRoutes(api fiber.Router) {
	schools := api.Group("/schools")

	// List and get - require school:read
	schools.Get("/", authz.Require(authz.PermSchoolRead), h.ListSchools)
	schools.Get("/:id", authz.Require(authz.PermSchoolRead), h.GetSchool)

	// Create, update, deactivate - require school:write
	schools.Post("/", authz.Require(authz.PermSchoolWrite), h.CreateSchool)
	schools.Patch("/:id", authz.Require(authz.PermSchoolWrite), h.UpdateSchool)
	schools.Delete("/", authz.Require(authz.PermSchoolWrite), h.DeactivateSchools)
}

// ListSchools handles GET /api/v1/schools
func (h *Handler) ListSchools(c fiber.Ctx) error {
	limit := int32(50)
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			limit = int32(n)
		}
	}

	var cursorName string
	var cursorID uuid.UUID
	if cursor := c.Query("cursor"); cursor != "" {
		parts := strings.SplitN(cursor, "|", 2)
		if len(parts) == 2 {
			cursorName = parts[0]
			if id, err := uuid.Parse(parts[1]); err == nil {
				cursorID = id
			}
		}
	}

	resp, err := h.svc.ListSchools(c.Context(), limit, cursorName, cursorID)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(ListSchoolsResponse{
		Items:      resp.Items,
		NextCursor: resp.NextCursor,
	})
}

// GetSchool handles GET /api/v1/schools/{id}
func (h *Handler) GetSchool(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperr.New(apperr.INVALID_REQUEST, "invalid school ID")
	}

	dto, err := h.svc.GetSchool(c.Context(), id)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(dto)
}

// CreateSchool handles POST /api/v1/schools
func (h *Handler) CreateSchool(c fiber.Ctx) error {
	var req CreateSchoolRequest
	if err := c.Bind().Body(&req); err != nil {
		return apperr.New(apperr.INVALID_REQUEST, "invalid request body")
	}

	// Get tenant ID from session context (frontend never sends tenant ID, FE-007)
	tenantID := getTenantID(c)
	if tenantID == uuid.Nil {
		return apperr.New(apperr.FORBIDDEN, "no active tenant in session")
	}

	dto, err := h.svc.CreateSchool(c.Context(), schoolsSvc.CreateSchoolRequest{
		TenantID:          tenantID,
		Name:              req.Name,
		Code:              req.Code,
		CountryID:         req.CountryID,
		EducationSystemID: req.EducationSystemID,
	})
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(dto)
}

// UpdateSchool handles PATCH /api/v1/schools/:id
func (h *Handler) UpdateSchool(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperr.New(apperr.INVALID_REQUEST, "invalid school ID")
	}

	var req UpdateSchoolRequest
	if err := c.Bind().Body(&req); err != nil {
		return apperr.New(apperr.INVALID_REQUEST, "invalid request body")
	}

	dto, err := h.svc.UpdateSchool(c.Context(), schoolsSvc.UpdateSchoolRequest{
		ID:                id,
		Name:              req.Name,
		Code:              req.Code,
		CountryID:         req.CountryID,
		EducationSystemID: req.EducationSystemID,
	})
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(dto)
}

// DeactivateSchools handles DELETE /api/v1/schools
func (h *Handler) DeactivateSchools(c fiber.Ctx) error {
	var req DeactivateSchoolsRequest
	if err := c.Bind().Body(&req); err != nil {
		return apperr.New(apperr.INVALID_REQUEST, "invalid request body")
	}

	if err := h.svc.DeactivateSchools(c.Context(), req.IDs); err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(DeactivateSchoolsResponse{Success: true})
}

// getTenantID returns the tenant ID from session context.
func getTenantID(c fiber.Ctx) uuid.UUID {
	if tid, ok := c.Locals("tenant_id").(uuid.UUID); ok {
		return tid
	}
	return uuid.Nil
}
