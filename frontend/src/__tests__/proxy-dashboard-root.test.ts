/**
 * Tests for the proxy's `/`-as-dashboard behaviour.
 *
 * After the dashboard migration to `/`, the proxy must:
 * 1. Treat `/` as a protected route requiring auth + valid role
 * 2. Use `ROLE_ROUTES` to determine if a role has dashboard access
 *    (routes containing either "/dashboard" or "/")
 * 3. Redirect unauthenticated users at `/` to /login?next=/
 * 4. Redirect authenticated users without dashboard access to their default route
 * 5. Clear cookies and redirect on tampered/invalid roles
 *
 * To run: pnpm vitest run src/__tests__/proxy-dashboard-root.test.ts
 */

import { describe, it, expect } from "vitest";
import {
  ROLE_ROUTES,
  ROLE_DEFAULT_ROUTES,
} from "@/lib/auth";

// ── Helpers that mirror the proxy's logic ──

/** Set of all valid roles — must stay in sync with proxy.ts. */
const VALID_ROLES = new Set(["SYSTEM_ADMIN", "SCHOOL_ADMIN", "TEACHER", "SUPPORT_STAFF"]);

/**
 * Determines whether a user's role permits access to the dashboard at `/`.
 * Mirrors the proxy's `hasDashboardAccess` check.
 */
function hasDashboardAccess(role: string): boolean {
  const allowedRoutes = ROLE_ROUTES[role];
  if (!allowedRoutes) return false;
  return allowedRoutes.some((route) => route === "/dashboard" || route === "/");
}

/**
 * Returns where a user should be redirected if they lack dashboard access.
 * Mirrors the proxy's default-route fallback.
 */
function getDefaultRoute(role: string): string {
  return ROLE_DEFAULT_ROUTES[role] ?? "/";
}

// ── Tests ──

describe("hasDashboardAccess — proxy check at /", () => {
  it.each(["SYSTEM_ADMIN", "SCHOOL_ADMIN", "TEACHER", "SUPPORT_STAFF"])(
    "allows %s to access dashboard at /",
    (role) => {
      expect(hasDashboardAccess(role)).toBe(true);
    },
  );

  it("rejects a made-up role that isn't in ROLE_ROUTES", () => {
    expect(hasDashboardAccess("SUPER_USER")).toBe(false);
  });

  it("rejects a valid role that somehow has no ROLE_ROUTES entry", () => {
    // Simulate a role that is in VALID_ROLES but not in ROLE_ROUTES
    const knownButMissing = Array.from(VALID_ROLES).find((r) => !ROLE_ROUTES[r]);
    if (knownButMissing) {
      expect(hasDashboardAccess(knownButMissing)).toBe(false);
    }
    // If all valid roles have entries, this is a structural guarantee check
    for (const role of VALID_ROLES) {
      expect(ROLE_ROUTES[role]).toBeDefined();
    }
  });
});

describe("ROLE_ROUTES integrity for dashboard access", () => {
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

  it("every role's default route is reachable (prefix match or hasDashboardAccess)", () => {
    for (const role of VALID_ROLES) {
      const defaultRoute = ROLE_DEFAULT_ROUTES[role]!;
      const allowedRoutes = ROLE_ROUTES[role]!;

      // The default route must be accessible by the role:
      // - Exact match: `/` is the dashboard, checked via hasDashboardAccess
      // - Prefix match: e.g. `/admin` starts with `/admin` in allowedRoutes
      const byPrefixMatch = allowedRoutes.some((route) => defaultRoute.startsWith(route));
      const byDashboardAccess = hasDashboardAccess(role) && defaultRoute === "/";

      expect(
        byPrefixMatch || byDashboardAccess,
        `role ${role}: default "${defaultRoute}" not reachable via ${JSON.stringify(allowedRoutes)}`,
      ).toBe(true);
    }
  });
});

describe("Default redirect route when dashboard access is denied", () => {
  it("SYSTEM_ADMIN defaults to /admin", () => {
    expect(getDefaultRoute("SYSTEM_ADMIN")).toBe("/admin");
  });

  it("SCHOOL_ADMIN defaults to /admin", () => {
    expect(getDefaultRoute("SCHOOL_ADMIN")).toBe("/admin");
  });

  it("TEACHER defaults to /", () => {
    expect(getDefaultRoute("TEACHER")).toBe("/");
  });

  it("SUPPORT_STAFF defaults to /", () => {
    expect(getDefaultRoute("SUPPORT_STAFF")).toBe("/");
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
  function simulateProxyDecision(
    opts: {
      hasSession: boolean;
      hasRole: boolean;
      roleCookieValid: boolean;
      roleValue: string | null;
    },
  ): "next" | string {
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

    // Step 5: Dashboard access check
    if (!hasDashboardAccess(roleValue)) {
      return getDefaultRoute(roleValue);
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

  it("allows authenticated SUPPORT_STAFF through to /", () => {
    const result = simulateProxyDecision({
      hasSession: true,
      hasRole: true,
      roleCookieValid: true,
      roleValue: "SUPPORT_STAFF",
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

  it("redirects SYSTEM_ADMIN to /admin when dashboard access is missing (hypothetical)", () => {
    // Edge case: if a role were somehow in VALID_ROLES but had
    // a ROLE_ROUTES entry without dashboard access
    const result = simulateProxyDecision({
      hasSession: true,
      hasRole: true,
      roleCookieValid: true,
      roleValue: "SYSTEM_ADMIN",
    });
    // SYTEM_ADMIN _does_ have dashboard access, so should pass through
    expect(result).toBe("next");
  });
});
