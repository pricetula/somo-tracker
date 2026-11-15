import { cookies } from 'next/headers'
import { NextResponse } from 'next/server'
import { goRequest, forwardSetCookieHeaders, getSessionCookieName } from './go-client'

/**
 * Auth gateway — server-side replacement for Next.js middleware.
 *
 * Runs on the server in API routes and Server Components. Reads the session
 * cookie, validates it against Go /api/v1/me, and returns the authenticated
 * user or null.
 */

export interface AuthUser {
  user_id: string
  full_name: string
  email: string
  avatar_url?: string | null
  role: string
  active_school?: { id: string; name: string } | null
  schools: Array<{ id: string; name: string }>
}

/**
 * Read the session cookie from the current request and validate it against
 * Go. Returns the authenticated user or null if not logged in.
 */
export async function getAuthUser(): Promise<AuthUser | null> {
  const cookieStore = await cookies()
  const sessionToken =
    cookieStore.get(getSessionCookieName())?.value ||
    cookieStore.get('session_token_development')?.value

  if (!sessionToken) {
    return null
  }

  const result = await goRequest('/api/v1/me', {
    method: 'GET',
    forwardCookies: true,
  })

  if (result.status !== 200 || !result.data) {
    return null
  }

  return result.data as AuthUser
}

/**
 * Require authentication. Returns a redirect response if the user is not
 * logged in, or null if they are. Use in Server Components to guard pages.
 */
export async function requireAuth(): Promise<NextResponse | null> {
  const user = await getAuthUser()
  if (user) {
    return null
  }
  return NextResponse.redirect(new URL('/login', process.env.PUBLIC_APP_URL || 'http://localhost:3000'))
}

/**
 * Require no authentication (guest only). Returns a redirect response if the
 * user is already logged in, or null if they are a guest. Use on login/
 * callback pages to bounce authenticated users to dashboard.
 */
export async function requireGuest(): Promise<NextResponse | null> {
  const user = await getAuthUser()
  if (!user) {
    return null
  }
  return NextResponse.redirect(new URL('/dashboard', process.env.PUBLIC_APP_URL || 'http://localhost:3000'))
}

/**
 * Forward an incoming Next.js request to Go with the session cookie and
 * Origin header. Returns the Go response with Set-Cookie headers forwarded
 * back to the browser. Use in API route handlers.
 */
export async function proxyToGo(
  path: string,
  req: Request,
  opts: { method?: string; body?: unknown } = {}
): Promise<Response> {
  const { method = req.method, body } = opts

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  }

  // Forward session cookie
  const cookieStore = await cookies()
  const sessionToken =
    cookieStore.get(getSessionCookieName())?.value ||
    cookieStore.get('session_token_development')?.value
  if (sessionToken) {
    headers['Cookie'] = `${getSessionCookieName()}=${sessionToken}`
  }

  // Forward Origin for CSRF on mutations
  if (['POST', 'PUT', 'DELETE', 'PATCH'].includes(method)) {
    headers['Origin'] = process.env.PUBLIC_APP_URL || 'http://localhost:3000'
  }

  const res = await fetch(`${process.env.PUBLIC_API_URL || 'http://127.0.0.1:8080'}${path}`, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
    cache: 'no-store',
  })

  const setCookieHeaders = res.headers.getSetCookie ? res.headers.getSetCookie() : res.headers.getAll('set-cookie')

  const nextRes = NextResponse.json(
    await res.json().catch(() => null),
    { status: res.status }
  )
  forwardSetCookieHeaders(nextRes, setCookieHeaders)
  return nextRes
}
