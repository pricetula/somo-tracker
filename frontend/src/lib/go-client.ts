import { cookies } from 'next/headers'
import { NextResponse } from 'next/server'

function sessionCookieName(): string {
  return process.env.NODE_ENV === 'production'
    ? '__Host-session_token'
    : 'session_token_development'
}

function pkceCookieName(): string {
  return process.env.NODE_ENV === 'production'
    ? '__Host-pkce_verifier'
    : 'pkce_verifier'
}

function onboardingCookieName(): string {
  return process.env.NODE_ENV === 'production'
    ? '__Host-onboarding'
    : 'onboarding'
}

/**
 * goRequest forwards a request to the Go backend, attaching the HttpOnly
 * session, PKCE verifier, and onboarding cookies from the browser so Go can
 * validate the session and authenticate magic-link tokens.
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
  const pkceVerifier = cookieStore.get(pkceCookieName())?.value
  const onboardingTicket = cookieStore.get(onboardingCookieName())?.value

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

  const cookiesToForward = [sessionToken, pkceVerifier, onboardingTicket]
    .filter(Boolean)
    .join(' ')
  if (cookiesToForward) {
    headers['Cookie'] = `${headers['Cookie'] ?? ''} ${cookiesToForward}`.trim()
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
