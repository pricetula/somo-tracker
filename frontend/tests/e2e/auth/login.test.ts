import { test, expect } from "@playwright/test";

test.describe("Auth Login Page", () => {
    test.beforeEach(async ({ page }) => {
        // Clear cookies and local storage to start from a clean state
        await page.context().clearCookies();
        await page.context().clearPermissions();
        await page.route("**/localStorage", async (route) => {
            await route.abort();
        });
    });

    test("should send magic link when email is submitted", async ({ page }) => {
        // Intercept the auth/discover API call and mock a successful response
        await page.route("**/api/auth/discover", async (route) => {
            const request = route.request();
            const postData = request.postDataJSON();
            expect(postData).toHaveProperty("email");
            // Mock successful response (200 OK with empty body)
            await route.fulfill({
                status: 200,
                contentType: "application/json",
                body: JSON.stringify({}),
            });
        });

        // Navigate to the login page
        await page.goto("/login");

        // Wait for the page to load and the email input to be visible
        const emailInput = page.getByLabel("Email");
        await expect(emailInput).toBeVisible();

        // Fill in a test email
        await emailInput.fill("test@example.com");

        // Click the submit button
        const submitButton = page.getByRole("button", { name: "Send Magic Link" });
        await expect(submitButton).toBeEnabled();
        await submitButton.click();

        // Wait for the success toast to appear
        const successToast = page.getByText("Magic link sent!", { exact: false });
        await expect(successToast).toBeVisible({ timeout: 5000 });

        // Optionally, we can also check that the description contains the email
        const toastDescription = page.getByText("Check test@example.com for your sign-in link.", {
            exact: false,
        });
        await expect(toastDescription).toBeVisible({ timeout: 5000 });
    });

    test("should show error when API call fails", async ({ page }) => {
        // Intercept the auth/discover API call and mock an error response
        await page.route("**/api/auth/discover", async (route) => {
            await route.fulfill({
                status: 400,
                contentType: "application/json",
                body: JSON.stringify({
                    code: "invalid_email",
                    message: "Invalid email address",
                    errors: { email: ["Invalid email format"] },
                }),
            });
        });

        // Navigate to the login page
        await page.goto("/login");

        // Fill in a valid email that will trigger an API error
        const emailInput = page.getByLabel("Email");
        await emailInput.fill("error@example.com");

        // Click the submit button
        const submitButton = page.getByRole("button", { name: "Send Magic Link" });
        await submitButton.click();

        // Wait for the error toast to appear
        const errorToast = page.getByText("Failed to send magic link", { exact: false });
        await expect(errorToast).toBeVisible({ timeout: 5000 });

        // Check that the error description contains the expected message
        const toastDescription = page.getByText("Invalid email address", { exact: false });
        await expect(toastDescription).toBeVisible({ timeout: 5000 });
    });
    test("should redirect to dashboard after successful magic link verification", async ({
        page,
    }) => {
        // Intercept discover to send magic link
        await page.route("**/api/auth/discover", async (route) => {
            await route.fulfill({
                status: 200,
                contentType: "application/json",
                body: JSON.stringify({}),
            });
        });

        // Intercept verify endpoint to simulate successful verification
        await page.route("**/api/auth/verify", async (route) => {
            const request = route.request();
            const url = new URL(request.url());
            const token = url.searchParams.get("token");
            expect(token).toBe("valid-token");
            await route.fulfill({
                status: 200,
                contentType: "application/json",
                body: JSON.stringify({ user: { id: "1", email: "test@example.com" } }),
            });
        });

        await page.goto("/login");
        const emailInput = page.getByLabel("Email");
        await emailInput.fill("test@example.com");
        const submitButton = page.getByRole("button", { name: "Send Magic Link" });
        await submitButton.click();

        // Wait for success toast (optional)
        await expect(page.getByText("Magic link sent!", { exact: false })).toBeVisible({
            timeout: 5000,
        });

        // Simulate clicking the magic link from email: navigate to verify URL with token
        await page.goto("/api/auth/verify?token=valid-token");

        // Wait for redirect to dashboard
        await expect(page).toHaveURL(/.*\/dashboard$/, { timeout: 5000 });
    });

    test("should show error for invalid magic link token", async ({ page }) => {
        await page.route("**/api/auth/verify", async (route) => {
            await route.fulfill({
                status: 400,
                contentType: "application/json",
                body: JSON.stringify({
                    code: "invalid_token",
                    message: "Invalid or expired token",
                    errors: { token: ["Invalid token"] },
                }),
            });
        });

        await page.goto("/api/auth/verify?token=invalid-token");
        const errorToast = page.getByText("Invalid or expired token", { exact: false });
        await expect(errorToast).toBeVisible({ timeout: 5000 });
    });

    test("should show error for expired magic link token", async ({ page }) => {
        await page.route("**/api/auth/verify", async (route) => {
            await route.fulfill({
                status: 401,
                contentType: "application/json",
                body: JSON.stringify({
                    code: "token_expired",
                    message: "Token has expired",
                    errors: { token: ["Token expired"] },
                }),
            });
        });

        await page.goto("/api/auth/verify?token=expired-token");
        const errorToast = page.getByText("Token has expired", { exact: false });
        await expect(errorToast).toBeVisible({ timeout: 5000 });
    });

    test("should enforce rate limiting on send magic link", async ({ page }) => {
        let callCount = 0;
        await page.route("**/api/auth/discover", async (route) => {
            callCount++;
            if (callCount <= 3) {
                await route.fulfill({
                    status: 200,
                    contentType: "application/json",
                    body: JSON.stringify({}),
                });
            } else {
                await route.fulfill({
                    status: 429,
                    contentType: "application/json",
                    body: JSON.stringify({
                        code: "rate_limit_exceeded",
                        message: "Too many requests",
                        errors: {},
                    }),
                });
            }
        });

        await page.goto("/login");
        const emailInput = page.getByLabel("Email");
        const submitButton = page.getByRole("button", { name: "Send Magic Link" });

        // First three attempts should succeed
        for (let i = 0; i < 3; i++) {
            await emailInput.fill(`test${i}@example.com`);
            await submitButton.click();
            await expect(page.getByText("Magic link sent!", { exact: false })).toBeVisible({
                timeout: 3000,
            });
            await expect(
                page.getByText("Failed to send magic link", { exact: false })
            ).toBeHidden();
            // Clear input for next
            await emailInput.fill("");
        }

        // Fourth attempt should be rate limited
        await emailInput.fill("test@example.com");
        await submitButton.click();
        const errorToast = page.getByText("Too many requests", { exact: false });
        await expect(errorToast).toBeVisible({ timeout: 3000 });
    });

    test("should allow resending magic link", async ({ page }) => {
        await page.route("**/api/auth/discover", async (route) => {
            await route.fulfill({
                status: 200,
                contentType: "application/json",
                body: JSON.stringify({}),
            });
        });

        await page.goto("/login");
        const emailInput = page.getByLabel("Email");
        const submitButton = page.getByRole("button", { name: "Send Magic Link" });
        await emailInput.fill("test@example.com");
        await submitButton.click();

        // Wait for success toast
        await expect(page.getByText("Magic link sent!", { exact: false })).toBeVisible({
            timeout: 3000,
        });

        // Click resend link (assuming it appears after toast or in UI)
        const resendLink = page.getByRole("link", { name: /resend magic link/i });
        await expect(resendLink).toBeVisible({ timeout: 3000 });
        await resendLink.click();

        // Verify another success toast appears
        await expect(page.getByText("Magic link sent!", { exact: false })).toBeVisible({
            timeout: 3000,
        });
    });

    test("should redirect authenticated user away from login page", async ({ page }) => {
        // Simulate an authenticated session by setting a cookie (assuming auth cookie name)
        await page.context().addCookies([
            {
                name: "session-token",
                value: "fake-jwt-token",
                domain: "localhost",
                path: "/",
                httpOnly: true,
                sameSite: "Lax",
            },
        ]);

        await page.goto("/login");
        // Should redirect to dashboard
        await expect(page).toHaveURL(/.*\/dashboard$/, { timeout: 5000 });
    });
});
