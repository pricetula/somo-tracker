import { NextResponse } from 'next/server'
import type { NextRequest } from 'next/server'

/**
 * Next.js 16 Proxy (replaces middleware.ts).
 *
 * Optimistic UX gate: checks cookie PRESENCE only — zero backend calls.
 * Session VALIDATION is enforced by the backend on every API request
 * via auth middleware.
 *
 * Cookie name comes from SESSION_COOKIE_NAME env var (matches backend).
 * In development: "session" (no __Host- prefix, works on HTTP)
 * In production: "__Host-session" (requires HTTPS)
 * Proxy runs on the server and can read the incoming Cookie header.
 */

function sessionCookieName(): string {
  const name = process.env.SESSION_COOKIE_NAME
  if (!name) {
    throw new Error('SESSION_COOKIE_NAME is not configured')
  }
  return name
}

function onboardingCookieName(): string {
  const sessionName = sessionCookieName()
  if (sessionName.startsWith('__Host-')) {
    return '__Host-needs_onboarding'
  }
  return 'needs_onboarding'
}

let configValidated = false

function validateConfig() {
  if (configValidated) return
  const sessionName = sessionCookieName()
  const appEnv = process.env.APP_ENV
  console.log('[proxy] config', { sessionCookie: sessionName, onboardingCookie: onboardingCookieName(), appEnv })

  if (appEnv === 'production' && !sessionName.startsWith('__Host-')) {
    throw new Error('SESSION_COOKIE_NAME must start with __Host- in production')
  }
  configValidated = true
}

const PUBLIC_ROUTES = ['/login', '/auth/callback']
const ONBOARDING_ROUTES = ['/onboarding', '/api/auth/onboard']

export function proxy(request: NextRequest) {
  // Validate config on first request (lazy, not at build time)
  validateConfig()

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
  const isOnboarding = ONBOARDING_ROUTES.some((p) => pathname === p || pathname.startsWith(p + '/'))

  // Read HttpOnly session cookie (server-side only)
  const sessionToken = request.cookies.get(sessionCookieName())?.value

  // Read onboarding flag cookie (set by backend when temp session issued)
  const needsOnboarding = request.cookies.get(onboardingCookieName())?.value === 'true'

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

  // Full session - redirect away from onboarding/login; root IS the dashboard so let it through
  if (isPublic || isOnboarding) {
    return NextResponse.redirect(new URL('/', request.url))
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