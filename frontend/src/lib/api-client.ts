import { cookies } from 'next/headers'
import { NextResponse } from 'next/server'

function sessionCookieName(): string {
  return '__Host-session'
}

/**
 * apiRequest forwards a request to the backend API, attaching the HttpOnly
 * session cookie from the browser. Set-Cookie headers from the backend
 * responses are forwarded back to the browser.
 * Origin is forwarded on mutating methods for CSRF middleware.
 */
export async function apiRequest(
  path: string,
  init: RequestInit = {}
): Promise<Response> {
  const apiUrl = process.env.PUBLIC_API_URL
  if (!apiUrl) {
    throw new Error('PUBLIC_API_URL is not configured')
  }

  const cookieStore = await cookies()
  const sessionToken = cookieStore.get(sessionCookieName())?.value

  // Forward cookies as proper "name=value" pairs so the backend can parse them.
  const cookieParts: string[] = []
  if (sessionToken) {
    cookieParts.push(`${sessionCookieName()}=${sessionToken}`)
  }

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

  if (cookieParts.length) {
    headers['Cookie'] = `${headers['Cookie'] ?? ''} ${cookieParts.join('; ')}`.trim()
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

  // Forward all upstream headers, preserving Set-Cookie (including multiple
  // Set-Cookie headers) which a plain Record would silently drop.
  const responseHeaders = new Headers()
  res.headers.forEach((v, k) => {
    responseHeaders.append(k, v)
  })

  return new NextResponse(await res.arrayBuffer(), {
    status: res.status,
    statusText: res.statusText,
    headers: responseHeaders,
  })
}
