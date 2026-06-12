import { NextRequest, NextResponse } from "next/server";
import { SESSION_COOKIE_NAME } from "@/lib/auth";

const PROTECTED_PREFIXES = ["/dashboard", "/settings", "/admin"];

/**
 * Auth state determination:
 * - **IST (Intermediate Session Token) stage**: No `somo_sid` cookie, but
 *   `session_ref` query param is present on `/register`. The user clicked a
 *   magic link but hasn't created their tenant yet.
 * - **Real token**: `somo_sid` cookie exists. The user has a valid session
 *   and can access protected routes.
 * - **Not authenticated**: Neither cookie nor valid IST query param.
 *
 * Middleware behaviour:
 * - Protected routes require the `somo_sid` cookie. Absent → redirect to /login.
 * - `/register` without `session_ref` → redirect to /login.
 * - `/login` with `somo_sid` cookie → redirect to /dashboard.
 * - `/` → rewrite based on auth state.
 */
export function middleware(req: NextRequest) {
  const { pathname, searchParams } = req.nextUrl;
  const hasSession = req.cookies.has(SESSION_COOKIE_NAME);
  const hasSessionRef = searchParams.has("session_ref");

  // ── `/` root: rewrite to dashboard if authenticated, login otherwise ──
  if (pathname === "/") {
    const dest = hasSession ? "/dashboard" : "/login";
    return NextResponse.rewrite(new URL(dest, req.url));
  }

  // ── Protected routes: require real session cookie ──
  const isProtected = PROTECTED_PREFIXES.some((p) => pathname.startsWith(p));
  if (isProtected) {
    if (!hasSession) {
      const loginUrl = new URL("/login", req.url);
      loginUrl.searchParams.set("next", pathname);
      return NextResponse.redirect(loginUrl);
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
  matcher: ["/((?!_next/static|_next/image|favicon.ico).*)"],
};
