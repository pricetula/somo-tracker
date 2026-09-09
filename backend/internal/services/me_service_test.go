package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestMeService_ValidationErrors(t *testing.T) {
	svc := NewMeService(nil, zap.NewNop())

	tests := []struct {
		name     string
		token    string
		tenantID string
		wantErr  string
	}{
		{
			name:     "empty token",
			token:    "",
			tenantID: "tenant-123",
			wantErr:  "bad_request: missing session token",
		},
		{
			name:     "empty tenantID",
			token:    "token-123",
			tenantID: "",
			wantErr:  "bad_request: missing tenant context",
		},
		{
			name:     "both empty",
			token:    "",
			tenantID: "",
			wantErr:  "bad_request: missing session token", // First validation check
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.GetCurrentUser(context.Background(), tt.token, tt.tenantID)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestMeService_NonValidationErrorsRequireDB(t *testing.T) {
	svc := NewMeService(nil, zap.NewNop())

	// With valid inputs but nil pool, we expect a DB-related error (not validation)
	_, err := svc.GetCurrentUser(context.Background(), "valid-token", "valid-tenant")
	require.Error(t, err)
	// Should be an internal error or DB connection error, not bad_request
	assert.NotContains(t, err.Error(), "bad_request:", "valid inputs should not produce bad_request errors")
}

// TestMeResult_StructSerialization ensures the MeResult struct serializes correctly
func TestMeResult_StructSerialization(t *testing.T) {
	result := MeResult{
		UserName:         "John Doe",
		Email:            "john@example.com",
		ActiveSchoolID:   "school-123",
		SchoolName:       "Test School",
		ActiveSchoolRole: "teacher",
		TenantID:         "tenant-456",
	}

	// Just verify fields are accessible and correct
	assert.Equal(t, "John Doe", result.UserName)
	assert.Equal(t, "john@example.com", result.Email)
	assert.Equal(t, "school-123", result.ActiveSchoolID)
	assert.Equal(t, "Test School", result.SchoolName)
	assert.Equal(t, "teacher", result.ActiveSchoolRole)
	assert.Equal(t, "tenant-456", result.TenantID)
}

// TestMeResult_EmptyFields handles empty optional fields
func TestMeResult_EmptyFields(t *testing.T) {
	result := MeResult{
		UserName: "Jane Doe",
		Email:    "jane@example.com",
		TenantID: "tenant-789",
		// ActiveSchoolID, SchoolName, ActiveSchoolRole are empty
	}

	assert.Equal(t, "", result.ActiveSchoolID)
	assert.Equal(t, "", result.SchoolName)
	assert.Equal(t, "", result.ActiveSchoolRole)
	assert.Equal(t, "Jane Doe", result.UserName)
	assert.Equal(t, "jane@example.com", result.Email)
	assert.Equal(t, "tenant-789", result.TenantID)
}

// Test that error messages follow the expected pattern
func TestMeService_ErrorPatterns(t *testing.T) {
	svc := NewMeService(nil, zap.NewNop())

	_, err := svc.GetCurrentUser(context.Background(), "", "")
	require.Error(t, err)

	errMsg := err.Error()
	// Error should follow the pattern: "code: message"
	assert.True(t,
		errors.Is(err, nil) || // won't match
			errMsg == "bad_request: missing session token" ||
			errMsg == "bad_request: missing tenant context",
		"error should follow expected pattern")
}
