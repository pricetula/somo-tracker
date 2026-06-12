import { NextRequest, NextResponse } from "next/server";
import { SESSION_COOKIE_NAME, ROLE_COOKIE_NAME, ROLE_ROUTES, ROLE_DEFAULT_ROUTES } from "@/lib/auth";

const PROTECTED_PREFIXES = ["/dashboard", "/settings", "/admin"];

/**
 * Verifies a signed cookie value using HMAC-SHA256 via the Web Crypto API.
 * The cookie format is: value.hexsignature
 * Returns the role string if valid, or null if tampered / malformed.
 */
async function verifySignedCookie(
  cookieValue: string,
  secret: string,
): Promise<string | null> {
  const lastDot = cookieValue.lastIndexOf(".");
  if (lastDot === -1) return null;

  const value = cookieValue.slice(0, lastDot);
  const expectedSig = cookieValue.slice(lastDot + 1);

  if (!value || !expectedSig) return null;

  try {
    // Import the secret as an HMAC key
    const encoder = new TextEncoder();
    const keyData = encoder.encode(secret);
    const key = await crypto.subtle.importKey(
      "raw",
      keyData,
      { name: "HMAC", hash: "SHA-256" },
      false,
      ["sign"],
    );

    // Sign the value
    const valueBytes = encoder.encode(value);
    const sigBytes = await crypto.subtle.sign("HMAC", key, valueBytes);

    // Convert signature to hex
    const sigHex = Array.from(new Uint8Array(sigBytes))
      .map((b) => b.toString(16).padStart(2, "0"))
      .join("");

    // Constant-time comparison via timing-safe string comparison
    if (sigHex.length !== expectedSig.length) return null;
    let mismatch = 0;
    for (let i = 0; i < sigHex.length; i++) {
      mismatch |= sigHex.charCodeAt(i) ^ expectedSig.charCodeAt(i);
    }

    return mismatch === 0 ? value : null;
  } catch {
    return null;
  }
}

/**
 * Creates a NextResponse that clears both auth cookies (somo_sid and somo_role)
 * and redirects to /login.
 */
function clearCookiesAndRedirect(req: NextRequest, pathname: string): NextResponse {
  const loginUrl = new URL("/login", req.url);
  loginUrl.searchParams.set("next", pathname);
  const res = NextResponse.redirect(loginUrl);

  res.cookies.set(SESSION_COOKIE_NAME, "", {
    httpOnly: true,
    secure: process.env.NODE_ENV !== "development",
    sameSite: "lax",
    path: "/",
    maxAge: 0,
  });
  res.cookies.set(ROLE_COOKIE_NAME, "", {
    httpOnly: false,
    secure: process.env.NODE_ENV !== "development",
    sameSite: "lax",
    path: "/",
    maxAge: 0,
  });

  return res;
}

/**
 * Auth state determination:
 * - **IST (Intermediate Session Token) stage**: No `somo_sid` cookie, but
 *   `session_ref` query param is present on `/register`. The user clicked a
 *   magic link but hasn't created their tenant yet.
 * - **Real token**: `somo_sid` + `somo_role` cookies exist. The user has a
 *   valid session and can access protected routes based on their role.
 * - **Not authenticated**: Neither cookie nor valid IST query param.
 *
 * Middleware behaviour:
 * - Protected routes require BOTH `somo_sid` and `somo_role` cookies.
 *   Missing either → redirect to /login.
 * - `somo_role` signature is verified using HMAC-SHA256.
 *   Tampered/invalid → cookies cleared → redirect to /login.
 * - Verified role is checked against ROLE_ROUTES for the requested path.
 *   Not permitted → redirect to that role's first allowed route.
 * - `/register` without `session_ref` → redirect to /login.
 * - `/login` with `somo_sid` cookie → redirect to /dashboard.
 * - `/` → rewrite based on auth state.
 */
export async function proxy(req: NextRequest) {
  const { pathname, searchParams } = req.nextUrl;
  const hasSession = req.cookies.has(SESSION_COOKIE_NAME);
  const hasRole = req.cookies.has(ROLE_COOKIE_NAME);
  const hasSessionRef = searchParams.has("session_ref");

  // ── `/` root: rewrite to dashboard if authenticated, login otherwise ──
  if (pathname === "/") {
    const dest = hasSession ? "/dashboard" : "/login";
    return NextResponse.rewrite(new URL(dest, req.url));
  }

  // ── Protected routes: require BOTH cookies + valid role ──
  const isProtected = PROTECTED_PREFIXES.some((p) => pathname.startsWith(p));
  if (isProtected) {
    // Both cookies must be present
    if (!hasSession || !hasRole) {
      const loginUrl = new URL("/login", req.url);
      loginUrl.searchParams.set("next", pathname);
      return NextResponse.redirect(loginUrl);
    }

    // Verify the role cookie signature
    const cookieSecret = process.env.COOKIE_SECRET;
    if (!cookieSecret) {
      // Misconfiguration — deny access
      return NextResponse.redirect(new URL("/login", req.url));
    }

    const roleCookieValue = req.cookies.get(ROLE_COOKIE_NAME)!.value;
    const verifiedRole = await verifySignedCookie(roleCookieValue, cookieSecret);

    if (!verifiedRole) {
      // Tampered or invalid signature — clear both cookies and redirect
      return clearCookiesAndRedirect(req, pathname);
    }

    // Check role is permitted on this path
    const allowedRoutes = ROLE_ROUTES[verifiedRole];
    if (allowedRoutes) {
      const isAllowed = allowedRoutes.some((route) => pathname.startsWith(route));
      if (!isAllowed) {
        // Redirect to this role's first allowed route
        const defaultRoute = ROLE_DEFAULT_ROUTES[verifiedRole] || "/dashboard";
        return NextResponse.redirect(new URL(defaultRoute, req.url));
      }
    }

    return NextResponse.next();
  }

  // ── Register page: IST stage if session_ref present ──
  if (pathname === "/register") {
    if (hasSession) {
      // Already authenticated → go to dashboard
      return NextResponse.redirect(new URL("/dashboard", req.url));
    }
    if (!hasSessionRef) {
      // No session_ref → redirect to login
      return NextResponse.redirect(new URL("/login", req.url));
    }
    // IST stage — allow through to /register?session_ref=...
    return NextResponse.next();
  }

  // ── Login page: if already authenticated, redirect to dashboard ──
  if (pathname === "/login") {
    if (hasSession) {
      return NextResponse.redirect(new URL("/dashboard", req.url));
    }
    return NextResponse.next();
  }

  return NextResponse.next();
}

export const config = {
  matcher: ["/((?!_next/static|_next/image|favicon.ico|api/|robots.txt).*)"],
};
