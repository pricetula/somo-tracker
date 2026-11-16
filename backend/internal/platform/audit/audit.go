// Package audit provides structured audit event logging for security-relevant actions.
// Audit events are emitted via slog with structured attributes including request ID,
// user, tenant, school, IP, and outcome. No sensitive values (tokens, cookies, etc.)
// are ever included in audit events.
package audit

import (
	"context"
	"log/slog"
)

// EventType represents the type of audit event.
type EventType string

const (
	EventLoginSuccess     EventType = "login_success"
	EventLoginFailure     EventType = "login_failure"
	EventLogout           EventType = "logout"
	EventSessionRevoked   EventType = "session_revoked"
	EventRoleChanged      EventType = "role_changed"
	EventMembershipChanged EventType = "membership_changed"
	EventActiveSchoolSwitched EventType = "active_school_switched"
	EventTenantProvisioned EventType = "tenant_provisioned"
	EventAuthorizationDenied EventType = "authorization_denied"
)

// Event represents a security-relevant audit event.
type Event struct {
	Type       EventType
	RequestID  string
	UserID     string
	TenantID   string
	SchoolID   string
	IP         string
	Outcome    string // "success", "failure", "denied"
	Details    map[string]string
}

// Recorder records audit events. The default implementation logs via slog.
type Recorder struct {
	logger *slog.Logger
}

// New creates a new audit recorder.
func New(logger *slog.Logger) *Recorder {
	return &Recorder{logger: logger}
}

// Record emits an audit event with structured attributes.
func (r *Recorder) Record(ctx context.Context, evt Event) {
	attrs := []any{
		"event_type", evt.Type,
		"outcome", evt.Outcome,
		"request_id", evt.RequestID,
	}
	if evt.UserID != "" {
		attrs = append(attrs, "user_id", evt.UserID)
	}
	if evt.TenantID != "" {
		attrs = append(attrs, "tenant_id", evt.TenantID)
	}
	if evt.SchoolID != "" {
		attrs = append(attrs, "school_id", evt.SchoolID)
	}
	if evt.IP != "" {
		attrs = append(attrs, "ip", evt.IP)
	}
	for k, v := range evt.Details {
		attrs = append(attrs, k, v)
	}

	switch evt.Outcome {
	case "failure", "denied":
		r.logger.WarnContext(ctx, "audit event", attrs...)
	default:
		r.logger.InfoContext(ctx, "audit event", attrs...)
	}
}
