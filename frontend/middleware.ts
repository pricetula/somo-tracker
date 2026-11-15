import { NextResponse } from 'next/server'
import type { NextRequest } from 'next/server'
import { cookies } from 'next/headers'

export function middleware(req: NextRequest) {
  const cookieStore = cookies()
  const sessionToken =
    cookieStore.get('__Host-session_token')?.value ||
    cookieStore.get('session_token_development')?.value

  const isAuthPage =
    req.nextUrl.pathname === '/login' ||
    req.nextUrl.pathname.startsWith('/auth/callback') ||
    req.nextUrl.pathname.startsWith('/api/auth') ||
    req.nextUrl.pathname.startsWith('/api/onboarding')

  // If no session and not on an auth page, redirect to login
  if (!sessionToken && !isAuthPage) {
    return NextResponse.redirect(new URL('/login', req.nextUrl))
  }

  // If has session and on login/callback page, redirect to dashboard
  if (sessionToken && (req.nextUrl.pathname === '/login' || req.nextUrl.pathname.startsWith('/auth/callback'))) {
    return NextResponse.redirect(new URL('/dashboard', req.nextUrl))
  }

  return NextResponse.next()
}

export const config = {
  matcher: ['/((?!_next/static|_next/image|favicon.ico).*)'],
}
