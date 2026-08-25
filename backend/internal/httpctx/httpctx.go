package httpctx

import (
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// Context keys for request-scoped values. Using custom types avoids
// collisions with other packages (SA1029).
type (
	requestIDKey string
	loggerKey    string
	tenantIDKey  string
	schoolIDKey  string
	userIDKey    string
)

const (
	// RequestIDHeader is the wire name of the correlation ID header.
	RequestIDHeader              = "X-Request-ID"
	RequestIDKey    requestIDKey = "request_id"
	LoggerKey       loggerKey    = "middleware.logger"
	TenantIDKey     tenantIDKey  = "tenant_id"
	SchoolIDKey     schoolIDKey  = "active_school_id"
	UserIDKey       userIDKey    = "user_id"
)

// GetRequestID returns the correlation ID for the current request.
// Returns empty string if not set (e.g. in unit tests).
func GetRequestID(c *fiber.Ctx) string {
	if id, ok := c.Locals(RequestIDKey).(string); ok {
		return id
	}
	return ""
}

// SetRequestID stores the correlation ID in the request context and
// echoes it in the response header.
func SetRequestID(c *fiber.Ctx, id string) {
	c.Locals(RequestIDKey, id)
	c.Set(RequestIDHeader, id)
}

// GetLogger returns the request-scoped sugared logger.
// Falls back to a no-op logger if not set (e.g. in unit tests).
func GetLogger(c *fiber.Ctx) *zap.SugaredLogger {
	if logger, ok := c.Locals(LoggerKey).(*zap.SugaredLogger); ok && logger != nil {
		return logger
	}
	return zap.NewNop().Sugar()
}

// SetLogger stores the logger in the request context.
func SetLogger(c *fiber.Ctx, logger *zap.SugaredLogger) {
	c.Locals(LoggerKey, logger)
}

// GetTenantID returns the tenant ID for the current request.
// Returns empty string if not set.
func GetTenantID(c *fiber.Ctx) string {
	if id, ok := c.Locals(TenantIDKey).(string); ok {
		return id
	}
	return ""
}

// SetTenantID stores the tenant ID in the request context.
func SetTenantID(c *fiber.Ctx, tenantID string) {
	c.Locals(TenantIDKey, tenantID)
}

// GetSchoolID returns the active school ID for the current request.
// Returns empty string if not set.
func GetSchoolID(c *fiber.Ctx) string {
	if id, ok := c.Locals(SchoolIDKey).(string); ok {
		return id
	}
	return ""
}

// SetSchoolID stores the active school ID in the request context.
func SetSchoolID(c *fiber.Ctx, schoolID string) {
	c.Locals(SchoolIDKey, schoolID)
}

// GetUserID returns the authenticated user ID for the current request.
// Returns empty string if not set.
func GetUserID(c *fiber.Ctx) string {
	if id, ok := c.Locals(UserIDKey).(string); ok {
		return id
	}
	return ""
}

// SetUserID stores the authenticated user ID in the request context.
func SetUserID(c *fiber.Ctx, userID string) {
	c.Locals(UserIDKey, userID)
}

// RequireTenantID returns the tenant ID or returns an error if not set.
// Use in handlers that must have a tenant context.
func RequireTenantID(c *fiber.Ctx) (string, error) {
	id := GetTenantID(c)
	if id == "" {
		return "", ErrMissingTenantID
	}
	return id, nil
}

// RequireSchoolID returns the active school ID or returns an error if not set.
func RequireSchoolID(c *fiber.Ctx) (string, error) {
	id := GetSchoolID(c)
	if id == "" {
		return "", ErrMissingSchoolID
	}
	return id, nil
}

// RequireUserID returns the authenticated user ID or returns an error if not set.
func RequireUserID(c *fiber.Ctx) (string, error) {
	id := GetUserID(c)
	if id == "" {
		return "", ErrMissingUserID
	}
	return id, nil
}

// ErrMissingTenantID is returned when a required tenant ID is not present.
var ErrMissingTenantID = &MissingContextError{Key: "tenant_id"}

// ErrMissingSchoolID is returned when a required school ID is not present.
var ErrMissingSchoolID = &MissingContextError{Key: "active_school_id"}

// ErrMissingUserID is returned when a required user ID is not present.
var ErrMissingUserID = &MissingContextError{Key: "user_id"}

// MissingContextError indicates a required context value is missing.
type MissingContextError struct {
	Key string
}

func (e *MissingContextError) Error() string {
	return "missing required context: " + e.Key
}
