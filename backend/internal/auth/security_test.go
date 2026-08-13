package auth

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// ============================================================================
// Security: cookie attributes
// ============================================================================

// TestHandler_Register_CookieSecurityFlags verifies that session-identifying
// cookies are HttpOnly (JS-inaccessible) while the CSRF cookie deliberately
// is not, since the frontend must be able to read it to attach it to requests.
// TestHandler_Register_CookieSecurityFlags verifies that session, role, and CSRF
// cookies carry the correct HttpOnly, Secure, and SameSite flags.
func TestHandler_Register_CookieSecurityFlags(t *testing.T) {
	h := newHandlerTestHarness(t)

	// Ensure AppEnv is set to non-development so Secure=true is evaluated
	h.handler.cfg.AppEnv = "test"

	sessionRef := "550e8400-e29b-41d4-a716-446655440020"
	istKey := "ist:test:" + sessionRef
	cacheData, _ := json.Marshal(istCacheData{IST: "ist_flags", Email: "flags@example.com"})
	if err := h.mr.Set(istKey, string(cacheData)); err != nil {
		t.Fatalf("set IST in redis: %v", err)
	}

	resp := h.doRequestWithBody("POST", "/api/auth/register", "", RegistrationPayload{
		SchoolName: "Flags School",
		SessionRef: sessionRef,
		FullName:   "Flags User",
	})

	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("expected 204 No Content, got %d", resp.StatusCode)
	}

	// Must use Values() because Fiber sets multiple Set-Cookie headers
	cookies := resp.Header.Values("Set-Cookie")
	if len(cookies) == 0 {
		t.Fatal("expected Set-Cookie headers, got none")
	}

	foundSessionCookie := false
	foundRoleCookie := false
	foundCSRFCookie := false

	for _, c := range cookies {
		cLower := strings.ToLower(c)

		// 1. Session ID Cookie (somo_sid) -> HttpOnly=true, Secure=true, SameSite=Lax
		if strings.Contains(c, "somo_sid=") {
			foundSessionCookie = true
			if !strings.Contains(cLower, "secure") {
				t.Errorf("somo_sid cookie missing 'Secure': %s", c)
			}
			if !strings.Contains(cLower, "httponly") {
				t.Errorf("somo_sid cookie missing 'HttpOnly': %s", c)
			}
			if !strings.Contains(cLower, "samesite=lax") {
				t.Errorf("somo_sid cookie missing 'SameSite=Lax': %s", c)
			}
		}

		// 2. Role Cookie (somo_role) -> HttpOnly=false, Secure=true
		if strings.Contains(c, "somo_role=") {
			foundRoleCookie = true
			if !strings.Contains(cLower, "secure") {
				t.Errorf("somo_role cookie missing 'Secure': %s", c)
			}
			if strings.Contains(cLower, "httponly") {
				t.Errorf("somo_role cookie should NOT be HttpOnly: %s", c)
			}
		}

		// 3. CSRF Cookie (csrf_token) -> HttpOnly=false, Secure=true
		if strings.Contains(c, "csrf_token=") {
			foundCSRFCookie = true
			if !strings.Contains(cLower, "secure") {
				t.Errorf("csrf_token cookie missing 'Secure': %s", c)
			}
		}
	}

	if !foundSessionCookie {
		t.Error("somo_sid cookie not found in response")
	}
	if !foundRoleCookie {
		t.Error("somo_role cookie not found in response")
	}
	if !foundCSRFCookie {
		t.Error("csrf_token cookie not found in response")
	}
}

// TestHandler_Register_CSRFTokenUnpredictable verifies that CSRF tokens
// differ across independent registrations, guarding against a static or
// predictable token generator.
func TestHandler_Register_CSRFTokenUnpredictable(t *testing.T) {
	getCSRF := func(sessionRef, ist, email string) string {
		h := newHandlerTestHarness(t)
		seedIST(t, h, sessionRef, ist, email)

		resp := h.doRequestWithBody("POST", "/api/auth/register", "", RegistrationPayload{
			SchoolName: "CSRF School",
			SessionRef: sessionRef,
			FullName:   "CSRF User",
		})
		for _, c := range resp.Cookies() {
			if c.Name == "csrf_token" {
				return c.Value
			}
		}
		t.Fatal("csrf cookie not set")
		return ""
	}

	token1 := getCSRF("550e8400-e29b-41d4-a716-446655440011", "ist_a", "a@example.com")
	token2 := getCSRF("550e8400-e29b-41d4-a716-446655440012", "ist_b", "b@example.com")

	if token1 == token2 {
		t.Fatal("CSRF tokens should not be identical across independent registrations")
	}
	if token1 == "ist_a" || token2 == "ist_b" {
		t.Fatal("CSRF token must not equal the raw IST value")
	}
}

// TestHandler_Register_SessionTokenNotIST verifies that the issued session
// token is a newly minted value, not a passthrough of the intermediate
// session token — guards against a session-fixation-style bug where the
// pre-auth token becomes the authenticated session.
func TestHandler_Register_SessionTokenNotIST(t *testing.T) {
	h := newHandlerTestHarness(t)

	sessionRef := "550e8400-e29b-41d4-a716-446655440013"
	seedIST(t, h, sessionRef, "ist_fixation_check", "fix@example.com")

	resp := h.doRequestWithBody("POST", "/api/auth/register", "", RegistrationPayload{
		SchoolName: "Fixation School",
		SessionRef: sessionRef,
		FullName:   "Fix User",
	})

	for _, c := range resp.Cookies() {
		if c.Name == "somo_sid" && c.Value == "ist_fixation_check" {
			t.Fatal("session cookie must not equal the raw IST value")
		}
	}
}

// seedIST is a small helper to reduce duplication across security/race tests.
func seedIST(t *testing.T, h *handlerTestHarness, sessionRef, ist, email string) {
	t.Helper()
	istKey := "ist:test:" + sessionRef
	cacheData, err := json.Marshal(istCacheData{IST: ist, Email: email})
	if err != nil {
		t.Fatalf("marshal ist cache data: %v", err)
	}
	if err := h.mr.Set(istKey, string(cacheData)); err != nil {
		t.Fatalf("set IST in redis: %v", err)
	}
}
