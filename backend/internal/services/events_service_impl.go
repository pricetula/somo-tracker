package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type eventsSvc struct {
	pool *pgxpool.Pool
}

func NewEventsService(pool *pgxpool.Pool) EventsService {
	return &eventsSvc{pool: pool}
}

func (s *eventsSvc) ListEvents(ctx context.Context, schoolID uuid.UUID, fromStr, toStr string) ([]EventItem, error) {
	if fromStr == "" {
		fromStr = time.Now().Format("2006-01-02")
	}
	if toStr == "" {
		toStr = time.Now().AddDate(0, 0, 7).Format("2006-01-02")
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, title, event_type, start_date, end_date, requires_attendance
		FROM school_events WHERE school_id = $1 AND start_date >= $2 AND start_date <= $3
		ORDER BY start_date ASC`, schoolID, fromStr, toStr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []EventItem
	for rows.Next() {
		var id uuid.UUID
		var title, eventType string
		var startDate, endDate time.Time
		var requiresAttendance bool
		if err := rows.Scan(&id, &title, &eventType, &startDate, &endDate, &requiresAttendance); err != nil {
			continue
		}
		items = append(items, EventItem{
			ID: id.String(), Title: title, EventType: eventType,
			StartDate: startDate.Format("2006-01-02"), EndDate: endDate.Format("2006-01-02"),
			RequiresAttendance: requiresAttendance,
		})
	}
	return items, rows.Err()
}

func (s *eventsSvc) CreateEvent(ctx context.Context, schoolID uuid.UUID, req CreateEventRequest) (uuid.UUID, error) {
	var id uuid.UUID
	err := s.pool.QueryRow(ctx, `
		INSERT INTO school_events (school_id, title, event_type, start_date, end_date, requires_attendance)
		VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`, schoolID, req.Title, req.EventType, req.StartDate, req.EndDate, req.RequiresAttendance).Scan(&id)
	return id, err
}

func (s *eventsSvc) UpdateEvent(ctx context.Context, schoolID, eventID uuid.UUID, req UpdateEventRequest) (EventItem, error) {
	var item EventItem
	var id uuid.UUID
	var title, eventType string
	var startDate, endDate time.Time
	var requiresAttendance bool
	err := s.pool.QueryRow(ctx, `UPDATE school_events SET title=$2,event_type=$3,start_date=$4,end_date=$5,requires_attendance=$6 WHERE id=$7 AND school_id=$1 RETURNING id,title,event_type,start_date,end_date,requires_attendance`, schoolID, req.Title, req.EventType, req.StartDate, req.EndDate, req.RequiresAttendance, eventID).Scan(&id, &title, &eventType, &startDate, &endDate, &requiresAttendance)
	if err != nil {
		if err == pgx.ErrNoRows {
			return EventItem{}, fmt.Errorf("not_found")
		}
		return EventItem{}, err
	}
	item = EventItem{ID: id.String(), Title: title, EventType: eventType, StartDate: startDate.Format("2006-01-02"), EndDate: endDate.Format("2006-01-02"), RequiresAttendance: requiresAttendance}
	return item, nil
}

func (s *eventsSvc) DeleteEvent(ctx context.Context, schoolID, eventID uuid.UUID) error {
	_, err := s.pool.Exec(ctx, "DELETE FROM school_events WHERE id=$1 AND school_id=$2", eventID, schoolID)
	return err
}
