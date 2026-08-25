// Package telemetry provides pluggable error telemetry sinks.
// This file contains error policy definitions and enrichment helpers.

package telemetry

import (
	"context"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"somotracker/backend/internal/xerrors"
)

// ─── Error Policy ────────────────────────────────────────────────────────

// ErrorPolicy defines how errors should be handled based on their type/code.
type ErrorPolicy struct {
	// ShouldLog determines if the error should be logged (via ZapSink).
	ShouldLog bool
	// ShouldTelemetry determines if the error should be sent to telemetry sinks.
	ShouldTelemetry bool
	// SanitizeMessage determines if the message should be replaced with generic.
	SanitizeMessage bool
	// GenericMessage is the replacement message when SanitizeMessage is true.
	GenericMessage string
	// Enrichers are functions that add context to errors before processing.
	Enrichers []ErrorEnricher
}

// ErrorEnricher adds contextual information to an error.
// The enriched error is returned (can be the same or a new instance).
type ErrorEnricher func(ctx context.Context, err *xerrors.DomainError, req *xerrors.TelemetryRequest) *xerrors.DomainError

// DefaultPolicy returns the standard error handling policy.
func DefaultPolicy() *ErrorPolicy {
	return &ErrorPolicy{
		ShouldLog:       true,
		ShouldTelemetry: true,
		SanitizeMessage: false,
		GenericMessage:  "an unexpected error occurred",
		Enrichers:       []ErrorEnricher{},
	}
}

// ValidationPolicy returns a policy for validation errors (4xx).
func ValidationPolicy() *ErrorPolicy {
	return &ErrorPolicy{
		ShouldLog:       true,
		ShouldTelemetry: false, // Don't spam telemetry with validation errors
		SanitizeMessage: false,
		GenericMessage:  "validation failed",
		Enrichers:       []ErrorEnricher{},
	}
}

// InternalErrorPolicy returns a policy for internal server errors (5xx).
func InternalErrorPolicy() *ErrorPolicy {
	return &ErrorPolicy{
		ShouldLog:       true,
		ShouldTelemetry: true,
		SanitizeMessage: true,
		GenericMessage:  "an unexpected error occurred",
		Enrichers:       []ErrorEnricher{},
	}
}

// AuthPolicy returns a policy for authentication/authorization errors.
func AuthPolicy() *ErrorPolicy {
	return &ErrorPolicy{
		ShouldLog:       true,
		ShouldTelemetry: true, // Track auth failures
		SanitizeMessage: false,
		GenericMessage:  "authentication error",
		Enrichers:       []ErrorEnricher{},
	}
}

// PolicyForError selects the appropriate policy based on the error.
func PolicyForError(err *xerrors.DomainError) *ErrorPolicy {
	switch {
	case err.Status >= 500:
		return InternalErrorPolicy()
	case err.Status == http.StatusUnauthorized || err.Status == http.StatusForbidden:
		return AuthPolicy()
	case err.Status >= 400:
		return ValidationPolicy()
	default:
		return DefaultPolicy()
	}
}

// ─── Enrichment Helpers ─────────────────────────────────────────────────

// EnrichWithRequestContext adds request context (method, path, headers) to the telemetry request.
// getRequestID is a function to extract the request ID from the fiber context.
func EnrichWithRequestContext(c *fiber.Ctx, getRequestID func(*fiber.Ctx) string) ErrorEnricher {
	return func(ctx context.Context, err *xerrors.DomainError, req *xerrors.TelemetryRequest) *xerrors.DomainError {
		req.Method = c.Method()
		req.Path = c.Path()
		// Query params handled separately via fiber URL parsing

		// Extract key headers
		req.Headers = make(map[string]string)
		for _, h := range []string{"user-agent", "referer", "x-forwarded-for", "x-real-ip"} {
			if v := c.Get(h); v != "" {
				req.Headers[h] = v
			}
		}

		// Extract request ID
		if getRequestID != nil {
			if rid := getRequestID(c); rid != "" {
				req.RequestID = rid
			}
		}

		return err
	}
}

