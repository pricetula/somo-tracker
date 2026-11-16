import { cookies } from 'next/headers'
import { NextResponse } from 'next/server'

function sessionCookieName(): string {
  return process.env.NODE_ENV === 'production'
    ? '__Host-session_token'
    : 'session_token_development'
}

/**
 * goRequest forwards a request to the Go backend, attaching the HttpOnly
 * session cookie from the browser so Go can validate the session.
 * Set-Cookie headers from Go responses are forwarded back to the browser.
 * Origin is forwarded on mutating methods for Go CSRF middleware.
 */
export async function goRequest(
  path: string,
  init: RequestInit = {}
): Promise<Response> {
  const apiUrl = process.env.PUBLIC_API_URL
  if (!apiUrl) {
    throw new Error('PUBLIC_API_URL is not configured')
  }

  const cookieStore = await cookies()
  const sessionToken = cookieStore.get(sessionCookieName())?.value

  const headers: Record<string, string> = {}
  if (init.headers) {
    if (init.headers instanceof Headers) {
      init.headers.forEach((v, k) => {
        headers[k] = v
      })
    } else if (Array.isArray(init.headers)) {
      for (const [k, v] of init.headers) {
        headers[k] = v
      }
    } else {
      Object.assign(headers, init.headers)
    }
  }

  if (sessionToken) {
    headers['Cookie'] = `${headers['Cookie'] ?? ''} ${sessionToken}`.trim()
  }

  const method = init.method ?? 'GET'
  if (['POST', 'PUT', 'PATCH', 'DELETE'].includes(method)) {
    headers['Origin'] = process.env.PUBLIC_APP_URL ?? ''
  }

  const res = await fetch(`${apiUrl}${path}`, {
    ...init,
    headers,
    cache: 'no-store',
  })

  const responseHeaders: Record<string, string> = {}
  res.headers.forEach((v, k) => {
    responseHeaders[k] = v
  })

  return new NextResponse(await res.arrayBuffer(), {
    status: res.status,
    statusText: res.statusText,
    headers: responseHeaders,
  })
}
