import { cookies } from 'next/headers'
import { NextResponse } from 'next/server'
import { getSessionCookieName, getBackendUrl } from './cookies'

export interface GoResponse<T = unknown> {
  status: number
  data: T
  setCookieHeaders: string[]
}

export async function goRequest<T = unknown>(
  path: string,
  opts: {
    method?: string
    body?: unknown
    forwardCookies?: boolean
  } = {}
): Promise<GoResponse<T>> {
  const { method = 'GET', body, forwardCookies = true } = opts

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  }

  if (forwardCookies) {
    const cookieStore = await cookies()
    const sessionToken =
      cookieStore.get(getSessionCookieName())?.value ||
      cookieStore.get('session_token_development')?.value
    if (sessionToken) {
      headers['Cookie'] = `${getSessionCookieName()}=${sessionToken}`
    }
  }

  // Forward Origin for CSRF on mutations (Go CSRF middleware checks Origin)
  if (['POST', 'PUT', 'DELETE', 'PATCH'].includes(method)) {
    headers['Origin'] = process.env.NEXT_PUBLIC_APP_URL || 'http://localhost:3000'
  }

  const res = await fetch(`${getBackendUrl()}${path}`, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
    cache: 'no-store',
  })

  const setCookieHeaders = res.headers.getSetCookie ? res.headers.getSetCookie() : res.headers.getAll('set-cookie')

  let data: T | null = null
  try {
    data = (await res.json()) as T
  } catch {
    // Non-JSON response
  }

  return {
    status: res.status,
    data: data as T,
    setCookieHeaders,
  }
}

export function forwardSetCookieHeaders(nextRes: NextResponse, setCookieHeaders: string[]): NextResponse {
  for (const cookie of setCookieHeaders) {
    nextRes.headers.append('Set-Cookie', cookie)
  }
  return nextRes
}
