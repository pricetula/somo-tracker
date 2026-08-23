package timetable

import (
	"context"
	"database/sql"
	"fmt"

	"somotracker/backend/internal/xerrors"
)

type PgRepository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *PgRepository {
	return &PgRepository{db: db}
}

func (r *PgRepository) ListBlocks(ctx context.Context, tenantID, schoolID, yearID string) ([]TimeBlock, error) {
	return nil, fmt.Errorf("timetable.Repository.ListBlocks: %w", xerrors.NotFound("not implemented"))
}
func (r *PgRepository) GetBlock(ctx context.Context, id, tenantID, schoolID string) (*TimeBlock, error) {
	return nil, fmt.Errorf("timetable.Repository.GetBlock: %w", xerrors.NotFound("not implemented"))
}
func (r *PgRepository) CreateBlock(ctx context.Context, tenantID, schoolID string, p CreateTimeBlockPayload) (*TimeBlock, error) {
	return nil, fmt.Errorf("timetable.Repository.CreateBlock: %w", xerrors.NotFound("not implemented"))
}
func (r *PgRepository) UpdateBlock(ctx context.Context, id, tenantID, schoolID string, p UpdateTimeBlockPayload) (*TimeBlock, error) {
	return nil, fmt.Errorf("timetable.Repository.UpdateBlock: %w", xerrors.NotFound("not implemented"))
}
func (r *PgRepository) DeleteBlock(ctx context.Context, id, tenantID, schoolID string) error {
	return fmt.Errorf("timetable.Repository.DeleteBlock: %w", xerrors.NotFound("not implemented"))
}

func (r *PgRepository) ListSlots(ctx context.Context, f SlotFilter) ([]Slot, error) {
	return nil, fmt.Errorf("timetable.Repository.ListSlots: %w", xerrors.NotFound("not implemented"))
}
func (r *PgRepository) GetSlot(ctx context.Context, id string) (*Slot, error) {
	return nil, fmt.Errorf("timetable.Repository.GetSlot: %w", xerrors.NotFound("not implemented"))
}
func (r *PgRepository) CreateSlot(ctx context.Context, tenantID, schoolID string, p SlotPayload) (*Slot, error) {
	return nil, fmt.Errorf("timetable.Repository.CreateSlot: %w", xerrors.NotFound("not implemented"))
}
func (r *PgRepository) BatchCreateSlots(ctx context.Context, tenantID, schoolID string, ps []SlotPayload) ([]Slot, error) {
	return nil, fmt.Errorf("timetable.Repository.BatchCreateSlots: %w", xerrors.NotFound("not implemented"))
}
func (r *PgRepository) UpdateSlot(ctx context.Context, id string, p UpdateSlotPayload) (*Slot, error) {
	return nil, fmt.Errorf("timetable.Repository.UpdateSlot: %w", xerrors.NotFound("not implemented"))
}
func (r *PgRepository) DeleteSlot(ctx context.Context, id string) error {
	return fmt.Errorf("timetable.Repository.DeleteSlot: %w", xerrors.NotFound("not implemented"))
}
