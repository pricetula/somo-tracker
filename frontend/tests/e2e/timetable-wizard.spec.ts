import { test, expect } from "@playwright/test";

test.describe("Timetable Template Wizard", () => {
    test.beforeEach(async ({ page }) => {
        // TODO: Replace with real auth flow
        // await page.goto("/login");
        // await page.fill('[name="email"]', process.env.TEST_USER_EMAIL!);
        // ...
        await page.goto("/(dashboard)/timetable");
    });

    test("metadata step enables Next after valid name", async ({ page }) => {
        const nextBtn = page.getByRole("button", { name: /next/i });
        await expect(nextBtn).toBeDisabled();

        await page.getByLabel(/template name/i).fill("Standard 6-Period Day");
        await expect(nextBtn).toBeEnabled();

        await nextBtn.click();
        await expect(page.getByRole("heading", { name: /configure time slots/i })).toBeVisible();
    });

    test("slot contiguity validation disables Save on gap", async ({ page }) => {
        await page.getByLabel(/template name/i).fill("Gap Test");
        await page.getByRole("button", { name: /next/i }).click();

        // Ensure at least two slots
        await page.getByRole("button", { name: /\+ add time slot/i }).click();

        const startInputs = page.locator('input[type="time"]').filter({ hasText: "" });
        // Change second slot start to create a gap
        const secondStart = page.locator('input[type="time"]').nth(2);
        await secondStart.fill("09:30");

        const saveBtn = page.getByRole("button", { name: /save template/i });
        await expect(saveBtn).toBeDisabled();

        // Fix the gap
        await secondStart.fill("09:00");
        await expect(saveBtn).toBeEnabled();
    });

    test("auto-cascade updates next slot start when end changes", async ({ page }) => {
        await page.getByLabel(/template name/i).fill("Cascade Test");
        await page.getByRole("button", { name: /next/i }).click();

        const endInputs = page.locator('input[type="time"]').filter({ hasText: "" });
        const firstEnd = page.locator('input[type="time"]').nth(1);
        await firstEnd.fill("09:30");

        const secondStart = page.locator('input[type="time"]').nth(2);
        await expect(secondStart).toHaveValue("09:30");
    });
});
