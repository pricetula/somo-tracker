package handler

import (
	"somotracker/internal/identity/service"
)

// Handler is HTTP handler for identity domain (thin, BE-006).
type Handler struct {
	svc service.Service
}

// New creates handler.
func New(svc service.Service) *Handler {
	return &Handler{svc: svc}
}

// DTOs are defined per endpoint, not domain structs (BE-007).
