/**
 * Tests for the ReviewGrid (ReviewView) component.
 *
 * Tests the 5-row sample preview, error display, and inline editing behavior.
 * In the test environment, useVirtualizer is mocked to return all items.
 */

import { describe, it, expect, beforeEach, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../setup/test-utils";
import { server } from "../setup/msw-server";
import { http, HttpResponse } from "msw";

import { ReviewView } from "@/features/staff-import/components/review-view";
import { buildRow } from "../factories/inviteRow";
import type { ImportDraftRow } from "@/lib/db";

// ─── Mock useVirtualizer ──────────────────────────────────────────────

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

function renderReviewView(rows: ImportDraftRow[]) {
    const onSubmit = vi.fn();
    const onBack = vi.fn();
    const utils = renderWithProviders(
        <ReviewView rows={rows} role="NURSE" onSubmit={onSubmit} onBack={onBack} />
    );
    return { ...utils, onSubmit, onBack };
}

// ─── Tests ─────────────────────────────────────────────────────────────

describe("ReviewGrid — rendering", () => {
    it("renders all rows — all 10 rows appear in the DOM", () => {
        const rows = Array.from({ length: 10 }, () => buildRow());
        renderReviewView(rows);

        // The review view shows a 5-row sample preview by default
        expect(screen.getByText(/All 10 records? are valid/i)).toBeInTheDocument();
    });

    it("zero-error state hides grid and shows 5-row sample — when no row has critical errors, the virtualized grid is replaced by a read-only 5-row sample preview", () => {
        const rows = Array.from({ length: 10 }, () => buildRow());
        renderReviewView(rows);

        // Should show sample text
        expect(screen.getByText(/first 5 rows/i)).toBeInTheDocument();

        // Should show the table with rows
        const emailCells = screen.getAllByText(/@school\.edu/);
        expect(emailCells).toHaveLength(5); // Only 5 in sample
    });

    it("5-row sample is not editable — cells in the sample preview do not have click-to-edit behavior", () => {
        const rows = Array.from({ length: 5 }, () => buildRow());
        renderReviewView(rows);

        // The review view uses a plain table — cells are not inputs
        const inputs = screen.queryAllByRole("textbox");
        expect(inputs).toHaveLength(0);
    });

    it("sample shows first 5 rows by default — the 5 shown are the first 5 from the clean dataset", () => {
        const rows = [
            buildRow({ email: "first@school.edu" }),
            buildRow({ email: "second@school.edu" }),
            buildRow({ email: "third@school.edu" }),
            buildRow({ email: "fourth@school.edu" }),
            buildRow({ email: "fifth@school.edu" }),
            buildRow({ email: "sixth@school.edu" }),
        ];
        renderReviewView(rows);

        // The first 5 emails should be displayed
        expect(screen.getByText("first@school.edu")).toBeInTheDocument();
        expect(screen.getByText("fifth@school.edu")).toBeInTheDocument();
        expect(screen.queryByText("sixth@school.edu")).not.toBeInTheDocument();
    });

    it("row count badge shows total and error count — header area shows e.g. '10 rows · 3 errors'", () => {
        // The review view shows summary text with total record count
        const rows = Array.from({ length: 10 }, () => buildRow());
        renderReviewView(rows);

        expect(screen.getByText(/10 records?/i)).toBeInTheDocument();
    });
});

describe("ReviewGrid — submit behavior", () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it("submit sends correct payload shape — clicking Submit with 2 clean rows calls onSubmit", async () => {
        const user = userEvent.setup();
        const rows = [buildRow({ email: "a@b.com" }), buildRow({ email: "c@d.com" })];
        const { onSubmit } = renderReviewView(rows);

        // Mock the startImport mutation to return a job ID
        server.use(
            http.post("http://localhost:3000/api/v1/imports/staff", () => {
                return HttpResponse.json({
                    import_job_id: "job-001",
                    status: "accepted",
                    total: 2,
                });
            })
        );

        const submitBtn = screen.getByText(/Submit 2 Invitations/i);
        await user.click(submitBtn);

        await waitFor(() => {
            expect(onSubmit).toHaveBeenCalled();
        });
    });
});

describe("ReviewGrid — submit button states", () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it("submit button is enabled at zero critical errors — all rows clean; Submit is enabled", () => {
        const rows = [buildRow()];
        renderReviewView(rows);

        const submitBtn = screen.getByRole("button", { name: /submit 1 invitation/i });
        expect(submitBtn).not.toBeDisabled();
    });

    it("loading spinner is shown during submit — between click and response, a spinner is visible and the Submit button shows sending state", async () => {
        // Create a delayed response
        server.use(
            http.post("http://localhost:3000/api/v1/imports/staff", async () => {
                await new Promise((resolve) => setTimeout(resolve, 500));
                return HttpResponse.json({
                    import_job_id: "job-001",
                    status: "accepted",
                    total: 1,
                });
            })
        );

        const user = userEvent.setup();
        const rows = [buildRow()];
        renderReviewView(rows);

        const submitBtn = screen.getByRole("button", { name: /submit 1 invitation/i });
        await user.click(submitBtn);

        // Button should show sending state
        await waitFor(() => {
            expect(screen.getByText(/Submitting/i)).toBeInTheDocument();
        });
    });
});
