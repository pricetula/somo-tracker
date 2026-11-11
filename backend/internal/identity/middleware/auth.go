package middleware

import (
	"context"
	"crypto/sha256"
	"errors"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"somotracker/internal/identity/service"
	"somotracker/internal/platform/config"
	"somotracker/internal/platform/httpserver"
)

// AuthMiddleware validates session token on protected routes
// Per ADR-0002: Go is sole security authority, validates every protected request
type AuthMiddleware struct {
	svc SessionValidator
	cfg *config.Config
}

// SessionValidator defines the interface needed by auth middleware.
type SessionValidator interface {
	ValidateSession(ctx context.Context, tokenHash []byte) (*service.SessionContext, error)
	ValidateSessionWithMemberships(ctx context.Context, tokenHash []byte) (*service.SessionContext, []service.MembershipContext, error)
	GetUserMemberships(ctx context.Context, userID, tenantID uuid.UUID) ([]service.MembershipContext, error)
	GetGuardianStudents(ctx context.Context, guardianUserID, tenantID uuid.UUID) ([]uuid.UUID, error)
	GetTeacherClasses(ctx context.Context, teacherUserID, tenantID uuid.UUID) ([]uuid.UUID, error)
	RevokeSession(ctx context.Context, sessionID uuid.UUID) error
}

// MembershipContext holds membership info for RBAC decisions
type MembershipContext struct {
	SchoolID uuid.UUID
	Role     string
	IsActive bool
}

// NewAuthMiddleware creates auth middleware.
func NewAuthMiddleware(svc SessionValidator, cfg *config.Config) *AuthMiddleware {
	return &AuthMiddleware{svc: svc, cfg: cfg}
}

// sessionCookieName returns the cookie name with environment suffix per ADR-0002
// Uses __Host- prefix in production (requires Secure, Path=/, no Domain)
func (m *AuthMiddleware) sessionCookieName() string {
	return m.cfg.SessionCookieName()
}

// RequireAuth middleware validates session token from cookie
// Fails closed if session invalid, expired, revoked, or user has no active membership
func (m *AuthMiddleware) RequireAuth() fiber.Handler {
	return func(c fiber.Ctx) error {
		cookieName := m.sessionCookieName()
		token := c.Cookies(cookieName)
		if token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
				Code:    "UNAUTHENTICATED",
				Message: "authentication required",
			})
		}

		tokenHash := sha256.Sum256([]byte(token))
		sessionCtx, memberships, err := m.svc.ValidateSessionWithMemberships(c.Context(), tokenHash[:])
		if err != nil {
			if errors.Is(err, service.ErrSessionNotFound) {
				return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
					Code:    "UNAUTHENTICATED",
					Message: "session not found or expired",
				})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
				Code:    "INTERNAL_ERROR",
				Message: "session validation failed",
			})
		}

		// Fail closed: no active membership = session invalid
		activeRoles := make([]string, 0)
		activeSchoolIDs := make([]uuid.UUID, 0)
		for _, ms := range memberships {
			if ms.IsActive {
				activeRoles = append(activeRoles, ms.Role)
				activeSchoolIDs = append(activeSchoolIDs, ms.SchoolID)
			}
		}
		if len(activeRoles) == 0 {
			// User removed from all schools - revoke session immediately
			_ = m.svc.RevokeSession(c.Context(), sessionCtx.SessionID)
			return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
				Code:    "UNAUTHENTICATED",
				Message: "no active membership",
			})
		}

		// Store in locals for handlers (role strings lowercased to match authz matrix keys)
		c.Locals("session_id", sessionCtx.SessionID)
		c.Locals("user_id", sessionCtx.UserID)
		c.Locals("tenant_id", sessionCtx.TenantID)
		lowerRoles := make([]string, 0, len(activeRoles))
		for _, r := range activeRoles {
			lowerRoles = append(lowerRoles, strings.ToLower(r))
		}
		c.Locals("roles", lowerRoles)
		c.Locals("school_ids", activeSchoolIDs)
		c.Locals("primary_role", strings.ToLower(activeRoles[0]))

		// Also store in context for service layer
		ctx := context.WithValue(c.Context(), httpserver.SessionContextKey{}, sessionCtx)
		ctx = context.WithValue(ctx, httpserver.MembershipsKey{}, memberships)
		c.SetContext(ctx)

		return c.Next()
	}
}

