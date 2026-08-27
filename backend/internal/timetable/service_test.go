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
	blocks      []TimeBlock
	allocations []Allocation
	err         error
	blockID     string
	allocID     string
}

func (m *mockRepo) GetTrack(ctx context.Context, id, tenantID, schoolID string) (*Track, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &Track{ID: id, Name: "Mock Track"}, nil
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
		ID:         m.blockID,
		TrackID:    p.TrackID,
		DayOfWeek:  p.DayOfWeek,
		PeriodName: p.PeriodName,
		StartTime:  p.StartTime,
		EndTime:    p.EndTime,
		IsBreak:    p.IsBreak,
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
			m.blocks[i].TrackID = p.TrackID
			m.blocks[i].DayOfWeek = p.DayOfWeek
			m.blocks[i].PeriodName = p.PeriodName
			m.blocks[i].StartTime = p.StartTime
			m.blocks[i].EndTime = p.EndTime
			m.blocks[i].IsBreak = p.IsBreak
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

func (m *mockRepo) ListAllocations(ctx context.Context, f AllocationFilter) ([]Allocation, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.allocations, nil
}

func (m *mockRepo) GetAllocation(ctx context.Context, id, tenantID, schoolID string) (*Allocation, error) {
	if m.err != nil {
		return nil, m.err
	}
	for _, a := range m.allocations {
		if a.ID == id {
			return &a, nil
		}
	}
	return nil, ErrNotFound
}

func (m *mockRepo) CreateAllocation(ctx context.Context, tenantID, schoolID string, p CreateAllocationPayload) (*Allocation, error) {
	if m.err != nil {
		return nil, m.err
	}
	a := Allocation{
		ID:             m.allocID,
		TenantID:       tenantID,
		SchoolID:       schoolID,
		BlockID:        p.BlockID,
		ClassID:        p.ClassID,
		LearningAreaID: p.LearningAreaID,
		TeacherID:      p.TeacherID,
		RoomIdentifier: p.RoomIdentifier,
	}
	m.allocations = append(m.allocations, a)
	return &a, nil
}

func (m *mockRepo) BatchCreateAllocations(ctx context.Context, tenantID, schoolID string, ps []CreateAllocationPayload) ([]Allocation, error) {
	if m.err != nil {
		return nil, m.err
	}
	var result []Allocation
	for _, p := range ps {
		a := Allocation{
			ID:             uuid.New().String(),
			TenantID:       tenantID,
			SchoolID:       schoolID,
			BlockID:        p.BlockID,
			ClassID:        p.ClassID,
			LearningAreaID: p.LearningAreaID,
			TeacherID:      p.TeacherID,
			RoomIdentifier: p.RoomIdentifier,
		}
		result = append(result, a)
	}
	m.allocations = append(m.allocations, result...)
	return result, nil
}

func (m *mockRepo) UpdateAllocation(ctx context.Context, id, tenantID, schoolID string, p UpdateAllocationPayload) (*Allocation, error) {
	if m.err != nil {
		return nil, m.err
	}
	for i, a := range m.allocations {
		if a.ID == id {
			if p.LearningAreaID != "" {
				m.allocations[i].LearningAreaID = p.LearningAreaID
			}
			if p.TeacherID != "" {
				m.allocations[i].TeacherID = p.TeacherID
			}
			if p.RoomIdentifier != nil {
				m.allocations[i].RoomIdentifier = p.RoomIdentifier
			}
			return &m.allocations[i], nil
		}
	}
	return nil, ErrNotFound
}

func (m *mockRepo) DeleteAllocation(ctx context.Context, id, tenantID, schoolID string) error {
	if m.err != nil {
		return m.err
	}
	for i, a := range m.allocations {
		if a.ID == id {
			m.allocations = append(m.allocations[:i], m.allocations[i+1:]...)
			return nil
		}
	}
	return ErrNotFound
}

func (m *mockRepo) ListTracks(ctx context.Context, tenantID, schoolID, yearID string) ([]Track, error) {
	if m.err != nil {
		return nil, m.err
	}
	return []Track{}, nil
}

func (m *mockRepo) UpdateBlockPeriod(ctx context.Context, tenantID, schoolID string, p UpdatePeriodPayload) ([]TimeBlock, error) {
	if m.err != nil {
		return nil, m.err
	}
	return []TimeBlock{}, nil
}

func (m *mockRepo) DeleteBlockPeriod(ctx context.Context, tenantID, schoolID string, p DeletePeriodPayload) (*DeleteResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &DeleteResult{Deleted: true}, nil
}

func (m *mockRepo) CreateTrack(ctx context.Context, tenantID, schoolID, academicYearID, academicTermID, name, description string, isDefault bool) (*Track, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &Track{ID: "t1", Name: name}, nil
}
func (m *mockRepo) UpdateTrack(ctx context.Context, id, tenantID, schoolID string, p UpdateTrackPayload) (*Track, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &Track{ID: id}, nil
}
func (m *mockRepo) DeleteTrack(ctx context.Context, id, tenantID, schoolID string) error {
	return m.err
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
		TrackID:    "track1",
		DayOfWeek:  1,
		PeriodName: "Lesson 1",
		StartTime:  "08:00",
		EndTime:    "08:40",
		IsBreak:    false,
	})
	require.NoError(t, err)
	require.Equal(t, "new-id", block.ID)
	require.Equal(t, "Lesson 1", block.PeriodName)
	require.Equal(t, 1, block.DayOfWeek)
	require.Equal(t, "track1", block.TrackID)
}

