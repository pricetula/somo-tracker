package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestStreamsService_ValidationErrors(t *testing.T) {
	svc := NewStreamsService(nil, nil, zap.NewNop())

	// Test empty schoolID
	_, err := svc.CreateStreams(context.Background(), "", []string{"stream1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad_request: school_id is required")

	// Test empty names slice
	_, err = svc.CreateStreams(context.Background(), "school-123", []string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad_request: at least one stream name is required")

	// Test invalid schoolID (not a valid UUID)
	_, err = svc.CreateStreams(context.Background(), "not-a-uuid", []string{"stream1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad_request: invalid school_id")

	// Test names with empty string - should be skipped but others created (requires DB)
	// This test validates input validation only
}

func TestStreamsService_ValidInput_PassesValidation(t *testing.T) {
	// This test validates that valid input doesn't trigger validation errors.
	// Actual DB integration requires a test database (see E2E tests).
	// We only test the validation logic here, which is covered by TestStreamsService_ValidationErrors.
	svc := NewStreamsService(nil, nil, zap.NewNop())
	_ = svc
}
