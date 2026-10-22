package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSetupClassTimetableSlot_Validation tests pure validation without DB.
func TestSetupClassTimetableSlot_Validation(t *testing.T) {
	svc := NewTimetableService(nil, nil).(*timetableService)
	_ = svc // service requires nil pool/queries for validation tests

	tests := []struct {
		name       string
		dayOfWeek  int
		wantErrSub string
	}{
		{"day too low", 0, "day_of_week must be between 1 and 7"},
		{"day too high", 8, "day_of_week must be between 1 and 7"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use a service with nil deps – SetupClassTimetableSlot will fail on day validation before DB
			err := svc.SetupClassTimetableSlot(context.Background(), [16]byte{}, "invalid-uuid", "invalid-uuid", "invalid-uuid", "invalid-uuid", tt.dayOfWeek, "")
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantErrSub)
		})
	}
}

// TestSetupClassTimetableSlot_Conflicts is a placeholder for conflict detection tests.
// The real implementation requires a test DB and mocks for sqlc.Queries and pgxpool.Pool.
// Test cases from FEATURE-timetable-conflict-detection.md Test Matrix:
// T-01 Teacher double-book -> 409 timetable_conflict:teacher_double_book
// T-02 Room double-book -> 409 timetable_conflict:room_double_book
// T-03 Class overlap -> 409 timetable_conflict:class_overlap
// T-04 Happy path -> success
func TestSetupClassTimetableSlot_Conflicts(t *testing.T) {
	t.Run("teacher_double_book", func(t *testing.T) {
		// TODO: spin up test container, seed existing slot, call SetupClassTimetableSlot
		// require.ErrorContains(t, err, "timetable_conflict:teacher_double_book")
	})
	t.Run("room_double_book", func(t *testing.T) {
		// TODO
	})
	t.Run("class_overlap", func(t *testing.T) {
		// TODO
	})
	t.Run("happy_path", func(t *testing.T) {
		// TODO
	})
}

// Note: Unit tests for pure validation logic should be extracted to a pure ValidateTimetableSlot function
// to avoid DB dependencies. Integration tests can use the existing Testcontainers setup in repo.
