package httpserver

import (
	"context"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestContextKeys_SessionContext(t *testing.T) {
	ctx := context.Background()

	session := &SessionContext{
		UserID:    uuid.New(),
		TenantID:  uuid.New(),
		Role:      "school_admin",
		SessionID: uuid.New(),
	}

	ctx = context.WithValue(ctx, SessionContextKey{}, session)

	retrieved := SessionContextFrom(ctx)
	require.NotNil(t, retrieved)
	require.Equal(t, session.UserID, retrieved.UserID)
	require.Equal(t, session.TenantID, retrieved.TenantID)
	require.Equal(t, session.Role, retrieved.Role)
	require.Equal(t, session.SessionID, retrieved.SessionID)
}

func TestContextKeys_SessionContext_NilWhenNotSet(t *testing.T) {
	ctx := context.Background()
	retrieved := SessionContextFrom(ctx)
	require.Nil(t, retrieved)
}

func TestContextKeys_Memberships(t *testing.T) {
	ctx := context.Background()

	schoolID := uuid.New()
	memberships := []MembershipContext{
		{SchoolID: schoolID, Role: "school_admin", IsActive: true},
		{SchoolID: uuid.New(), Role: "teacher", IsActive: true},
	}

	ctx = context.WithValue(ctx, MembershipsKey{}, memberships)

	retrieved := MembershipsFrom(ctx)
	require.NotNil(t, retrieved)
	require.Len(t, retrieved, 2)
	require.Equal(t, memberships[0].SchoolID, retrieved[0].SchoolID)
	require.Equal(t, memberships[0].Role, retrieved[0].Role)
	require.Equal(t, memberships[1].Role, retrieved[1].Role)
}

func TestContextKeys_Memberships_NilWhenNotSet(t *testing.T) {
	ctx := context.Background()
	retrieved := MembershipsFrom(ctx)
	require.Nil(t, retrieved)
}

func TestContextKeys_TenantID(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New().String()

	ctx = context.WithValue(ctx, TenantIDKey{}, tenantID)

	retrieved := TenantIDFrom(ctx)
	require.Equal(t, tenantID, retrieved)
}

func TestContextKeys_TenantID_EmptyWhenNotSet(t *testing.T) {
	ctx := context.Background()
	retrieved := TenantIDFrom(ctx)
	require.Empty(t, retrieved)
}

func TestContextKeys_RequestID(t *testing.T) {
	ctx := context.Background()
	reqID := "req_test123"

	ctx = context.WithValue(ctx, requestIDKey, reqID)

	retrieved := RequestIDFrom(ctx)
	require.Equal(t, reqID, retrieved)
}

func TestContextKeys_Logger(t *testing.T) {
	ctx := context.Background()

	logger := slog.Default().With("test", "value")
	ctx = context.WithValue(ctx, loggerKey, logger)

	retrieved := LoggerFrom(ctx)
	require.NotNil(t, retrieved)
}

func TestContextKeys_Isolation(t *testing.T) {
	// Different keys should not collide
	ctx := context.Background()

	session := &SessionContext{UserID: uuid.New()}
	memberships := []MembershipContext{{SchoolID: uuid.New(), Role: "admin", IsActive: true}}

	ctx = context.WithValue(ctx, SessionContextKey{}, session)
	ctx = context.WithValue(ctx, MembershipsKey{}, memberships)

	// Each key retrieves only its own value
	require.Equal(t, session.UserID, SessionContextFrom(ctx).UserID)
	require.Len(t, MembershipsFrom(ctx), 1)

	// Wrong key type returns nil/empty
	require.Nil(t, SessionContextFrom(context.Background()))
	require.Nil(t, MembershipsFrom(context.Background()))
}