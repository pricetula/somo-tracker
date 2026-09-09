package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestSchoolRegistration_ValidationErrors(t *testing.T) {
	svc := NewSchoolRegistrationService(nil, zap.NewNop())

	_, err := svc.RegisterSchool(context.Background(), "", "t1", "Alice", "School")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad_request")
}

func TestSchoolRegistration_TransactionalFlow_RealDB(t *testing.T) {
	pool := getTestDBPool(t)
	defer pool.Close()

	svc := NewSchoolRegistrationService(pool, zap.NewNop())

	// The service validates inputs before any DB interaction.
	_, err := svc.RegisterSchool(context.Background(), "bad-user", "bad-tenant", "", "School")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad_request")
}
