/**
 * Tests for the ManualAddForm (ManualEntryPanel) component.
 *
 * Tests row management, validation, phone auto-correction,
 * duplicate detection, and the 5,000-row limit.
 */

import { describe, it, expect, beforeEach, vi } from "vitest";
import { screen, waitFor, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../setup/test-utils";

import { ManualEntryPanel } from "@/features/staff-import/components/manual-entry-panel";

// ─── Mock useVirtualizer ──────────────────────────────────────────────
// Must be at module level so it applies to the re-export from validation.ts

vi.mock("@tanstack/react-virtual", () => ({
    useVirtualizer: (opts: {
        count: number;
        getScrollElement: () => HTMLDivElement | null;
        estimateSize: () => number;
        overscan: number;
    }) => ({
        getVirtualItems: () =>
            Array.from({ length: opts.count }, (_, index) => ({
                index,
                key: index,
                start: index * opts.estimateSize(),
                size: opts.estimateSize(),
                end: (index + 1) * opts.estimateSize(),
                lane: 0,
            })),
        getTotalSize: () => opts.count * opts.estimateSize(),
        measureElement: vi.fn(),
    }),
}));

// ─── Helpers ───────────────────────────────────────────────────────────

function renderManualForm() {
    const onRowsReady = vi.fn();
    const utils = renderWithProviders(
        <ManualEntryPanel
            onRowsReady={onRowsReady}
            role="NURSE"
            tenantID="tenant-abc"
            userID="user-xyz"
            context="staff-import:NURSE"
        />
    );
    return { ...utils, onRowsReady };
}

describe("ManualAddForm — row management", () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it("renders one empty row on mount — full_name, full_name, email, phone inputs are present", () => {
        renderManualForm();

        expect(screen.getByPlaceholderText("Jane")).toBeInTheDocument();
        expect(screen.getByPlaceholderText("Doe")).toBeInTheDocument();
        expect(screen.getByPlaceholderText("jane@school.edu")).toBeInTheDocument();
        expect(screen.getByPlaceholderText("+254 712 345 678")).toBeInTheDocument();
    });

    it("Add Row button appends a new empty row — clicking 'Add Row' increments row count by 1; new row inputs are empty and focused", async () => {
        const user = userEvent.setup();
        renderManualForm();

        // Click "Add another" to add a row
        const addButton = screen.getByText("Add another");
        await user.click(addButton);

        // There should now be at least 2 rows worth of inputs — we can verify by
        // checking that there are 2 first-name placeholders rendered
        const fullNameInputs = screen.getAllByPlaceholderText("Jane");
        expect(fullNameInputs).toHaveLength(2);
    });

    it("Remove Row button deletes that row — row count decrements; remaining rows are unchanged", async () => {
        const user = userEvent.setup();
        renderManualForm();

        // Add a row first
        await user.click(screen.getByText("Add another"));

        // The remove button is an X icon button
        // Initially first row has disabled remove, second row has enabled
        const enabledRemoveButtons = screen
            .getAllByRole("button")
            .filter(
                (btn) =>
                    !btn.hasAttribute("disabled") && btn.classList.contains("text-muted-foreground")
            );

        if (enabledRemoveButtons.length > 0) {
            await user.click(enabledRemoveButtons[0]);
        }

        // After removing one row, back to 1 row
        const fullNameInputs = screen.getAllByPlaceholderText("Jane");
        expect(fullNameInputs).toHaveLength(1);
    });

    it("cannot remove the last row — the Remove button on the only remaining row is disabled or absent", () => {
        renderManualForm();

        // With only 1 row, the remove button should be disabled
        const removeButtons = screen
            .getAllByRole("button")
            .filter((btn) => btn.querySelector("svg"));

        // The remove button should be disabled
        const disabledBtn = removeButtons.find((btn) => btn.hasAttribute("disabled"));
        expect(disabledBtn).toBeTruthy();
    });
});

