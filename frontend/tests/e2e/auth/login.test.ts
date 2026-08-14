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
});
