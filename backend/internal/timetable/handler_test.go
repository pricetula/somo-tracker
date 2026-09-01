package timetable

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Mock Service
// ============================================================================

type mockService struct {
	svcErr error

	// Track
	createTrackFn     func(ctx context.Context, tenantID, schoolID, yearID, termID, name, description string, isDefault bool) (*Track, error)
	updateTrackFn     func(ctx context.Context, id, tenantID, schoolID string, p UpdateTrackPayload) (*Track, error)
	deleteTrackFn     func(ctx context.Context, id, tenantID, schoolID string) (*DeleteResult, error)
	bulkDeleteTrackFn func(ctx context.Context, ids []string, tenantID, schoolID string) (*DeleteResult, error)

	// Blocks
	createBlockFn     func(ctx context.Context, tenantID, schoolID string, p CreateTimeBlockPayload) (*TimeBlock, error)
	updateBlockFn     func(ctx context.Context, id, tenantID, schoolID string, p UpdateTimeBlockPayload) (*TimeBlock, error)
	deleteBlockFn     func(ctx context.Context, id, tenantID, schoolID string) (*DeleteResult, error)
	bulkDeleteBlockFn func(ctx context.Context, ids []string, tenantID, schoolID string) (*DeleteResult, error)
	listBlocksFn      func(ctx context.Context, tenantID, schoolID, yearID string) ([]TimeBlock, error)
	getBlockFn        func(ctx context.Context, id, tenantID, schoolID string) (*TimeBlock, error)

	// Allocations
	createAllocationFn       func(ctx context.Context, tenantID, schoolID string, p CreateAllocationPayload) (*Allocation, error)
	updateAllocationFn       func(ctx context.Context, id, tenantID, schoolID string, p UpdateAllocationPayload) (*Allocation, error)
	deleteAllocationFn       func(ctx context.Context, id, tenantID, schoolID string) error
	bulkDeleteAllocationFn   func(ctx context.Context, ids []string, tenantID, schoolID string) (*DeleteResult, error)
	listAllocationsFn        func(ctx context.Context, f AllocationFilter) ([]Allocation, error)
	getAllocationFn          func(ctx context.Context, id, tenantID, schoolID string) (*Allocation, error)
	batchCreateAllocationsFn func(ctx context.Context, tenantID, schoolID string, ps []CreateAllocationPayload) ([]Allocation, error)
}

func (m *mockService) CreateTrack(ctx context.Context, tenantID, schoolID, yearID, termID, name, description string, isDefault bool) (*Track, error) {
	if m.createTrackFn != nil {
		return m.createTrackFn(ctx, tenantID, schoolID, yearID, termID, name, description, isDefault)
	}
	if m.svcErr != nil {
		return nil, m.svcErr
	}
	return &Track{ID: "track_1", Name: name}, nil
}

func (m *mockService) UpdateTrack(ctx context.Context, id, tenantID, schoolID string, p UpdateTrackPayload) (*Track, error) {
	if m.updateTrackFn != nil {
		return m.updateTrackFn(ctx, id, tenantID, schoolID, p)
	}
	if m.svcErr != nil {
		return nil, m.svcErr
	}
	return &Track{ID: id}, nil
}

func (m *mockService) DeleteTrack(ctx context.Context, id, tenantID, schoolID string) (*DeleteResult, error) {
	if m.deleteTrackFn != nil {
		return m.deleteTrackFn(ctx, id, tenantID, schoolID)
	}
	if m.svcErr != nil {
		return nil, m.svcErr
	}
	return &DeleteResult{Deleted: true}, nil
}

func (m *mockService) BulkDeleteTracks(ctx context.Context, ids []string, tenantID, schoolID string) (*DeleteResult, error) {
	if m.bulkDeleteTrackFn != nil {
		return m.bulkDeleteTrackFn(ctx, ids, tenantID, schoolID)
	}
	if m.svcErr != nil {
		return nil, m.svcErr
	}
	return &DeleteResult{Deleted: true}, nil
}

func (m *mockService) CreateBlock(ctx context.Context, tenantID, schoolID string, p CreateTimeBlockPayload) (*TimeBlock, error) {
	if m.createBlockFn != nil {
		return m.createBlockFn(ctx, tenantID, schoolID, p)
	}
	if m.svcErr != nil {
		return nil, m.svcErr
	}
	return &TimeBlock{ID: "block_1", TrackID: p.TrackID}, nil
}

