package timetable

import (
	"context"

	"github.com/gofiber/fiber/v2"

	"somotracker/backend/internal/middleware"
)

type Handler struct {
	svc              Service
	academicYearsSvc interface {
		GetCurrentYearAndTermID(ctx context.Context, tenantID, schoolID string) (yearID, termID string, err error)
	}
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) SetAcademicYearsService(svc interface {
	GetCurrentYearAndTermID(ctx context.Context, tenantID, schoolID string) (yearID, termID string, err error)
}) {
	h.academicYearsSvc = svc
}

func (h *Handler) RegisterRoutes(router fiber.Router) {
	base := router.Group("/api/v1/timetable")

	// Structure mutations
	base.Post("/structure", middleware.RequireAuth, h.CreateBlock)
	base.Get("/structure", middleware.RequireAuth, h.ListBlocks)
	base.Get("/structure/:id", middleware.RequireAuth, h.GetBlock)
	base.Put("/structure/:id", middleware.RequireAuth, h.UpdateBlock)
	base.Delete("/structure/:id", middleware.RequireAuth, h.DeleteBlock)

	// Slot mutations
	base.Post("/slots", middleware.RequireAuth, h.CreateSlot)
	base.Get("/slots", middleware.RequireAuth, h.ListSlots)
	base.Put("/slots/:id", middleware.RequireAuth, h.UpdateSlot)
	base.Delete("/slots/:id", middleware.RequireAuth, h.DeleteSlot)

	// Read-only timetable views
	base.Get("/timetable", middleware.RequireAuth, h.GetTimetable)
}

func (h *Handler) CreateBlock(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"code": "ok", "message": "structure endpoint"})
}
func (h *Handler) ListBlocks(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"code": "ok", "message": "structure list"})
}
func (h *Handler) GetBlock(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"code": "ok", "message": "structure get"})
}
func (h *Handler) UpdateBlock(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"code": "ok", "message": "structure update"})
}
func (h *Handler) DeleteBlock(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"code": "ok", "message": "structure delete"})
}
func (h *Handler) CreateSlot(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"code": "ok", "message": "slot endpoint"})
}
func (h *Handler) ListSlots(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"code": "ok", "message": "slots list"})
}
func (h *Handler) UpdateSlot(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"code": "ok", "message": "slot update"})
}
func (h *Handler) DeleteSlot(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"code": "ok", "message": "slot delete"})
}
func (h *Handler) GetTimetable(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"code": "ok", "message": "timetable view"})
}
