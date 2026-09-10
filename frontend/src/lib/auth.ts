/**
 * Auth utilities for the Somotracker frontend.
 *
 * Session management architecture:
 * - Session IDs are stored exclusively in an HttpOnly; Secure; SameSite=Lax cookie
 *   named `session_token`, managed by the Go backend.
 * - React/Next.js code must NEVER read the session ID from document.cookie
 *   (it cannot due to HttpOnly) and must NEVER store it in localStorage/sessionStorage.
 * - The backend owns session creation (via Set-Cookie at callback) and invalidation
 *   (DEL in Redis + cookie expiry on logout).
 * - Next.js proxy checks only for cookie *presence* as a UX guard.
 *   The Go backend's session validation + fingerprint check is the actual security gate.
 * - Client-side auth state should be derived from API responses (when available),
 *   not from document.cookie (HttpOnly cookies are invisible to JS).
 */

/** Name of the HttpOnly session cookie set by the Go backend. */
export const SESSION_COOKIE_NAME = "session_token";

/** Name of the CSRF cookie set by the Go backend. */
export const CSRF_COOKIE_NAME = "csrf_token";