func (m *mockService) UpdateBlock(ctx context.Context, id, tenantID, schoolID string, p UpdateTimeBlockPayload) (*TimeBlock, error) {
	if m.updateBlockFn != nil {
		return m.updateBlockFn(ctx, id, tenantID, schoolID, p)
	}
	if m.svcErr != nil {
		return nil, m.svcErr
	}
	return &TimeBlock{ID: id}, nil
}

func (m *mockService) DeleteBlock(ctx context.Context, id, tenantID, schoolID string) (*DeleteResult, error) {
	if m.deleteBlockFn != nil {
		return m.deleteBlockFn(ctx, id, tenantID, schoolID)
	}
	if m.svcErr != nil {
		return nil, m.svcErr
	}
	return &DeleteResult{Deleted: true}, nil
}

func (m *mockService) BulkDeleteBlocks(ctx context.Context, ids []string, tenantID, schoolID string) (*DeleteResult, error) {
	if m.bulkDeleteBlockFn != nil {
		return m.bulkDeleteBlockFn(ctx, ids, tenantID, schoolID)
	}
	if m.svcErr != nil {
		return nil, m.svcErr
	}
	return &DeleteResult{Deleted: true}, nil
}

func (m *mockService) ListBlocks(ctx context.Context, tenantID, schoolID, yearID string) ([]TimeBlock, error) {
	if m.listBlocksFn != nil {
		return m.listBlocksFn(ctx, tenantID, schoolID, yearID)
	}
	if m.svcErr != nil {
		return nil, m.svcErr
	}
	return []TimeBlock{}, nil
}

func (m *mockService) GetBlock(ctx context.Context, id, tenantID, schoolID string) (*TimeBlock, error) {
	if m.getBlockFn != nil {
		return m.getBlockFn(ctx, id, tenantID, schoolID)
	}
	if m.svcErr != nil {
		return nil, m.svcErr
	}
	return nil, ErrNotFound
}

func (m *mockService) CreateAllocation(ctx context.Context, tenantID, schoolID string, p CreateAllocationPayload) (*Allocation, error) {
	if m.createAllocationFn != nil {
		return m.createAllocationFn(ctx, tenantID, schoolID, p)
	}
	if m.svcErr != nil {
		return nil, m.svcErr
	}
	return &Allocation{ID: "alloc_1"}, nil
}

func (m *mockService) BatchCreateAllocations(ctx context.Context, tenantID, schoolID string, ps []CreateAllocationPayload) ([]Allocation, error) {
	if m.batchCreateAllocationsFn != nil {
		return m.batchCreateAllocationsFn(ctx, tenantID, schoolID, ps)
	}
	if m.svcErr != nil {
		return nil, m.svcErr
	}
	return []Allocation{{ID: "alloc_1"}}, nil
}

func (m *mockService) UpdateAllocation(ctx context.Context, id, tenantID, schoolID string, p UpdateAllocationPayload) (*Allocation, error) {
	if m.updateAllocationFn != nil {
		return m.updateAllocationFn(ctx, id, tenantID, schoolID, p)
	}
	if m.svcErr != nil {
		return nil, m.svcErr
	}
	return &Allocation{ID: id}, nil
}

func (m *mockService) DeleteAllocation(ctx context.Context, id, tenantID, schoolID string) error {
	if m.deleteAllocationFn != nil {
		return m.deleteAllocationFn(ctx, id, tenantID, schoolID)
	}
	if m.svcErr != nil {
		return m.svcErr
	}
	return nil
}

func (m *mockService) BulkDeleteAllocations(ctx context.Context, ids []string, tenantID, schoolID string) (*DeleteResult, error) {
	if m.bulkDeleteAllocationFn != nil {
		return m.bulkDeleteAllocationFn(ctx, ids, tenantID, schoolID)
	}
	if m.svcErr != nil {
		return nil, m.svcErr
	}
	return &DeleteResult{Deleted: true}, nil
}

func (m *mockService) ListAllocations(ctx context.Context, f AllocationFilter) ([]Allocation, error) {
	if m.listAllocationsFn != nil {
		return m.listAllocationsFn(ctx, f)
	}
	if m.svcErr != nil {
		return nil, m.svcErr
	}
	return []Allocation{}, nil
}

