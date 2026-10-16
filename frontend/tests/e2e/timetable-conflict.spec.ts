import { test, expect } from "@playwright/test";

test.describe("Timetable Conflict Detection E2E", () => {
    test.beforeEach(async ({ page }) => {
        // TODO: Replace with real auth flow
        // await page.goto("/login");
        await page.goto("/(dashboard)/timetable");
    });

    test("assign slot with teacher double-book shows 409 conflict error", async ({ page }) => {
        // Navigate to class assignment
        await page.getByRole("button", { name: /assign slot/i }).click();
        await expect(page.getByRole("dialog", { name: /assign timetable slot/i })).toBeVisible();

        // Fill form with conflicting teacher/time
        await page.getByLabel(/class/i).selectOption({ label: "Class 1" });
        await page.getByLabel(/day/i).selectOption({ label: "Monday" });
        await page.getByLabel(/time slot/i).selectOption({ label: "08:00 - 09:00" });
        await page.getByLabel(/teacher/i).selectOption({ label: "Teacher A" });
        await page.getByLabel(/room/i).selectOption({ label: "Room 101" });
        await page.getByLabel(/subject/i).selectOption({ label: "Math" });

        await page.getByRole("button", { name: /save/i }).click();

        // Expect conflict error from API 409
        await expect(page.getByText(/timetable conflict detected/i)).toBeVisible();
        await expect(page.getByText(/teacher is already booked/i)).toBeVisible();
        await expect(page.getByRole("button", { name: /save/i })).toBeDisabled();
    });

    test("assign slot with room double-book shows conflict error", async ({ page }) => {
        await page.getByRole("button", { name: /assign slot/i }).click();
        await expect(page.getByRole("dialog", { name: /assign timetable slot/i })).toBeVisible();

        await page.getByLabel(/class/i).selectOption({ label: "Class 2" });
        await page.getByLabel(/day/i).selectOption({ label: "Monday" });
        await page.getByLabel(/time slot/i).selectOption({ label: "08:00 - 09:00" });
        await page.getByLabel(/teacher/i).selectOption({ label: "Teacher B" });
        await page.getByLabel(/room/i).selectOption({ label: "Room 101" });
        await page.getByLabel(/subject/i).selectOption({ label: "Science" });

        await page.getByRole("button", { name: /save/i }).click();

        await expect(page.getByText(/timetable conflict detected/i)).toBeVisible();
        await expect(page.getByText(/room is already booked/i)).toBeVisible();
    });

    test("assign slot with class overlap shows conflict error", async ({ page }) => {
        await page.getByRole("button", { name: /assign slot/i }).click();
        await expect(page.getByRole("dialog", { name: /assign timetable slot/i })).toBeVisible();

        await page.getByLabel(/class/i).selectOption({ label: "Class 1" });
        await page.getByLabel(/day/i).selectOption({ label: "Monday" });
        await page.getByLabel(/time slot/i).selectOption({ label: "08:00 - 09:00" });
        await page.getByLabel(/teacher/i).selectOption({ label: "Teacher C" });
        await page.getByLabel(/room/i).selectOption({ label: "Room 102" });
        await page.getByLabel(/subject/i).selectOption({ label: "English" });

        await page.getByRole("button", { name: /save/i }).click();

        await expect(page.getByText(/timetable conflict detected/i)).toBeVisible();
        await expect(page.getByText(/class already has a slot/i)).toBeVisible();
    });

    test("assign slot with no conflict succeeds", async ({ page }) => {
        await page.getByRole("button", { name: /assign slot/i }).click();

        await page.getByLabel(/class/i).selectOption({ label: "Class 3" });
        await page.getByLabel(/day/i).selectOption({ label: "Tuesday" });
        await page.getByLabel(/time slot/i).selectOption({ label: "10:00 - 11:00" });
        await page.getByLabel(/teacher/i).selectOption({ label: "Teacher D" });
        await page.getByLabel(/room/i).selectOption({ label: "Room 103" });
        await page.getByLabel(/subject/i).selectOption({ label: "History" });

        await page.getByRole("button", { name: /save/i }).click();

        await expect(page.getByText(/slot created successfully/i)).toBeVisible();
        await expect(
            page.getByRole("dialog", { name: /assign timetable slot/i })
        ).not.toBeVisible();
    });
});
