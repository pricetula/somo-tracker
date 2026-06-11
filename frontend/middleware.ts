import { NextRequest, NextResponse } from "next/server";

const PROTECTED_PREFIXES = ["/dashboard", "/settings", "/admin"];

/**
 * Next.js edge middleware that checks for the presence of the `somo_session`
 * cookie on every protected route. If absent, redirects to `/login`.
 *
 * This is a UX guard only — it is not a security boundary.
 * The Go backend's ValidateSession middleware is the actual security gate.
 */
export function middleware(req: NextRequest) {
  const isProtected = PROTECTED_PREFIXES.some((p) =>
    req.nextUrl.pathname.startsWith(p),
  );

  if (isProtected && !req.cookies.get("somo_session")) {
    const loginUrl = new URL("/login", req.url);
    loginUrl.searchParams.set("next", req.nextUrl.pathname);
    return NextResponse.redirect(loginUrl);
  }

  return NextResponse.next();
}

export const config = {
  matcher: ["/((?!_next/static|_next/image|favicon.ico).*)"],
};
