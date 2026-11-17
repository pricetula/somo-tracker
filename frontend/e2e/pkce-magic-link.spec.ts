import { test, expect } from '@playwright/test'

/**
 * E2E: PKCE cookie forwarding in magic-link authentication flow.
 *
 * Reproduces the production bug: clicking a magic link POSTed the token to
 * /api/auth/authenticate-token but the __Host-pkce_verifier cookie was not
 * forwarded, so the backend returned 400 MISSING_PKCE.
 *
 * These tests run against a mock backend (PUBLIC_API_URL) that mimics the Go
 * handlers: it sets pkce_verifier on /request-magic-link and requires it on
 * /authenticate-token.
 */
test.describe('PKCE magic-link flow', () => {
  const API_URL = process.env.PUBLIC_API_URL ?? 'http://localhost:8080'

  test('sets pkce_verifier cookie after requesting a magic link', async ({
    page,
    context,
  }) => {
    await page.goto('/login')
    await page.fill('input[type="email"]', 'test@example.com')
    await page.click('button:has-text("Send magic link")')

    await page.waitForResponse((res) =>
      res.url().includes('/api/auth/request-magic-link')
    )

    const cookies = await context.cookies()
    const pkceCookie = cookies.find((c) => c.name === 'pkce_verifier')
    expect(pkceCookie).toBeTruthy()
  })

  test('authenticate-token returns MISSING_PKCE without the cookie', async ({
    request,
  }) => {
    const res = await request.post(`${API_URL}/api/auth/authenticate-token`, {
      data: { token: 'mock-token-123' },
    })

    expect(res.status()).toBe(400)
    const body = await res.json()
    expect(body.code).toBe('MISSING_PKCE')
    expect(body.message).toContain('PKCE verifier missing')
  })

  test('authenticate-token succeeds when callback page forwards the cookie', async ({
    page,
    context,
  }) => {
    // 1. Request magic link → backend sets pkce_verifier cookie.
    await page.goto('/login')
    await page.fill('input[type="email"]', 'test@example.com')
    await page.click('button:has-text("Send magic link")')
    await page.waitForResponse((res) =>
      res.url().includes('/api/auth/request-magic-link')
    )

    const cookies = await context.cookies()
    expect(cookies.find((c) => c.name === 'pkce_verifier')).toBeTruthy()

    // 2. Navigate to the callback page. It POSTs the token + pkce_verifier
    //    cookie to authenticate-token. Set up the response listener before
    //    navigation because the fetch fires on mount.
    const [authenticateResponse] = await Promise.all([
      page.waitForResponse((res) =>
        res.url().includes('/api/auth/authenticate-token')
      ),
      page.goto('/auth/callback?token=mock-token-123'),
    ])
    expect(authenticateResponse.status()).toBe(200)
  })
})
