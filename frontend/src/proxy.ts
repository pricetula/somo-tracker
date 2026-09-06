/**
 * Next.js Proxy — Auth Guard
 *
 * Runs before every matched request. Checks for the presence of the HttpOnly
 * `session_token` session cookie on protected dashboard routes. Redirects to
 * /login when the cookie is missing.
 *
 * This is a **UX guard only** — the actual security gate is the Go backend's
 * session validation + fingerprint check. The proxy performs NO session verification
 * (no Redis/Postgres calls) because:
 *   1. HttpOnly cookies cannot be forged by client-side JS
 *   2. The backend validates every authenticated API request via
 *      session middleware with fingerprint validation
 *   3. Expired/stale cookies are handled reactively: the API client's
 *      global 401 handler redirects to /logout → clears cookies → /login
 *
 * ─── Infinite Redirect Prevention ──────────────────────────────────────
 * - Public routes (login, logout, unauthorized, docs, API) are
 *   never redirected regardless of auth state.
 * - When redirecting to /login, we do NOT add a "redirect" query param
 *   because the magic-link flow always lands on the backend callback which
 *   redirects to "/" (dashboard root) after setting cookies.
 * - The /logout page explicitly clears all auth cookies before redirecting,
 *   so a subsequent proxy check on /login never enters a redirect cycle.
 *
 * @see frontend/src/lib/auth.ts      — Client-side auth utilities
 * @see backend/internal/api/middleware/session/session_middleware.go — Backend session validation
 * @see https://nextjs.org/docs/app/api-reference/file-conventions/proxy
 */

import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

// ═══════════════════════════════════════════════════════════════════════════
// Constants — must match frontend/src/lib/auth.ts and backend session middleware
// ═══════════════════════════════════════════════════════════════════════════

/** HttpOnly session cookie set by the Go backend after callback. */
const SESSION_COOKIE = "session_token";

/**
 * Routes that NEVER require authentication.
 * These always pass through without a session check.
 */
const PUBLIC_ROUTES = new Set(["/login", "/logout", "/unauthorized"]);

/**
 * Protected route prefixes — any path starting with one of these (or the
 * root `/`) requires a `session_token` cookie.
 *
 * These correspond to all routes under `app/(dashboard)/`:
 *   app/(dashboard)/page.tsx         → /
 *   app/(dashboard)/students/…       → /students/…
 *   app/(dashboard)/classes/…        → /classes/…
 *   …etc.
 */
const PROTECTED_PREFIXES = [
    "/admins",
    "/assessments",
    "/classes",
    "/curriculum",
    "/finance",
    "/nurses",
    "/parents",
    "/reports",
    "/settings",
    "/students",
    "/teachers",
];

// ═══════════════════════════════════════════════════════════════════════════
// Config — Matcher (required)
// ═══════════════════════════════════════════════════════════════════════════

/**
 * Matcher: run proxy on all request paths EXCEPT API routes, Next.js
 * static assets, image optimizations, and common metadata files.
 *
 * API routes are excluded because they are proxied to the Go backend
 * (via next.config.ts rewrites), and the backend handles its own auth.
 */
export const config = {
    matcher: ["/((?!api|_next/static|_next/image|favicon.ico|sitemap.xml|robots.txt|.*\\.png$).*)"],
};

// ═══════════════════════════════════════════════════════════════════════════
// Proxy
// ═══════════════════════════════════════════════════════════════════════════

/**
 * Auth guard that runs before every matched request.
 *
 * Decision flow:
 *   1. If the path is public (login, logout, unauthorized, docs, etc.) → allow.
 *   2. If the path is protected AND `session_token` cookie is missing → redirect
 *      to /login.
 *   3. Otherwise → allow (cookie exists or path is not explicitly protected).
 */
export function proxy(request: NextRequest) {
    const { pathname } = request.nextUrl;

    // ── Step 1: Always allow public routes ──────────────────────────────
    if (isPublicRoute(pathname)) {
        return NextResponse.next();
    }

    // ── Step 2: Check for session cookie on protected routes ────────────
    if (!request.cookies.has(SESSION_COOKIE)) {
        const loginUrl = new URL("/login", request.url);
        return NextResponse.redirect(loginUrl);
    }

    // ── Step 3: Cookie exists — allow through ───────────────────────────
    return NextResponse.next();
}

// ═══════════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════════

/**
 * Returns `true` if the given pathname should bypass the auth check.
 *
 * A path is considered public if it is:
 * - An exact match in PUBLIC_ROUTES (login, logout, unauthorized)
 * - A prefix match for docs, API, or Next.js internal paths
 * - NOT an explicit protected prefix
 * - The root `/` IS protected (dashboard root)
 */
function isPublicRoute(pathname: string): boolean {
    // Root is the dashboard — requires auth
    if (pathname === "/") {
        return false;
    }

    // Exact public route matches
    if (PUBLIC_ROUTES.has(pathname)) {
        return true;
    }

    // Public prefix-based routes
    if (
        pathname.startsWith("/docs") ||
        pathname.startsWith("/api") ||
        pathname.startsWith("/_next")
    ) {
        return true;
    }

    // Explicitly protected dashboard routes
    for (const prefix of PROTECTED_PREFIXES) {
        if (pathname === prefix || pathname.startsWith(prefix + "/")) {
            return false;
        }
    }

    // Unknown/unguessed routes → allow (they'll 404 naturally if they don't
    // exist, and we don't want to block valid future routes)
    return true;
}