func TestServiceImpl_CreateBlock_Error(t *testing.T) {
	m := &mockRepo{err: ErrBlockOverlap}
	s := newService(m)

	_, err := s.CreateBlock(context.Background(), "t", "s", CreateTimeBlockPayload{
		TrackID:    "track1",
		DayOfWeek:  1,
		PeriodName: "Lesson 1",
		StartTime:  "08:00",
		EndTime:    "08:40",
		IsBreak:    false,
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrBlockOverlap)
}

func TestServiceImpl_UpdateBlock(t *testing.T) {
	m := &mockRepo{blocks: []TimeBlock{{ID: "1", PeriodName: "Old"}}}
	s := newService(m)

	block, err := s.UpdateBlock(context.Background(), "1", "t", "s", UpdateTimeBlockPayload{
		TrackID:    "track1",
		DayOfWeek:  2,
		PeriodName: "New",
		StartTime:  "09:00",
		EndTime:    "09:40",
		IsBreak:    true,
	})
	require.NoError(t, err)
	require.Equal(t, "1", block.ID)
	require.Equal(t, "New", block.PeriodName)
	require.Equal(t, 2, block.DayOfWeek)
	require.True(t, block.IsBreak)
	require.Equal(t, "track1", block.TrackID)
}

func TestServiceImpl_UpdateBlock_NotFound(t *testing.T) {
	m := &mockRepo{}
	s := newService(m)

	_, err := s.UpdateBlock(context.Background(), "missing", "t", "s", UpdateTimeBlockPayload{
		TrackID:    "track1",
		DayOfWeek:  1,
		PeriodName: "Lesson 1",
		StartTime:  "08:00",
		EndTime:    "08:40",
		IsBreak:    false,
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestServiceImpl_UpdateBlock_Error(t *testing.T) {
	m := &mockRepo{err: ErrBlockOverlap}
	s := newService(m)

	_, err := s.UpdateBlock(context.Background(), "1", "t", "s", UpdateTimeBlockPayload{
		TrackID:    "track1",
		DayOfWeek:  1,
		PeriodName: "Lesson 1",
		StartTime:  "08:00",
		EndTime:    "08:40",
		IsBreak:    false,
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

func TestServiceImpl_ListAllocations(t *testing.T) {
	m := &mockRepo{allocations: []Allocation{{ID: "1", ClassID: "c1"}}}
	s := newService(m)

	allocs, err := s.ListAllocations(context.Background(), AllocationFilter{TenantID: "t", SchoolID: "s"})
	require.NoError(t, err)
	require.Len(t, allocs, 1)
	require.Equal(t, "c1", allocs[0].ClassID)
}

func TestServiceImpl_ListAllocations_Error(t *testing.T) {
	m := &mockRepo{err: errors.New("db error")}
	s := newService(m)

	_, err := s.ListAllocations(context.Background(), AllocationFilter{TenantID: "t", SchoolID: "s"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "timetable.ServiceImpl.ListAllocations")
}

func TestServiceImpl_GetAllocation(t *testing.T) {
	m := &mockRepo{allocations: []Allocation{{ID: "1", ClassID: "c1", TeacherID: "t1"}}}
	s := newService(m)

	alloc, err := s.GetAllocation(context.Background(), "1", "t", "s")
	require.NoError(t, err)
	require.Equal(t, "1", alloc.ID)
	require.Equal(t, "c1", alloc.ClassID)
	require.Equal(t, "t1", alloc.TeacherID)
}

func TestServiceImpl_GetAllocation_NotFound(t *testing.T) {
	m := &mockRepo{}
	s := newService(m)

	_, err := s.GetAllocation(context.Background(), "missing", "t", "s")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestServiceImpl_GetAllocation_Error(t *testing.T) {
	m := &mockRepo{err: errors.New("db error")}
	s := newService(m)

	_, err := s.GetAllocation(context.Background(), "1", "t", "s")
	require.Error(t, err)
	require.Contains(t, err.Error(), "timetable.ServiceImpl.GetAllocation")
}

func TestServiceImpl_CreateAllocation(t *testing.T) {
	m := &mockRepo{allocID: "new-alloc"}
	s := newService(m)

	alloc, err := s.CreateAllocation(context.Background(), "t", "s", CreateAllocationPayload{
		BlockID:        "struct1",
		ClassID:        "c1",
		LearningAreaID: "la1",
		TeacherID:      "t1",
		RoomIdentifier: ptr("Room 101"),
	})
	require.NoError(t, err)
	require.Equal(t, "new-alloc", alloc.ID)
	require.Equal(t, "t", alloc.TenantID)
	require.Equal(t, "s", alloc.SchoolID)
	require.Equal(t, "struct1", alloc.BlockID)
	require.Equal(t, "c1", alloc.ClassID)
	require.Equal(t, "la1", alloc.LearningAreaID)
	require.Equal(t, "t1", alloc.TeacherID)
	require.NotNil(t, alloc.RoomIdentifier)
	require.Equal(t, "Room 101", *alloc.RoomIdentifier)
}

func TestServiceImpl_CreateAllocation_Error(t *testing.T) {
	m := &mockRepo{err: ErrTeacherDoubleBooked}
	s := newService(m)

	_, err := s.CreateAllocation(context.Background(), "t", "s", CreateAllocationPayload{
		BlockID:        "struct1",
		ClassID:        "c1",
		LearningAreaID: "la1",
		TeacherID:      "t1",
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrTeacherDoubleBooked)
}

func TestServiceImpl_BatchCreateAllocations(t *testing.T) {
	m := &mockRepo{}
	s := newService(m)

	allocs, err := s.BatchCreateAllocations(context.Background(), "t", "s", []CreateAllocationPayload{
		{BlockID: "s1", ClassID: "c1", LearningAreaID: "la1", TeacherID: "t1"},
		{BlockID: "s2", ClassID: "c2", LearningAreaID: "la2", TeacherID: "t2"},
	})
	require.NoError(t, err)
	require.Len(t, allocs, 2)
	require.Equal(t, "s1", allocs[0].BlockID)
	require.Equal(t, "c1", allocs[0].ClassID)
	require.Equal(t, "s2", allocs[1].BlockID)
	require.Equal(t, "c2", allocs[1].ClassID)
}

func TestServiceImpl_BatchCreateAllocations_Empty(t *testing.T) {
	m := &mockRepo{}
	s := newService(m)

	allocs, err := s.BatchCreateAllocations(context.Background(), "t", "s", []CreateAllocationPayload{})
	require.NoError(t, err)
	require.Empty(t, allocs)
}

func TestServiceImpl_BatchCreateAllocations_Error(t *testing.T) {
	m := &mockRepo{err: ErrConflict}
	s := newService(m)

	_, err := s.BatchCreateAllocations(context.Background(), "t", "s", []CreateAllocationPayload{
		{BlockID: "s1", ClassID: "c1", LearningAreaID: "la1", TeacherID: "t1"},
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrConflict)
}

func TestServiceImpl_UpdateAllocation(t *testing.T) {
	m := &mockRepo{allocations: []Allocation{{ID: "1", LearningAreaID: "old", TeacherID: "old", RoomIdentifier: ptr("Old")}}}
	s := newService(m)

	newRoom := "New Room"
	alloc, err := s.UpdateAllocation(context.Background(), "1", "t", "s", UpdateAllocationPayload{
		LearningAreaID: "new",
		TeacherID:      "newt",
		RoomIdentifier: &newRoom,
	})
	require.NoError(t, err)
	require.Equal(t, "1", alloc.ID)
	require.Equal(t, "new", alloc.LearningAreaID)
	require.Equal(t, "newt", alloc.TeacherID)
	require.NotNil(t, alloc.RoomIdentifier)
	require.Equal(t, "New Room", *alloc.RoomIdentifier)
}

func TestServiceImpl_UpdateAllocation_NotFound(t *testing.T) {
	m := &mockRepo{}
	s := newService(m)

	_, err := s.UpdateAllocation(context.Background(), "missing", "t", "s", UpdateAllocationPayload{
		LearningAreaID: "la1",
		TeacherID:      "t1",
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestServiceImpl_UpdateAllocation_Error(t *testing.T) {
	m := &mockRepo{err: ErrTeacherDoubleBooked}
	s := newService(m)

	_, err := s.UpdateAllocation(context.Background(), "1", "t", "s", UpdateAllocationPayload{
		LearningAreaID: "la1",
		TeacherID:      "t1",
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrTeacherDoubleBooked)
}

func TestServiceImpl_DeleteAllocation(t *testing.T) {
	m := &mockRepo{allocations: []Allocation{{ID: "1", ClassID: "c1"}}}
	s := newService(m)

	err := s.DeleteAllocation(context.Background(), "1", "t", "s")
	require.NoError(t, err)
}

func TestServiceImpl_DeleteAllocation_NotFound(t *testing.T) {
	m := &mockRepo{}
	s := newService(m)

	err := s.DeleteAllocation(context.Background(), "missing", "t", "s")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestServiceImpl_DeleteAllocation_Error(t *testing.T) {
	m := &mockRepo{err: errors.New("db error")}
	s := newService(m)

	err := s.DeleteAllocation(context.Background(), "1", "t", "s")
	require.Error(t, err)
	require.Contains(t, err.Error(), "timetable.ServiceImpl.DeleteAllocation")
}
