package timetable

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type mockRepo struct {
	blocks  []TimeBlock
	slots   []Slot
	err     error
	blockID string
	slotID  string
}

func (m *mockRepo) ListBlocks(ctx context.Context, tenantID, schoolID, yearID string) ([]TimeBlock, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.blocks, nil
}

func (m *mockRepo) GetBlock(ctx context.Context, id, tenantID, schoolID string) (*TimeBlock, error) {
	if m.err != nil {
		return nil, m.err
	}
	for _, b := range m.blocks {
		if b.ID == id {
			return &b, nil
		}
	}
	return nil, ErrNotFound
}

func (m *mockRepo) CreateBlock(ctx context.Context, tenantID, schoolID string, p CreateTimeBlockPayload) (*TimeBlock, error) {
	if m.err != nil {
		return nil, m.err
	}
	b := TimeBlock{
		ID:             m.blockID,
		DayOfWeek:      p.DayOfWeek,
		PeriodName:     p.PeriodName,
		StartTime:      p.StartTime,
		EndTime:        p.EndTime,
		IsBreak:        p.IsBreak,
		AcademicYearID: p.AcademicYearID,
		Order:          p.Order,
	}
	m.blocks = append(m.blocks, b)
	return &b, nil
}

func (m *mockRepo) UpdateBlock(ctx context.Context, id, tenantID, schoolID string, p UpdateTimeBlockPayload) (*TimeBlock, error) {
	if m.err != nil {
		return nil, m.err
	}
	for i, b := range m.blocks {
		if b.ID == id {
			m.blocks[i].DayOfWeek = p.DayOfWeek
			m.blocks[i].PeriodName = p.PeriodName
			m.blocks[i].StartTime = p.StartTime
			m.blocks[i].EndTime = p.EndTime
			m.blocks[i].IsBreak = p.IsBreak
			m.blocks[i].AcademicYearID = p.AcademicYearID
			m.blocks[i].Order = p.Order
			return &m.blocks[i], nil
		}
	}
	return nil, ErrNotFound
}

func (m *mockRepo) DeleteBlock(ctx context.Context, id, tenantID, schoolID string) error {
	if m.err != nil {
		return m.err
	}
	for i, b := range m.blocks {
		if b.ID == id {
			m.blocks = append(m.blocks[:i], m.blocks[i+1:]...)
			return nil
		}
	}
	return ErrNotFound
}

func (m *mockRepo) ListSlots(ctx context.Context, f SlotFilter) ([]Slot, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.slots, nil
}

func (m *mockRepo) GetSlot(ctx context.Context, id, tenantID, schoolID string) (*Slot, error) {
	if m.err != nil {
		return nil, m.err
	}
	for _, s := range m.slots {
		if s.ID == id {
			return &s, nil
		}
	}
	return nil, ErrNotFound
}

func (m *mockRepo) CreateSlot(ctx context.Context, tenantID, schoolID, academicYearID string, p SlotPayload) (*Slot, error) {
	if m.err != nil {
		return nil, m.err
	}
	s := Slot{
		ID:             m.slotID,
		TenantID:       tenantID,
		SchoolID:       schoolID,
		AcademicYearID: academicYearID,
		BlockID:        p.BlockID,
		ClassID:        p.ClassID,
		LearningAreaID: p.LearningAreaID,
		TeacherID:      p.TeacherID,
		RoomIdentifier: p.RoomIdentifier,
	}
	m.slots = append(m.slots, s)
	return &s, nil
}

func (m *mockRepo) BatchCreateSlots(ctx context.Context, tenantID, schoolID, academicYearID string, ps []SlotPayload) ([]Slot, error) {
	if m.err != nil {
		return nil, m.err
	}
	var result []Slot
	for _, p := range ps {
		s := Slot{
			ID:             uuid.New().String(),
			TenantID:       tenantID,
			SchoolID:       schoolID,
			AcademicYearID: academicYearID,
			BlockID:        p.BlockID,
			ClassID:        p.ClassID,
			LearningAreaID: p.LearningAreaID,
			TeacherID:      p.TeacherID,
			RoomIdentifier: p.RoomIdentifier,
		}
		result = append(result, s)
	}
	m.slots = append(m.slots, result...)
	return result, nil
}

