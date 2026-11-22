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

describe('apiRequest — Render hibernation cold-start', () => {
  it('triggers readiness handling and retries successfully after backend becomes healthy', async () => {
    const hibernationRes = makeJsonResponse(
      429,
      { error: 'rate limited' },
      { 'x-render-routing': 'hibernate-rate-limited' }
    )
    const readyRes = makeResponse(200, {}, 'Ready')
    const retryRes = makeJsonResponse(200, { ok: true, message: 'success after retry' })

    mockFetch
      .mockResolvedValueOnce(hibernationRes)
      .mockResolvedValueOnce(readyRes)
      .mockResolvedValueOnce(retryRes)

    const result = await apiRequest('/api/v1/auth/session')

    // Call 1: original request to backend
    const [url1] = mockFetch.mock.calls[0]
    expect(String(url1)).toContain('/api/v1/auth/session')
    expect(String(url1)).not.toContain('/readyz')

    // Call 2: readiness poll
    const [url2] = mockFetch.mock.calls[1]
    expect(String(url2)).toContain('/readyz')

    // Call 3: retry original request
    const [url3] = mockFetch.mock.calls[2]
    expect(String(url3)).toContain('/api/v1/auth/session')
    expect(String(url3)).not.toContain('/readyz')

    expect(mockFetch).toHaveBeenCalledTimes(3)
    expect(result.status).toBe(200)
    const body = await result.json()
    expect(body).toEqual({ ok: true, message: 'success after retry' })
  })

  it('preserves original body on retry', async () => {
    const hibernationRes = makeJsonResponse(
      429,
      { error: 'rate limited' },
      { 'x-render-routing': 'hibernate-rate-limited' }
    )
    const readyRes = makeResponse(200, {}, 'Ready')
    const retryRes = makeJsonResponse(200, { ok: true })

    const capturedInits: Array<RequestInit | undefined> = []
    const capturedUrls: Array<string> = []
    mockFetch.mockImplementation((...args: any[]) => {
      capturedUrls.push(String(args[0]))
      capturedInits.push(args[1])
      const callIndex = capturedUrls.length - 1
      if (callIndex === 0) return Promise.resolve(hibernationRes)
      if (callIndex === 1) return Promise.resolve(readyRes)
      return Promise.resolve(retryRes)
    })

    const originalBody = JSON.stringify({ email: 'test@example.com', code: '123456' })
    await apiRequest('/api/v1/auth/otp/verify', {
      method: 'POST',
      body: originalBody,
      headers: { 'content-type': 'application/json' },
    })

    expect(mockFetch).toHaveBeenCalledTimes(3)
    // Retry call (index 2) should have the same body as the original (index 0)
    expect(capturedInits[2]?.body).toBe(originalBody)
    expect(capturedUrls[2]).toContain('/api/v1/auth/otp/verify')
  })

  it('returns structured 503 when backend never becomes ready', async () => {
    vi.useFakeTimers()

    const hibernationRes = makeJsonResponse(
      429,
      { error: 'rate limited' },
      { 'x-render-routing': 'hibernate-rate-limited' }
    )
    const notReadyRes = makeResponse(503, {}, 'DB not ready')

    // Every fetch returns hibernation for API paths, not-ready for /readyz
    mockFetch.mockImplementation(() => {
      const url = mockFetch.mock.calls[mockFetch.mock.calls.length - 1]?.[0]
      if (String(url).includes('/readyz')) {
        return Promise.resolve(notReadyRes)
      }
      return Promise.resolve(hibernationRes)
    })

    const promise = apiRequest('/api/v1/auth/session')

    // Flush microtasks so the initial fetch resolves and we enter readiness polling
    await Promise.resolve()
    await Promise.resolve()

    // Advance time past the 30s cold-start deadline
    vi.advanceTimersByTime(31000)

    const result = await promise
    expect(result.status).toBe(503)
    const body = await result.json()
    expect(body.code).toBe('BACKEND_UNAVAILABLE')
    expect(body.message).toContain('temporarily unavailable')

    vi.useRealTimers()
  })
})

describe('apiRequest — idempotency and side-effect safety', () => {
  it('does not retry more than once for a single hibernation response', async () => {
    const hibernationRes = makeJsonResponse(
      429,
      { error: 'rate limited' },
      { 'x-render-routing': 'hibernate-rate-limited' }
    )
    const readyRes = makeResponse(200, {}, 'Ready')

    // Even if the retry also returns hibernation, we do not retry again
    const retryHibernationRes = makeJsonResponse(
      429,
      { error: 'rate limited' },
      { 'x-render-routing': 'hibernate-rate-limited' }
    )

    mockFetch
      .mockResolvedValueOnce(hibernationRes)
      .mockResolvedValueOnce(readyRes)
      .mockResolvedValueOnce(retryHibernationRes)

    const result = await apiRequest('/api/v1/auth/otp/request', {
      method: 'POST',
      body: JSON.stringify({ email: 'test@example.com' }),
    })

    // Original + readiness + one retry = 3 calls max
    expect(mockFetch).toHaveBeenCalledTimes(3)
    expect(result.status).toBe(429)
  })

  it('retried request uses same method and headers as original', async () => {
    const hibernationRes = makeJsonResponse(
      429,
      { error: 'rate limited' },
      { 'x-render-routing': 'hibernate-rate-limited' }
    )
    const readyRes = makeResponse(200, {}, 'Ready')
    const retryRes = makeJsonResponse(200, { ok: true })

    const capturedInits: Array<RequestInit | undefined> = []
    mockFetch.mockImplementation((...args: any[]) => {
      capturedInits.push(args[1])
      const callIndex = capturedInits.length - 1
      if (callIndex === 0) return Promise.resolve(hibernationRes)
      if (callIndex === 1) return Promise.resolve(readyRes)
      return Promise.resolve(retryRes)
    })

    await apiRequest('/api/v1/auth/onboard', {
      method: 'POST',
      body: JSON.stringify({
        organization_name: 'Org',
        school_name: 'School',
        user_name: 'User',
      }),
      headers: { 'content-type': 'application/json', 'x-idempotency-key': 'key-123' },
    })

    const originalInit = capturedInits[0]
    const retryInit = capturedInits[2]

    expect(originalInit?.headers).toBeDefined()
    expect(retryInit?.method).toBe(originalInit?.method)
    expect(retryInit?.body).toBe(originalInit?.body)
    expect(retryInit?.headers).toMatchObject(originalInit?.headers ?? {})
  })
})
