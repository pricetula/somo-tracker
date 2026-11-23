import { defineConfig, devices } from "@playwright/test";

/**
 * E2E config for magic-link flow tests.
 *
 * Requires two services running before tests:
 *   - Frontend dev server:  pnpm dev        (http://localhost:3000)
 *   - Mock backend:         node server.js  (http://localhost:8080)
 *   - API_URL=http://localhost:8080
 */
export default defineConfig({
  testDir: "./e2e",
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 2 : undefined,
  reporter: "html",
  use: {
    baseURL: "http://localhost:3000",
    trace: "on-first-retry",
  },
  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
    },
  ],
  webServer: undefined, // services started manually; do not auto-launch
});
