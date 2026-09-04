package api

import (
	"github.com/gofiber/fiber/v3"
	"somotracker/backend/internal/services"
)

// Router wires all delivery-layer routes. It depends only on the service
// interfaces (not concrete implementations), which makes it fully testable
// with mock services.
type Router struct {
	User   *userHandler
	Tenant *tenantHandler
}

// NewRouter creates a Router from the injected services. In production this
// is built by Uber Fx; in tests a test can construct it directly.
func NewRouter(userSvc services.UserService, tenantSvc services.TenantService) *Router {
	return &Router{
		User:   newUserHandler(userSvc),
		Tenant: newTenantHandler(tenantSvc),
	}
}

// RegisterRoutes attaches the grouped endpoints to the Fiber app.
func (r *Router) RegisterRoutes(app *fiber.App) {
	// Tenant routes — non-RLS, direct sqlc lookup.
	app.Get("/api/tenants/slug/:slug", r.Tenant.getBySlug)

	// User routes — RLS-backed via WithTenantTx inside UserService.
	app.Get("/api/users/:id", r.User.getByID)
	app.Get("/api/users/email/:email", r.User.getByEmail)
}
