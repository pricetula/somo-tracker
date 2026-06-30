/**
 * Tests for the proxy's `/`-as-dashboard behaviour.
 *
 * After the dashboard migration to `/`, the proxy must:
 * 1. Treat `/` as a protected route requiring auth + valid role
 * 2. Redirect unauthenticated users at `/` to /login?next=/
 * 3. Clear cookies and redirect on tampered/invalid roles
 *
 * All authenticated users with a valid role can access `/` (the dashboard).
 *
 * To run: pnpm vitest run src/__tests__/proxy-dashboard-root.test.ts
 */

import { describe, it, expect } from "vitest";
import { ROLE_ROUTES, ROLE_DEFAULT_ROUTES } from "@/lib/auth";

// ── Helpers that mirror the proxy's logic ──

/** Set of all valid roles — must stay in sync with proxy.ts. */
const VALID_ROLES = new Set(["SYSTEM_ADMIN", "SCHOOL_ADMIN", "TEACHER", "NURSE", "FINANCE"]);

/**
 * Returns where a user should be redirected if they lack dashboard access.
 * Mirrors the proxy's default-route fallback.
 */
function getDefaultRoute(role: string): string {
    return ROLE_DEFAULT_ROUTES[role] ?? "/";
}

// ── Tests ──

describe("ROLE_ROUTES integrity", () => {
    it("every valid role has a ROLE_ROUTES entry", () => {
        for (const role of VALID_ROLES) {
            expect(ROLE_ROUTES[role]).toBeDefined();
            expect(Array.isArray(ROLE_ROUTES[role])).toBe(true);
            expect(ROLE_ROUTES[role]!.length).toBeGreaterThan(0);
        }
    });

    it("every valid role has a ROLE_DEFAULT_ROUTES entry", () => {
        for (const role of VALID_ROLES) {
            expect(ROLE_DEFAULT_ROUTES[role]).toBeDefined();
        }
    });

    it("every role's default route is reachable via prefix match or is /", () => {
        for (const role of VALID_ROLES) {
            const defaultRoute = ROLE_DEFAULT_ROUTES[role]!;
            const allowedRoutes = ROLE_ROUTES[role]!;

            // `/` is served by the proxy's dedicated `/` handler (dashboard)
            // and is accessible to all authenticated users — no ROLE_ROUTES entry needed.
            if (defaultRoute === "/") continue;

            const byPrefixMatch = allowedRoutes.some((route) => defaultRoute.startsWith(route));

            expect(
                byPrefixMatch,
                `role ${role}: default "${defaultRoute}" not reachable via ${JSON.stringify(allowedRoutes)}`
            ).toBe(true);
        }
    });
});

describe("ROLE_DEFAULT_ROUTES lookup helper", () => {
    it("SYSTEM_ADMIN defaults to /admin", () => {
        expect(getDefaultRoute("SYSTEM_ADMIN")).toBe("/admin");
    });

    it("SCHOOL_ADMIN defaults to /admin", () => {
        expect(getDefaultRoute("SCHOOL_ADMIN")).toBe("/admin");
    });

    it("TEACHER defaults to /", () => {
        expect(getDefaultRoute("TEACHER")).toBe("/");
    });

    it("NURSE defaults to /", () => {
        expect(getDefaultRoute("NURSE")).toBe("/");
    });

    it("FINANCE defaults to /", () => {
        expect(getDefaultRoute("FINANCE")).toBe("/");
    });

    it("unknown role falls back to /", () => {
        expect(getDefaultRoute("NONEXISTENT")).toBe("/");
    });
});

