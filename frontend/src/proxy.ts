import { NextRequest, NextResponse } from "next/server";
import { SESSION_COOKIE_NAME, ROLE_COOKIE_NAME, ROLE_ROUTES, ROLE_DEFAULT_ROUTES } from "@/lib/auth";

const PROTECTED_PREFIXES = ["/dashboard", "/settings", "/admin"];

// Exhaustive set of valid roles — verifiedRole must be one of these or access is denied.
// Keeps role checking honest even if ROLE_ROUTES is missing an entry.
const VALID_ROLES = new Set(["SYSTEM_ADMIN", "SCHOOL_ADMIN", "TEACHER", "SUPPORT_STAFF"]);

/**
 * Converts a hex string back to a Uint8Array.
 * Used to decode the stored signature before passing to crypto.subtle.verify.
 */
function hexToBytes(hex: string): Uint8Array<ArrayBuffer> {
  if (hex.length % 2 !== 0) return new Uint8Array(new ArrayBuffer(0));
  const buf = new ArrayBuffer(hex.length / 2);
  const bytes = new Uint8Array(buf);
  for (let i = 0; i < hex.length; i += 2) {
    bytes[i / 2] = parseInt(hex.slice(i, i + 2), 16);
  }
  return bytes;
}

/**
 * Verifies a signed cookie value using HMAC-SHA256 via the Web Crypto API.
 * The cookie format is: value.hexsignature
 *
 * Uses crypto.subtle.verify (timing-safe by spec) rather than re-signing and
 * doing a manual character comparison.
 *
 * Returns the role string if the signature is valid, or null if tampered/malformed.
 */
async function verifySignedCookie(
  cookieValue: string,
  secret: string,
): Promise<string | null> {
  const lastDot = cookieValue.lastIndexOf(".");
  if (lastDot === -1) return null;

  const value = cookieValue.slice(0, lastDot);
  const expectedSigHex = cookieValue.slice(lastDot + 1);

  if (!value || !expectedSigHex) return null;

  try {
    const encoder = new TextEncoder();
    const keyData = encoder.encode(secret);

    // Import with "verify" usage — correct for signature verification
    const key = await crypto.subtle.importKey(
      "raw",
      keyData,
      { name: "HMAC", hash: "SHA-256" },
      false,
      ["verify"],
    );

    const valueBytes = encoder.encode(value);
    const sigBytes: Uint8Array<ArrayBuffer> = hexToBytes(expectedSigHex);

    // crypto.subtle.verify is timing-safe by spec — no manual loop needed
    const isValid = await crypto.subtle.verify("HMAC", key, sigBytes, valueBytes);

    return isValid ? value : null;
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
  // NOTE: `pathname` is always a relative path from req.nextUrl — not user-supplied.
  // The login page handler must still validate the `next` param before redirecting
  // to it, to prevent open-redirect if that handler ever accepts external input.
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
 * Proxy behaviour:
 * - Protected routes require BOTH `somo_sid` and `somo_role` cookies.
 *   Missing either → redirect to /login.
 * - `somo_role` signature is verified using HMAC-SHA256 via crypto.subtle.verify.
 *   Tampered/invalid → cookies cleared → redirect to /login.
 * - Verified role must be a known valid role (VALID_ROLES set).
 *   Unknown role → cookies cleared → redirect to /login.
 * - Verified role is checked against ROLE_ROUTES for the requested path.
 *   No entry in ROLE_ROUTES → deny (not silently allow).
 *   Not permitted → redirect to that role's default route.
 * - `/register` without `session_ref` → redirect to /login.
 * - `/login` with BOTH `somo_sid` and `somo_role` cookies → redirect to /dashboard.
 *   (Requiring both prevents a redirect loop when only one cookie is present.)
 * - `/` → redirect (not rewrite) based on auth state so the URL updates correctly.
 */
export async function proxy(req: NextRequest) {
  const { pathname, searchParams } = req.nextUrl;
  const hasSession = req.cookies.has(SESSION_COOKIE_NAME);
  const hasRole = req.cookies.has(ROLE_COOKIE_NAME);
  const hasSessionRef = searchParams.has("session_ref");

  // ── `/` root: redirect to dashboard if authenticated, login otherwise ──
  // Using redirect (not rewrite) so the browser URL updates correctly.
  if (pathname === "/") {
    const dest = hasSession && hasRole ? "/dashboard" : "/login";
    return NextResponse.redirect(new URL(dest, req.url));
  }

  // ── Protected routes: require BOTH cookies + valid, permitted role ──
  const isProtected = PROTECTED_PREFIXES.some((p) => pathname.startsWith(p));
  if (isProtected) {
    if (!hasSession || !hasRole) {
      const loginUrl = new URL("/login", req.url);
      loginUrl.searchParams.set("next", pathname);
      return NextResponse.redirect(loginUrl);
    }

    const cookieSecret = process.env.COOKIE_SECRET;
    if (!cookieSecret) {
      // Misconfiguration — log loudly and deny access rather than failing open
      console.error("[proxy] COOKIE_SECRET is not set — blocking all protected route access");
      return NextResponse.redirect(new URL("/login", req.url));
    }

    const roleCookieValue = req.cookies.get(ROLE_COOKIE_NAME)!.value;
    const verifiedRole = await verifySignedCookie(roleCookieValue, cookieSecret);

    if (!verifiedRole) {
      // Tampered or invalid signature
      return clearCookiesAndRedirect(req, pathname);
    }

    // Guard against a validly-signed but unrecognised role value
    if (!VALID_ROLES.has(verifiedRole)) {
      return clearCookiesAndRedirect(req, pathname);
    }

    const allowedRoutes = ROLE_ROUTES[verifiedRole];
    if (!allowedRoutes) {
      // Role exists in VALID_ROLES but has no entry in ROLE_ROUTES — deny, don't allow
      return clearCookiesAndRedirect(req, pathname);
    }

    const isAllowed = allowedRoutes.some((route) => pathname.startsWith(route));
    if (!isAllowed) {
      const defaultRoute = ROLE_DEFAULT_ROUTES[verifiedRole] || "/dashboard";
      return NextResponse.redirect(new URL(defaultRoute, req.url));
    }

    return NextResponse.next();
  }

  // ── Register page: IST stage if session_ref present ──
  if (pathname === "/register") {
    if (hasSession) {
      return NextResponse.redirect(new URL("/dashboard", req.url));
    }
    if (!hasSessionRef) {
      return NextResponse.redirect(new URL("/login", req.url));
    }
    // IST stage — allow through to /register?session_ref=...
    return NextResponse.next();
  }

  // ── Login page: only redirect away if BOTH cookies are present ──
  // Checking only hasSession caused a redirect loop when somo_role was
  // missing or tampered — the protected route handler would immediately
  // bounce back to /login.
  if (pathname === "/login") {
    if (hasSession && hasRole) {
      return NextResponse.redirect(new URL("/dashboard", req.url));
    }
    return NextResponse.next();
  }

  return NextResponse.next();
}

export const config = {
  matcher: ["/((?!_next/static|_next/image|favicon.ico|api/|robots.txt).*)"],
};