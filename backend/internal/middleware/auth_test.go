package middleware

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// TestRequireAuth_NoSession verifies that RequireAuth returns
// ErrUnauthorized (not nil, not a raw ad-hoc JSON response) when no
// session is present in context.
func TestRequireAuth_NoSession(t *testing.T) {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			// Verify the error matches our sentinel
			if !errors.Is(err, ErrUnauthorized) {
				t.Fatalf("expected error wrapping ErrUnauthorized, got %v", err)
			}
			return c.Status(fiber.StatusUnauthorized).SendString("unauthorized")
		},
	})

	app.Get("/protected", RequireAuth, func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", resp.StatusCode)
	}
}

// TestRequireAuth_WithSession verifies that RequireAuth passes through to the
// next handler when a valid session is present in context.
func TestRequireAuth_WithSession(t *testing.T) {
	app := fiber.New()

	app.Get("/protected", func(c *fiber.Ctx) error {
		// Inject a session (simulating what the global session-loading middleware does)
		c.Locals("session", &SessionInfo{
			UserID:   "user_001",
			TenantID: "tenant_001",
			Role:     "SCHOOL_ADMIN",
		})
		return c.Next()
	}, RequireAuth, func(c *fiber.Ctx) error {
		// Verify locals were set
		if tid := c.Locals("tenant_id"); tid != "tenant_001" {
			t.Fatalf("expected tenant_id 'tenant_001', got %v", tid)
		}
		if uid := c.Locals("user_id"); uid != "user_001" {
			t.Fatalf("expected user_id 'user_001', got %v", uid)
		}
		if r := c.Locals("role"); r != "SCHOOL_ADMIN" {
			t.Fatalf("expected role 'SCHOOL_ADMIN', got %v", r)
		}
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}
}

// TestRequireRole_NoSession verifies that a route wrapped with RequireRole
// returns 401 (not 403) when there is no session at all.
func TestRequireRole_NoSession(t *testing.T) {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			if !errors.Is(err, ErrUnauthorized) {
				t.Fatalf("expected error wrapping ErrUnauthorized, got %v", err)
			}
			return c.Status(fiber.StatusUnauthorized).SendString("unauthorized")
		},
	})

	app.Get("/admin", RequireRole("SCHOOL_ADMIN"), func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/admin", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", resp.StatusCode)
	}
}

// TestRequireRole_ForbiddenRole verifies that RequireRole returns 403
// when authenticated but lacking the required role.
func TestRequireRole_ForbiddenRole(t *testing.T) {
	app := fiber.New()

	app.Get("/admin", func(c *fiber.Ctx) error {
		c.Locals("session", &SessionInfo{
			UserID:   "user_001",
			TenantID: "tenant_001",
			Role:     "TEACHER",
		})
		return c.Next()
	}, RequireRole("SCHOOL_ADMIN"), func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/admin", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusForbidden {
		t.Fatalf("expected 403 Forbidden, got %d", resp.StatusCode)
	}
}