func (m *mockRepo) UpdateSlot(ctx context.Context, id, tenantID, schoolID string, p UpdateSlotPayload) (*Slot, error) {
	if m.err != nil {
		return nil, m.err
	}
	for i, s := range m.slots {
		if s.ID == id {
			if p.LearningAreaID != "" {
				m.slots[i].LearningAreaID = p.LearningAreaID
			}
			if p.TeacherID != "" {
				m.slots[i].TeacherID = p.TeacherID
			}
			if p.RoomIdentifier != nil {
				m.slots[i].RoomIdentifier = p.RoomIdentifier
			}
			return &m.slots[i], nil
		}
	}
	return nil, ErrNotFound
}

func (m *mockRepo) DeleteSlot(ctx context.Context, id, tenantID, schoolID string) error {
	if m.err != nil {
		return m.err
	}
	for i, s := range m.slots {
		if s.ID == id {
			m.slots = append(m.slots[:i], m.slots[i+1:]...)
			return nil
		}
	}
	return ErrNotFound
}

func newService(m *mockRepo) *ServiceImpl {
	return &ServiceImpl{repo: m, logger: zap.NewNop().Sugar()}
}

func TestServiceImpl_ListBlocks(t *testing.T) {
	m := &mockRepo{blocks: []TimeBlock{{ID: "1", PeriodName: "Lesson 1"}}}
	s := newService(m)

	blocks, err := s.ListBlocks(context.Background(), "t", "s", "y")
	require.NoError(t, err)
	require.Len(t, blocks, 1)
	require.Equal(t, "Lesson 1", blocks[0].PeriodName)
}

func TestServiceImpl_ListBlocks_Error(t *testing.T) {
	m := &mockRepo{err: errors.New("db error")}
	s := newService(m)

	_, err := s.ListBlocks(context.Background(), "t", "s", "y")
	require.Error(t, err)
	require.Contains(t, err.Error(), "timetable.ServiceImpl.ListBlocks")
}

func TestServiceImpl_GetBlock(t *testing.T) {
	m := &mockRepo{blocks: []TimeBlock{{ID: "1", PeriodName: "Lesson 1"}}}
	s := newService(m)

	block, err := s.GetBlock(context.Background(), "1", "t", "s")
	require.NoError(t, err)
	require.Equal(t, "1", block.ID)
	require.Equal(t, "Lesson 1", block.PeriodName)
}

