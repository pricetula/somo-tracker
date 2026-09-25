package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	"somotracker/backend/internal/database/sqlc"
)

type RoomsService interface {
	CreateRoom(ctx context.Context, schoolID string, name string, capacity *int, roomType string) (*sqlc.Room, error)
	UpdateRoom(ctx context.Context, schoolID string, id string, name *string, capacity *int, roomType *string) (*sqlc.Room, error)
	DeleteRoom(ctx context.Context, schoolID string, id string) error
}

type roomsService struct {
	queries *sqlc.Queries
	logger  *zap.Logger
}

func NewRoomsService(queries *sqlc.Queries, logger *zap.Logger) RoomsService {
	return &roomsService{
		queries: queries,
		logger:  logger.With(zap.String("service", "rooms")),
	}
}

func (s *roomsService) CreateRoom(ctx context.Context, schoolID string, name string, capacity *int, roomType string) (*sqlc.Room, error) {
	if schoolID == "" {
		return nil, fmt.Errorf("bad_request: school_id is required")
	}
	if name == "" {
		return nil, fmt.Errorf("bad_request: name is required")
	}
	schoolUUID, err := uuid.Parse(schoolID)
	if err != nil {
		return nil, fmt.Errorf("bad_request: invalid school_id")
	}
	params := sqlc.CreateRoomParams{
		SchoolID: pgtype.UUID{Bytes: schoolUUID, Valid: true},
		Name:     name,
		RoomType: sqlc.RoomType(roomType),
	}
	if capacity != nil {
		params.Capacity = pgtype.Int4{Int32: int32(*capacity), Valid: true}
	} else {
		params.Capacity = pgtype.Int4{Valid: false}
	}
	room, err := s.queries.CreateRoom(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("internal_error: failed to create room: %w", err)
	}
	return &room, nil
}

func (s *roomsService) UpdateRoom(ctx context.Context, schoolID string, id string, name *string, capacity *int, roomType *string) (*sqlc.Room, error) {
	if schoolID == "" {
		return nil, fmt.Errorf("bad_request: school_id is required")
	}
	if id == "" {
		return nil, fmt.Errorf("bad_request: id is required")
	}
	schoolUUID, err := uuid.Parse(schoolID)
	if err != nil {
		return nil, fmt.Errorf("bad_request: invalid school_id")
	}
	roomUUID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("bad_request: invalid room id")
	}

	nameVal := ""
	if name != nil {
		nameVal = *name
	}
	roomTypeVal := ""
	if roomType != nil {
		roomTypeVal = *roomType
	}
	updateParams := sqlc.UpdateRoomParams{
		ID:       pgtype.UUID{Bytes: roomUUID, Valid: true},
		SchoolID: pgtype.UUID{Bytes: schoolUUID, Valid: true},
		Column3:  nameVal,
		Column5:  roomTypeVal,
	}
	if capacity != nil {
		capPg := pgtype.Int4{Int32: int32(*capacity), Valid: true}
		updateParams.Capacity = capPg
	} else {
		updateParams.Capacity = pgtype.Int4{Valid: false}
	}

	room, err := s.queries.UpdateRoom(ctx, updateParams)
	if err != nil {
		return nil, fmt.Errorf("internal_error: failed to update room: %w", err)
	}
	return &room, nil
}

func (s *roomsService) DeleteRoom(ctx context.Context, schoolID string, id string) error {
	if schoolID == "" {
		return fmt.Errorf("bad_request: school_id is required")
	}
	if id == "" {
		return fmt.Errorf("bad_request: id is required")
	}
	schoolUUID, err := uuid.Parse(schoolID)
	if err != nil {
		return fmt.Errorf("bad_request: invalid school_id")
	}
	roomUUID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("bad_request: invalid room id")
	}
	params := sqlc.DeleteRoomParams{
		ID:       pgtype.UUID{Bytes: roomUUID, Valid: true},
		SchoolID: pgtype.UUID{Bytes: schoolUUID, Valid: true},
	}
	if err := s.queries.DeleteRoom(ctx, params); err != nil {
		return fmt.Errorf("internal_error: failed to delete room: %w", err)
	}
	return nil
}
