import { test, expect } from "@playwright/test";

test.describe("Proxy auth guard", () => {
    test("redirects unauthenticated requests to login", async ({ page }) => {
        // Assuming proxy protects /app/* routes
        await page.goto("/app/dashboard");
        await expect(page).toHaveURL(/\/login/);
    });

    test("allows authenticated requests to pass through", async ({ page }) => {
        // This test would require authenticated session setup
        // Placeholder for future implementation
        test.skip();
    });
});