// OptionalAuth middleware validates session if present, but doesn't require it
func (m *AuthMiddleware) OptionalAuth() fiber.Handler {
	return func(c fiber.Ctx) error {
		cookieName := m.sessionCookieName()
		token := c.Cookies(cookieName)
		if token == "" {
			return c.Next()
		}

		tokenHash := sha256.Sum256([]byte(token))
		sessionCtx, memberships, err := m.svc.ValidateSessionWithMemberships(c.Context(), tokenHash[:])
		if err != nil {
			return c.Next()
		}

		activeRoles := make([]string, 0)
		activeSchoolIDs := make([]uuid.UUID, 0)
		for _, ms := range memberships {
			if ms.IsActive {
				activeRoles = append(activeRoles, ms.Role)
				activeSchoolIDs = append(activeSchoolIDs, ms.SchoolID)
			}
		}

		c.Locals("session_id", sessionCtx.SessionID)
		c.Locals("user_id", sessionCtx.UserID)
		c.Locals("tenant_id", sessionCtx.TenantID)
		lowerRoles := make([]string, 0, len(activeRoles))
		for _, r := range activeRoles {
			lowerRoles = append(lowerRoles, strings.ToLower(r))
		}
		c.Locals("roles", lowerRoles)
		c.Locals("school_ids", activeSchoolIDs)

		// Also store in context for service layer
		ctx := context.WithValue(c.Context(), httpserver.SessionContextKey{}, sessionCtx)
		ctx = context.WithValue(ctx, httpserver.MembershipsKey{}, memberships)
		c.SetContext(ctx)

		return c.Next()
	}
}

// CSRFMiddleware validates Origin header for mutations
// Rejects if both Origin and Sec-Fetch-Site are absent on cookie-authenticated mutations
func (m *AuthMiddleware) CSRFMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		// Only validate on mutating methods
		if c.Method() == fiber.MethodGet || c.Method() == fiber.MethodHead || c.Method() == fiber.MethodOptions {
			return c.Next()
		}

		origin := c.Get("Origin")
		secFetchSite := c.Get("Sec-Fetch-Site")

		// If Origin is missing, check Sec-Fetch-Site
		if origin == "" {
			if secFetchSite == "same-origin" {
				return c.Next()
			}
			// Reject: no Origin and not same-origin navigation
			return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{
				Code:    "CSRF_FAILED",
				Message: "missing origin header",
			})
		}

		// Validate origin against allowed origins from config
		allowedOrigins := []string{}
		if m.cfg.CORSOrigins != "" {
			parts := strings.Split(m.cfg.CORSOrigins, ",")
			for _, p := range parts {
				if v := strings.TrimSpace(p); v != "" {
					allowedOrigins = append(allowedOrigins, v)
				}
			}
		}
		// Development fallback: allow localhost:3000
		if m.cfg.Env == "development" || m.cfg.Env == "test" {
			allowedOrigins = append(allowedOrigins, "http://localhost:3000")
		}
		// Production: NO fallback - must be explicitly configured
		if m.cfg.Env == "production" && len(allowedOrigins) == 0 {
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
				Code:    "INTERNAL_ERROR",
				Message: "CORS origins not configured for production",
			})
		}

		// Exact match only (no prefix/suffix)
		allowed := false
		for _, allowedOrigin := range allowedOrigins {
			if origin == allowedOrigin {
				allowed = true
				break
			}
		}

		if !allowed {
			return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{
				Code:    "CSRF_FAILED",
				Message: "invalid origin",
			})
		}

		return c.Next()
	}
}

// TenantMiddleware extracts tenant from session and validates access
func (m *AuthMiddleware) TenantMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		tenantID, ok := c.Locals("tenant_id").(uuid.UUID)
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{
				Code:    "FORBIDDEN",
				Message: "tenant not found in session",
			})
		}

		ctx := context.WithValue(c.Context(), httpserver.TenantIDKey{}, tenantID)
		c.SetContext(ctx)

		return c.Next()
	}
}

