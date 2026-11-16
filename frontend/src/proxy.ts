import { NextResponse } from 'next/server'
import type { NextRequest } from 'next/server'

/**
 * Next.js 16 Proxy (replaces middleware.ts).
 *
 * Optimistic UX gate: checks cookie PRESENCE only — zero backend calls.
 * Session VALIDATION is enforced by the Go backend on every API request
 * via AuthMiddleware.RequireAuth().
 *
 * Cookie __Host-session_token is HttpOnly, so browser JS cannot read it.
 * Proxy runs on the server and can read the incoming Cookie header.
 */

const PUBLIC_ROUTES = ['/login', '/auth/callback', '/']
const PROTECTED_PREFIXES = ['/dashboard', '/students', '/schools']

export function proxy(request: NextRequest) {
  const { pathname } = request.nextUrl

  // Never proxy API routes, static assets, or Next.js internals
  if (
    pathname.startsWith('/api/') ||
    pathname.startsWith('/_next/') ||
    pathname.startsWith('/favicon.ico')
  ) {
    return NextResponse.next()
  }

  const isPublic = PUBLIC_ROUTES.includes(pathname)
  const isProtected = PROTECTED_PREFIXES.some((p) => pathname.startsWith(p))

  // Read HttpOnly session cookie (server-side only)
  const sessionToken =
    request.cookies.get('__Host-session_token')?.value ??
    request.cookies.get('session_token_development')?.value

  // Protected route without session cookie → redirect to login
  if (isProtected && !sessionToken) {
    return NextResponse.redirect(new URL('/login', request.url))
  }

  // Public route with session cookie → redirect to dashboard
  if (isPublic && sessionToken) {
    return NextResponse.redirect(new URL('/dashboard', request.url))
  }

  return NextResponse.next()
}

export const config = {
  matcher: [
    /*
     * Match all request paths except:
     * - /api/* (BFF API routes — Go handles auth)
     * - /_next/static, /_next/image (static assets)
     * - /favicon.ico, sitemap.xml, robots.txt
     */
    '/((?!api|_next/static|_next/image|favicon.ico|sitemap.xml|robots.txt).*)',
  ],
}