// EnrichWithUserContext adds user/tenant context to the error.
func EnrichWithUserContext(userID, tenantID string) ErrorEnricher {
	return func(ctx context.Context, err *xerrors.DomainError, req *xerrors.TelemetryRequest) *xerrors.DomainError {
		req.UserID = userID
		req.TenantID = tenantID
		return err
	}
}

// EnrichWithSessionContext extracts session info from Fiber context.
// The sessionLocalsKey is the locals key for the session (defined in middleware).
func EnrichWithSessionContext(c *fiber.Ctx, sessionLocalsKey string) ErrorEnricher {
	return func(ctx context.Context, err *xerrors.DomainError, req *xerrors.TelemetryRequest) *xerrors.DomainError {
		if sess, ok := c.Locals(sessionLocalsKey).(*SessionInfo); ok && sess != nil {
			req.UserID = sess.UserID
			req.TenantID = sess.TenantID
			if req.Context == nil {
				req.Context = make(map[string]any)
			}
			req.Context["role"] = sess.Role
			req.Context["stytch_member_id"] = sess.StytchMemberID
		}
		return err
	}
}

// EnrichWithBusinessContext adds business-specific context.
func EnrichWithBusinessContext(key string, value any) ErrorEnricher {
	return func(ctx context.Context, err *xerrors.DomainError, req *xerrors.TelemetryRequest) *xerrors.DomainError {
		if req.Context == nil {
			req.Context = make(map[string]any)
		}
		req.Context[key] = value
		return err
	}
}

// EnrichFromMeta copies Meta from the error to the telemetry request context.
func EnrichFromMeta() ErrorEnricher {
	return func(ctx context.Context, err *xerrors.DomainError, req *xerrors.TelemetryRequest) *xerrors.DomainError {
		if err.Meta != nil && len(err.Meta) > 0 {
			if req.Context == nil {
				req.Context = make(map[string]any)
			}
			for k, v := range err.Meta {
				req.Context["error_meta_"+k] = v
			}
		}
		return err
	}
}

// ApplyEnrichers applies all enrichers to an error in sequence.
func ApplyEnrichers(ctx context.Context, err *xerrors.DomainError, req *xerrors.TelemetryRequest, enrichers []ErrorEnricher) *xerrors.DomainError {
	for _, enrich := range enrichers {
		err = enrich(ctx, err, req)
	}
	return err
}

// ─── Policy Application ──────────────────────────────────────────────────

// Apply applies the error policy: logs if needed, sends to telemetry if needed.
// The enrichers are configured via the policy's Enrichers field.
// Use ApplyFull when you also need request/session context enrichment.
func (p *ErrorPolicy) Apply(ctx context.Context, req *xerrors.TelemetryRequest, err *xerrors.DomainError) *xerrors.DomainError {
	// Apply enrichers
	err = ApplyEnrichers(ctx, err, req, p.Enrichers)

	// Send to telemetry if policy says so
	if p.ShouldTelemetry && err != nil {
		Registry.ProcessAll(ctx, err, req)
	}

	return err
}

// ─── SessionInfo (for telemetry enrichment) ──────────────────────────────

// SessionInfo represents session data for telemetry enrichment.
// This mirrors the session type from the middleware package to avoid circular imports.
type SessionInfo struct {
	ID                 string
	UserID             string
	TenantID           string
	Role               string
	StytchMemberID     string
	StytchOrgID        string
	StytchSessionToken string
	DeviceFingerprint  string
}

// SessionResolver is a pluggable function for extracting session info from fiber context.
// Set this via SetSessionResolver in middleware initialization to avoid import cycles.
var SessionResolver func(*fiber.Ctx) *SessionInfo

// RequestIDResolver is a pluggable function for extracting request IDs.
// Set this via SetRequestIDResolver in middleware initialization.
var RequestIDResolver func(*fiber.Ctx) string