// RBACMiddleware provides centralized role-based access control
// Usage: m.RequireRole("school_admin", "finance_officer") or m.RequireAnyRole(...)
// For object-level: m.RequireGuardianAccess(studentIDParam) or m.RequireTeacherAccess(classIDParam)
type RBACMiddleware struct {
	svc SessionValidator
}

// NewRBACMiddleware creates RBAC middleware.
func NewRBACMiddleware(svc SessionValidator) *RBACMiddleware {
	return &RBACMiddleware{svc: svc}
}

// RequireRole creates middleware that requires one of the specified roles
// Roles checked: tenant_owner, school_admin, finance_officer, teacher, guardian
func (r *RBACMiddleware) RequireRole(roles ...string) fiber.Handler {
	roleSet := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		roleSet[role] = struct{}{}
	}

	return func(c fiber.Ctx) error {
		userRoles, ok := c.Locals("roles").([]string)
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{
				Code:    "FORBIDDEN",
				Message: "roles not found in session",
			})
		}

		hasRole := false
		for _, ur := range userRoles {
			if _, ok := roleSet[ur]; ok {
				hasRole = true
				break
			}
		}

		if !hasRole {
			return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{
				Code:    "FORBIDDEN",
				Message: "insufficient role",
			})
		}

		return c.Next()
	}
}

// RequireAnyRole is an alias for RequireRole
func (r *RBACMiddleware) RequireAnyRole(roles ...string) fiber.Handler {
	return r.RequireRole(roles...)
}

// RequireAllRoles requires ALL specified roles (user must have all)
func (r *RBACMiddleware) RequireAllRoles(roles ...string) fiber.Handler {
	return func(c fiber.Ctx) error {
		userRoles, ok := c.Locals("roles").([]string)
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{
				Code:    "FORBIDDEN",
				Message: "roles not found in session",
			})
		}

		userRoleSet := make(map[string]struct{}, len(userRoles))
		for _, ur := range userRoles {
			userRoleSet[ur] = struct{}{}
		}

		for _, required := range roles {
			if _, ok := userRoleSet[required]; !ok {
				return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{
					Code:    "FORBIDDEN",
					Message: "insufficient role",
				})
			}
		}

		return c.Next()
	}
}

// RequireSchoolAccess ensures user has an active membership in the target school
// schoolIDParam is the path parameter name (e.g., "school_id") or "current" for session's school
func (r *RBACMiddleware) RequireSchoolAccess(schoolIDParam string) fiber.Handler {
	return func(c fiber.Ctx) error {
		schoolIDs, ok := c.Locals("school_ids").([]uuid.UUID)
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{
				Code:    "FORBIDDEN",
				Message: "school memberships not found",
			})
		}

		var targetSchoolID uuid.UUID
		if schoolIDParam == "current" {
			// Use first active school (primary)
			if len(schoolIDs) == 0 {
				return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{
					Code:    "FORBIDDEN",
					Message: "no active school membership",
				})
			}
			targetSchoolID = schoolIDs[0]
		} else {
			// Parse from path params
			paramVal := c.Params(schoolIDParam)
			if paramVal == "" {
				return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
					Code:    "INVALID_REQUEST",
					Message: "school_id parameter required",
				})
			}
			parsed, err := uuid.Parse(paramVal)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
					Code:    "INVALID_REQUEST",
					Message: "invalid school_id",
				})
			}
			targetSchoolID = parsed
		}

		// Check if user has access to this school
		hasAccess := false
		for _, sid := range schoolIDs {
			if sid == targetSchoolID {
				hasAccess = true
				break
			}
		}

		if !hasAccess {
			return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{
				Code:    "FORBIDDEN",
				Message: "no access to this school",
			})
		}

		// Store for downstream handlers
		c.Locals("authorized_school_id", targetSchoolID)
		return c.Next()
	}
}

