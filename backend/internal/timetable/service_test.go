package timetable

import (
	"context"
	"testing"

	"go.uber.org/zap"
)

type mockRepo struct{}

func (m *mockRepo) ListBlocks(ctx context.Context, tenantID, schoolID, yearID string) ([]TimeBlock, error) {
	return nil, nil
}
func (m *mockRepo) GetBlock(ctx context.Context, id, tenantID, schoolID string) (*TimeBlock, error) {
	return nil, nil
}
func (m *mockRepo) CreateBlock(ctx context.Context, tenantID, schoolID string, p CreateTimeBlockPayload) (*TimeBlock, error) {
	return nil, nil
}
func (m *mockRepo) UpdateBlock(ctx context.Context, id, tenantID, schoolID string, p UpdateTimeBlockPayload) (*TimeBlock, error) {
	return nil, nil
}
func (m *mockRepo) DeleteBlock(ctx context.Context, id, tenantID, schoolID string) error { return nil }
func (m *mockRepo) ListSlots(ctx context.Context, f SlotFilter) ([]Slot, error)          { return nil, nil }
func (m *mockRepo) GetSlot(ctx context.Context, id string) (*Slot, error)                { return nil, nil }
func (m *mockRepo) CreateSlot(ctx context.Context, tenantID, schoolID, academicYearID string, p SlotPayload) (*Slot, error) {
	return nil, nil
}
func (m *mockRepo) BatchCreateSlots(ctx context.Context, tenantID, schoolID, academicYearID string, ps []SlotPayload) ([]Slot, error) {
	return nil, nil
}
func (m *mockRepo) UpdateSlot(ctx context.Context, id string, p UpdateSlotPayload) (*Slot, error) {
	return nil, nil
}
func (m *mockRepo) DeleteSlot(ctx context.Context, id string) error { return nil }

func TestServiceImpl_ListBlocks(t *testing.T) {
	s := &ServiceImpl{repo: &mockRepo{}, logger: zap.NewNop().Sugar()}
	_, err := s.ListBlocks(nil, "t", "s", "y")
	if err != nil {
		t.Fatal(err)
	}
}
