import { test, expect } from "@playwright/test";

test.describe("Auth Logout Flow", () => {
    // Helper to set auth cookies to simulate logged-in state
    const setAuthCookies = async (page) => {
        const domain = "localhost"; // adjust if needed
        await page.context().addCookies([
            {
                name: "somo_sid",
                value: "valid-session-token",
                domain,
                path: "/",
                httpOnly: true,
                sameSite: "Lax",
            },
            { name: "somo_role", value: "TEACHER", domain, path: "/" },
            { name: "somo_school_id", value: "1", domain, path: "/" },
            { name: "csrf_token", value: "fake-csrf-token", domain, path: "/" },
        ]);
    };

    test.beforeEach(async ({ page }) => {
        // Clear cookies and local storage before each test
        await page.context().clearCookies();
        await page.context().clearPermissions();
        await page.route("**/localStorage", async (route) => {
            await route.abort();
        });
        // Mock user session
        await page.route("**/api/auth/me", async (route) => {
            await route.fulfill({
                status: 200,
                contentType: "application/json",
                body: JSON.stringify({
                    id: "1",
                    name: "Test Teacher",
                    full_name: "Test Teacher",
                    email: "teacher@example.com",
                }),
            });
        });
    });

    test("should log out user when clicking logout link in user menu", async ({ page }) => {
        // Simulate authenticated session
        await setAuthCookies(page);

        // Intercept logout API call
        await page.route("**/api/auth/session", async (route) => {
            await route.fulfill({
                status: 200,
                contentType: "application/json",
                body: JSON.stringify({}),
                headers: {
                    "Set-Cookie": [
                        "somo_sid=deleted; Path=/; Domain=localhost; SameSite=Lax; Expires=Thu, 01 Jan 1970 00:00:00 GMT",
                        "somo_school_id=deleted; Path=/; Domain=localhost; SameSite=Lax; Expires=Thu, 01 Jan 1970 00:00:00 GMT",
                        "somo_role=deleted; Path=/; Domain=localhost; SameSite=Lax; Expires=Thu, 01 Jan 1970 00:00:00 GMT",
                        "csrf_token=deleted; Path=/; Domain=localhost; SameSite=Lax; Expires=Thu, 01 Jan 1970 00:00:00 GMT",
                    ],
                },
            });
        });

        // Navigate to a protected page (dashboard) to see the user menu
        await page.goto("/dashboard");
        await expect(page).toHaveURL(/.*\/dashboard$/, { timeout: 5000 });

        // Open user dropdown: click the button that contains an avatar
        const avatarButton = page.locator("button:has(.group\\/avatar)");
        await expect(avatarButton).toBeVisible({ timeout: 3000 });
        await avatarButton.click({ force: true });

        // Click logout link inside the dropdown
        const logoutLink = page.getByText(/log out/i);
        await expect(logoutLink).toBeVisible({ timeout: 3000 });
        await logoutLink.click();

        // Wait for redirect to login page
        await expect(page).toHaveURL(/.*\/login$/, { timeout: 5000 });

        // Verify success toast appears
        await expect(page.getByText("Logged out", { exact: false }).first()).toBeVisible({
            timeout: 3000,
        });
    });

    test("should handle logout API failure and still redirect to login", async ({ page }) => {
        await setAuthCookies(page);

        // Mock logout endpoint to return error
        await page.route("**/api/auth/session", async (route) => {
            await route.fulfill({
                status: 500,
                contentType: "application/json",
                body: JSON.stringify({
                    code: "internal_error",
                    message: "Internal server error",
                }),
                headers: {
                    "Set-Cookie": [
                        "somo_sid=deleted; Path=/; Domain=localhost; SameSite=Lax; Expires=Thu, 01 Jan 1970 00:00:00 GMT",
                        "somo_school_id=deleted; Path=/; Domain=localhost; SameSite=Lax; Expires=Thu, 01 Jan 1970 00:00:00 GMT",
                        "somo_role=deleted; Path=/; Domain=localhost; SameSite=Lax; Expires=Thu, 01 Jan 1970 00:00:00 GMT",
                        "csrf_token=deleted; Path=/; Domain=localhost; SameSite=Lax; Expires=Thu, 01 Jan 1970 00:00:00 GMT",
                    ],
                },
            });
        });

        await page.goto("/dashboard");
        await expect(page).toHaveURL(/.*\/dashboard$/, { timeout: 5000 });

        // Open user dropdown and click logout
        const avatarButton = page.locator("button:has(.group\\/avatar)");
        await expect(avatarButton).toBeVisible({ timeout: 5000 });
        await avatarButton.click({ force: true });

        const logoutLink = page.getByText(/log out/i);
        await expect(logoutLink).toBeVisible({ timeout: 5000 });
        await logoutLink.click();

        //
        // Even on error, should redirect to login (as per logout page finally block)
        await expect(page).toHaveURL(/.*\/login$/, { timeout: 5000 });
    });

    test("should redirect to login after logout when accessing protected route", async ({
        page,
    }) => {
        await setAuthCookies(page);

        // Intercept logout API
        await page.route("**/api/auth/session", async (route) => {
            console.log("Logout API intercepted");
            await route.fulfill({
                status: 200,
                headers: {
                    "Set-Cookie": [
                        "somo_sid=deleted; Path=/; Domain=localhost; SameSite=Lax; Expires=Thu, 01 Jan 1970 00:00:00 GMT",
                        "somo_school_id=deleted; Path=/; Domain=localhost; SameSite=Lax; Expires=Thu, 01 Jan 1970 00:00:00 GMT",
                        "somo_role=deleted; Path=/; Domain=localhost; SameSite=Lax; Expires=Thu, 01 Jan 1970 00:00:00 GMT",
                        "csrf_token=deleted; Path=/; Domain=localhost; SameSite=Lax; Expires=Thu, 01 Jan 1970 00:00:00 GMT",
                    ],
                },
            });
        });

        // Perform logout via visiting /logout directly (simulate clicking link)
        await page.goto("/logout");
        // Logout page redirects to login after async logout
        await expect(page).toHaveURL(/.*\/login$/, { timeout: 5000 });
        // Debug: check cookies after logout
        const cookiesAfterLogout = await page.context().cookies();
        console.log(
            "Cookies after logout:",
            cookiesAfterLogout.map((c) => ({ name: c.name, value: c.value }))
        );

        // Now try to access a protected route (e.g., /dashboard) - should redirect to login
        await page.goto("/dashboard");
        // Should redirect to login because session cleared
        await expect(page).toHaveURL(/.*\/login$/, { timeout: 5000 });
    });

    test("should clear cookies on client-side when logout API fails (network error)", async ({
        page,
    }) => {
        await setAuthCookies(page);

        // Simulate network error by aborting the request
        await page.route("**/api/auth/session", async (route) => {
            await route.abort();
        });

        await page.goto("/dashboard");
        await expect(page).toHaveURL(/.*\/dashboard$/, { timeout: 5000 });

        // Open dropdown and click logout
        const avatarButton = page.locator("button:has(.group\\/avatar)");
        await expect(avatarButton).toBeVisible({ timeout: 5000 });
        await avatarButton.click({ force: true });
        const logoutLink = page.getByText(/log out/i);
        await expect(logoutLink).toBeVisible({ timeout: 5000 });
        await logoutLink.click();

        // Wait for redirect to login
        await expect(page).toHaveURL(/.*\/login$/, { timeout: 5000 });

        // Verify that non-HttpOnly cookies are removed (client-side cleared)
        const cookies = await page.context().cookies();
        const cookieNames = cookies.map((c) => c.name);
        expect(cookieNames).not.toContain("somo_role");
        expect(cookieNames).not.toContain("somo_school_id");
        expect(cookieNames).not.toContain("csrf_token");
        // Note: somo_sid is HttpOnly; cannot be cleared via document.cookie, but backend would clear on success.
    });
});
