package httpserver

// Context keys for request-scoped values.
// Using custom types avoids collisions with other packages (staticcheck SA1029).
type ctxKey int

const (
	requestIDKey ctxKey = iota
	loggerKey
	tenantIDKey
	sessionCtxKey
	membershipsKey
)

// SessionContextKey is the context key for session context.
type SessionContextKey struct{}

// MembershipsKey is the context key for memberships.
type MembershipsKey struct{}

// TenantIDKey is the context key for tenant ID.
type TenantIDKey struct{}