describe("Proxy decision matrix at / (simulated)", () => {
    /**
     * Simulates the proxy's decision at `/` given auth state.
     * Returns the redirect URL path or null (NextResponse.next()).
     */
    function simulateProxyDecision(opts: {
        hasSession: boolean;
        hasRole: boolean;
        roleCookieValid: boolean;
        roleValue: string | null;
    }): "next" | string {
        const { hasSession, hasRole, roleCookieValid, roleValue } = opts;

        // Step 1: Cookie presence check
        if (!hasSession || !hasRole) {
            return "/login?next=/";
        }

        // Step 2: Cookie signature verification
        if (!roleCookieValid || !roleValue) {
            // clearCookiesAndRedirect → /login
            return "/login";
        }

        // Step 3: Valid role check
        if (!VALID_ROLES.has(roleValue)) {
            return "/login";
        }

        // Step 4: ROLE_ROUTES entry exists
        if (!ROLE_ROUTES[roleValue]) {
            return "/login";
        }

        return "next";
    }

    it("allows authenticated TEACHER with valid cookie through to /", () => {
        const result = simulateProxyDecision({
            hasSession: true,
            hasRole: true,
            roleCookieValid: true,
            roleValue: "TEACHER",
        });
        expect(result).toBe("next");
    });

    it("allows authenticated SYSTEM_ADMIN with valid cookie through to /", () => {
        const result = simulateProxyDecision({
            hasSession: true,
            hasRole: true,
            roleCookieValid: true,
            roleValue: "SYSTEM_ADMIN",
        });
        expect(result).toBe("next");
    });

    it("allows authenticated SCHOOL_ADMIN through to /", () => {
        const result = simulateProxyDecision({
            hasSession: true,
            hasRole: true,
            roleCookieValid: true,
            roleValue: "SCHOOL_ADMIN",
        });
        expect(result).toBe("next");
    });

    it("allows authenticated NURSE through to /", () => {
        const result = simulateProxyDecision({
            hasSession: true,
            hasRole: true,
            roleCookieValid: true,
            roleValue: "NURSE",
        });
        expect(result).toBe("next");
    });

    it("allows authenticated FINANCE through to /", () => {
        const result = simulateProxyDecision({
            hasSession: true,
            hasRole: true,
            roleCookieValid: true,
            roleValue: "FINANCE",
        });
        expect(result).toBe("next");
    });

    it("redirects unauthenticated user to /login?next=/", () => {
        const result = simulateProxyDecision({
            hasSession: false,
            hasRole: false,
            roleCookieValid: false,
            roleValue: null,
        });
        expect(result).toBe("/login?next=/");
    });

    it("redirects when only session cookie is present (missing role)", () => {
        const result = simulateProxyDecision({
            hasSession: true,
            hasRole: false,
            roleCookieValid: false,
            roleValue: null,
        });
        expect(result).toBe("/login?next=/");
    });

    it("redirects when only role cookie is present (missing session)", () => {
        const result = simulateProxyDecision({
            hasSession: false,
            hasRole: true,
            roleCookieValid: true,
            roleValue: "TEACHER",
        });
        expect(result).toBe("/login?next=/");
    });

    it("redirects on tampered role cookie signature", () => {
        const result = simulateProxyDecision({
            hasSession: true,
            hasRole: true,
            roleCookieValid: false,
            roleValue: "TEACHER",
        });
        expect(result).toBe("/login");
    });

    it("redirects on unknown/invalid role value (even if signed)", () => {
        const result = simulateProxyDecision({
            hasSession: true,
            hasRole: true,
            roleCookieValid: true,
            roleValue: "HACKER",
        });
        expect(result).toBe("/login");
    });
});

describe("Proxy unauthorized redirect on protected routes (simulated)", () => {
    /**
     * Simulates the proxy's decision for a protected route given auth state.
     * Returns the redirect URL path or "next".
     */
    function simulateProtectedRouteDecision(opts: {
        hasSession: boolean;
        hasRole: boolean;
        roleCookieValid: boolean;
        roleValue: string | null;
        pathname: string;
    }): "next" | string {
        const { hasSession, hasRole, roleCookieValid, roleValue, pathname } = opts;

        if (!hasSession || !hasRole) {
            return `/login?next=${pathname}`;
        }

        if (!roleCookieValid || !roleValue) {
            return "/login";
        }

        if (!VALID_ROLES.has(roleValue)) {
            return "/login";
        }

        const allowedRoutes = ROLE_ROUTES[roleValue];
        if (!allowedRoutes) {
            return "/login";
        }

        const isAllowed = allowedRoutes.some((route) => pathname.startsWith(route));
        if (!isAllowed) {
            return "/unauthorized";
        }

        return "next";
    }

    it("redirects TEACHER to /unauthorized when hitting /settings", () => {
        const result = simulateProtectedRouteDecision({
            hasSession: true,
            hasRole: true,
            roleCookieValid: true,
            roleValue: "TEACHER",
            pathname: "/settings",
        });
        expect(result).toBe("/unauthorized");
    });

    it("redirects TEACHER to /unauthorized when hitting /schools", () => {
        const result = simulateProtectedRouteDecision({
            hasSession: true,
            hasRole: true,
            roleCookieValid: true,
            roleValue: "TEACHER",
            pathname: "/schools",
        });
        expect(result).toBe("/unauthorized");
    });

    it("allows SYSTEM_ADMIN to access /settings", () => {
        const result = simulateProtectedRouteDecision({
            hasSession: true,
            hasRole: true,
            roleCookieValid: true,
            roleValue: "SYSTEM_ADMIN",
            pathname: "/settings",
        });
        expect(result).toBe("next");
    });

    it("allows SYSTEM_ADMIN to access /schools", () => {
        const result = simulateProtectedRouteDecision({
            hasSession: true,
            hasRole: true,
            roleCookieValid: true,
            roleValue: "SYSTEM_ADMIN",
            pathname: "/schools",
        });
        expect(result).toBe("next");
    });

    it("allows SYSTEM_ADMIN to access /admin", () => {
        const result = simulateProtectedRouteDecision({
            hasSession: true,
            hasRole: true,
            roleCookieValid: true,
            roleValue: "SYSTEM_ADMIN",
            pathname: "/admin",
        });
        expect(result).toBe("next");
    });

    it("redirects SYSTEM_ADMIN to /unauthorized when hitting a non-existent prefix", () => {
        const result = simulateProtectedRouteDecision({
            hasSession: true,
            hasRole: true,
            roleCookieValid: true,
            roleValue: "SYSTEM_ADMIN",
            pathname: "/other",
        });
        // /other is not protected, so the proxy would not match it
        // This simulates if /other were in PROTECTED_PREFIXES
        // Since SYSTEM_ADMIN doesn't have /other in ROLE_ROUTES, it would be unauthorized
        expect(result).toBe("/unauthorized");
    });

    it("redirects unauthenticated to /login for protected routes", () => {
        const result = simulateProtectedRouteDecision({
            hasSession: false,
            hasRole: false,
            roleCookieValid: false,
            roleValue: null,
            pathname: "/settings",
        });
        expect(result).toBe("/login?next=/settings");
    });
});
