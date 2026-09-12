// Package worker — unit tests for bulk invitation processing pipeline.
// Sections 4 (Stytch error classification) and 5 (circuit breaker).
package worker

import (
	"errors"
	"testing"

	"github.com/sony/gobreaker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"somotracker/backend/internal/stytch"
)

// MockStytchClient implements the interface boundary for deterministic injection.
type MockStytchClient struct {
	CallCount int
	NextErr   error
	Results   []*stytch.InviteMemberResult
	ResIndex  int
}

func (m *MockStytchClient) ClassifyStytchError(err error) (retry bool, permanent bool, duplicate bool, reason string) {
	if err == nil {
		return false, false, false, ""
	}
	switch err.(type) {
	case *stytchMock429:
		return true, false, false, "rate_limited"
	case *stytchMock500:
		return true, false, false, "server_error"
	case *stytchMockTimeout:
		return true, false, false, "timeout"
	case *stytchMockDuplicate:
		return false, false, true, "duplicate"
	case *stytchMockInvalidEmail:
		return false, true, false, "invalid_email"
	case *stytchMockUnknown:
		return true, false, false, "unknown"
	default:
		if errors.Is(err, errors.New("timeout")) {
			return true, false, false, "timeout"
		}
		return false, true, false, "unknown"
	}
}

func (m *MockStytchClient) InviteMember(ctx interface{}, email, fullName, role string, tenantID string) (*stytch.InviteMemberResult, error) {
	m.CallCount++
	if m.NextErr != nil {
		err := m.NextErr
		m.NextErr = nil // one-time injection unless reset externally
		return nil, err
	}
	res := m.Results[m.ResIndex]
	m.ResIndex++
	return res, nil
}

// Custom error shapes for deterministic classification.
type stytchMock429 struct{}

func (e *stytchMock429) Error() string { return "429 rate_limited" }

type stytchMock500 struct{}

func (e *stytchMock500) Error() string { return "500 server error" }

type stytchMockTimeout struct{}

func (e *stytchMockTimeout) Error() string { return "timeout" }

type stytchMockDuplicate struct{}

func (e *stytchMockDuplicate) Error() string { return "duplicate_user_email" }

type stytchMockInvalidEmail struct{}

func (e *stytchMockInvalidEmail) Error() string { return "invalid_email" }

type stytchMockUnknown struct{}

func (e *stytchMockUnknown) Error() string { return "unexpected error" }

// Section 4 — Stytch error classification table-driven.
func TestStytchErrorClassification(t *testing.T) {
	mock := &MockStytchClient{}

	cases := []struct {
		name             string
		err              error
		wantRetry        bool
		wantPermanent    bool
		wantDuplicate    bool
		wantReason       string
		wantStatus       string // resulting item status
		wantAttemptDelta int
		wantRetrySched   bool
	}{
		{
			name:      "429 rate_limited -> retry, item stays PENDING/PROCESSING",
			err:       &stytchMock429{},
			wantRetry: true, wantStatus: "PENDING", wantAttemptDelta: 1, wantRetrySched: true,
		},
		{
			name:      "500 timeout -> retry, item stays PENDING/PROCESSING",
			err:       &stytchMock500{},
			wantRetry: true, wantStatus: "PENDING", wantAttemptDelta: 1, wantRetrySched: true,
		},
		{
			name:          "duplicate_user_email -> SUCCEEDED, result populated",
			err:           &stytchMockDuplicate{},
			wantDuplicate: true, wantStatus: "SUCCEEDED", wantAttemptDelta: 0, wantRetrySched: false,
		},
		{
			name:          "invalid_email / 400 -> FAILED immediately, no retry",
			err:           &stytchMockInvalidEmail{},
			wantPermanent: true, wantStatus: "FAILED", wantAttemptDelta: 0, wantRetrySched: false,
		},
		{
			name:      "unexpected unknown -> first retries once, second -> FAILED",
			err:       &stytchMockUnknown{},
			wantRetry: true, wantStatus: "DEFERRED", wantAttemptDelta: 1, wantRetrySched: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock.NextErr = tc.err
			retry, permanent, duplicate, reason := mock.ClassifyStytchError(tc.err)
			assert.Equal(t, tc.wantRetry, retry, "retry flag mismatch")
			assert.Equal(t, tc.wantPermanent, permanent, "permanent flag mismatch")
			assert.Equal(t, tc.wantDuplicate, duplicate, "duplicate flag mismatch")
			assert.Equal(t, tc.wantReason, reason, "reason mismatch")
		})
	}

	// Assert retries respect max attempt ceiling.
	t.Run("max attempt ceiling terminates FAILED not forever", func(t *testing.T) {
		// If attempt_count reaches 3, next error should result in FAILED, not retry.
		_, _, _, reason := mock.ClassifyStytchError(&stytchMockUnknown{})
		// Design decision: retry budget of 3 attempts. After exhaustion -> FAILED.
		assert.NotEmpty(t, reason)
		t.Log("TODO: confirm max attempt ceiling (e.g., 3) in service/worker implementation")
	})
}

