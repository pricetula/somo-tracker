import { NextResponse } from 'next/server'
import type { NextRequest } from 'next/server'

/**
 * Next.js 16 Proxy (replaces middleware.ts).
 *
 * Optimistic UX gate: checks cookie PRESENCE only — zero backend calls.
 * Session VALIDATION is enforced by the backend on every API request
 * via auth middleware.
 *
 * Cookie __Host-session_token is HttpOnly, so browser JS cannot read it.
 * Proxy runs on the server and can read the incoming Cookie header.
 */

const PUBLIC_ROUTES = ['/login', '/auth/callback']
const PROTECTED_PREFIXES = ['/dashboard', '/students', '/schools']
const ONBOARDING_ROUTES = ['/onboarding', '/api/auth/onboard']

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
  const isOnboarding = ONBOARDING_ROUTES.some((p) => pathname === p || pathname.startsWith(p + '/'))
  const isRoot = pathname === '/'

  // Read HttpOnly session cookie (server-side only)
  const sessionToken =
    request.cookies.get('__Host-session_token')?.value ??
    request.cookies.get('session_token_development')?.value

  // Read onboarding flag cookie (set by backend when temp session issued)
  const needsOnboarding = request.cookies.get('needs_onboarding')?.value === 'true'

  // No session at all
  if (!sessionToken) {
    if (isPublic) {
      return NextResponse.next()
    }
    // Root or any other path with no session → redirect to login
    return NextResponse.redirect(new URL('/login', request.url))
  }

  // Has temp session (onboarding not complete)
  if (needsOnboarding) {
    if (isOnboarding) {
      return NextResponse.next()
    }
    return NextResponse.redirect(new URL('/onboarding', request.url))
  }

  // Full session - redirect away from onboarding/login/root
  if (isPublic || isOnboarding || isRoot) {
    return NextResponse.redirect(new URL('/dashboard', request.url))
  }

  return NextResponse.next()
}

export const config = {
  matcher: [
    /*
     * Match all request paths except:
     * - /api/* (BFF API routes — backend handles auth)
     * - /_next/static, /_next/image (static assets)
     * - /favicon.ico, sitemap.xml, robots.txt
     */
    '/((?!api|_next/static|_next/image|favicon.ico|sitemap.xml|robots.txt).*)',
  ],
}