// RequireGuardianAccess ensures guardian can only access their own children
// studentIDParam is the path parameter name (e.g., "student_id")
func (r *RBACMiddleware) RequireGuardianAccess(studentIDParam string) fiber.Handler {
	return func(c fiber.Ctx) error {
		userID, ok := c.Locals("user_id").(uuid.UUID)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
				Code:    "UNAUTHENTICATED",
				Message: "user not in session",
			})
		}

		tenantID, ok := c.Locals("tenant_id").(uuid.UUID)
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{
				Code:    "FORBIDDEN",
				Message: "tenant not in session",
			})
		}

		// Check if user has guardian role
		userRoles, ok := c.Locals("roles").([]string)
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{
				Code:    "FORBIDDEN",
				Message: "roles not found",
			})
		}

		isGuardian := false
		for _, role := range userRoles {
			if role == "guardian" {
				isGuardian = true
				break
			}
		}

		if !isGuardian {
			return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{
				Code:    "FORBIDDEN",
				Message: "guardian role required",
			})
		}

		// Get student ID from path
		paramVal := c.Params(studentIDParam)
		if paramVal == "" {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Code:    "INVALID_REQUEST",
				Message: "student_id parameter required",
			})
		}
		studentID, err := uuid.Parse(paramVal)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Code:    "INVALID_REQUEST",
				Message: "invalid student_id",
			})
		}

		// Fetch guardian's authorized students
		authorizedStudents, err := r.svc.GetGuardianStudents(c.Context(), userID, tenantID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
				Code:    "INTERNAL_ERROR",
				Message: "failed to verify guardian access",
			})
		}

		// Check if student is in guardian's list
		authorized := false
		for _, sid := range authorizedStudents {
			if sid == studentID {
				authorized = true
				break
			}
		}

		if !authorized {
			return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{
				Code:    "FORBIDDEN",
				Message: "not authorized to access this student",
			})
		}

		return c.Next()
	}
}

// RequireTeacherAccess ensures teacher can only access their assigned classes
// classIDParam is the path parameter name (e.g., "class_id")
func (r *RBACMiddleware) RequireTeacherAccess(classIDParam string) fiber.Handler {
	return func(c fiber.Ctx) error {
		userID, ok := c.Locals("user_id").(uuid.UUID)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
				Code:    "UNAUTHENTICATED",
				Message: "user not in session",
			})
		}

		tenantID, ok := c.Locals("tenant_id").(uuid.UUID)
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{
				Code:    "FORBIDDEN",
				Message: "tenant not in session",
			})
		}

		// Check if user has teacher role
		userRoles, ok := c.Locals("roles").([]string)
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{
				Code:    "FORBIDDEN",
				Message: "roles not found",
			})
		}

		isTeacher := false
		for _, role := range userRoles {
			if role == "teacher" {
				isTeacher = true
				break
			}
		}

		if !isTeacher {
			return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{
				Code:    "FORBIDDEN",
				Message: "teacher role required",
			})
		}

		// Get class ID from path
		paramVal := c.Params(classIDParam)
		if paramVal == "" {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Code:    "INVALID_REQUEST",
				Message: "class_id parameter required",
			})
		}
		classID, err := uuid.Parse(paramVal)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Code:    "INVALID_REQUEST",
				Message: "invalid class_id",
			})
		}

		// Fetch teacher's authorized classes
		authorizedClasses, err := r.svc.GetTeacherClasses(c.Context(), userID, tenantID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
				Code:    "INTERNAL_ERROR",
				Message: "failed to verify teacher access",
			})
		}

		// Check if class is in teacher's list
		authorized := false
		for _, cid := range authorizedClasses {
			if cid == classID {
				authorized = true
				break
			}
		}

		if !authorized {
			return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{
				Code:    "FORBIDDEN",
				Message: "not authorized to access this class",
			})
		}

		return c.Next()
	}
}

// RequireFinanceAccess ensures user has finance_officer or school_admin or tenant_owner role
// for the current school
func (r *RBACMiddleware) RequireFinanceAccess() fiber.Handler {
	return r.RequireRole("finance", "admin", "owner")
}

// RequireAdminAccess ensures user has school_admin or tenant_owner role
func (r *RBACMiddleware) RequireAdminAccess() fiber.Handler {
	return r.RequireRole("admin", "owner")
}

// ErrorResponse standardizes error shape for middleware
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}