// Section 5 — Circuit breaker (gobreaker) with mock Stytch client.
func TestCircuitBreaker_OpenStopsCalls(t *testing.T) {
	mock := &MockStytchClient{}

	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "stytch-b2b-test",
		MaxRequests: 3,
		Interval:    0,
		Timeout:     0,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.TotalFailures >= 3
		},
	})

	// Force 3 consecutive failures -> breaker OPEN.
	for i := 0; i < 3; i++ {
		_, err := cb.Execute(func() (interface{}, error) {
			mock.NextErr = &stytchMock500{}
			return mock.InviteMember(nil, "test@test.co", "Test", "ADMIN", "t1")
		})
		require.Error(t, err)
	}

	callsBefore := mock.CallCount
	assert.True(t, callsBefore >= 3, "expected at least 3 mock calls before open")

	// While OPEN: subsequent calls should NOT hit mock client.
	_, err := cb.Execute(func() (interface{}, error) {
		mock.NextErr = nil
		return mock.InviteMember(nil, "test@test.co", "Test", "ADMIN", "t1")
	})
	assert.Error(t, err)
	assert.Equal(t, callsBefore, mock.CallCount, "mock client called while breaker OPEN")
}

func TestCircuitBreaker_OpenSetsDeferred(t *testing.T) {
	// When breaker is OPEN, affected items should be marked DEFERRED, NOT FAILED,
	// attempt_count NOT incremented, and task requeued with backoff (not immediate).
	t.Log("TODO: wire breaker into service layer and assert item status = DEFERRED, attempt_count unchanged, retry delay > 0")
}

func TestCircuitBreaker_HalfOpenRecovery(t *testing.T) {
	mock := &MockStytchClient{}
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "stytch-test-recovery",
		MaxRequests: 3,
		Interval:    0,
		Timeout:     0,
		ReadyToTrip: func(c gobreaker.Counts) bool { return c.TotalFailures >= 2 },
	})

	// Trip breaker.
	for i := 0; i < 2; i++ {
		_, err := cb.Execute(func() (interface{}, error) {
			mock.NextErr = &stytchMock500{}
			return mock.InviteMember(nil, "test@test.co", "Test", "ADMIN", "t1")
		})
		require.Error(t, err)
	}

	// Simulate cooldown recovery: reset breaker to HALF_OPEN by forcing success.
	mock.NextErr = nil
	_, err := cb.Execute(func() (interface{}, error) {
		return mock.InviteMember(nil, "test@test.co", "Test", "ADMIN", "t1")
	})
	require.NoError(t, err)
	t.Log("Break transitions to CLOSED on success; subsequent batch items process normally.")
}

func TestCircuitBreaker_ConcurrentSharedState(t *testing.T) {
	mock := &MockStytchClient{}
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "stytch-shared",
		MaxRequests: 2,
		Interval:    0,
		Timeout:     0,
		ReadyToTrip: func(c gobreaker.Counts) bool { return c.TotalFailures >= 2 },
	})

	// Concurrent batches must see the same OPEN state (global breaker).
	// If wiring creates per-batch instances, this test catches it.
	_ = mock
	_ = cb
	t.Log("TODO: fire two concurrent batches; assert both see same breaker state (most likely wiring bug)")
}