func (m *mockService) GetAllocation(ctx context.Context, id, tenantID, schoolID string) (*Allocation, error) {
	if m.getAllocationFn != nil {
		return m.getAllocationFn(ctx, id, tenantID, schoolID)
	}
	if m.svcErr != nil {
		return nil, m.svcErr
	}
	return nil, ErrNotFound
}

func (m *mockService) GetTrack(ctx context.Context, id, tenantID, schoolID string) (*Track, error) {
	if m.svcErr != nil {
		return nil, m.svcErr
	}
	return &Track{ID: id}, nil
}

func (m *mockService) ListTracks(ctx context.Context, tenantID, schoolID, yearID string) ([]Track, error) {
	if m.svcErr != nil {
		return nil, m.svcErr
	}
	return []Track{}, nil
}

func (m *mockService) UpdateBlockPeriod(ctx context.Context, tenantID, schoolID string, p UpdatePeriodPayload) ([]TimeBlock, error) {
	if m.svcErr != nil {
		return nil, m.svcErr
	}
	return []TimeBlock{}, nil
}

func (m *mockService) DeleteBlockPeriod(ctx context.Context, tenantID, schoolID string, p DeletePeriodPayload) (*DeleteResult, error) {
	if m.svcErr != nil {
		return nil, m.svcErr
	}
	return &DeleteResult{Deleted: true}, nil
}

func (m *mockService) GetTimetable(ctx context.Context, tenantID, schoolID string) ([]TimeBlock, []Allocation, error) {
	if m.svcErr != nil {
		return nil, nil, m.svcErr
	}
	return []TimeBlock{}, []Allocation{}, nil
}

// ============================================================================
// Handler Test Harness
// ============================================================================

type handlerTestHarness struct {
	app     *fiber.App
	svc     *mockService
	handler *Handler
}

func newHandlerTestHarness() *handlerTestHarness {
	svc := &mockService{}
	handler := NewHandler(svc)
	// Note: Not setting academic years service for unit tests;
	// resolution tests would need integration setup

	app := fiber.New()

	// Test middleware that sets auth locals (bypasses requireAuth)
	testAuth := func(c *fiber.Ctx) error {
		c.Locals("tenant_id", "tenant_001")
		c.Locals("user_id", "user_001")
		c.Locals("active_school_id", "school_001")
		return c.Next()
	}

	// Register handler methods directly with test auth (bypassing RequireAuth)
	base := app.Group("", testAuth)
	base.Post("/api/v1/timetable", handler.CreateTrackWithBlocks)
	base.Put("/api/v1/timetable", handler.UpdateTrack)
	base.Delete("/api/v1/timetable", handler.BulkDeleteTracks)
	base.Post("/api/v1/timetable/blocks", handler.CreateBlocks)
	base.Put("/api/v1/timetable/blocks", handler.UpdateBlock)
	base.Delete("/api/v1/timetable/blocks", handler.BulkDeleteBlocks)
	base.Post("/api/v1/timetable/allocations", handler.CreateAllocations)
	base.Put("/api/v1/timetable/allocations", handler.UpdateAllocation)
	base.Delete("/api/v1/timetable/allocations", handler.BulkDeleteAllocations)
	base.Get("/api/v1/timetable", handler.GetTimetable)
	base.Get("/api/v1/timetable/allocations/:id", handler.GetAllocation)

	return &handlerTestHarness{
		app:     app,
		svc:     svc,
		handler: handler,
	}
}

