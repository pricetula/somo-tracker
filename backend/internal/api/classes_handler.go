package api

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"somotracker/backend/internal/services"
)

type ClassesHandler struct {
	svc    services.ClassesService
	logger *zap.Logger
}

func NewClassesHandler(svc services.ClassesService, logger *zap.Logger) *ClassesHandler {
	return &ClassesHandler{svc: svc, logger: logger.With(zap.String("handler", "classes"))}
}

type createClassRequest struct {
	Name     string `json:"name"`
	GradeID  string `json:"gradeId"`
	StreamID string `json:"streamId"`
}

// ListClasses lists classes for the active school
// @Summary List classes
// @Tags Classes
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Page size" default(50)
// @Param search query string false "Search by name"
// @Param grade query string false "Comma separated grade labels"
// @Param stream query string false "Comma separated stream names"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /school/classes [get]
func (h *ClassesHandler) ListClasses(c fiber.Ctx) error {
	schoolIDStr, ok := c.Locals("active_school_id").(string)
	if !ok || schoolIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"code": "unauthorized", "message": "active school not found", "errors": fiber.Map{}})
	}
	schoolID, err := uuid.Parse(schoolIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": "bad_request", "message": "invalid school id", "errors": fiber.Map{}})
	}
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	search := c.Query("search")
	gradeParam := c.Query("grade")
	streamParam := c.Query("stream")

	var grades []string
	if strings.TrimSpace(gradeParam) != "" {
		for _, g := range strings.Split(gradeParam, ",") {
			g = strings.TrimSpace(g)
			if g != "" {
				grades = append(grades, g)
			}
		}
	}
	var streams []string
	if strings.TrimSpace(streamParam) != "" {
		for _, s := range strings.Split(streamParam, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				streams = append(streams, s)
			}
		}
	}

	items, total, err := h.svc.ListClasses(c.Context(), schoolID, page, limit, search, grades, streams)
	if err != nil {
		h.logger.Error("list classes failed", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"code": "internal_error", "message": "failed to list classes", "errors": fiber.Map{}})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"items": items,
		"total": total,
	})
}

// GetClass returns a single class
// @Summary Get class
// @Tags Classes
// @Produce json
// @Param id path string true "Class ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /school/classes/{id} [get]
func (h *ClassesHandler) GetClass(c fiber.Ctx) error {
	schoolIDStr, ok := c.Locals("active_school_id").(string)
	if !ok || schoolIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"code": "unauthorized", "message": "active school not found", "errors": fiber.Map{}})
	}
	schoolID, _ := uuid.Parse(schoolIDStr)
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": "bad_request", "message": "invalid id", "errors": fiber.Map{}})
	}
	detail, err := h.svc.GetClass(c.Context(), schoolID, id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"code": "not_found", "message": "class not found", "errors": fiber.Map{}})
	}
	return c.Status(fiber.StatusOK).JSON(detail)
}

// CreateClass creates a new class
// @Summary Create class
// @Tags Classes
// @Accept json
// @Produce json
// @Param body body createClassRequest true "Class payload"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /school/classes [post]
func (h *ClassesHandler) CreateClass(c fiber.Ctx) error {
	schoolIDStr, ok := c.Locals("active_school_id").(string)
	if !ok || schoolIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"code": "unauthorized", "message": "active school not found", "errors": fiber.Map{}})
	}
	schoolID, _ := uuid.Parse(schoolIDStr)
	var req createClassRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": "bad_request", "message": "invalid body", "errors": fiber.Map{}})
	}
	item, err := h.svc.CreateClass(c.Context(), schoolID, services.CreateClassRequest{
		Name: req.Name, GradeID: req.GradeID, StreamID: req.StreamID,
	})
	if err != nil {
		h.logger.Error("create class failed", zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": "bad_request", "message": err.Error(), "errors": fiber.Map{}})
	}
	return c.Status(fiber.StatusCreated).JSON(item)
}
