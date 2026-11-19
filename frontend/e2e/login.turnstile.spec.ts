import { test, expect } from '@playwright/test'

/**
 * Turnstile login E2E tests.
 *
 * Strategy: mock the backend API routes with `page.route` so no real Go
 * backend, Postgres, or Redis is required. Mock the Cloudflare Turnstile
 * script so we never hit the real challenges.cloudflare.com during tests.
 *
 * Prerequisites:
 *   - Frontend dev server running:  pnpm dev  (http://localhost:3000)
 *   - NEXT_PUBLIC_TURNSTILE_SITE_KEY must be set (via Doppler) so the
 *     widget renders. In local dev you can use a Cloudflare test key:
 *       NEXT_PUBLIC_TURNSTILE_SITE_KEY=1x0000000000000000000000000000000000000000
 */

const TURNSTILE_SCRIPT =
  'https://challenges.cloudflare.com/turnstile/v0/api.js'

function mockTurnstileScript(opts?: {
  token?: string
  autoInvoke?: boolean
}) {
  const token = opts?.token ?? 'mock-turnstile-token'
  const autoInvoke = opts?.autoInvoke ?? true

  return `
    window.turnstile = {
      render: (_el, callbacks) => {
        if (autoInvoke && callbacks.callback) {
          callbacks.callback('${token}')
        }
        return 1
      },
      remove: () => {},
      getResponse: () => '${token}',
    }
  `
}

test.beforeEach(async ({ page }) => {
  // Mock Turnstile script so tests don't depend on real Cloudflare
  await page.route(TURNSTILE_SCRIPT, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/javascript',
      body: mockTurnstileScript(),
    })
  })
})

test('sends turnstile token with email to /api/auth/otp/request', async ({
  page,
}) => {
  let capturedBody: any = null

  await page.route('/api/auth/otp/request', async (route) => {
    capturedBody = JSON.parse(route.request().postData() ?? '{}')
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ ok: true, message: 'code sent' }),
    })
  })

  await page.goto('/login')

  // Turnstile widget renders
  await expect(page.locator('.cf-turnstile')).toBeVisible()

  await page.fill('input[type="email"]', 'user@example.com')
  await page.click('button[type="submit"]')

  expect(capturedBody).not.toBeNull()
  expect(capturedBody.email).toBe('user@example.com')
  expect(capturedBody.turnstile_token).toBe('mock-turnstile-token')
})

test('shows error when user tries to submit without completing Turnstile', async ({
  page,
}) => {
  // Mock Turnstile that does NOT auto-invoke the callback (user didn't complete)
  await page.route(TURNSTILE_SCRIPT, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/javascript',
      body: mockTurnstileScript({ autoInvoke: false, token: '' }),
    })
  })

  await page.goto('/login')

  await page.fill('input[type="email"]', 'user@example.com')
  await page.click('button[type="submit"]')

  await expect(
    page.locator('text=Please complete the security check above')
  ).toBeVisible()
})

test('proceeds to OTP step after successful OTP request', async ({ page }) => {
  await page.route('/api/auth/otp/request', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ ok: true }),
    })
  })

  await page.goto('/login')

  await page.fill('input[type="email"]', 'user@example.com')
  await page.click('button[type="submit"]')

  await expect(page.locator('h1')).toContainText('Verify code')
  await expect(page.locator('strong')).toContainText('user@example.com')
})

test('submit button is disabled until Turnstile token is present', async ({
  page,
}) => {
  // Turnstile that never invokes callback → token stays empty
  await page.route(TURNSTILE_SCRIPT, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/javascript',
      body: mockTurnstileScript({ autoInvoke: false }),
    })
  })

  await page.goto('/login')

  await page.fill('input[type="email"]', 'user@example.com')

  const submit = page.locator('button[type="submit"]')
  await expect(submit).toBeDisabled()
})
