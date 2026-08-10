package auth

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// ============================================================================
// Security: cookie attributes
// ============================================================================

// TestHandler_Register_CookieSecurityFlags verifies that session-identifying
// cookies are HttpOnly (JS-inaccessible) while the CSRF cookie deliberately
// is not, since the frontend must be able to read it to attach it to requests.
func TestHandler_Register_CookieSecurityFlags(t *testing.T) {
	h := newHandlerTestHarness(t)

	sessionRef := "550e8400-e29b-41d4-a716-446655440010"
	seedIST(t, h, sessionRef, "ist_sec", "sec@example.com")

	resp := h.doRequestWithBody("POST", "/api/auth/register", "", RegistrationPayload{
		SchoolName: "Security School",
		SessionRef: sessionRef,
		FullName:   "Sec User",
	})
	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}

	cookies := resp.Cookies()
	wantHTTPOnly := map[string]bool{
		"somo_sid":       true,
		"somo_role":      true,
		"somo_school_id": true,
		"somo_csrf":      false, // frontend JS must read this
	}

	seen := map[string]bool{}
	for _, c := range cookies {
		want, tracked := wantHTTPOnly[c.Name]
		if !tracked {
			continue
		}
		seen[c.Name] = true

		if c.HttpOnly != want {
			t.Errorf("cookie %s: HttpOnly = %v, want %v", c.Name, c.HttpOnly, want)
		}
		if !c.Secure {
			t.Errorf("cookie %s: Secure = false, want true", c.Name)
		}
		if c.SameSite != http.SameSiteStrictMode && c.SameSite != http.SameSiteLaxMode {
			t.Errorf("cookie %s: SameSite = %v, want Strict or Lax", c.Name, c.SameSite)
		}
	}
	for name := range wantHTTPOnly {
		if !seen[name] {
			t.Errorf("expected cookie %s to be set, but it was missing", name)
		}
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
			if c.Name == "somo_csrf" {
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
