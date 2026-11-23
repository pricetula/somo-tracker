import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'

// Mock next/headers before importing the module under test
vi.mock('next/headers', () => ({
  cookies: vi.fn(),
}))

import { apiRequest } from './api-client'
import { cookies } from 'next/headers'

const mockCookies = cookies as ReturnType<typeof vi.fn>
const mockFetch = vi.fn()

// Replace global.fetch with a vitest mock
global.fetch = mockFetch as any

function makeResponse(
  status: number,
  headers: Record<string, string> = {},
  body: string = ''
): Response {
  return new Response(body, { status, headers })
}

function makeJsonResponse(
  status: number,
  body: unknown,
  extraHeaders: Record<string, string> = {}
): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'content-type': 'application/json', ...extraHeaders },
  })
}

beforeEach(() => {
  vi.restoreAllMocks()
  mockFetch.mockClear()
  mockCookies.mockReturnValue({
    get: vi.fn(() => ({ value: undefined })),
  })
  // Default fetch: successful response
  mockFetch.mockResolvedValue(makeJsonResponse(200, { ok: true }))
  // Configure the backend URL required by apiRequest
  process.env.PUBLIC_API_URL = 'https://api.somotracker.com'
  process.env.SESSION_COOKIE_NAME = 'session'
})

afterEach(() => {
  vi.useRealTimers()
  delete process.env.PUBLIC_API_URL
  delete process.env.SESSION_COOKIE_NAME
})

describe('apiRequest — normal requests (no hibernation)', () => {
  it('succeeds without additional health checks', async () => {
    mockFetch.mockResolvedValue(makeJsonResponse(200, { ok: true }))

    const result = await apiRequest('/api/v1/auth/session')

    expect(mockFetch).toHaveBeenCalledTimes(1)
    const [url] = mockFetch.mock.calls[0]
    expect(String(url)).not.toContain('/readyz')
    expect(result.status).toBe(200)
    const body = await result.json()
    expect(body).toEqual({ ok: true })
  })

  it('passes through ordinary application-level 429 unchanged', async () => {
    // 429 without the Render hibernation header
    mockFetch.mockResolvedValue(
      makeJsonResponse(429, { error: 'rate limited' }, { 'x-ratelimit-limit': '10' })
    )

    const result = await apiRequest('/api/v1/auth/otp/request', {
      method: 'POST',
      body: JSON.stringify({ email: 'test@example.com' }),
    })

    expect(mockFetch).toHaveBeenCalledTimes(1)
    expect(result.status).toBe(429)
    const body = await result.json()
    expect(body).toEqual({ error: 'rate limited' })
    // No readiness polling should happen
    const urls = mockFetch.mock.calls.map(([u]) => String(u))
    expect(urls).not.toContain(expect.stringContaining('/readyz'))
  })

  it('forwards original method, headers, cookies and body', async () => {
    mockCookies.mockReturnValue({
      get: vi.fn(() => ({ value: 'session-token-123' })),
    })
    mockFetch.mockResolvedValue(makeJsonResponse(200, { ok: true }))

    const requestBody = JSON.stringify({ email: 'test@example.com', code: '123456' })
    await apiRequest('/api/v1/auth/otp/verify', {
      method: 'POST',
      body: requestBody,
      headers: { 'content-type': 'application/json', 'x-custom': 'custom-value' },
    })

    const [url, init] = mockFetch.mock.calls[0]
    expect(String(url)).toContain('/api/v1/auth/otp/verify')
    expect(init?.method).toBe('POST')
    expect(init?.body).toBe(requestBody)
    const headers = init?.headers ?? {}
    // Header keys may vary in case (Cookie vs cookie); check case-insensitively
    const cookieHeader = Object.entries(headers).find(
      ([k]) => k.toLowerCase() === 'cookie'
    )
    expect(cookieHeader).toBeTruthy()
    expect(cookieHeader?.[1]).toContain('session=session-token-123')
    expect(headers['content-type']).toBe('application/json')
    expect(headers['x-custom']).toBe('custom-value')
    // Origin header added for mutating methods (key may be 'Origin' or 'origin')
    const originHeader = Object.entries(headers).find(
      ([k]) => k.toLowerCase() === 'origin'
    )
    expect(originHeader).toBeTruthy()
    expect(typeof originHeader?.[1]).toBe('string')
  })
})
