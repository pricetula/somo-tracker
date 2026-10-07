import { test, expect } from "@playwright/test";

test.describe("Admin Bulk Invite", () => {
    test("upload CSV, map fields and see progress", async ({ page }) => {
        // TODO: login flow depends on your auth setup
        await page.goto("/admins/invite");
        await expect(page.getByRole("button", { name: /Upload/i })).toBeVisible();

        // Click Upload
        await page.getByRole("button", { name: /Upload/i }).click();

        // Wait for file input
        const fileInput = page.locator('input[type="file"]');
        await expect(fileInput).toBeVisible();

        // Upload a test CSV
        // Create a small CSV in the test assets folder
        await fileInput.setInputFiles("tests/e2e/fixtures/admin_invites.csv");

        // Wait for mapping UI
        await expect(page.getByText(/Map fields/i)).toBeVisible();

        // Save mapping
        await page.getByRole("button", { name: /Save mapping/i }).click();

        // Wait for job creation toast
        await expect(page.getByText(/Invitation job queued/i)).toBeVisible();

        // Progress view should appear
        await expect(page.getByText(/Job/i)).toBeVisible();
        await expect(page.getByText(/Status:/i)).toBeVisible();

        // Wait for progress to reach completed
        await expect(page.getByText(/COMPLETED|COMPLETED_WITH_ERRORS/i)).toBeVisible({
            timeout: 120000,
        });
    });
});
