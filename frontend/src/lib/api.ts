import { redirect } from 'next/navigation'

function apiUrl(path: string): string {
  return new URL(path, process.env.PUBLIC_APP_URL ?? 'http://localhost:3000').href
}

/**
 * Thrown when the backend returns 401 — session expired or revoked.
 * Callers (Server Components / Route Handlers) should catch this and
 * redirect to /login after clearing the session cookie.
 */
export class SessionExpiredError extends Error {
  constructor() {
    super('Session expired or revoked')
    this.name = 'SessionExpiredError'
  }
}

/**
 * apiRequest wraps fetch for calls to Next.js BFF API routes.
 * On 401, throws SessionExpiredError so the caller can redirect to /login.
 *
 * Use this instead of raw fetch for all calls to /api/* routes.
 * Works in Server Components and Route Handlers.
 */
export async function apiRequest(
  input: string,
  init?: RequestInit
): Promise<Response> {
  const res = await fetch(apiUrl(input), init)

  if (res.status === 401) {
    throw new SessionExpiredError()
  }

  return res
}

/**
 * clearSession calls the BFF logout route, which proxies to Go
 * /api/v1/auth/logout. Go revokes the server-side session and forwards
 * a Set-Cookie header to clear the HttpOnly session cookie in the browser.
 *
 * Must be called from a Server Component or Route Handler (uses redirect).
 */
export async function clearSession(): Promise<never> {
  await fetch(apiUrl('/api/auth/logout'), { method: 'POST' })
  redirect('/login')
}
