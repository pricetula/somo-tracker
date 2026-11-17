import { describe, it } from 'node:test'
import assert from 'node:assert'

/**
 * Unit tests for the cookie-header handling logic in goRequest.
 *
 * These tests validate the fix for the PKCE cookie issue:
 *
 *   BUG: goRequest converted upstream response headers through
 *   `Headers.forEach()` into a plain `Record<string, string>`:
 *     const headers: Record<string, string> = {}
 *     res.headers.forEach((v, k) => { headers[k] = v })
 *   This clobbers duplicate Set-Cookie headers — only the last one
 *   survives — so the browser never receives pkce_verifier.
 *
 *   FIX: Use `new Headers()` with `.append()` so every Set-Cookie
 *   header is preserved.
 *
 * No Next.js modules are imported; we test the header-handling pattern
 * directly.
 */

function oldRecordBehavior(setCookieHeaders: string[]): string | undefined {
  const headers: Record<string, string> = {}
  for (const cookie of setCookieHeaders) {
    headers['set-cookie'] = cookie  // clobbers previous value
  }
  return headers['set-cookie']
}

function newHeadersBehavior(setCookieHeaders: string[]): string[] {
  const headers = new Headers()
  for (const cookie of setCookieHeaders) {
    headers.append('set-cookie', cookie)
  }
  return headers.getSetCookie()
}

function buildCookieHeader(cookies: { name: string; value }[]): string {
  const parts: string[] = []
  for (const c of cookies) {
    parts.push(`${c.name}=${c.value}`)
  }
  return parts.join('; ')
}

describe('goRequest cookie-header handling', () => {
  it('old Record behavior clobbers duplicate Set-Cookie headers', () => {
    const cookies = [
      'pkce_verifier=abc123; Path=/; HttpOnly; Max-Age=600',
      '__Host-session_token=def456; Path=/; HttpOnly; Max-Age=2592000',
    ]

    const result = oldRecordBehavior(cookies)

    // Only the last Set-Cookie survives — the first is lost.
    assert.strictEqual(result, cookies[cookies.length - 1])
    assert.ok(!result!.includes('pkce_verifier'), 'pkce_verifier is clobbered')
  })

  it('new Headers.append behavior preserves all Set-Cookie headers', () => {
    const cookies = [
      'pkce_verifier=abc123; Path=/; HttpOnly; Max-Age=600',
      '__Host-session_token=def456; Path=/; HttpOnly; Max-Age=2592000',
    ]

    const result = newHeadersBehavior(cookies)

    assert.strictEqual(result.length, 2, 'both Set-Cookie headers preserved')
    assert.ok(
      result.some((c) => c.startsWith('pkce_verifier=abc123')),
      'pkce_verifier present',
    )
    assert.ok(
      result.some((c) => c.startsWith('__Host-session_token=def456')),
      'session_token present',
    )
  })

  it('builds Cookie header from multiple browser cookies', () => {
    const cookies = [
      { name: 'pkce_verifier', value: 'verifier-123' },
      { name: 'session_token_development', value: 'sess-456' },
    ]

    const header = buildCookieHeader(cookies)

    assert.ok(header.includes('pkce_verifier=verifier-123'))
    assert.ok(header.includes('session_token_development=sess-456'))
  })
})