func TestServiceImpl_GetBlock_NotFound(t *testing.T) {
	m := &mockRepo{}
	s := newService(m)

	_, err := s.GetBlock(context.Background(), "missing", "t", "s")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestServiceImpl_GetBlock_Error(t *testing.T) {
	m := &mockRepo{err: errors.New("db error")}
	s := newService(m)

	_, err := s.GetBlock(context.Background(), "1", "t", "s")
	require.Error(t, err)
	require.Contains(t, err.Error(), "timetable.ServiceImpl.GetBlock")
}

func TestServiceImpl_CreateBlock(t *testing.T) {
	m := &mockRepo{blockID: "new-id"}
	s := newService(m)

	block, err := s.CreateBlock(context.Background(), "t", "s", CreateTimeBlockPayload{
		DayOfWeek:      1,
		PeriodName:     "Lesson 1",
		StartTime:      "08:00",
		EndTime:        "08:40",
		IsBreak:        false,
		AcademicYearID: "y1",
		Order:          1,
	})
	require.NoError(t, err)
	require.Equal(t, "new-id", block.ID)
	require.Equal(t, "Lesson 1", block.PeriodName)
	require.Equal(t, 1, block.DayOfWeek)
}

func TestServiceImpl_CreateBlock_Error(t *testing.T) {
	m := &mockRepo{err: ErrBlockOverlap}
	s := newService(m)

	_, err := s.CreateBlock(context.Background(), "t", "s", CreateTimeBlockPayload{
		DayOfWeek:      1,
		PeriodName:     "Lesson 1",
		StartTime:      "08:00",
		EndTime:        "08:40",
		IsBreak:        false,
		AcademicYearID: "y1",
		Order:          1,
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrBlockOverlap)
}

func TestServiceImpl_UpdateBlock(t *testing.T) {
	m := &mockRepo{blocks: []TimeBlock{{ID: "1", PeriodName: "Old"}}}
	s := newService(m)

	block, err := s.UpdateBlock(context.Background(), "1", "t", "s", UpdateTimeBlockPayload{
		DayOfWeek:      2,
		PeriodName:     "New",
		StartTime:      "09:00",
		EndTime:        "09:40",
		IsBreak:        true,
		AcademicYearID: "y1",
		Order:          5,
	})
	require.NoError(t, err)
	require.Equal(t, "1", block.ID)
	require.Equal(t, "New", block.PeriodName)
	require.Equal(t, 2, block.DayOfWeek)
	require.True(t, block.IsBreak)
}

func TestServiceImpl_UpdateBlock_NotFound(t *testing.T) {
	m := &mockRepo{}
	s := newService(m)

	_, err := s.UpdateBlock(context.Background(), "missing", "t", "s", UpdateTimeBlockPayload{
		DayOfWeek:      1,
		PeriodName:     "Lesson 1",
		StartTime:      "08:00",
		EndTime:        "08:40",
		IsBreak:        false,
		AcademicYearID: "y1",
		Order:          1,
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestServiceImpl_UpdateBlock_Error(t *testing.T) {
	m := &mockRepo{err: ErrBlockOverlap}
	s := newService(m)

	_, err := s.UpdateBlock(context.Background(), "1", "t", "s", UpdateTimeBlockPayload{
		DayOfWeek:      1,
		PeriodName:     "Lesson 1",
		StartTime:      "08:00",
		EndTime:        "08:40",
		IsBreak:        false,
		AcademicYearID: "y1",
		Order:          1,
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrBlockOverlap)
}

func TestServiceImpl_DeleteBlock(t *testing.T) {
	m := &mockRepo{blocks: []TimeBlock{{ID: "1", PeriodName: "Lesson 1"}}}
	s := newService(m)

	result, err := s.DeleteBlock(context.Background(), "1", "t", "s")
	require.NoError(t, err)
	require.True(t, result.Deleted)
}

func TestServiceImpl_DeleteBlock_NotFound(t *testing.T) {
	m := &mockRepo{}
	s := newService(m)

	_, err := s.DeleteBlock(context.Background(), "missing", "t", "s")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestServiceImpl_ListSlots(t *testing.T) {
	m := &mockRepo{slots: []Slot{{ID: "1", ClassID: "c1"}}}
	s := newService(m)

	slots, err := s.ListSlots(context.Background(), SlotFilter{TenantID: "t", SchoolID: "s"})
	require.NoError(t, err)
	require.Len(t, slots, 1)
	require.Equal(t, "c1", slots[0].ClassID)
}

func TestServiceImpl_ListSlots_Error(t *testing.T) {
	m := &mockRepo{err: errors.New("db error")}
	s := newService(m)

	_, err := s.ListSlots(context.Background(), SlotFilter{TenantID: "t", SchoolID: "s"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "timetable.ServiceImpl.ListSlots")
}

func TestServiceImpl_GetSlot(t *testing.T) {
	m := &mockRepo{slots: []Slot{{ID: "1", ClassID: "c1", TeacherID: "t1"}}}
	s := newService(m)

	slot, err := s.GetSlot(context.Background(), "1", "t", "s")
	require.NoError(t, err)
	require.Equal(t, "1", slot.ID)
	require.Equal(t, "c1", slot.ClassID)
	require.Equal(t, "t1", slot.TeacherID)
}

func TestServiceImpl_GetSlot_NotFound(t *testing.T) {
	m := &mockRepo{}
	s := newService(m)

	_, err := s.GetSlot(context.Background(), "missing", "t", "s")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestServiceImpl_GetSlot_Error(t *testing.T) {
	m := &mockRepo{err: errors.New("db error")}
	s := newService(m)

	_, err := s.GetSlot(context.Background(), "1", "t", "s")
	require.Error(t, err)
	require.Contains(t, err.Error(), "timetable.ServiceImpl.GetSlot")
}

func TestServiceImpl_CreateSlot(t *testing.T) {
	m := &mockRepo{slotID: "new-slot"}
	s := newService(m)

	slot, err := s.CreateSlot(context.Background(), "t", "s", "y1", SlotPayload{
		BlockID:        "struct1",
		ClassID:        "c1",
		LearningAreaID: "la1",
		TeacherID:      "t1",
		RoomIdentifier: ptr("Room 101"),
	})
	require.NoError(t, err)
	require.Equal(t, "new-slot", slot.ID)
	require.Equal(t, "t", slot.TenantID)
	require.Equal(t, "s", slot.SchoolID)
	require.Equal(t, "y1", slot.AcademicYearID)
	require.Equal(t, "struct1", slot.BlockID)
	require.Equal(t, "c1", slot.ClassID)
	require.Equal(t, "la1", slot.LearningAreaID)
	require.Equal(t, "t1", slot.TeacherID)
	require.NotNil(t, slot.RoomIdentifier)
	require.Equal(t, "Room 101", *slot.RoomIdentifier)
}

func TestServiceImpl_CreateSlot_Error(t *testing.T) {
	m := &mockRepo{err: ErrTeacherDoubleBooked}
	s := newService(m)

	_, err := s.CreateSlot(context.Background(), "t", "s", "y1", SlotPayload{
		BlockID:        "struct1",
		ClassID:        "c1",
		LearningAreaID: "la1",
		TeacherID:      "t1",
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrTeacherDoubleBooked)
}

func TestServiceImpl_BatchCreateSlots(t *testing.T) {
	m := &mockRepo{}
	s := newService(m)

	slots, err := s.BatchCreateSlots(context.Background(), "t", "s", "y1", []SlotPayload{
		{BlockID: "s1", ClassID: "c1", LearningAreaID: "la1", TeacherID: "t1"},
		{BlockID: "s2", ClassID: "c2", LearningAreaID: "la2", TeacherID: "t2"},
	})
	require.NoError(t, err)
	require.Len(t, slots, 2)
	require.Equal(t, "s1", slots[0].BlockID)
	require.Equal(t, "c1", slots[0].ClassID)
	require.Equal(t, "s2", slots[1].BlockID)
	require.Equal(t, "c2", slots[1].ClassID)
}

func TestServiceImpl_BatchCreateSlots_Empty(t *testing.T) {
	m := &mockRepo{}
	s := newService(m)

	slots, err := s.BatchCreateSlots(context.Background(), "t", "s", "y1", []SlotPayload{})
	require.NoError(t, err)
	require.Empty(t, slots)
}

func TestServiceImpl_BatchCreateSlots_Error(t *testing.T) {
	m := &mockRepo{err: ErrConflict}
	s := newService(m)

	_, err := s.BatchCreateSlots(context.Background(), "t", "s", "y1", []SlotPayload{
		{BlockID: "s1", ClassID: "c1", LearningAreaID: "la1", TeacherID: "t1"},
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrConflict)
}

func TestServiceImpl_UpdateSlot(t *testing.T) {
	m := &mockRepo{slots: []Slot{{ID: "1", LearningAreaID: "old", TeacherID: "old", RoomIdentifier: ptr("Old")}}}
	s := newService(m)

	newRoom := "New Room"
	slot, err := s.UpdateSlot(context.Background(), "1", "t", "s", UpdateSlotPayload{
		LearningAreaID: "new",
		TeacherID:      "newt",
		RoomIdentifier: &newRoom,
	})
	require.NoError(t, err)
	require.Equal(t, "1", slot.ID)
	require.Equal(t, "new", slot.LearningAreaID)
	require.Equal(t, "newt", slot.TeacherID)
	require.NotNil(t, slot.RoomIdentifier)
	require.Equal(t, "New Room", *slot.RoomIdentifier)
}

func TestServiceImpl_UpdateSlot_NotFound(t *testing.T) {
	m := &mockRepo{}
	s := newService(m)

	_, err := s.UpdateSlot(context.Background(), "missing", "t", "s", UpdateSlotPayload{
		LearningAreaID: "la1",
		TeacherID:      "t1",
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestServiceImpl_UpdateSlot_Error(t *testing.T) {
	m := &mockRepo{err: ErrTeacherDoubleBooked}
	s := newService(m)

	_, err := s.UpdateSlot(context.Background(), "1", "t", "s", UpdateSlotPayload{
		LearningAreaID: "la1",
		TeacherID:      "t1",
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrTeacherDoubleBooked)
}

func TestServiceImpl_DeleteSlot(t *testing.T) {
	m := &mockRepo{slots: []Slot{{ID: "1", ClassID: "c1"}}}
	s := newService(m)

	err := s.DeleteSlot(context.Background(), "1", "t", "s")
	require.NoError(t, err)
}

func TestServiceImpl_DeleteSlot_NotFound(t *testing.T) {
	m := &mockRepo{}
	s := newService(m)

	err := s.DeleteSlot(context.Background(), "missing", "t", "s")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestServiceImpl_DeleteSlot_Error(t *testing.T) {
	m := &mockRepo{err: errors.New("db error")}
	s := newService(m)

	err := s.DeleteSlot(context.Background(), "1", "t", "s")
	require.Error(t, err)
	require.Contains(t, err.Error(), "timetable.ServiceImpl.DeleteSlot")
}