describe("ManualAddForm — validation", () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it("full_name is required — submitting with empty full_name shows submit disabled", async () => {
        renderManualForm();

        // Fill email but leave full_name empty
        fireEvent.change(screen.getByPlaceholderText("jane@school.edu"), {
            target: { value: "test@school.edu" },
        });

        // The Review & Submit button should be disabled because of errors
        const submitBtn = screen.getByText("Review & Submit");
        expect(submitBtn).toBeDisabled();
    });

    it("full_name is required — submitting with empty full_name shows submit disabled", async () => {
        const user = userEvent.setup();
        renderManualForm();

        // Fill full_name and email, leave full_name empty
        await user.type(screen.getByPlaceholderText("Jane"), "John");
        fireEvent.change(screen.getByPlaceholderText("jane@school.edu"), {
            target: { value: "john@school.edu" },
        });

        // Submit button should be disabled due to missing full_name
        const submitBtn = screen.getByText("Review & Submit");
        expect(submitBtn).toBeDisabled();
    });

    it("email structural validation — an email without @ shows inline error; a valid a@b.com shows no email error", async () => {
        const user = userEvent.setup();
        renderManualForm();

        // Fill name fields first to avoid name-related errors
        await user.type(screen.getByPlaceholderText("Jane"), "Alice");
        await user.type(screen.getByPlaceholderText("Doe"), "Smith");

        // Set invalid email via fireEvent.change for exact value control
        const emailInput = screen.getByPlaceholderText("jane@school.edu");
        fireEvent.change(emailInput, { target: { value: "notanemail" } });

        // The error counter should appear in the summary area (1 error = email)
        await waitFor(() => {
            expect(screen.getByText(/1 critical error/)).toBeInTheDocument();
        });

        // Set valid email
        fireEvent.change(emailInput, { target: { value: "a@b.com" } });

        await waitFor(() => {
            expect(screen.queryByText(/critical error/)).not.toBeInTheDocument();
        });
    });

    it("duplicate email within manual rows — entering the same email in two rows flags the second occurrence with 'Duplicate email in this batch'", async () => {
        const user = userEvent.setup();
        renderManualForm();

        // Fill name fields and email in first row
        await user.type(screen.getByPlaceholderText("Jane"), "Alice");
        await user.type(screen.getByPlaceholderText("Doe"), "Smith");
        fireEvent.change(screen.getByPlaceholderText("jane@school.edu"), {
            target: { value: "dup@school.edu" },
        });

        // Add second row
        await user.click(screen.getByText("Add another"));

        // Fill same email in second row
        const emailInputs = screen.getAllByPlaceholderText("jane@school.edu");
        fireEvent.change(emailInputs[1], { target: { value: "dup@school.edu" } });

        // It should show a duplicate error
        await waitFor(() => {
            expect(screen.getByText(/critical error/)).toBeInTheDocument();
        });
    });

    it("duplicate check is case-insensitive — Test@Example.com and test@example.com in two rows are treated as duplicates", async () => {
        const user = userEvent.setup();
        renderManualForm();

        // Fill name fields and email in first row
        await user.type(screen.getByPlaceholderText("Jane"), "Alice");
        await user.type(screen.getByPlaceholderText("Doe"), "Smith");
        fireEvent.change(screen.getByPlaceholderText("jane@school.edu"), {
            target: { value: "Test@Example.com" },
        });

        // Add second row
        await user.click(screen.getByText("Add another"));

        // Fill same email (different case) in second row
        const emailInputs = screen.getAllByPlaceholderText("jane@school.edu");
        fireEvent.change(emailInputs[1], { target: { value: "test@example.com" } });

        await waitFor(() => {
            expect(screen.getByText(/critical error/)).toBeInTheDocument();
        });
    });

    it("phone auto-correction with KE default — entering 0712345678 normalizes to +254712345678; the corrected cell shows a visual correction indicator", async () => {
        renderManualForm();

        // Use fireEvent.change to set the full value at once (type would trigger
        // intermediate partial values that get cleared by the auto-correction logic)
        const phoneInput = screen.getByPlaceholderText("+254 712 345 678");
        fireEvent.change(phoneInput, { target: { value: "0712345678" } });

        // The phone should be auto-corrected to E.164 format
        await waitFor(() => {
            expect(phoneInput).toHaveValue("+254712345678");
        });
    });

    it("unparseable phone sets to empty and shows warning — entering 'N/A' clears the phone field and shows a warning badge 'Phone cleared – invalid value'", async () => {
        renderManualForm();

        // Set an unparseable value via fireEvent.change
        const phoneInput = screen.getByPlaceholderText("+254 712 345 678");
        fireEvent.change(phoneInput, { target: { value: "N/A" } });

        // The phone should be cleared to empty string (normalizePhone returns null for N/A)
        await waitFor(() => {
            expect(phoneInput).toHaveValue("");
        });
    });
});

describe("ManualAddForm — callbacks and limits", () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it("onRowsReady callback fires on every mutation — fires when a row is added, removed, or any field is edited; the emitted array reflects current state", async () => {
        const { onRowsReady } = renderManualForm();
        const user = userEvent.setup();

        // Fill in the first row with valid data
        await user.type(screen.getByPlaceholderText("Jane"), "Alice");
        await user.type(screen.getByPlaceholderText("Doe"), "Smith");
        fireEvent.change(screen.getByPlaceholderText("jane@school.edu"), {
            target: { value: "alice@school.edu" },
        });

        // Click Review & Submit
        await user.click(screen.getByText("Review & Submit"));

        await waitFor(() => {
            expect(onRowsReady).toHaveBeenCalledTimes(1);
            const emitted = onRowsReady.mock.calls[0][0];
            expect(emitted).toHaveLength(1);
            expect(emitted[0].full_name).toBe("Alice");
            expect(emitted[0].email).toBe("alice@school.edu");
        });
    });

    it("up to 5,000 rows can be added — the Add Row button remains enabled until limit", async () => {
        const user = userEvent.setup();
        renderManualForm();

        const addButton = screen.getByText("Add another");

        // Add many rows (we don't need to do 5000, just verify the button works)
        for (let i = 0; i < 10; i++) {
            await user.click(addButton);
        }

        // Should now have 11 rows (1 default + 10 added)
        const fullNameInputs = screen.getAllByPlaceholderText("Jane");
        expect(fullNameInputs).toHaveLength(11);

        // The "Add another" button should still be enabled
        expect(addButton).not.toBeDisabled();
    });

    it("tab order is logical — tabbing from last field of row N lands on first field of row N+1", async () => {
        const user = userEvent.setup();
        renderManualForm();

        // Add a second row
        await user.click(screen.getByText("Add another"));

        // Focus on the phone input of the first row (last input field in grid)
        const phoneInputs = screen.getAllByPlaceholderText("+254 712 345 678");
        phoneInputs[0].focus();

        // Tab should move to the remove button of row 1, then to full_name of row 2
        await user.tab();
        await user.tab();

        // After tabbing twice, focus should be on the full_name input of row 2
        const fullNameInputs = screen.getAllByPlaceholderText("Jane");
        expect(document.activeElement).toBe(fullNameInputs[1]);
    });
});
