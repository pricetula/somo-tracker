package tenant

import (
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

// Handler exposes tenant-related HTTP endpoints.
type Handler struct {
	svc *Service
}

// NewHandler creates a new Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts tenant routes on the given router.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	tenants := router.Group("/tenants")
	_ = tenants // placeholder — routes will be registered here
}

// Module is an fx-compatible module for the tenant domain.
var Module = fx.Module("tenant",
	fx.Provide(
		NewRepository,
		NewService,
		NewHandler,
	),
)
