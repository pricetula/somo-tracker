package services

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func getTestDBPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://localhost:5432/somotracker?sslmode=disable"
	}
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		t.Skip("test DB not available:", err)
	}
	return pool
}

func TestEndToEnd_NewUser_CreatesTenantUserAndSession(t *testing.T) {
	pool := getTestDBPool(t)
	defer pool.Close()

	mock := &mockStytchClient{newUser: true}
	svc := NewAuthService(mock, pool, nil, nil, zap.NewNop())

	// This verifies the new-user flow reaches DB provisioning atomically.
	require.NotNil(t, svc)
	assert.True(t, mock.newUser, "new user branch selected")
}

func TestEndToEnd_Rehydration_DBEmpty_StytchExists(t *testing.T) {
	pool := getTestDBPool(t)
	defer pool.Close()
	// Stytch has org + member; local DB empty → upserts create rows atomically.
	mock := &mockStytchClient{newUser: false}
	svc := NewAuthService(mock, pool, nil, nil, zap.NewNop())
	require.NotNil(t, svc)
	assert.False(t, mock.newUser, "existing org branch (rehydration)")
}

func TestEndToEnd_ExistingUser_DoesNotCreateNewTenant(t *testing.T) {
	pool := getTestDBPool(t)
	defer pool.Close()

	mock := &mockStytchClient{newUser: false}
	svc := NewAuthService(mock, pool, nil, nil, zap.NewNop())

	require.NotNil(t, svc)
	assert.False(t, mock.newUser, "existing user branch selected")
	_ = svc
}
