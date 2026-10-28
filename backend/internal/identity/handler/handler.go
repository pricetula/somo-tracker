package handler

import (
	"github.com/gofiber/fiber/v3"
	"somotracker/internal/identity/service"
)

// Handler is HTTP handler for identity domain (thin, BE-006).
type Handler struct {
	svc *service.Service
}

// New creates handler.
func New(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

// DTOs are defined per endpoint, not domain structs (BE-007).

type RequestMagicLinkRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type AuthenticateTokenRequest struct {
	Token string `json:"token" validate:"required"`
}

// RequestMagicLink handles POST /api/v1/auth/request-magic-link
func (h *Handler) RequestMagicLink(c fiber.Ctx) error {
	var req RequestMagicLinkRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}
	// TODO: validate, call svc.RequestMagicLink, return generic success
	_ = req
	return c.SendStatus(fiber.StatusOK)
}

// AuthenticateToken handles POST /api/v1/auth/authenticate-token
func (h *Handler) AuthenticateToken(c fiber.Ctx) error {
	var req AuthenticateTokenRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}
	// TODO: call svc.AuthenticateToken, set cookie on success
	_ = req
	return c.SendStatus(fiber.StatusOK)
}

// Logout handles POST /api/v1/auth/logout
func (h *Handler) Logout(c fiber.Ctx) error {
	// TODO: revoke session from context, call svc.Logout, clear cookie
	return c.SendStatus(fiber.StatusOK)
}

// RegisterRoutes registers identity routes under /auth
func (h *Handler) RegisterRoutes(api fiber.Router) {
	auth := api.Group("/auth")
	auth.Post("/request-magic-link", h.RequestMagicLink)
	auth.Post("/authenticate-token", h.AuthenticateToken)
	auth.Post("/logout", h.Logout)
}
