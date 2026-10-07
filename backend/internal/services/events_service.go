package services

import (
	"context"

	"github.com/google/uuid"
)

type EventsService interface {
	ListEvents(ctx context.Context, schoolID uuid.UUID, fromStr, toStr string) ([]EventItem, error)
	CreateEvent(ctx context.Context, schoolID uuid.UUID, req CreateEventRequest) (uuid.UUID, error)
	UpdateEvent(ctx context.Context, schoolID, eventID uuid.UUID, req UpdateEventRequest) (EventItem, error)
	DeleteEvent(ctx context.Context, schoolID, eventID uuid.UUID) error
}

type EventItem struct {
	ID                 string `json:"id"`
	Title              string `json:"title"`
	EventType          string `json:"event_type"`
	StartDate          string `json:"start_date"`
	EndDate            string `json:"end_date"`
	RequiresAttendance bool   `json:"requires_attendance"`
}

type CreateEventRequest struct {
	Title              string `json:"title"`
	EventType          string `json:"event_type"`
	StartDate          string `json:"start_date"`
	EndDate            string `json:"end_date"`
	RequiresAttendance bool   `json:"requires_attendance"`
}

type UpdateEventRequest struct {
	Title              string `json:"title"`
	EventType          string `json:"event_type"`
	StartDate          string `json:"start_date"`
	EndDate            string `json:"end_date"`
	RequiresAttendance bool   `json:"requires_attendance"`
}
