/**
 * Auth utilities for the Somotracker frontend.
 *
 * Session management architecture:
 * - Session IDs are stored exclusively in an HttpOnly; Secure; SameSite=Lax cookie
 *   named `somo_sid`, managed by the Go backend.
 * - React/Next.js code must NEVER read the session ID from document.cookie
 *   (it cannot due to HttpOnly) and must NEVER store it in localStorage/sessionStorage.
 * - The backend owns session creation (via Set-Cookie at register) and invalidation
 *   (DEL in Redis + cookie expiry on logout).
 * - Next.js proxy checks only for cookie *presence* as a UX guard.
 *   The Go backend's CSRF + session validation is the actual security gate.
 * - Client-side auth state should be derived from the /me API endpoint response,
 *   not from document.cookie (HttpOnly cookies are invisible to JS).
 *
 * ## IST vs Real Token
 * - **IST (Intermediate Session Token) stage**: The user clicked a magic link and was
 *   redirected to /register?session_ref=.... No `somo_sid` cookie exists yet.
 *   The frontend should show the registration form to let the user create their tenant.
 * - **Real token stage**: After POST /api/auth/register succeeds, the backend sets the
 *   `somo_sid` cookie. The user has access to `/` (dashboard) and other protected routes.
 *
 * To determine which stage you're in on the server side (proxy):
 *   cookie `somo_sid` exists    → Real token → allow access to dashboard
 *   query param `session_ref`    → IST stage  → on /register page
 *   neither                      → Not authenticated → redirect to /login
 */

/** Name of the HttpOnly session cookie set by the Go backend. */
export const SESSION_COOKIE_NAME = "somo_sid";

/** Name of the signed role cookie set by the Go backend (not HttpOnly). */
export const ROLE_COOKIE_NAME = "somo_role";

/**
 * Role-to-routes mapping for Next.js middleware guards.
 * Each role lists the path prefixes they are allowed to access.
 * If a role is not listed, they get the default access from PROTECTED_PREFIXES.
 */
export const ROLE_ROUTES: Record<string, string[]> = {
    SYSTEM_ADMIN: ["/admin", "/admins", "/settings", "/schools", "/docs"],
    SCHOOL_ADMIN: ["/admin", "/admins", "/settings", "/schools", "/docs"],
    TEACHER: ["/docs"],
    NURSE: ["/docs"],
    FINANCE: ["/docs"],
};

/**
 * Per-role first allowed route for redirecting users who hit a forbidden path.
 */
export const ROLE_DEFAULT_ROUTES: Record<string, string> = {
    SYSTEM_ADMIN: "/admin",
    SCHOOL_ADMIN: "/admin",
    TEACHER: "/",
    NURSE: "/",
    FINANCE: "/",
};