func doRequest(app *fiber.App, method, path string, body []byte) *http.Response {
	req := httptest.NewRequest(method, path, nil)
	if body != nil {
		req = httptest.NewRequest(method, path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	resp, _ := app.Test(req)
	return resp
}

func decodeError(t *testing.T, resp *http.Response) map[string]interface{} {
	t.Helper()
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	return result
}

// ============================================================================
// Helpers
// ============================================================================

func newValidTrackPayload() []byte {
	p := CreateTrackPayload{
		Name:        "Test Track",
		Description: "A test track",
		IsDefault:   false,
	}
	b, _ := json.Marshal(p)
	return b
}

func newValidUpdateTrackPayload() []byte {
	p := UpdateTrackPayload{
		ID:          "track_001",
		Name:        "Updated Track",
		Description: "Updated description",
	}
	b, _ := json.Marshal(p)
	return b
}

func newValidBlockPayload() []byte {
	p := CreateTimeBlockPayload{
		TrackID:    "track_001",
		DayOfWeek:  1,
		PeriodName: "Period 1",
		StartTime:  "08:00",
		EndTime:    "08:40",
		IsBreak:    false,
		OrderIndex: 1,
	}
	b, _ := json.Marshal(p)
	return b
}

func newValidUpdateBlockPayload() []byte {
	p := UpdateTimeBlockPayload{
		ID:         "block_001",
		TrackID:    "track_001",
		DayOfWeek:  1,
		PeriodName: "Period 1 Updated",
		StartTime:  "08:00",
		EndTime:    "08:40",
		IsBreak:    false,
		OrderIndex: 1,
	}
	b, _ := json.Marshal(p)
	return b
}

func newValidAllocationPayload() []byte {
	room := "Room A"
	p := CreateAllocationPayload{
		BlockID:        "block_001",
		ClassID:        "class_001",
		LearningAreaID: "la_001",
		TeacherID:      "teacher_001",
		RoomIdentifier: &room,
	}
	b, _ := json.Marshal(p)
	return b
}

func newValidUpdateAllocationPayload() []byte {
	room := "Room B"
	p := UpdateAllocationPayload{
		ID:             "alloc_001",
		LearningAreaID: "la_002",
		TeacherID:      "teacher_002",
		RoomIdentifier: &room,
	}
	b, _ := json.Marshal(p)
	return b
}

// ============================================================================
// Tests: POST /api/v1/timetable (CreateTrackWithBlocks)
// ============================================================================

func TestHandler_CreateTrack_HappyPath(t *testing.T) {
	h := newHandlerTestHarness()

	createdTrack := &Track{
		ID:             "track_001",
		Name:           "Test Track",
		Description:    "A test track",
		IsDefault:      false,
		TenantID:       "tenant_001",
		SchoolID:       "school_001",
		AcademicYearID: "year_001",
		AcademicTermID: "term_001",
	}
	h.svc.createTrackFn = func(ctx context.Context, tenantID, schoolID, yearID, termID, name, description string, isDefault bool) (*Track, error) {
		require.Equal(t, "tenant_001", tenantID)
		require.Equal(t, "school_001", schoolID)
		require.Equal(t, "year_001", yearID)
		require.Equal(t, "term_001", termID)
		require.Equal(t, "Test Track", name)
		return createdTrack, nil
	}

	resp := doRequest(h.app, "POST", "/api/v1/timetable", newValidTrackPayload())
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	require.Equal(t, "track created successfully", result["message"])
}

func TestHandler_CreateTrack_WithInitialBlocks(t *testing.T) {
	h := newHandlerTestHarness()

	createdTrack := &Track{
		ID:             "track_001",
		Name:           "Test Track",
		Description:    "A test track",
		IsDefault:      false,
		TenantID:       "tenant_001",
		SchoolID:       "school_001",
		AcademicYearID: "year_001",
		AcademicTermID: "term_001",
	}
	h.svc.createTrackFn = func(ctx context.Context, tenantID, schoolID, yearID, termID, name, description string, isDefault bool) (*Track, error) {
		return createdTrack, nil
	}
	h.svc.createBlockFn = func(ctx context.Context, tenantID, schoolID string, p CreateTimeBlockPayload) (*TimeBlock, error) {
		require.Equal(t, "track_001", p.TrackID)
		return &TimeBlock{ID: "block_001", TrackID: p.TrackID}, nil
	}

	payload := CreateTrackPayload{
		Name:        "Test Track",
		Description: "A test track",
		IsDefault:   false,
		InitialBlocks: []CreateTimeBlockPayload{
			{DayOfWeek: 1, PeriodName: "Period 1", StartTime: "08:00", EndTime: "08:40", IsBreak: false, OrderIndex: 1},
		},
	}
	body, _ := json.Marshal(payload)

	resp := doRequest(h.app, "POST", "/api/v1/timetable", body)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	require.Equal(t, "track created with initial blocks", result["message"])
}

func TestHandler_CreateTrack_ValidationError_MissingName(t *testing.T) {
	h := newHandlerTestHarness()

	payload := CreateTrackPayload{
		Name:        "",
		Description: "A test track",
		IsDefault:   false,
	}
	body, _ := json.Marshal(payload)

	resp := doRequest(h.app, "POST", "/api/v1/timetable", body)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	err := decodeError(t, resp)
	require.Equal(t, "VALIDATION_ERROR", err["code"])
	require.Equal(t, "track name is required", err["message"])
}

// ============================================================================
// Tests: PUT /api/v1/timetable (UpdateTrack)
// ============================================================================

func TestHandler_UpdateTrack_HappyPath(t *testing.T) {
	h := newHandlerTestHarness()

	updatedTrack := &Track{
		ID:          "track_001",
		Name:        "Updated Track",
		Description: "Updated description",
	}
	h.svc.updateTrackFn = func(ctx context.Context, id, tenantID, schoolID string, p UpdateTrackPayload) (*Track, error) {
		require.Equal(t, "track_001", id)
		require.Equal(t, "tenant_001", tenantID)
		require.Equal(t, "school_001", schoolID)
		require.Equal(t, "Updated Track", p.Name)
		return updatedTrack, nil
	}

	resp := doRequest(h.app, "PUT", "/api/v1/timetable", newValidUpdateTrackPayload())
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	require.True(t, result["updated"].(bool))
}

// ============================================================================
// Tests: DELETE /api/v1/timetable (BulkDeleteTracks)
// ============================================================================

func TestHandler_BulkDeleteTracks_HappyPath(t *testing.T) {
	h := newHandlerTestHarness()

	h.svc.bulkDeleteTrackFn = func(ctx context.Context, ids []string, tenantID, schoolID string) (*DeleteResult, error) {
		require.Equal(t, []string{"track_001", "track_002"}, ids)
		require.Equal(t, "tenant_001", tenantID)
		require.Equal(t, "school_001", schoolID)
		return &DeleteResult{Deleted: true}, nil
	}

	body, _ := json.Marshal(map[string][]string{"ids": {"track_001", "track_002"}})
	resp := doRequest(h.app, "DELETE", "/api/v1/timetable", body)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	require.Equal(t, float64(2), result["deleted"])
	require.Equal(t, float64(2), result["total"])
}

// ============================================================================
// Tests: POST /api/v1/timetable/blocks (CreateBlocks)
// ============================================================================

func TestHandler_CreateBlocks_HappyPath(t *testing.T) {
	h := newHandlerTestHarness()

	// Each input block should be replicated across 7 days (Monday-Sunday).
	daysSeen := make(map[int]int)
	h.svc.createBlockFn = func(ctx context.Context, tenantID, schoolID string, p CreateTimeBlockPayload) (*TimeBlock, error) {
		require.Equal(t, "tenant_001", tenantID)
		require.Equal(t, "school_001", schoolID)
		require.Equal(t, "track_001", p.TrackID)
		require.GreaterOrEqual(t, p.DayOfWeek, 1)
		require.LessOrEqual(t, p.DayOfWeek, 7)
		daysSeen[p.DayOfWeek]++
		return &TimeBlock{ID: "block_001", TrackID: p.TrackID, DayOfWeek: p.DayOfWeek}, nil
	}

	payload := []CreateTimeBlockPayload{
		{TrackID: "track_001", DayOfWeek: 1, PeriodName: "Period 1", StartTime: "08:00", EndTime: "08:40", IsBreak: false, OrderIndex: 1},
	}
	body, _ := json.Marshal(payload)

	resp := doRequest(h.app, "POST", "/api/v1/timetable/blocks", body)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	require.NotNil(t, result["blocks"])
	require.Equal(t, float64(7), result["replicated_days"])

	// One block was provided, replicated to 7 days
	require.Len(t, daysSeen, 7)
	for day := 1; day <= 7; day++ {
		require.Equal(t, 1, daysSeen[day], "day %d should have been created once", day)
	}
}

func TestHandler_CreateBlocks_ReplicatesMultipleBlocksAcrossAllDays(t *testing.T) {
	h := newHandlerTestHarness()

	// Two periods * 7 days = 14 block creations
	periodsSeen := make(map[string]map[int]int)
	h.svc.createBlockFn = func(ctx context.Context, tenantID, schoolID string, p CreateTimeBlockPayload) (*TimeBlock, error) {
		require.Equal(t, "track_001", p.TrackID)
		require.GreaterOrEqual(t, p.DayOfWeek, 1)
		require.LessOrEqual(t, p.DayOfWeek, 7)
		if periodsSeen[p.PeriodName] == nil {
			periodsSeen[p.PeriodName] = make(map[int]int)
		}
		periodsSeen[p.PeriodName][p.DayOfWeek]++
		return &TimeBlock{ID: "block_001", TrackID: p.TrackID, PeriodName: p.PeriodName, DayOfWeek: p.DayOfWeek}, nil
	}

	payload := []CreateTimeBlockPayload{
		{TrackID: "track_001", PeriodName: "Period 1", StartTime: "08:00", EndTime: "08:40", IsBreak: false, OrderIndex: 1},
		{TrackID: "track_001", PeriodName: "Period 2", StartTime: "08:40", EndTime: "09:20", IsBreak: false, OrderIndex: 2},
	}
	body, _ := json.Marshal(payload)

	resp := doRequest(h.app, "POST", "/api/v1/timetable/blocks", body)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// Each period should be present for every day of the week
	require.Len(t, periodsSeen, 2)
	for _, periodName := range []string{"Period 1", "Period 2"} {
		require.Len(t, periodsSeen[periodName], 7, "%s should exist on all 7 days", periodName)
		for day := 1; day <= 7; day++ {
			require.Equal(t, 1, periodsSeen[periodName][day], "%s on day %d should have been created once", periodName, day)
		}
	}
}

// ============================================================================
// Tests: GET /api/v1/timetable (GetTimetable)
// ============================================================================

func TestHandler_GetAllocation_HappyPath(t *testing.T) {
	h := newHandlerTestHarness()

	h.svc.getAllocationFn = func(ctx context.Context, id, tenantID, schoolID string) (*Allocation, error) {
		require.Equal(t, "alloc_1", id)
		require.Equal(t, "tenant_001", tenantID)
		require.Equal(t, "school_001", schoolID)
		return &Allocation{
			ID:               id,
			TenantID:         tenantID,
			SchoolID:         schoolID,
			BlockID:          "block_1",
			ClassID:          "class_1",
			LearningAreaID:   "la_1",
			TeacherID:        "teacher_1",
			ClassName:        "Grade 10A",
			LearningAreaName: "Mathematics",
			TeacherName:      "Jane Doe",
			RoomName:         "Room 101",
		}, nil
	}

	resp := doRequest(h.app, http.MethodGet, "/api/v1/timetable/allocations/alloc_1", nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result Allocation
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	require.Equal(t, "alloc_1", result.ID)
	require.Equal(t, "Grade 10A", result.ClassName)
	require.Equal(t, "Mathematics", result.LearningAreaName)
	require.Equal(t, "Jane Doe", result.TeacherName)
}

func TestHandler_GetAllocation_NotFound(t *testing.T) {
	h := newHandlerTestHarness()

	h.svc.getAllocationFn = func(ctx context.Context, id, tenantID, schoolID string) (*Allocation, error) {
		return nil, ErrNotFound
	}

	resp := doRequest(h.app, http.MethodGet, "/api/v1/timetable/allocations/missing", nil)
	require.NotEqual(t, http.StatusOK, resp.StatusCode)
}

func TestHandler_GetTimetable_HappyPath(t *testing.T) {
	h := newHandlerTestHarness()

	h.svc.listBlocksFn = func(ctx context.Context, tenantID, schoolID, yearID string) ([]TimeBlock, error) {
		require.Equal(t, "tenant_001", tenantID)
		require.Equal(t, "school_001", schoolID)
		return []TimeBlock{{ID: "block_001", PeriodName: "Period 1"}}, nil
	}
	h.svc.listAllocationsFn = func(ctx context.Context, f AllocationFilter) ([]Allocation, error) {
		require.Equal(t, "tenant_001", f.TenantID)
		require.Equal(t, "school_001", f.SchoolID)
		return []Allocation{{ID: "alloc_001"}}, nil
	}

	resp := doRequest(h.app, "GET", "/api/v1/timetable", nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	require.Len(t, result["blocks"].([]interface{}), 1)
	require.Len(t, result["allocations"].([]interface{}), 1)
}
