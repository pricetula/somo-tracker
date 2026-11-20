import { NextResponse, type NextRequest } from 'next/server'

const BACKEND_URL = process.env.PUBLIC_API_URL ?? 'http://localhost:8080'

// Paths that don't require authentication
const PUBLIC_PATHS = ['/login', '/api/auth/otp/request', '/api/auth/otp/verify']

// Paths that require onboarding completion
const ONBOARDING_PATHS = ['/onboarding', '/api/auth/onboard']

export async function middleware(req: NextRequest) {
  const { pathname } = req.nextUrl

  // Skip static assets and _next
  if (pathname.startsWith('/_next') || pathname.startsWith('/api/_next')) {
    return NextResponse.next()
  }

  // Read session cookie
  const sessionToken = req.cookies.get('__Host-session')?.value

  let authenticated = false
  let sessionType = ''
  let needsOnboarding = false

  if (sessionToken) {
    try {
      const res = await fetch(`${BACKEND_URL}/api/v1/auth/session`, {
        headers: {
          Cookie: `__Host-session=${sessionToken}`,
        },
        cache: 'no-store',
      })
      if (res.ok) {
        const data = await res.json()
        authenticated = data.authenticated
        sessionType = data.session_type ?? ''
        needsOnboarding = data.needs_onboarding ?? false
      }
    } catch {
      // Backend unreachable - treat as unauthenticated
    }
  }

  const isPublicPath = PUBLIC_PATHS.some((p) => pathname === p || pathname.startsWith(p + '/'))
  const isOnboardingPath = ONBOARDING_PATHS.some((p) => pathname === p || pathname.startsWith(p + '/'))
  const isApiPath = pathname.startsWith('/api/')

  // No session
  if (!authenticated) {
    if (isPublicPath || isApiPath) {
      return NextResponse.next()
    }
    return NextResponse.redirect(new URL('/login', req.nextUrl))
  }

  // Has temp session (onboarding not complete)
  if (sessionType === 'temporary' || needsOnboarding) {
    if (isOnboardingPath || pathname === '/api/auth/onboard') {
      return NextResponse.next()
    }
    return NextResponse.redirect(new URL('/onboarding', req.nextUrl))
  }

  // Full session - redirect away from onboarding/login
  if (isOnboardingPath || pathname === '/login') {
    return NextResponse.redirect(new URL('/dashboard', req.nextUrl))
  }

  return NextResponse.next()
}

export const config = {
  matcher: ['/((?!_next|.*\\..*).*)'],
}
